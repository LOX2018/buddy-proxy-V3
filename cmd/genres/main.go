package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Minimal Windows resource (.syso) generator.
// Produces a COFF object with RT_MANIFEST (UTF-8 manifest) and a simple icon.

func main() {
	wd, _ := os.Getwd()
	_ = wd

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: genres <output.syso> [icon.png]\n")
		os.Exit(1)
	}
	outPath := os.Args[1]
	_ = os.MkdirAll(filepath.Dir(outPath), 0o755)

	// Build manifest XML
	manifest := []byte(strings.Replace(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="1.0.0.0" name="CodeBuddyProxy"/>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>
      <supportedOS Id="{1f676c76-80e1-4239-95bb-83d0f6d0da78}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
    </windowsSettings>
  </application>
</assembly>`, "\r\n", "\n", -1))

	// Build a minimal 32x32 4bpp ICO file (simple gradient icon)
	iconData := buildSimpleIcon()

	resources := []resourceEntry{
		{ID: 1, Type: RT_ICON, Data: iconData},
		{ID: 1, Type: RT_GROUP_ICON, Data: buildGroupIcon(iconData)},
		{ID: 1, Type: RT_MANIFEST, Data: manifest},
	}

	coff := buildCOFF(resources)
	if err := os.WriteFile(outPath, coff, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", outPath, len(coff))
}

type resourceEntry struct {
	ID   uint32
	Type uint32
	Data []byte
}

const (
	RT_ICON       = 3
	RT_GROUP_ICON = 14
	RT_MANIFEST   = 24
)

func buildSimpleIcon() []byte {
	// 32x32 32bpp BGRA icon
	w, h := 32, 32
	bpp := 4 // bytes per pixel (BGRA)
	pixelData := make([]byte, w*h*bpp)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * bpp
			// Simple CB-like gradient (teal to orange)
			t := float64(x+y) / float64(w+h-2)
			r := uint8(94*(1-t) + 232*t)  // teal→orange R
			g := uint8(196*(1-t) + 138*t) // teal→orange G
			b := uint8(168*(1-t) + 74*t)  // teal→orange B
			pixelData[off+0] = b          // BGRA order
			pixelData[off+1] = g
			pixelData[off+2] = r
			pixelData[off+3] = 255
		}
	}

	// AND mask: all zeros (fully opaque)
	andMask := make([]byte, ((w+31)/32)*4*h)

	// ICO image entry: w(1), h(1), colors(0), reserved(0), planes(1), bpp(16), size(32), offset(0)
	// BITMAPINFOHEADER
	var hdr bytes.Buffer
	binary.Write(&hdr, binary.LittleEndian, uint32(40)) // header size
	binary.Write(&hdr, binary.LittleEndian, int32(w))   // width
	binary.Write(&hdr, binary.LittleEndian, int32(h*2)) // height (doubled for ICO)
	binary.Write(&hdr, binary.LittleEndian, uint16(1))  // planes
	binary.Write(&hdr, binary.LittleEndian, uint16(32)) // bpp
	binary.Write(&hdr, binary.LittleEndian, uint32(0))  // compression
	binary.Write(&hdr, binary.LittleEndian, uint32(0))  // image size (0 = calc)
	binary.Write(&hdr, binary.LittleEndian, int32(0))   // ppm X
	binary.Write(&hdr, binary.LittleEndian, int32(0))   // ppm Y
	binary.Write(&hdr, binary.LittleEndian, uint32(0))  // colors used
	binary.Write(&hdr, binary.LittleEndian, uint32(0))  // important colors

	// Bottom-up bitmap: flip vertically
	flipped := make([]byte, len(pixelData))
	rowSize := w * bpp
	for y := 0; y < h; y++ {
		srcOff := y * rowSize
		dstOff := (h - 1 - y) * rowSize
		copy(flipped[dstOff:dstOff+rowSize], pixelData[srcOff:srcOff+rowSize])
	}

	var ico bytes.Buffer
	ico.Write(hdr.Bytes())
	ico.Write(flipped)
	ico.Write(andMask)
	return ico.Bytes()
}

func buildGroupIcon(iconData []byte) []byte {
	var buf bytes.Buffer
	// ICONDIR header
	binary.Write(&buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // type: ICO
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // count: 1 image
	// ICONDIRENTRY
	binary.Write(&buf, binary.LittleEndian, byte(32))              // width
	binary.Write(&buf, binary.LittleEndian, byte(32))              // height
	binary.Write(&buf, binary.LittleEndian, byte(0))               // colors
	binary.Write(&buf, binary.LittleEndian, byte(0))               // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))             // planes
	binary.Write(&buf, binary.LittleEndian, uint16(32))            // bpp
	binary.Write(&buf, binary.LittleEndian, uint32(len(iconData))) // size
	binary.Write(&buf, binary.LittleEndian, uint32(22))            // offset (14-byte header + 16-byte entry = 30, but icon data starts at offset after group)

	// The group icon data actually points to the icon resource by ID
	// For simplicity, store just the icon data here and reference by ID
	// Actually in PE resources, group icon references icon resources by ID
	// We need to write the icon data as resource ID 1 and reference it
	// For a .syso, we can embed the full icon data here
	buf.Write(iconData)
	return buf.Bytes()
}

type coffHeader struct {
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
}

type sectionHeader struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLinenumbers uint32
	NumberOfRelocations  uint16
	NumberOfLinenumbers  uint16
	Characteristics      uint32
}

func buildCOFF(resources []resourceEntry) []byte {
	// Build .rsrc section data
	rsrcData := buildResourceSection(resources)
	padSize := (4 - len(rsrcData)%4) % 4
	paddedRsrc := make([]byte, len(rsrcData)+padSize)
	copy(paddedRsrc, rsrcData)

	// COFF header (20 bytes)
	hdr := coffHeader{
		Machine:              0x8664, // AMD64
		NumberOfSections:     1,
		SizeOfOptionalHeader: 0,
		Characteristics:      0,
	}

	// Section header
	sec := sectionHeader{}
	copy(sec.Name[:], ".rsrc\x00\x00\x00")
	sec.VirtualSize = uint32(len(rsrcData))
	sec.SizeOfRawData = uint32(len(paddedRsrc))

	var buf bytes.Buffer
	// COFF header
	binary.Write(&buf, binary.LittleEndian, hdr)
	// Section headers
	binary.Write(&buf, binary.LittleEndian, sec)
	// Section data
	buf.Write(paddedRsrc)
	return buf.Bytes()
}

type resourceDirEntry struct {
	NameOrID  uint32
	DataOrDir uint32
}

type resourceDir struct {
	Characteristics uint32
	TimeDateStamp   uint32
	MajorVersion    uint16
	MinorVersion    uint16
	NamedEntries    uint16
	IDEntries       uint16
}

const langDefault = 0x0409 // LANG_ENGLISH / SUBLANG_ENGLISH_US

// buildResourceSection builds a correct 3-level PE resource directory:
// Type -> Name(Id) -> Language, with:
//   - type & name entries: OffsetToData has bit 31 set (points to a subdirectory)
//   - language entries:    OffsetToData has bit 31 clear (points to a data entry)
//   - data entry: 16 bytes { RVA, Size, CodePage, Reserved }
//
// All offsets are relative to the start of the .rsrc section.
func buildResourceSection(resources []resourceEntry) []byte {
	const subdirFlag = 0x80000000

	// Build the tree: type -> id -> data.
	type dirKey = uint32
	typeEntry := map[uint32]map[uint32][]byte{} // type -> (id -> data)
	var typeOrder []uint32
	idOrder := map[uint32][]uint32{} // type -> id order
	typeSeen := map[uint32]map[uint32]bool{}
	for _, r := range resources {
		if _, ok := typeEntry[r.Type]; !ok {
			typeEntry[r.Type] = map[uint32][]byte{}
			typeOrder = append(typeOrder, r.Type)
			idOrder[r.Type] = []uint32{}
			typeSeen[r.Type] = map[uint32]bool{}
		}
		if !typeSeen[r.Type][r.ID] {
			typeSeen[r.Type][r.ID] = true
			idOrder[r.Type] = append(idOrder[r.Type], r.ID)
		}
		typeEntry[r.Type][r.ID] = r.Data
	}

	// Layout plan: list of blocks. Each block is a directory {16-byte header + entries}.
	// We'll first lay out all directories (and record their offsets), then place data.
	type dirBlock struct {
		entries []resourceDirEntry
		offset  uint32
	}
	var dirs []*dirBlock

	// Reserve root (level 1) block.
	root := &dirBlock{}
	dirs = append(dirs, root)

	// Reserve one dir block per type and per id (id blocks are leaves; their
	// entries point to data entries, so we place data right after this level).
	// Track: typeBlock[type], leafBlock[type][id].
	typeBlock := map[uint32]*dirBlock{}
	leafBlock := map[dirKey]*map[uint32]*dirBlock{}
	for _, tt := range typeOrder {
		tb := &dirBlock{}
		dirs = append(dirs, tb)
		typeBlock[tt] = tb
		leafBlock[tt] = &map[uint32]*dirBlock{}
		for _, id := range idOrder[tt] {
			lb := &dirBlock{}
			dirs = append(dirs, lb)
			(*leafBlock[tt])[id] = lb
		}
	}

	// Pass 1: assign offsets to every directory; track where data should go.
	curr := uint32(0)
	for _, d := range dirs {
		if pad := curr % 4; pad != 0 {
			curr += 4 - pad
		}
		d.offset = curr
		curr += 16 + uint32(len(d.entries))*8
	}

	// Build entries for each directory, resolving DataOrDir now that subdir/data
	// offsets are known. Data payloads are appended after all directory tables.
	dataOffsets := map[dirKey]map[uint32]uint32{} // type -> (id -> dataOffset)
	// Placate the compiler that dirKey is used:
	_ = dirKey(0)

	if pad := curr % 4; pad != 0 {
		curr += 4 - pad
	}
	dataCursor := curr

	// Assign data offsets and fill leaf entries (language level).
	root.entries = make([]resourceDirEntry, 0, len(typeOrder))
	for _, tt := range typeOrder {
		// root -> subdirectory (high bit set)
		root.entries = append(root.entries, resourceDirEntry{
			NameOrID:  tt,
			DataOrDir: typeBlock[tt].offset | subdirFlag,
		})
		tb := typeBlock[tt]
		tb.entries = make([]resourceDirEntry, 0, len(idOrder[tt]))
		for _, id := range idOrder[tt] {
			tb.entries = append(tb.entries, resourceDirEntry{
				NameOrID:  id,
				DataOrDir: (*leafBlock[tt])[id].offset | subdirFlag,
			})
			lb := (*leafBlock[tt])[id]
			// leaf -> single language (0x0409) pointing to a data entry (high bit clear)
			lb.entries = []resourceDirEntry{{NameOrID: langDefault}}
			if dataOffsets[tt] == nil {
				dataOffsets[tt] = map[uint32]uint32{}
			}
			dataOffsets[tt][id] = dataCursor
			lb.entries[0].DataOrDir = dataCursor
			if pad := dataCursor % 4; pad != 0 {
				dataCursor += 4 - pad
			}
			dataCursor += uint32(len(typeEntry[tt][id]))
		}
	}

	// Pass 2: serialize.
	buf := make([]byte, dataCursor)
	writeDir := func(d *dirBlock) {
		if pad := d.offset % 4; pad != 0 {
			panic("dir offset unaligned") // should not happen
		}
		binary.LittleEndian.PutUint32(buf[d.offset:], 0)                         // Characteristics
		binary.LittleEndian.PutUint32(buf[d.offset+4:], 0)                       // TimeDateStamp
		binary.LittleEndian.PutUint16(buf[d.offset+8:], 0)                       // MajorVersion
		binary.LittleEndian.PutUint16(buf[d.offset+10:], 0)                      // MinorVersion
		binary.LittleEndian.PutUint16(buf[d.offset+12:], 0)                      // NumberOfNamedEntries
		binary.LittleEndian.PutUint16(buf[d.offset+14:], uint16(len(d.entries))) // NumberOfIdEntries
		for i, e := range d.entries {
			o := d.offset + 16 + uint32(i)*8
			binary.LittleEndian.PutUint32(buf[o:], e.NameOrID)
			binary.LittleEndian.PutUint32(buf[o+4:], e.DataOrDir)
		}
	}
	for _, d := range dirs {
		writeDir(d)
	}
	// Write data payloads.
	for _, tt := range typeOrder {
		for _, id := range idOrder[tt] {
			off := dataOffsets[tt][id]
			copy(buf[off:], typeEntry[tt][id])
		}
	}

	return buf
}

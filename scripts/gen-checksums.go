package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	dir := "releases"
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir: %v\n", err)
		os.Exit(1)
	}
	type entry struct{ name, hash string }
	var items []entry
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasPrefix(e.Name(), "codebuddy-proxy-gui-windows-x64-") || !strings.HasSuffix(e.Name(), ".exe") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			os.Exit(1)
		}
		h := sha256.Sum256(data)
		items = append(items, entry{name: e.Name(), hash: fmt.Sprintf("%x", h)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	var out strings.Builder
	for _, it := range items {
		fmt.Fprintf(&out, "%s  %s\n", it.hash, it.name)
	}
	checksumsPath := filepath.Join(dir, "SHA256SUMS.txt")
	if err := os.WriteFile(checksumsPath, []byte(out.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write checksums: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(out.String())
}

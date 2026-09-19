//go:build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/tray"
)

func main() {
	app := tray.New("CodeBuddy Proxy TRAY-TEST")
	app.Open = func() { fmt.Println(">>> left-click: open console") }
	app.Tip = "CodeBuddy Proxy TRAY-TEST"
	if err := app.Start(); err != nil {
		fmt.Println("TRAY START FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("tray Start() returned OK - icon should be in notification area")
	fmt.Println("waiting 10s on message pump... (check tray now)")
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("done, exiting")
		os.Exit(0)
	}()
	app.Loop()
}

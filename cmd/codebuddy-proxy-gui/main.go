//go:build windows

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/config"
	"github.com/wnddd839/codebuddy-proxy/internal/gateway"
	"github.com/wnddd839/codebuddy-proxy/internal/openprivate"
	"github.com/wnddd839/codebuddy-proxy/internal/server"
	"github.com/wnddd839/codebuddy-proxy/internal/tray"
)

func main() {
	hideConsoleWindow()
	cfg := config.Load()

	if strings.TrimSpace(cfg.APIKey) == "" && len(cfg.APIKeys) == 0 {
		if key, err := server.GenerateProxyAPIKey(); err == nil {
			envPath := config.ResolveEnvFilePath()
			_ = config.UpsertEnvFile(envPath, map[string]string{
				"CODEBUDDY_PROXY_API_KEY":         key,
				"CODEBUDDY_PROXY_REQUIRE_API_KEY": "true",
			})
			cfg.APIKey = key
			cfg.RequireAPIKey = true
		}
	}

	config.ScrubSecretEnv()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if !cfg.RequireAPIKey {
		logger.Warn("api key enforcement disabled: /v1 is open to any client that can reach this machine", "addr", cfg.Addr(), "fix", "set CODEBUDDY_PROXY_REQUIRE_API_KEY=true")
	}
	if cfg.AdminPassword == "" {
		logger.Warn("admin console has no password: anyone who can reach the admin URL can manage the pool", "admin", adminURLFor(cfg), "fix", "set CODEBUDDY_PROXY_ADMIN_PASSWORD")
	}

	svc := gateway.New(cfg, logger)
	srv := server.New(cfg, svc)
	srv.LogActivity("server-start", map[string]any{
		"mode":  "gui-tray",
		"addr":  cfg.Addr(),
		"admin": adminURLFor(cfg),
	})

	adminURL := adminURLFor(cfg)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("server exited", "error", err)
			srv.LogActivity("server-error", map[string]any{"event": "server-exit", "error": err.Error()})
		}
	}()

	app := tray.New("CodeBuddy Proxy")
	app.Open = func() { tray.OpenURL(adminURL) }
	app.OpenURL = adminURL
	app.OpenConfigDir = func() {
		dir, dirErr := openprivate.ConfigDir()
		if dirErr != nil {
			logger.Warn("locate config dir failed", "error", dirErr)
			return
		}
		if openErr := openprivate.OpenDirectory(dir); openErr != nil {
			logger.Warn("open config dir failed", "error", openErr)
			return
		}
	}
	app.Quit = func() {
		srv.LogActivity("server-stop", map[string]any{"mode": "gui-tray"})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := srv.Shutdown(ctx); shutdownErr != nil {
			logger.Error("shutdown error", "error", shutdownErr)
		}
	}

	// 接收系统关闭信号（如关机/注销前）。
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		app.RequestQuit()
	}()

	if err := app.Start(); err != nil {
		logger.Error("tray start failed", "error", err)
		select {}
	}
	app.Notify("CodeBuddy Proxy 已启动", "服务运行中，托盘图标常驻。\n"+"管理台: "+adminURL)
	tray.OpenURL(adminURL)
	app.Loop()
	app.Cleanup()
}

func adminURLFor(cfg config.Config) string {
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/direct-admin/"
	}
	return "http://" + cfg.Addr() + "/direct-admin/"
}

// hideConsoleWindow 隐藏宿主控制台窗口（SW_HIDE）。GUI 版按 GUI 子系统
// 构建时本无控制台，GetConsoleWindow 返回 0，调用为空操作；若误用控制台
// 子系统构建，则启动瞬间就会被隐藏，避免常驻黑窗口。
func hideConsoleWindow() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		const swHide = 0
		_, _, _ = showWindow.Call(hwnd, swHide)
	}
}

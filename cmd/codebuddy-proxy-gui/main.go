//go:build windows

package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wnddd839/codebuddy-proxy/internal/config"
	"github.com/wnddd839/codebuddy-proxy/internal/gateway"
	"github.com/wnddd839/codebuddy-proxy/internal/openprivate"
	"github.com/wnddd839/codebuddy-proxy/internal/server"
	"github.com/wnddd839/codebuddy-proxy/internal/singleinstance"
	"github.com/wnddd839/codebuddy-proxy/internal/tray"
)

const singletonName = "CodeBuddyProxySingleton"

func main() {
	hideConsoleWindow()

	// 单实例锁：命名互斥量跨进程唯一，第二次启动直接弹窗提示退出，
	// 避免"有托盘的进程被抢端口后静默残留、无托盘进程无法关闭"的僵尸态。
	m, err := singleinstance.Acquire(singletonName)
	if errors.Is(err, singleinstance.ErrAlreadyRunning) {
		singleinstance.Popup(
			"CodeBuddy Proxy 已在运行",
			"检测到已有 CodeBuddy Proxy 进程在运行，请在系统托盘右键打开管理台；本次启动已退出。",
			singleinstance.MB_ICONWARNING,
		)
		return
	}
	if err != nil {
		singleinstance.Popup(
			"CodeBuddy Proxy 启动失败",
			"获取单实例锁失败："+err.Error(),
			singleinstance.MB_ICONWARNING,
		)
		return
	}
	defer m.Release()

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

	// 先启动托盘，再放服务上线：这样端口冲突等启动错误发生时窗口已就绪，
	// 可以通过 RequestQuit 平滑结束消息循环，不会留下"僵尸进程 + 无托盘"。
	if err := app.Start(); err != nil {
		logger.Error("tray start failed", "error", err)
		srv.LogActivity("server-error", map[string]any{"event": "tray-start-failed", "error": err.Error()})
		singleinstance.Popup(
			"CodeBuddy Proxy 托盘启动失败",
			"托盘图标创建失败："+err.Error()+"\n服务已停止，请重启程序后再试。",
			singleinstance.MB_ICONWARNING,
		)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		return
	}

	// 服务启动失败（端口占用、bind 错误等）时请求托盘退出，而不是静默挂死；
	// 失败原因通过弹窗告知用户，避免"图标还在但接口已经废了"的隐形状态。
	// 用 atomic.Value 传递错误，避免 goroutine 写 / 主协程读之间的 data race。
	var serverErr atomic.Value
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			serverErr.Store(err)
			logger.Error("server exited", "error", err)
			srv.LogActivity("server-error", map[string]any{"event": "server-exit", "error": err.Error()})
			app.RequestQuit()
		}
	}()

	// 接收系统关闭信号（如关机/注销前）。
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		app.RequestQuit()
	}()

	app.Notify("CodeBuddy Proxy 已启动", "服务运行中，托盘图标常驻。\n"+"管理台: "+adminURL)
	tray.OpenURL(adminURL)
	app.Loop()
	app.Cleanup()

	if v := serverErr.Load(); v != nil {
		singleinstance.Popup(
			"CodeBuddy Proxy 服务启动失败",
			"HTTP 服务未能启动："+v.(error).Error()+"\n可能是端口被其他程序占用，或配置有误。",
			singleinstance.MB_ICONWARNING,
		)
	}
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

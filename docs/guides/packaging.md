# 打包说明

本项目只发布 **Windows 托盘版**（GUI 子系统，系统托盘常驻）。控制台版与 Linux / macOS 版不再打包。

## 前置条件

| 项 | 要求 |
|----|------|
| 操作系统 | Windows（`cmd/codebuddy-proxy-gui` 带 `//go:build windows`，无法在其他平台编译） |
| Go | ≥ 1.26（`go.mod` 约束） |
| make | 可选；Windows 上也可直接用下面的 `go build` 命令 |

## 一键打包

```powershell
# 方式 A：make（需安装 make）
make release VERSION=v5.0

# 方式 B：纯 go 命令（Windows PowerShell，无需 make）
$VERSION = "v5.0"  # 发布时改成实际版本号
New-Item -ItemType Directory -Path releases -Force
go build -trimpath `
  -ldflags="-s -w -X github.com/wnddd839/codebuddy-proxy/internal/version.Version=$VERSION -H=windowsgui" `
  -o "releases/codebuddy-proxy-gui-windows-x64-$VERSION.exe" `
  ./cmd/codebuddy-proxy-gui
Copy-Item .env.example releases/.env.example
Get-FileHash -Algorithm SHA256 releases/codebuddy-proxy-* |
  ForEach-Object { "$($_.Hash.ToLower())  $([System.IO.Path]::GetFileName($_.Path))" } |
  Set-Content releases/SHA256SUMS.txt
```

## 关键构建标志

| 标志 | 作用 | 不能省略的原因 |
|------|------|----------------|
| `-H=windowsgui` | 链接为 GUI 子系统 | 省略后启动会常驻一个 CMD 黑窗口 |
| `-trimpath` | 去除构建机路径 | 产物可复现 |
| `-s -w` | 去除符号表与 DWARF | 减小体积 |
| `-X ...version.Version=dev` | 注入版本号 | 发布时把 `dev` 改成实际版本号 |

> **不要**对托盘版设置 `CGO_ENABLED=0` 以外的 `GOOS` / `GOARCH`。
> `//go:build windows` 约束意味着 `GOOS` 必须是 `windows`；改 `GOARCH` 会导致在目标机器上出现「平台不支持」类错误。

## 产物

| 文件 | 说明 |
|------|------|
| `releases/codebuddy-proxy-gui-windows-x64-$(VERSION).exe` | Windows 托盘版，双击启动（`$(VERSION)` 为发布版本号，如 `v5.0`） |
| `releases/.env.example` | 配置模板 |
| `releases/SHA256SUMS.txt` | 产物校验和 |

## 验证

```powershell
# 启动后应在托盘看到图标，端口 32126 开始监听
.\releases\codebuddy-proxy-gui-windows-x64.exe
# 另开终端检查
Get-NetTCPConnection -LocalPort 32126 -State Listen
```

管理台地址：`http://127.0.0.1:32126/direct-admin/`

## 常见问题

**Q：运行时提示「平台不支持」 / 闪退**

检查构建命令是否满足以上全部标志，尤其 `-H=windowsgui` 与 `GOOS=windows`。重新执行 `make release` 或上面的纯 `go build` 命令即可。

**Q：双击后没有窗口**

托盘版不会弹窗，检查系统托盘（右下角）是否有图标，或访问 `http://127.0.0.1:32126/direct-admin/`。

**Q：端口 32126 被占用**

设置环境变量 `CODEBUDDY_PROXY_PORT` 改端口，或在 `.env` 中配置。

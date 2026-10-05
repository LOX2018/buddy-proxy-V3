## v0.5.6 · 2026-10-04 · 国际站授权改走 Edge 无痕窗口 · 加第二个国际站账号不再"消失"

### 下载哪个文件？

| 你的系统 | 下载 |
|----------|------|
| **Windows 64 位（托盘版，推荐）** | `codebuddy-proxy-gui-windows-x64.exe` |
| **Windows 64 位（命令行）** | `codebuddy-proxy-windows-x64.exe` |

本次只打 Windows 包；Linux / macOS 产物未构建（需要时用同一 `VERSION=v0.5.6` 跑 `make release`）。

校验：`SHA256SUMS.txt`

---

### 改了什么

- `site=global` 的 OAuth 授权由服务端拉起 **Edge 无痕窗口**（`msedge --inprivate`，Edge 缺失时退 `chrome --incognito`），管理台不再另开普通标签页，「启动登录」链接与 `/oauth/launch` 兜底路径同样走无痕窗口。
  原因：普通标签页带着浏览器里已登录的 CodeBuddy 会话，"再授权一个账号"实际是把同一个账号授权第二次；号池按「身份 + 站点」判重（`internal/accounts/pool.go` 的 `Upsert`），于是列表里不会多出第二行，看起来像新账号消失了。
  本机 Edge 只注册在 App Paths 注册表、不在 PATH 上，所以按 `LOCALAPPDATA` / `ProgramFiles` / `ProgramFiles(x86)` 扫描安装目录定位。打不开浏览器时退回原重定向，并把原因写进管理台提示；非 Windows 平台回"不支持"，由用户自己开无痕窗口。国内站行为不变。
- 想加第二个国际站账号，必须在无痕窗口里登录另一个邮箱；用同一身份再授权只会刷新那一行（换 token、保留 ID 与计数）。

本包同时包含上一版尚未发布的两项修复：上游产品主机默认直连、不再被机器上的 `HTTPS_PROXY` 劫持（新变量 `CODEBUDDY_PROXY_UPSTREAM_PROXY` 可显式指定），以及 GitHub 版本检查支持令牌、失败结论缓存由 6 小时缩短到 15 分钟。

### 升级

覆盖旧二进制后重启。账号池与用量文件不用迁移。管理台指纹：`cbp-ui-revision = 2026.10.04-oauth-inprivate`。

完整说明见 [`CHANGELOG.md`](https://github.com/wnddd839/buddy-proxy/blob/main/CHANGELOG.md)。

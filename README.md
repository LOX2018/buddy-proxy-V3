<p align="center">
  <img src="docs/logo.svg" width="96" height="96" alt="Buddy Proxy" />
</p>

<h1 align="center">Buddy Proxy V3</h1>

<p align="center">
  <strong>把你的 CodeBuddy / WorkBuddy 账号，变成任何 OpenAI 客户端都能直连的 <code>/v1</code> 渠道。</strong>
</p>

<p align="center">
  <a href="https://github.com/LOX2018/buddy-proxy-V3/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-BSD--3--Clause-1C1C1C?style=flat-square" alt="License" /></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-%E2%89%A51.26-1C1C1C?style=flat-square&logo=go&logoColor=white" alt="Go" /></a>
  <img src="https://img.shields.io/badge/Transport-protocol__direct-1C1C1C?style=flat-square" alt="protocol_direct" />
</p>

<p align="center">
  <a href="#-快速开始">快速开始</a> ·
  <a href="#-接入客户端">接入客户端</a> ·
  <a href="#-模型白名单">模型白名单</a> ·
  <a href="#-文档">文档</a> ·
  <a href="#%EF%B8%8F-免责声明与合规">免责声明</a>
</p>

---

## 为什么需要它

你有腾讯 CodeBuddy / WorkBuddy 账号，也有一堆只认 OpenAI `/v1` 格式的工具——NewAPI、ZCode、Sub2API、各类 SDK 和客户端。它们互相不认识。

**Buddy Proxy 是中间那一层协议翻译器。**

用你自己的账号（OAuth 登录）直连上游，对外暴露标准 OpenAI 接口。管理台可以切国内 / 国际，也可以切 CodeBuddy / WorkBuddy——同一套 token，产品和站点正交。不需要改客户端，不需要 `codebuddy --serve`，不需要碰任何浏览器插件。

```text
你的客户端  ──►  Buddy Proxy  ──►  CodeBuddy 或 WorkBuddy
(OpenAI格式)     (协议翻译/账号池)     (protocol_direct)
```

一个 Go 写的单文件二进制，跑在你自己的机器上。请求只从你自己的机器发往你所选的上游。

---

## 核心能力

| 能力 | 说明 |
| :--- | :--- |
| **协议直连** | OAuth 登录后直连上游，不依赖 `codebuddy --serve` 等本地中间进程 |
| **标准 OpenAI 形状** | `GET /v1/models` · `POST /v1/chat/completions` · `POST /v1/responses`（Responses API，Codex CLI 可直连），流式与非流式都支持 |
| **多账号调度** | 同会话钉在同一账号；钉号被禁用/删除会换号。新会话按额度快照选最大（缺快照或超过 5 分钟才探活）；失败换号前再探活拿最大。6004 按错误文案 `will reset at` 长冷却。凭据以 `0600` 权限落盘 |
| **真实余额** | 管理台直读官网 Credits，显示「剩余 / 总额」 |
| **国内 / 国际双区** | 同一进程可同时持有两区账号。默认区域可在管理台切换；单请求用 Key 绑定、`X-Site` 或 `cn:` / `global:` 前缀选区。**端点以账号自身区域为准** |
| **CodeBuddy / WorkBuddy** | 一键切产品：CodeBuddy 走 CLI 头，WorkBuddy 走 IDE 头；模型目录随之切换 |
| **模型列表** | 走协议 `/v3/config`，60 秒缓存，可强制刷新；**按付费倍率排序**，无 `creditMultiplier` 的模型自动隐藏 |
| **模型白名单** | `~/.codebuddy/proxy-modelpolicy.json`，`allow` / `deny` 两级过滤，支持**国内 / 国际分区独立覆盖**，请求与目录同时生效，改文件即时热加载 |
| **管理台** | 章节式单页：概览监控、账号池、用量明细、签到、模型目录、日志、设置，顶栏实时显示服务/上游/号池/产品状态 |
| **一键签到** | 查看每账号签到状态，支持批量一键签到 |
| **用量明细** | 按账号 / 模型可筛选的用量表，Token 与 credit 明细透传，含缓存命中统计 |
| **活动日志** | `~/.codebuddy/activity/activity.log`，JSONL 追加式，每日零点自动清空（超额自动轮转） |
| **Token 用量透传** | 流式收尾补 usage chunk，缓存字段兼容多上游别名 |
| **Windows 托盘版** | 系统托盘常驻，启动即自动打开管理台 |

---

## 🚀 快速开始

### 方式 A：下载即用（推荐）

从 [GitHub Releases](https://github.com/LOX2018/buddy-proxy-V3/releases/latest) 下载对应平台文件，直接运行。

| 形态 | 文件 |
|------|------|
| **Windows 桌面托盘版（推荐）** | `codebuddy-proxy-gui-windows-x64.exe` |
| **Windows 64 位（控制台）** | `codebuddy-proxy-windows-x64.exe` |
| Linux 64 位 | `codebuddy-proxy-linux-amd64` |
| macOS Apple 芯片 | `codebuddy-proxy-darwin-arm64` |
| macOS Intel | `codebuddy-proxy-darwin-amd64` |

```powershell
# Windows 托盘版：双击即启动，托盘图标常驻，自动打开管理台
.\codebuddy-proxy-gui-windows-x64.exe
```

```bash
# Linux / macOS
chmod +x ./codebuddy-proxy-linux-amd64
./codebuddy-proxy-linux-amd64
```

首次启动自动生成 API Key 并写入 `~/.codebuddy/proxy.env`，日志里也会打印。

### 方式 B：从源码

```bash
git clone https://github.com/LOX2018/buddy-proxy-V3.git
cd buddy-proxy-V3
go run ./cmd/codebuddy-proxy          # 控制台版
go run ./cmd/codebuddy-proxy-gui      # Windows 托盘版
```

### 三个入口

| | 地址 |
| :--- | :--- |
| API | `http://127.0.0.1:32126/v1` |
| 管理台 | `http://127.0.0.1:32126/direct-admin/` |
| 健康检查 | `http://127.0.0.1:32126/health`（存活）· `/readyz`（上游可达） |

### 三步跑起来

1. 打开管理台 → 选「国内」或「国际」、选「CodeBuddy」或「WorkBuddy」→ 点「开始 OAuth」，浏览器完成授权
2. 回管理台点「检查登录」，账号进入账号池
3. 复制页面上的 **Base URL + API Key**，填进你的客户端

管理台密码留空即为免密（本地推荐）。`/v1` 的 API Key 建议保持开启。

> 管理台「生成 API Key」会写入 `.env` 并**立即生效**——旧主 Key 当场失效，客户端必须同步更换。`CODEBUDDY_PROXY_API_KEYS` 里的绑定 Key 不受影响。

---

## 🔌 接入客户端

任何 OpenAI 兼容客户端都行：

```text
Base URL   http://<host>:32126/v1
API Key    管理台复制的那个
Model      auto  或 GET /v1/models 返回的 id
```

```bash
curl http://127.0.0.1:32126/v1/chat/completions \
  -H "Authorization: Bearer $CODEBUDDY_PROXY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","stream":true,"messages":[{"role":"user","content":"你好"}]}'
```

模型 `id` 不带 `codebuddy/` 前缀，但请求时 `codebuddy/auto` 和 `auto` 都接受。`GET /v1/models` 只返回带付费倍率、且通过白名单的模型；倍率决定排序。

> 只提供列表接口，**不支持** `GET /v1/models/{id}` 单模型查询（返回 404）。

### Codex CLI（Responses API）

`POST /v1/responses` 已按 OpenAI Responses API 实现，Codex CLI 可直连：

```toml
# ~/.codex/config.toml
model_provider = "buddy-proxy"
model = "auto"

[model_providers.buddy-proxy]
name = "Buddy Proxy"
base_url = "http://127.0.0.1:32126/v1"
env_key = "CODEBUDDY_PROXY_API_KEY"
wire_api = "responses"
```

支持 `input` item 数组 / `instructions` / function 工具调用 / `reasoning.effort`；流式为命名 SSE 事件（`response.output_text.delta` 等）。`background`、服务端存储（`store` / `previous_response_id`）与内置工具（`web_search` 等）不支持，详见 [HTTP API](docs/api/http.md)。

### 站点与产品配置

国内 / 国际、CodeBuddy / WorkBuddy 切换只需改 `.env`（或在管理台点切换）：

```env
# 站点
CODEBUDDY_SITE=domestic          # 或 global
CODEBUDDY_INTERNET_ENVIRONMENT=internal   # 国际用 public

# 产品（与站点正交；同一套 token）
CODEBUDDY_PRODUCT=codebuddy      # 或 workbuddy

# 可选：一把 Key 绑一个区域。ZCode 等客户端换 Key 等于换区，
# GET /v1/models 只返回该区无前缀目录，不再出现 cn: / global: 三份。
# CODEBUDDY_PROXY_API_KEYS=cbp_aaa:global,cbp_bbb:domestic
```

完整变量见 [配置参考](docs/guides/configuration.md)。

---

## 🗂️ 模型白名单

策略文件 `~/.codebuddy/proxy-modelpolicy.json`（可用环境变量 `CODEBUDDY_PROXY_MODELPOLICY_PATH` 指定路径）：

```json
{
  "enabled": true,
  "allow": ["hy3", "hy4-preview", "glm-5.3-flash"],
  "deny":  ["glm-5.2"],
  "domestic": {
    "allow": ["hy3", "hy4-preview"],
    "deny":  ["glm-5.2"]
  },
  "global": {
    "allow": ["gpt-5", "deepseek-v4.1-flash"],
    "deny":  []
  }
}
```

规则（优先级：enabled > auto/default > deny > allow）：

- `enabled: false` → 不限制
- **deny 命中** → 拒绝该模型（目录隐藏、请求 4xx）
- **allow 非空且未命中** → 拒绝，并提示当前白名单
- `allow` 为空 → 仅 `deny` 生效
- `auto` / `default` 始终放行
- **分区覆盖**：`domestic` / `global` 各自可独立设置 `allow` / `deny`；分区 `allow` 非空时**覆盖**全局 `allow`，分区 `deny` **累加**全局 `deny`

文件改动即时生效（mtime 感知热加载），无需重启。批量过滤与单请求判定共用同一份策略。

---

## 📚 文档

| 资源 | 链接 |
| :--- | :--- |
| 文档索引 | [`docs/README.md`](docs/README.md) |
| 快速开始 | [`guides/getting-started.md`](docs/guides/getting-started.md) |
| 配置参考 | [`guides/configuration.md`](docs/guides/configuration.md) |
| HTTP API | [`api/http.md`](docs/api/http.md) |
| 架构说明 | [`architecture/overview.md`](docs/architecture/overview.md) |
| 运维排障 | [`operations/runbook.md`](docs/operations/runbook.md) |
| 预编译包 | [GitHub Releases](https://github.com/LOX2018/buddy-proxy-V3/releases/latest) |
| 更新日记 | [`CHANGELOG.md`](CHANGELOG.md) · [安全说明](SECURITY.md) |

开发命令（仓库根目录）：

```bash
make test      # go test ./...
make build     # 产出 bin/codebuddy-proxy
make build-gui # 产出 bin/codebuddy-proxy-gui.exe（Windows 托盘版）
make release   # 四平台交叉编译 + SHA256SUMS.txt
```

另有 `make fmt` / `vet` / `test-race` / `check`。

---

## 🙏 贡献者致谢

感谢这些同学用 issue 把真实问题送上门（原仓库 [wnddd839/buddy-proxy](https://github.com/wnddd839/buddy-proxy)）：

- [@dyed-fanxing](https://github.com/dyed-fanxing) · [#2](https://github.com/wnddd839/buddy-proxy/issues/2) · [#4](https://github.com/wnddd839/buddy-proxy/issues/4) · [#27](https://github.com/wnddd839/buddy-proxy/pull/27) · [#28](https://github.com/wnddd839/buddy-proxy/issues/28) · [#30](https://github.com/wnddd839/buddy-proxy/issues/30)
- [@carter003](https://github.com/carter003) · [#6](https://github.com/wnddd839/buddy-proxy/issues/6) · [#8](https://github.com/wnddd839/buddy-proxy/issues/8) · [#9](https://github.com/wnddd839/buddy-proxy/issues/9) · [#10](https://github.com/wnddd839/buddy-proxy/issues/10) · [#11](https://github.com/wnddd839/buddy-proxy/issues/11)
- [@tearslee](https://github.com/tearslee) · [#7](https://github.com/wnddd839/buddy-proxy/issues/7)
- [@240xu](https://github.com/240xu) · [#12](https://github.com/wnddd839/buddy-proxy/pull/12)
- [@zeonseoi](https://github.com/zeonseoi) · [#13](https://github.com/wnddd839/buddy-proxy/issues/13) · [#19](https://github.com/wnddd839/buddy-proxy/issues/19)
- [@itaid](https://github.com/itaid) · [#14](https://github.com/wnddd839/buddy-proxy/issues/14)
- [@kouekikin24](https://github.com/kouekikin24) · [#15](https://github.com/wnddd839/buddy-proxy/issues/15) · [#17](https://github.com/wnddd839/buddy-proxy/issues/17) · [#18](https://github.com/wnddd839/buddy-proxy/issues/18) · [#20](https://github.com/wnddd839/buddy-proxy/issues/20) · [#22](https://github.com/wnddd839/buddy-proxy/issues/22) · [#23](https://github.com/wnddd839/buddy-proxy/issues/23) · [#24](https://github.com/wnddd839/buddy-proxy/issues/24) · [#25](https://github.com/wnddd839/buddy-proxy/issues/25) · [#29](https://github.com/wnddd839/buddy-proxy/issues/29) · [#31](https://github.com/wnddd839/buddy-proxy/pull/31) · [#32](https://github.com/wnddd839/buddy-proxy/pull/32) · [#33](https://github.com/wnddd839/buddy-proxy/issues/33)
- [@kkl31415926](https://github.com/kkl31415926) · [#16](https://github.com/wnddd839/buddy-proxy/issues/16)

---

## ⚠️ 免责声明与合规

**请用一分钟读完这一节。** 它是本项目持续开源的前提。

### 这是什么

一个**自行托管、本地运行的协议转换工具**。它不提供任何模型服务，不代理任何第三方 API，不托管任何账号，不中转任何流量到本项目维护者——**你的请求只从你自己的机器发往你所选的上游（CodeBuddy 或 WorkBuddy）**。

### 你的责任

使用本项目即表示你确认并同意：

1. **你只对拥有合法授权、且符合其服务条款的账号使用本工具。** 账号是否允许此类接入，由你与该服务的协议决定。
2. **遵守服务条款是你的责任。** 本项目无法代你判断某个账号、某个地区、某类套餐是否允许此类用法，也**不为你的账号被限制、降权、封禁承担任何责任**。
3. **合规自查在你这边。** 包括但不限于：服务条款、地区法律法规、单位或组织的内部规定。
4. **不要绕过付费。** 本项目无意也不应被用于规避订阅费用或用量计费。

若你所在的环境不允许此类接入，**请不要使用本项目**。

### 我们不提供什么

| 不提供 | 说明 |
| :--- | :--- |
| 不提供账号 | 不售卖、不赠送、不代注册任何 CodeBuddy / WorkBuddy 账号或额度 |
| 不提供托管服务 | 没有公共实例，没有官方部署，不接收你的流量 |
| 不提供担保 | 软件按「原样」提供，不保证可用性、不保证与上游的持续兼容 |
| 不承担损失 | 对使用造成的任何直接或间接损失（含账号风险、数据损失、业务中断）不承担责任 |

### 安全使用建议

- 默认绑定 `127.0.0.1`，需要局域网访问时自行评估暴露面
- 暴露 `/v1` 时务必设置 `CODEBUDDY_PROXY_API_KEY`
- **不要把 `.env`、账号 JSON、token、API Key 提交进仓库或分享给他人**
- 管理台密码与 API Key 分开管理
- 定期备份账号池 JSON，但注意其中包含凭据
- 建议一进程一份 `proxy-accounts.json`；加号优先走管理台

漏洞报告请勿开公开 issue，见 [SECURITY.md](SECURITY.md)。

---

## License

[BSD-3-Clause](LICENSE) · 开源分享，不含任何担保与责任。

---

<p align="center">
  <sub>Buddy Proxy 与 CodeBuddy、WorkBuddy 官方无关联、无隶属、无背书关系。「CodeBuddy」「WorkBuddy」为其各自持有者的商标，此处仅作技术兼容性描述之用。</sub>
</p>

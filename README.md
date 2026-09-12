# opencode2api

<p>
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="Go 1.24" />
  <img src="https://img.shields.io/badge/platform-linux%2Famd64-blue" alt="linux/amd64" />
  <img src="https://img.shields.io/docker/pulls/moewsama/opencode2api" alt="Docker pulls" />
  <img src="https://img.shields.io/badge/license-MIT-green" alt="MIT" />
</p>

把 **OpenCode Zen / Zen Go** 的私有协议，转成标准的 **OpenAI** 与 **Anthropic** API。
一个 Go 二进制、一个 8080 端口，开箱即用。

```text
你的客户端 ── OpenAI / Anthropic ──► opencode2api ── Zen / Zen Go ──► 上游
                                        │
                                        └── /admin（内置管理后台）
```

---

## ✨ 特性

**协议**

- OpenAI Chat Completions、Responses、Models API
- Anthropic Messages API
- 普通响应 + SSE 流式，同协议透传 / 跨协议转码
- 文本、图片、thinking/reasoning、工具定义 / 调用 / 结果全量转换
- 无法无损表达的内容直接报错，绝不静默丢弃

**上游调度**

- Zen / Go 双 key 池，自动均衡 + 故障冷却 + 指数退避
- Zen 匿名模式：免费模型先走 `public` 匿名通道，失败按 `prefer` 回退 key
- 模型目录按周期同步 Zen/Go `/v1/models`，原生协议来自 OpenCode 官方能力目录，不硬编码模型 ID
- 成本与弃用信息每日从 models.dev 同步
- 401/403/429 判定为 key 粒度失败，不拖全池陪跑；429 尊重 `Retry-After`

**网络**

- `direct` / HTTP(S) / SOCKS5(H) 代理，key 与代理亲和绑定
- 代理池文件（`proxyfile`，仅限配置目录内相对路径）+ 行尾注释
- 会话级 key/proxy 亲和，故障自动迁移，异常代理每 15 分钟复查

**可观测**

- stdout 单行 JSON 结构化日志，key 只露尾 5 位，短 key 全脱敏
- 内存滚动指标：请求统计、Token 用量、逐次上游尝试、实时日志 SSE

**管理后台**（`/admin`，与 API 同端口）

- 运行桌面、首次运行检查、接入手册、Token 用量、三协议 Playground、路由诊断、配置、事件日志、账号安全
- Argon2id 密码哈希 + HttpOnly Session Cookie + CSRF + 登录/密码限速
- 配置保存先验证后切换，无效配置不影响当前流量；`listen` 类字段提示重启

---

## 🚀 快速开始

```bash
git clone https://github.com/MoewSama/opencode2api.git
cd opencode2api
cp config.example.json config.json
# 编辑 config.json：填 server_keys、zen_keys/go_keys，改 webui.password
docker compose pull && docker compose up -d
```

```bash
curl http://127.0.0.1:8080/healthz        # 健康检查
# 浏览器打开 http://127.0.0.1:8080/admin   # 管理后台
docker compose logs -f                     # 日志
```

换版本 / 换端口：

```bash
OPENCODE2API_VERSION=v2.0.6 OPENCODE2API_PORT=18080 docker compose up -d
```

不用 Compose，直接跑镜像：

```bash
docker run -d --name opencode2api --restart unless-stopped \
  -p 8080:8080 \
  -e LISTEN_ADDRESS=0.0.0.0:8080 \
  -v "$(pwd)/config.json:/var/lib/opencode2api/config.json" \
  ghcr.io/moewsama/opencode2api:latest
```

> `config.json` 通过 bind mount 直接挂载，改完 `docker compose restart` 即生效。
> 宿主机没有 `config.json` 时 Docker 会建出空目录，entrypoint 会自动改用
> `config.local.json` 并从示例生成，服务照常启动。

源码编译（Go 1.24+）：

```bash
go build -o opencode2api ./
```

预编译 `linux/amd64` 二进制见 [GitHub Releases](https://github.com/MoewSama/opencode2api/releases)，
容器镜像 `ghcr.io/moewsama/opencode2api` 只有 `latest` + 版本号两个标签。

---

## 🔌 API

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| `GET` | `/v1/models` | OpenAI 模型列表 |
| `POST` | `/v1/chat/completions` | OpenAI Chat Completions |
| `POST` | `/v1/responses` | OpenAI Responses |
| `POST` | `/v1/messages` | Anthropic Messages |
| `GET` | `/healthz` | 健康检查（免鉴权） |

`/healthz` 返回版本、模型目录状态、key/代理池汇总，不含任何 secret。
刚启动、模型目录未就绪时会短暂返回 `503 starting`，首次刷新成功后变 `200 ok`，
属于正常现象。

---

## 🖥️ 管理后台

地址：`http://服务器地址:8080/admin`

首次账号 `admin`，密码取自 `webui.password`。首次启动成功后密码会被转成
Argon2id 哈希写入 `webui.password_hash`，明文自动删除——**登录后第一件事就是改密码**。

后台能力：运行桌面 · 六步首次检查 · 接入手册（含 Chat / Responses / Anthropic /
Python / JS 示例，key 统一占位不写真值）· Token 用量（覆盖率、分钟趋势、模型排行、
Tier 分布）· 三协议 Playground · 路由诊断 · 配置热更新 · 实时日志 SSE · 账号安全。

监控数据只在内存（lifetime + 最近一小时，请求路由保留 1 万、上游尝试保留 2 万，
接口最多返回 500 条），**重启清空**；要持久化请收集 stdout 日志。

诊断接口（均需 Session，POST 还需 CSRF，每 IP 每分钟 12 次）：

| 方法 | 路径 | 用途 |
| ---- | ---- | ---- |
| `GET` | `/admin/api/debug/models` | 模型路由、原生协议、Zen/Go 可用性、匿名资格、成本状态 |
| `POST` | `/admin/api/debug/inference` | 经真实 Gateway 发起非流式诊断请求 |

诊断请求强制非流式，结果含 `ok` / 真实 `http_status` / `duration_ms` / `request_id` /
路由 / 原始响应；敏感字段在返回浏览器前清除，只保存在当前进程内存。

---

## ⚙️ 配置

`config.json` 支持 `//` 与 `/* ... */` 注释（保存后会被规范化，注释不保留）。

```json
{
  "listen": "0.0.0.0:8080",
  "server_keys": ["change-this-local-key"],
  "zen_keys": [],
  "go_keys": [],
  "anonymous": true,
  "prefer": "go",
  "proxyfile": "",
  "proxies": ["direct"],
  "upstream": {
    "zen": "https://opencode.ai/zen",
    "go": "https://opencode.ai/zen/go"
  },
  "retry": { "max_attempts": 3, "timeout_seconds": 300 },
  "models": { "refresh_seconds": 300, "protocols": {} },
  "performance": {
    "max_idle_conns": 2048,
    "max_idle_conns_per_host": 256,
    "max_conns_per_host": 0,
    "idle_conn_timeout_seconds": 120,
    "connect_timeout_seconds": 5,
    "failure_cooldown_seconds": 15
  },
  "logging": { "level": "info", "ring_size": 2000 },
  "webui": {
    "enabled": true,
    "username": "admin",
    "password": "change-this-admin-password",
    "session_ttl_minutes": 720
  }
}
```

### 基础字段

| 字段 | 说明 |
| ---- | ---- |
| `listen` | 监听地址，`host:port` 形式。API 与 `/admin` 共用此端口 |
| `server_keys` | 本地鉴权 key（至少 1 个），只验本地身份，不发给上游 |
| `zen_keys` / `go_keys` | Zen / Go 上游 key 池，可多配 |
| `anonymous` | Zen 匿名模式。关闭时两池至少一非空；开启后可全空 |
| `prefer` | 双上游都有模型时的 key 尝试顺序，`go` 或 `zen` |
| `proxyfile` | 代理池文件，**仅允许配置目录内的相对路径**（防任意文件读取） |
| `proxies` | 代理列表：`direct` / `http(s)://` / `socks5(h)://`，可带认证信息 |

### 匿名模式

免费模型（models.dev 零成本，或 ID 含 `free`）先走匿名 Zen，失败后按 `prefer`
回退 key；非免费模型跳过匿名。匿名阶段把每个可用代理各试一次，不受
`max_attempts` 截断。双 key 池全空时 `/v1/models` 只列可匿名模型。

### 代理

```json
"proxies": ["http://user:password@127.0.0.1:7890", "socks5://127.0.0.1:1080"]
```

`proxyfile`（如 `proxies.txt`，与 `config.json` 同目录）每行一个，支持空行、
`#` / `;` / `//` 整行与行尾注释。文件内容追加到 `proxies` 并去重，全空则回退 `direct`。

### 重试与模型

| 字段 | 说明 |
| ---- | ---- |
| `retry.max_attempts` | 每 Tier 最大尝试次数 1–10（key 轮换间带退避 + 抖动） |
| `retry.timeout_seconds` | 非流式请求总超时 1–3600s；流式响应建连后不限整流时长，断连即止 |
| `models.refresh_seconds` | 模型列表 + 能力目录刷新间隔 1–86400s |
| `models.protocols` | 手动覆盖模型原生协议（`chat` / `responses` / `anthropic`），如 `{"custom-model": "chat"}` |

非流式上游响应超过 64MB 直接 502，不会把截断 JSON 当 200 返回。
`models.protocols` 覆盖、能力快照存于 `<config>.models.catalog.json`（`0600`）。

### 性能 / 日志 / 后台

| 字段 | 说明 |
| ---- | ---- |
| `performance.*` | 连接池与超时（均有上界，见示例）；`max_conns_per_host: 0` 为不限；冷却随连续失败指数增长 |
| `logging.level` | `debug` / `info` / `warn` / `error`，WebUI 可热切 |
| `logging.ring_size` | WebUI 日志环 100–50000 |
| `webui.enabled` | 是否挂载 `/admin` |
| `webui.password` | 仅首次初始化用（≥10 字符），启动后转哈希删除 |
| `webui.session_ttl_minutes` | Session 有效期 5–10080 分钟 |

WebUI 保存走"验证 → 建新实例 → 原子写盘（`config.json.bak` 兜底）→ 切换流量"，
失败自动回滚。keys / 代理 / 上游 / 重试 / 模型 / 日志即时生效；
`listen`、`webui.enabled` 需重启。bind mount 下写盘自动降级为原位覆盖，
宿主机文件保持同步。

### 跨协议注意

Responses 的 `instructions` / developer 消息只收纯文本，system / developer 里带
image / file 会直接报错（指引改放 message input），不会静默丢。转 Anthropic 的
file 块必须带文件数据或 URL。Chat / Responses / Anthropic 互转的 tool_choice、
reasoning effort、停止原因、SSE 工具分片都会映射；上游错误转成目标协议的结构化
错误事件，不伪装正常结束。

---

## 🔑 会话亲和

上游请求自动带 `User-Agent`、`x-opencode-client`、`x-opencode-session`、
`x-session-affinity`、`X-Session-Id`、`x-opencode-request`、`x-opencode-project`
等头。会话 ID 优先用客户端显式头（`x-opencode-session` / `x-session-id` /
`conversation-id` 等），否则由首条用户消息稳定派生——两个首消息完全相同的独立
会话请显式传不同 `x-session-id` 以免串会话。

---

## 🙏 致谢

感谢 [LINUX DO](https://linux.do) 社区一直以来的支持。

# Tavily 代理池 & 管理面板

简体中文 | [English](./README_EN.md)

一个透明的 Tavily API 反向代理：将多个 Tavily API Key（额度/credits）汇聚在一个 **Master Key** 之后，并提供内置 Web UI 用于管理 Key、用量与请求日志。

---

## 🚀 功能特性

- **透明代理**：完整转发至 `https://api.tavily.com`（支持所有路径与方法）。
- **Master Key 鉴权**：客户端通过 `Authorization: Bearer <MasterKey>` 安全访问。
- **分发 User Key**：
  - 后台创建调用专用 Key（可备注、可停用、可设置过期时间）。
  - 每个 User Key 独立限流（`rate_limit_per_minute`，`0` 表示不限流）。
  - 按 User Key 统计总请求数与状态码分布（2xx/4xx/5xx）。
- **智能 Key 池管理**：
  - 优先使用剩余额度最高的 Key。
  - 同额度 Key 随机打散，有效防止请求过于集中触发频率限制。
- **自动故障切换**：遇到 `401` / `429` / `432` / `433` 等错误时，自动尝试 Key 池中的下一个可用 Key。
- **MCP 支持**：内置 HTTP MCP (Model Context Protocol) 端点，可轻松接入 Claude、VS Code 等 AI 工具。
- **可视化管理面板**：
  - **Key 管理**：便捷添加、删除及同步多个 Tavily Key 的额度信息。
  - **用量统计**：通过图表直观展示请求量与额度消耗趋势。
  - **请求日志**：详细记录每次请求，支持过滤筛选与手动清理。
- **自动化任务**：每月 1 号自动重置额度，定期清理历史日志。
- **开箱即用**：Go 二进制单文件部署，内嵌 Web UI（Vite + Vue 3 + Naive UI）。

---

## 🛠️ 环境要求

- **Docker / Docker Compose** (推荐部署方式，无需本地环境)
- **Go**: `1.23+` & **Bun**: `1.2+`（或 **Node.js**: `20+`，仅用于本地手动编译）

---

## 📦 快速部署 (Docker)

直接使用 GHCR 镜像部署，**无需本地编译**。

### 1. 使用 Docker Compose (推荐)

创建 `docker-compose.yml` 文件：

```yaml
version: "3.8"
services:
  tavily-proxy:
    image: ghcr.io/xuncv/tavilyproxymanager:main
    container_name: tavily-proxy
    ports:
      - "8080:8080"
    environment:
      - LISTEN_ADDR=:8080
      - DATABASE_PATH=/app/data/proxy.db
      - TAVILY_BASE_URL=https://api.tavily.com
      - UPSTREAM_TIMEOUT=30s
      - MASTER_KEY=replace_with_your_master_key
      - USER_KEY_ENCRYPTION_KEY=replace_with_32_byte_or_base64_key
    volumes:
      - ./data:/app/data
      - /etc/localtime:/etc/localtime:ro
    restart: unless-stopped
```

执行启动：

```bash
docker-compose up -d
```

### 2. 使用 Docker 原生命令

```bash
docker run -d \
  --name tavily-proxy \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e DATABASE_PATH=/app/data/proxy.db \
  -e MASTER_KEY=replace_with_your_master_key \
  -e USER_KEY_ENCRYPTION_KEY=replace_with_32_byte_or_base64_key \
  ghcr.io/xuncv/tavilyproxymanager:main
```

---

## 🔑 首次运行：获取 Master Key

服务在**首次启动**时：

- 如果配置了 `MASTER_KEY`，将使用该值作为初始 Master Key。
- 如果未配置 `MASTER_KEY`，会自动生成一个随机 Master Key。

该 Key 用于后续登录管理面板和调用 API。

您可以通过以下命令查看控制台日志来获取它：

```bash
docker logs tavily-proxy 2>&1 | grep "master key"
```

**日志示例：**
`level=INFO msg="no master key found, generated a new one" key=your_generated_master_key_here`

> **提示**：建议首次登录后在管理面板或通过数据库备份妥善保存此 Key。

---

## 🛠️ 本地开发与手动编译

如果您需要修改源码并自行构建：

1.  **启动后端**:
    ```bash
    go run ./server
    ```
2.  **启动前端**:
    ```bash
    cd web && bun install && bun run dev
    ```

**手动编译二进制产物**:

- **Windows**: `.\scripts\build_all.ps1`
- **Linux/macOS**: `./scripts/build_all.sh`

**使用 Dockerfile 本地构建镜像**:

```bash
docker build -t my-tavily-proxy .
```

---

## 📖 使用指南

### REST API 代理

客户端调用方式与 Tavily 官方 API 完全一致，只需将 API 地址替换为代理地址，并使用 **Master Key**：

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <MASTER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "最新 AI 技术趋势", "search_depth": "basic"}'
```

**兼容性说明**:

- `Master Key` 支持 `{"api_key": "<MASTER_KEY>"}` 或 `{"apiKey": "<MASTER_KEY>"}`。
- `Master Key` 支持 GET 参数 `?api_key=<MASTER_KEY>`。
- `User Key` 仅支持 `Authorization: Bearer <USER_KEY>`，不支持 body/query 传参。

### 分发 User Key 调用

在 Web UI 的“调用密钥”页面创建并分发 User Key 后，调用示例：

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <USER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "AI agent 安全实践", "search_depth": "basic"}'
```

### MCP (Model Context Protocol)

服务在 `http://localhost:8080/mcp` 提供 HTTP MCP 端点。

默认启用无状态模式（`MCP_STATELESS=true`），可避免客户端出现 `session not found`。
如需有状态会话，请将 `MCP_STATELESS=false`，并确保上游反向代理正确透传 `Mcp-Session-Id` 且启用会话粘性（sticky）。

#### VS Code 配置示例 (配合 mcp-remote)

```json
{
  "servers": {
    "tavily-proxy": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "http://localhost:8080/mcp",
        "--header",
        "Authorization: Bearer 您的_MASTER_KEY"
      ]
    }
  }
}
```

---

## ⚙️ 配置项 (环境变量)

| 变量名             | 说明                 | 默认值                   |
| :----------------- | :------------------- | :----------------------- |
| `LISTEN_ADDR`      | 服务监听地址         | `:8080`                  |
| `DATABASE_PATH`    | SQLite 数据库路径    | `/app/data/proxy.db`     |
| `TAVILY_BASE_URL`  | 上游 Tavily API 地址 | `https://api.tavily.com` |
| `UPSTREAM_TIMEOUT` | 上游请求超时时间     | `150s`                   |
| `MCP_STATELESS`    | MCP 是否无状态模式   | `true`                   |
| `MCP_SESSION_TTL`  | MCP 会话空闲超时     | `10m`                    |
| `MASTER_KEY` | 首次启动时可选指定初始 Master Key（数据库已存在时忽略） | 空 |
| `USER_KEY_ENCRYPTION_KEY` | User Key 加密主密钥（仅在启用分发 User Key 功能时需要） | 空（未配置则分发 User Key 功能关闭） |
| `USER_KEY_RATE_LIMIT_WINDOW` | User Key 限流窗口 | `1m` |
| `USER_KEY_RATE_LIMIT_DEFAULT` | 新建 User Key 默认每分钟限额（`0` 表示不限流） | `60` |

### `USER_KEY_ENCRYPTION_KEY` 格式要求

- 可选；留空表示关闭分发 User Key 功能。
- 若设置，则必须满足以下之一：
  - 原始密钥字节长度为 `16` / `24` / `32`（对应 AES-128/192/256）。
  - Base64 / Base64URL 解码后字节长度为 `16` / `24` / `32`。
- 若配置了但长度不合法，服务会启动失败并报错。
- 建议使用 `32` 字节随机值（AES-256），并使用 Base64 保存。

PowerShell 生成 32 字节随机 Base64 示例：

```powershell
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToBase64String($bytes)
```

---

## 📄 开源协议

本项目基于 MIT 协议开源。

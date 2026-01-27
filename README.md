# Tavily 代理池 & 管理面板

简体中文 | [English](./README_EN.md)

一个透明的 Tavily API 反向代理：将多个 Tavily API Key（额度/credits）汇聚在一个 **Master Key** 之后，并提供内置 Web UI 用于管理 Key、用量与请求日志。

---

## 🚀 功能特性

- **透明代理**：完整转发至 `https://api.tavily.com`（支持所有路径与方法）。
- **Master Key 鉴权**：客户端通过 `Authorization: Bearer <MasterKey>` 安全访问。
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

- **Go**: `1.23+`
- **Node.js**: `20+`（仅用于前端 Web UI 构建）
- **Docker**:（推荐部署方式）

---

## 🚦 快速开始 (开发环境)

1.  **启动后端**:

    ```bash
    go run ./server
    ```

    _首次启动会自动生成 Master Key，请查看控制台日志。_

2.  **启动 frontend**:
    ```bash
    cd web
    npm install
    npm run dev
    ```
    访问 `http://localhost:5173`，按页面提示输入 Master Key。

---

## 📦 部署说明

### 1. 编译二进制

使用项目自带脚本进行构建（需要安装 Go 和 Node.js，脚本会自动完成前端构建并内嵌）：

- **Windows (PowerShell)**:
  ```powershell
  .\scripts\build_all.ps1
  ```
- **Linux/macOS (Bash)**:
  ```bash
  chmod +x ./scripts/build_all.sh
  ./scripts/build_all.sh
  ```
  编译产物位于 `build/` 目录。

### 2. Docker 部署 (推荐)

项目提供多阶段构建的 `Dockerfile`，可自动完成前后端编译。

#### 使用 Docker Compose 构建并运行

1. 确保项目根目录下存在 `docker-compose.yml` 和 `Dockerfile`。
2. 执行构建并启动：
   ```bash
   docker-compose up -d --build
   ```

#### 使用 Docker 原生命令构建并运行
```bash
docker build -t tavily-proxy .
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  --name tavily-proxy \
  tavily-proxy
```

> **注意**: 容器内部默认使用 `/app/data/proxy.db` 存储数据。请务必挂载该目录以实现数据持久化。对于 Windows/macOS 的 Docker Desktop 用户，建议手动设置环境变量 `TZ`（如 `TZ=Asia/Shanghai`）。

---

## 📖 使用指南

### REST API 代理

客户端调用方式与 Tavily 官方 API 完全一致，只需更改请求地址并使用 **Master Key**：

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <MASTER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "最新 AI 技术趋势", "search_depth": "basic"}'
```

**兼容性说明**:

- **POST JSON**: 支持 `{"api_key": "<MASTER_KEY>"}` 或 `{"apiKey": "<MASTER_KEY>"}`。
- **GET Query**: 支持 `?api_key=<MASTER_KEY>` 或 `?apiKey=<MASTER_KEY>`。

### MCP (Model Context Protocol)

服务在 `http://localhost:8080/mcp` 提供 Streamable HTTP MCP 端点。它暴露的工具与官方 `tavily-mcp` 一致（如 `tavily-search`）。

#### VS Code 配置示例

在您的 MCP 配置文件中添加如下内容（配合 `mcp-remote` 使用）：

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

---

## 📄 开源协议

本项目基于 MIT 协议开源。

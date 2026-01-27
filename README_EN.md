# Tavily Proxy & Management Dashboard

简体中文 | English

A transparent reverse proxy for the Tavily API that aggregates multiple Tavily API Keys into a single **Master Key**. It features a built-in Web UI for managing keys, monitoring usage, and inspecting request logs.

---

## 🚀 Features

- **Transparent Proxy**: Seamlessly forwards requests to `https://api.tavily.com` (supports all endpoints/methods).
- **Master Key Authentication**: Secure access via `Authorization: Bearer <MasterKey>`.
- **Intelligent Key Pooling**:
  - Prioritizes keys with the highest remaining quota.
  - Randomly distributes requests among keys with equal quota to prevent rate limiting.
- **Automatic Failover**: Automatically retries with the next available key upon receiving `401`, `429`, `432`, or `433` errors.
- **MCP Support**: Built-in HTTP MCP (Model Context Protocol) endpoint for easy integration with AI tools (e.g., Claude, VS Code).
- **Comprehensive Dashboard**:
  - **Key Management**: Add, delete, and sync quotas for multiple Tavily keys.
  - **Usage Statistics**: Visualized charts for request volume and quota consumption.
  - **Request Logs**: Detailed logs with filtering and manual cleanup options.
- **Automated Tasks**: Monthly quota resets and periodic log cleaning.
- **Self-Contained**: Single binary deployment with embedded Web UI (Vite + Vue 3 + Naive UI).

---

## 🛠️ Prerequisites

- **Go**: `1.23+`
- **Node.js**: `20+` (Only required for building the Web UI)
- **Docker**: (Recommended for deployment)

---

## 🚦 Quick Start (Development)

1.  **Start Backend**:

    ```bash
    go run ./server
    ```

    _On the first run, a Master Key will be generated and printed in the logs._

2.  **Start Frontend**:
    ```bash
    cd web
    npm install
    npm run dev
    ```
    Open `http://localhost:5173` and enter the Master Key when prompted.

---

## 📦 Deployment

### 1. Build from Source

Use the provided scripts to build for your target platform (requires Go and Node.js installed):

- **Windows (PowerShell)**:
  ```powershell
  .\scripts\build_all.ps1
  ```
- **Linux/macOS (Bash)**:
  ```bash
  chmod +x ./scripts/build_all.sh
  ./scripts/build_all.sh
  ```
  The binaries will be generated in the `build/` directory.

### 2. Docker Deployment (Recommended)

The project includes a multi-stage `Dockerfile` that builds both the frontend and backend.

#### Build and Run with Docker Compose

1. Ensure `docker-compose.yml` and `Dockerfile` are in the project root.
2. Build and start:
   ```bash
   docker-compose up -d --build
   ```

#### Build and Run with Docker

```bash
docker build -t tavily-proxy .
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  --name tavily-proxy \
  tavily-proxy
```

> **Note**: The container uses `/app/data/proxy.db` for storage by default. Ensure the directory is correctly mounted to persist your data. For Windows/macOS Docker Desktop, it's recommended to set the `TZ` environment variable (e.g., `TZ=Asia/Shanghai`).

---

## 📖 Usage Guide

### REST API Proxy

Use the proxy as you would the official Tavily API, but replace the endpoint and authentication:

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <MASTER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "Latest AI trends", "search_depth": "basic"}'
```

**Legacy Compatibility**:

- **POST JSON**: Supports `{"api_key": "<MASTER_KEY>"}` or `{"apiKey": "<MASTER_KEY>"}`.
- **GET Query**: Supports `?api_key=<MASTER_KEY>` or `?apiKey=<MASTER_KEY>`.

### MCP (Model Context Protocol)

The server provides a Streamable HTTP MCP endpoint at `http://localhost:8080/mcp`. It exposes tools compatible with the official `tavily-mcp` (e.g., `tavily-search`).

#### Integration with VS Code

Add the following to your MCP configuration (e.g., via `mcp-remote`):

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
        "Authorization: Bearer YOUR_MASTER_KEY"
      ]
    }
  }
}
```

---

## ⚙️ Configuration (Environment Variables)

| Variable           | Description                   | Default                  |
| :----------------- | :---------------------------- | :----------------------- |
| `LISTEN_ADDR`      | Server listening address      | `:8080`                  |
| `DATABASE_PATH`    | Path to SQLite database       | `/app/data/proxy.db`     |
| `TAVILY_BASE_URL`  | Upstream Tavily API URL       | `https://api.tavily.com` |
| `UPSTREAM_TIMEOUT` | Timeout for upstream requests | `150s`                   |

---

## 📄 License

This project is licensed under the MIT License.

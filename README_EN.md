# Tavily Proxy & Management Dashboard

简体中文 | English

A transparent reverse proxy for the Tavily API that aggregates multiple Tavily API Keys into a single **Master Key**. It features a built-in Web UI for managing keys, monitoring usage, and inspecting request logs.

---

## 🚀 Features

- **Transparent Proxy**: Seamlessly forwards requests to `https://api.tavily.com` (supports all endpoints/methods).
- **Master Key Authentication**: Secure access via `Authorization: Bearer <MasterKey>`.
- **Distributed User Keys**:
  - Create invocation-only user keys in the dashboard (with note, disable, and expiration support).
  - Per-key independent rate limit (`rate_limit_per_minute`, where `0` means unlimited).
  - Per-key usage analytics (total calls + 2xx/4xx/5xx breakdown).
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

## 🛠️ Requirements

- **Docker / Docker Compose** (Recommended deployment method, no local environment needed)
- **Go**: `1.23+` & **Bun**: `1.2+` (or **Node.js**: `20+`, only for manual builds)

---

## 📦 Quick Deployment (Docker)

Deploy directly using the GHCR image, **no local compilation required**.

### 1. Using Docker Compose (Recommended)

Create a `docker-compose.yml` file:

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

Start the service:

```bash
docker-compose up -d
```

### 2. Using Docker CLI

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

## 🔑 First Run: Obtaining the Master Key

On **first startup**:

- If `MASTER_KEY` is provided, it is used as the initial Master Key.
- If `MASTER_KEY` is not provided, the service generates a random Master Key.

This key is required to log into the dashboard and authenticate API calls.

You can retrieve it by checking the container logs:

```bash
docker logs tavily-proxy 2>&1 | grep "master key"
```

**Log Example:**
`level=INFO msg="no master key found, generated a new one" key=your_generated_master_key_here`

> **Tip**: It is highly recommended to save this key in a secure location after your first login.

---

## 🛠️ Local Development & Manual Building

If you need to modify the code and build it yourself:

1.  **Start Backend**:
    ```bash
    go run ./server
    ```
2.  **Start Frontend**:
    ```bash
    cd web && bun install && bun run dev
    ```

**Manual Binary Build**:

- **Windows**: `.\scripts\build_all.ps1`
- **Linux/macOS**: `./scripts/build_all.sh`

**Local Image Build with Dockerfile**:

```bash
docker build -t my-tavily-proxy .
```

---

## 📖 Usage Guide

### REST API Proxy

Call the proxy exactly as you would the official Tavily API, simply replacing the API base URL and using your **Master Key**:

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <MASTER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "Latest AI trends", "search_depth": "basic"}'
```

**Compatibility Notes**:

- `Master Key` supports `{"api_key": "<MASTER_KEY>"}` or `{"apiKey": "<MASTER_KEY>"}` in JSON bodies.
- `Master Key` supports the `api_key=<MASTER_KEY>` query parameter.
- `User Key` supports only `Authorization: Bearer <USER_KEY>` (no body/query auth).

### Calling with a Distributed User Key

After creating a user key in the dashboard, call the proxy like this:

```bash
curl -X POST "http://localhost:8080/search" \
  -H "Authorization: Bearer <USER_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"query": "AI agent security practices", "search_depth": "basic"}'
```

### MCP (Model Context Protocol)

The server provides an HTTP MCP endpoint at `http://localhost:8080/mcp`.

Stateless mode is enabled by default (`MCP_STATELESS=true`) to avoid `session not found` errors.
If you need stateful sessions, set `MCP_STATELESS=false` and ensure your reverse proxy forwards `Mcp-Session-Id` and uses sticky sessions.

#### VS Code Configuration (with mcp-remote)

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

| Variable           | Description              | Default                  |
| :----------------- | :----------------------- | :----------------------- |
| `LISTEN_ADDR`      | Server listening address | `:8080`                  |
| `DATABASE_PATH`    | Path to SQLite database  | `/app/data/proxy.db`     |
| `TAVILY_BASE_URL`  | Upstream Tavily API URL  | `https://api.tavily.com` |
| `UPSTREAM_TIMEOUT` | Upstream request timeout | `150s`                   |
| `MCP_STATELESS`    | Enable stateless MCP mode | `true`                  |
| `MCP_SESSION_TTL`  | Idle timeout for MCP session | `10m`               |
| `MASTER_KEY` | Optional initial Master Key on first startup (ignored if DB already has one) | empty |
| `USER_KEY_ENCRYPTION_KEY` | Encryption key for distributed user keys (only needed when this feature is enabled) | empty (feature disabled if missing) |
| `USER_KEY_RATE_LIMIT_WINDOW` | User-key rate-limit window | `1m` |
| `USER_KEY_RATE_LIMIT_DEFAULT` | Default per-minute limit for newly created user keys (`0` = unlimited) | `60` |

### `USER_KEY_ENCRYPTION_KEY` Requirements

- Optional; leave empty to disable distributed user-key feature.
- If set, it must satisfy one of:
  - Raw key byte length is `16` / `24` / `32` (AES-128/192/256).
  - Base64 / Base64URL decoded byte length is `16` / `24` / `32`.
- If provided with invalid length, startup fails with an error.
- Recommended: use a random `32`-byte key (AES-256), stored as Base64.

PowerShell example to generate a random 32-byte Base64 key:

```powershell
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToBase64String($bytes)
```

---

## 📄 License

This project is licensed under the MIT License.

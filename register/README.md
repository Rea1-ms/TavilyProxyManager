# tavily-register

## 目录结构

- `batch_signup.py`：批量任务入口（CLI），负责邮箱生成、调用注册流程、验证与保存结果
- `signup.py`：核心注册/登录/取 Key 逻辑（`requests.Session` 驱动）
- `gptmail_client.py`：临时邮箱（GPTMail 兼容接口）客户端

## 环境要求

- Python `>= 3.12`
- 推荐使用 `uv` 管理依赖与虚拟环境

## 安装

```bash
uv sync
```

## 配置

### 1) `config.yaml`

`signup.py` 会从仓库根目录读取 `config.yaml`（已在 `.gitignore` 中忽略）。示例：

```yaml
# YesCaptcha 验证码识别服务（推荐，速度快、准确率高）
YESCAPTCHA_API_KEY: "your_yescaptcha_api_key"

# OpenAI 兼容的 Chat Completions 接口（备用方案，如果未配置 YesCaptcha 则使用）
OPENAI_BASEURL: "https://example.com/v1"
OPENAI_API_KEY: "YOUR_API_KEY"
OPENAI_MODEL: "YOUR_MODEL"
```

**验证码识别方案说明：**

- **YesCaptcha（推荐）**：专业验证码识别服务，速度快（3-10秒），准确率高
  - 获取 API Key：访问 [YesCaptcha](https://yescaptcha.com/) 注册账号
  - 配置后程序会自动优先使用 YesCaptcha

- **视觉大模型（备用）**：使用支持视觉的大模型识别验证码，速度较慢（10-30秒）
  - 如果未配置 YesCaptcha，程序会自动回退到此方案
  - 需要配置 OpenAI 兼容的 API 接口

### 2) 临时邮箱环境变量（可选）

`batch_signup.py` 支持通过环境变量配置邮箱服务：

- `GPTMAIL_BASE_URL`
- `GPTMAIL_API_KEY`
- `GPTMAIL_TIMEOUT`
- `GPTMAIL_PREFIX`
- `GPTMAIL_DOMAIN`

## 快速开始

### 1. 测试配置

运行测试脚本验证 YesCaptcha 配置是否正确：

```bash
uv run python test_yescaptcha.py
```

测试脚本会检查：
- ✓ 配置文件是否正确
- ✓ SVG 转 PNG 功能是否可用
- ✓ YesCaptcha API 连接是否正常

### 2. 批量注册

查看参数：

```bash
uv run python batch_signup.py --help
```

脚本内置临时邮箱的共享key，批量注册：

```bash
uv run python batch_signup.py --count 10
```

如共享key额度用完，可以到 https://mail.chatgpt.org.uk 获取：

```bash
uv run python batch_signup.py --gptmail-api-key your_own_key --count 10
```

### 3. 重试失败的注册

如果有注册失败的记录，可以重试：

```bash
uv run python batch_signup.py --retry
```



## 输出文件

- `api_keys.txt`：成功记录（邮箱与 key）
- `failed.txt`：失败记录（邮箱与错误信息）
- `banned_domains.txt`：被判定为不可用的域名黑名单

## 常见问题

### 验证码识别相关

- **YesCaptcha 识别失败**：
  - 检查 API Key 是否正确配置
  - 确认账户余额是否充足
  - 运行 `python test_yescaptcha.py` 进行诊断

- **想切换回视觉模型**：
  - 从 `config.yaml` 中删除或注释 `YESCAPTCHA_API_KEY`
  - 确保 `OPENAI_*` 配置正确

- **识别速度慢**：
  - YesCaptcha 通常 3-10 秒完成识别
  - 视觉模型可能需要 10-30 秒
  - 建议使用 YesCaptcha 以提升效率

### 注册相关

- `ip-signup-blocked`：表示当前出口 IP 被禁止注册。脚本会终止批量流程
- `custom-script-error-code_extensibility_error`：通常表示当前邮箱域名被禁止。脚本会将域名写入 `banned_domains.txt` 并在自动生成邮箱模式下重新获取邮箱重试
- `invalid-captcha`：验证码识别结果不正确。可考虑降低并发、增加重试间隔
- `tavily` 调整了策略，一个 IP 一段时间内只能注册 5 个，请勿滥用

## 更多文档

- [YesCaptcha 使用说明](YESCAPTCHA_USAGE.md) - 详细的 YesCaptcha 配置和使用指南

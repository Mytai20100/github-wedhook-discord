# github-wedhook-discord

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![GitHub Stars](https://img.shields.io/github/stars/mytai20100/github-wedhook-discord?style=social)
![GitHub Issues](https://img.shields.io/github/issues/mytai20100/github-wedhook-discord)
![GitHub Forks](https://img.shields.io/github/forks/mytai20100/github-wedhook-discord?style=social)

Hmm wedhook github forward to wedhook discord 

## Features
- Colorful embeds - Different colors for different events
- User avatars - Shows GitHub user profile pictures
- Easy config - Simple YAML configuration file

## Installation

### Requirements

- Go 1.25 or higher
- Discord webhook URL

### Quick Start

1. Clone the repository

```bash
git clone https://github.com/mytai20100/github-wedhook-discord.git
cd github-wedhook-discord
```

2. Install dependencies

```bash
go mod download
```

3. Run the service

```bash
go run main.go
```

On first run, it will create a `config.yml` file. Edit it with your settings and run again.

## Configuration

Edit `config.yml`:

```yaml
server:
  host: "0.0.0.0"  # Server host (0.0.0.0 for all interfaces)
  port: 8080        # Server port

discord:
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN"
```

## GitHub Setup

1. Go to your GitHub repository
2. Navigate to Settings → Webhooks → Add webhook
3. Configure:
   - Payload URL: `https://your-domain.com/webhook`
   - Content type: `application/json`
   - SSL verification: Enable (recommended)
   - Events: Choose what you want (or "Send me everything")
4. Click Add webhook

## Supported Events

| Event | Color | Hex | Description |
|-------|-------|-----|-------------|
| Push | Blue | `#7289DA` | New commits pushed |
| Issues | Red | `#DC143C` | Issue opened/closed |
| Issue Comment | Pink | `#FF69B4` | Comments on issues |
| PR Open | Green | `#28A745` | Pull request opened |
| PR Review | Light Green | `#90EE90` | Review submitted |
| PR Comment | White | `#FFFFFF` | Comment on PR |
| PR Closed | Gray | `#6E7681` | Pull request closed |
| Actions | Pink | `#FF69B4` | Other actions |

Also supports: Star, Fork, Create, Delete, and more.

## Build

Build for production:

```bash
go build -o github
```

Run the binary:

```bash
./github
```
## Testing

Test with curl:

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: ping" \
  -d '{"zen":"GitHub webhook test","hook_id":123}'
```
You should see "pong" response and a log message.
## Endpoints
- `POST /webhook` - Receives GitHub webhooks
- `GET /` - Health check
## License
???? fuck 
## Change logs

- 27/12/2025: v0.0000000000000000001 

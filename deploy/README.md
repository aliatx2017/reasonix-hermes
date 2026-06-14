# Reasonix-Hermes Deployment

One-command deploy to any Kubernetes cluster or single-node VPS.

## Quick Start — $5 VPS (docker-compose)

```bash
# Clone and deploy
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes/deploy

# Set secrets
export DEEPSEEK_API_KEY="sk-..."
export DISCORD_BOT_TOKEN="..."  # optional

# Start all services
docker-compose up -d
```

Services:
- `reasonix-mcpbridge` — MCP bridge on :9090
- `reasonix-memoryserver` — Hindsight memory on :9091
- `reasonix-bot` — Discord bot (if `DISCORD_BOT_TOKEN` set)

## Kubernetes (Helm)

```bash
helm repo add reasonix https://aliatx2017.github.io/reasonix-hermes
helm install reasonix reasonix/reasonix \
  --set secrets.deepseekApiKey="sk-..." \
  --set components.bot.enabled=true \
  --set secrets.discordBotToken="..."
```

Or with a values file:

```bash
helm install reasonix ./deploy/helm/reasonix -f my-values.yaml
```

## Components

| Component | Env var | Description |
|-----------|---------|-------------|
| `mcpbridge` | — | Exposes Reasonix as MCP tools over HTTP |
| `memoryserver` | — | Hindsight memory (SQLite + vector search) |
| `bot` | `DISCORD_BOT_TOKEN` | Discord bot gateway |

## Minimum VPS

- **CPU**: 1 vCPU
- **RAM**: 512 MB (256 MB works for single-component)
- **Disk**: 5 GB
- **Price**: ~$5/month (Hetzner, DigitalOcean, Linode)

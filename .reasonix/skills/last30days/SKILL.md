---
name: last30days
description: "Research what people say about any topic in the last 30 days across Reddit, X, YouTube, TikTok, HN, Polymarket, GitHub, and the web. Pulls posts and engagement data, then synthesizes a grounded summary. Requires Python 3.12+."
argument-hint: "last30days nvidia earnings reaction | last30days AI video tools | last30days what users want in react"
allowed-tools: Bash, Read, Write
user-invocable: true
metadata:
  tags:
    - research
    - deep-research
    - reddit
    - twitter
    - youtube
    - tiktok
    - hackernews
    - polymarket
    - social-media
    - trends
    - news
    - analysis
  requires:
    bins:
      - python3
      - curl
---

# /last30days — Cross-Platform Social Research

This is a Reasonix wrapper for the [mvanhorn/last30days-skill](https://github.com/mvanhorn/last30days-skill) (MIT license, v3.3.2).

It researches any topic across Reddit, X, YouTube, TikTok, HN, Polymarket, Bluesky, GitHub, and general web search — then synthesizes a grounded, citation-backed summary.

## Installation (one-time)

```bash
# Clone the skill
git clone https://github.com/mvanhorn/last30days-skill.git /tmp/last30days-skill

# Copy the skill into Reasonix
cp -r /tmp/last30days-skill/skills/last30days ~/.reasonix/skills/last30days

# Install dependencies
cd ~/.reasonix/skills/last30days
python3 -m pip install -r scripts/requirements.txt 2>/dev/null || true

# Verify
python3 scripts/last30days.py --diagnose
```

## Usage

Invoke in Reasonix chat with:
```
/last30days <topic>
```

Or with options:
```
/last30days "best mechanical keyboards 2025" --search=reddit,youtube
/last30days "AI news" --days=7 --deep
```

## Available Sources

### Free (no API key required)
- **Reddit** — Public posts and comment threads
- **Hacker News** — Tech discussions via Algolia API
- **Polymarket** — Prediction market data
- **YouTube** — Search + transcripts (requires `yt-dlp`: `brew install yt-dlp`)
- **Bluesky** — Public posts (requires `BSKY_HANDLE` + `BSKY_APP_PASSWORD`)
- **GitHub** — Repository and issue search

### Requires API Key
- **X/Twitter** — `XAI_API_KEY` or browser cookies
- **TikTok** — `SCRAPECREATORS_API_KEY` (100 free credits)
- **Instagram** — `SCRAPECREATORS_API_KEY`
- **Web Search** — `BRAVE_API_KEY`

Set API keys as environment variables or in `~/.reasonix/.env`:
```bash
export XAI_API_KEY="xai-..."
export BRAVE_API_KEY="BSA..."
export SCRAPECREATORS_API_KEY="sc_..."
```

## What It Returns

A research brief with:
- **Executive summary** (2-3 paragraphs)
- **Key themes** extracted across platforms
- **Notable posts** with engagement metrics (upvotes, retweets, comments)
- **Sentiment breakdown** per platform
- **Source citations** (URLs, dates, engagement counts)
- **Confidence score** based on source diversity and recency

## Tips

- First run is slow (~90s for full sweep) — results are cached
- Narrow with `--search=reddit,youtube` for faster focused research
- Use `--deep` for thorough analysis (slower but more comprehensive)
- Run `--diagnose` to check which sources are configured
- Skill auto-detects browser cookies for X/Twitter if you're logged in

## Troubleshooting

```bash
# Check Python version (needs 3.12+)
python3 --version

# Install yt-dlp for YouTube
brew install yt-dlp

# Full diagnostic
cd ~/.reasonix/skills/last30days
python3 scripts/last30days.py --diagnose
```

## Upstream

- **Repo**: https://github.com/mvanhorn/last30days-skill
- **Author**: mvanhorn
- **License**: MIT
- **Version**: 3.3.2

---
name: crawl4ai-local
description: "Crawl and extract content from web pages using the local Crawl4AI instance at http://192.168.1.214:11235. Get cleaned HTML, markdown, fit-text, screenshots, or structured LLM extraction. Use when you need to read web pages programmatically."
argument-hint: "crawl4ai-local https://example.com | crawl4ai-local https://docs.python.org --mode markdown"
allowed-tools: Bash
user-invocable: true
metadata:
  tags:
    - crawl
    - web
    - scraper
    - extraction
    - local
    - lan
  requires:
    bins:
      - curl
---

# Crawl4AI Local Crawler

Use the local Crawl4AI instance running on the home LAN for web page crawling and content extraction.

**Base URL**: `http://192.168.1.214:11235`  
**Health**: `GET /health`  
**API Docs**: `http://192.168.1.214:11235/docs`

## Quick Crawl (single URL)

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["https://example.com"], "priority": 10}'
```

### Response fields

| Field | Description |
|-------|-------------|
| `html` | Raw HTML |
| `cleaned_html` | Cleaned HTML (scripts, styles removed) |
| `fit_html` | Semantic-fit HTML (most relevant content) |
| `markdown` | Markdown version (use `"output_format": "markdown"`) |
| `media.images` | Extracted image URLs |
| `media.videos` | Extracted video URLs |
| `links.external` | External links with text and scores |

## Crawl with Options

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{
    "urls": ["https://example.com"],
    "priority": 10,
    "extract_markdown": true,
    "bypass_cache": false,
    "timeout": 30000
  }'
```

## Extract as Markdown

```bash
curl -s -X POST http://192.168.1.214:11235/md \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}'
```

## Screenshot a Page

```bash
curl -s -X POST http://192.168.1.214:11235/screenshot \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}' | base64 -d > screenshot.png
```

## Structured Extraction (LLM)

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{
    "urls": ["https://example.com/products"],
    "priority": 10,
    "extraction_config": {
      "type": "llm",
      "params": {
        "instruction": "Extract all product names and prices as a JSON array"
      }
    }
  }'
```

## Parse Results with Python

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls":["https://example.com"],"priority":10}' \
  | python3 -c "
import sys, json
d = json.load(sys.stdin)
for r in d['results']:
    if r['success']:
        print(r.get('cleaned_html', '')[:500])
"
```

## Notes

- Crawl4AI uses headless browsers — JavaScript-rendered pages work.
- Cache is enabled by default; use `"bypass_cache": true` for fresh content.
- Large pages may take 10-30 seconds.

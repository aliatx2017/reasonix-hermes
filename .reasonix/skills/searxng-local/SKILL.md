---
name: searxng-local
description: "Search the web using the local SearXNG instance at http://192.168.1.214:30053. Returns JSON results from multiple engines (DuckDuckGo, Startpage, Wikipedia, etc.). Use when you need to search the web privately."
argument-hint: "searxng-local <query> | searxng-local latest ai news"
allowed-tools: Bash
user-invocable: true
metadata:
  tags:
    - search
    - web
    - local
    - private
    - lan
  requires:
    bins:
      - curl
---

# SearXNG Local Search

Use the local SearXNG instance running on the home LAN for private web searches.

**Endpoint**: `http://192.168.1.214:30053/search?q=<query>&format=json`

## Usage

```bash
curl -s 'http://192.168.1.214:30053/search?q=<URL-ENCODED-QUERY>&format=json'
```

## Parameters

| Param | Description |
|-------|-------------|
| `q` | Search query (URL-encoded) |
| `format` | `json` for machine-readable output |
| `categories` | Optional: `general`, `news`, `videos`, `images`, `files`, `map`, `music`, `science`, `social+media` |
| `engines` | Optional: comma-separated engine list (e.g. `google,wikipedia,duckduckgo`) |
| `pageno` | Page number (default: 1) |
| `language` | Language code (e.g. `en`, `zh`, `auto`) |
| `time_range` | `day`, `week`, `month`, `year` or empty |

## Response Format

```json
{
  "query": "search term",
  "results": [
    {
      "url": "https://...",
      "title": "Page Title",
      "content": "Snippet text...",
      "engine": "duckduckgo",
      "score": 1.0,
      "category": "general"
    }
  ],
  "answers": [],
  "corrections": [],
  "suggestions": []
}
```

## Examples

**Basic search:**
```bash
curl -s 'http://192.168.1.214:30053/search?q=rust+programming+language&format=json'
```

**News from last week:**
```bash
curl -s 'http://192.168.1.214:30053/search?q=openai+announcements&format=json&categories=news&time_range=week'
```

**Parse with jq for titles only:**
```bash
curl -s 'http://192.168.1.214:30053/search?q=golang+generics&format=json' | python3 -c "import sys,json; [print(r['title']) for r in json.load(sys.stdin)['results'][:10]]"
```

## Notes

- This instance may have rate limits on certain engines (Google often hits CAPTCHA).
- SearXNG aggregates results from multiple engines and deduplicates.
- All searches go through the local LAN — no external tracking.

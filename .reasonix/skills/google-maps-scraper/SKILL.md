---
name: google-maps-scraper
description: "Scrape Google Maps for business listings using the local Google Maps Scraper at http://192.168.1.214:8080. Search by keywords, location, language, and zoom level. Get business names, addresses, phones, websites, ratings, and reviews."
argument-hint: "google-maps-scraper 'coffee shops in Athens' | google-maps-scraper 'hotels near Manhattan' --lang en --depth 2"
allowed-tools: Bash
user-invocable: true
metadata:
  tags:
    - maps
    - google-maps
    - scraper
    - business
    - local
    - lan
    - geolocation
  requires:
    bins:
      - curl
---

# Google Maps Scraper (Local)

Scrape Google Maps business listings using the local instance at `http://192.168.1.214:8080`.

**Base URL**: `http://192.168.1.214:8080`  
**API Docs**: `http://192.168.1.214:8080/api/docs`  
**GitHub**: `https://github.com/gosom/google-maps-scraper`

## Create a Scrape Job

```bash
JOB=$(curl -s -X POST http://192.168.1.214:8080/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Coffee shops Athens",
    "keywords": ["coffee shops in athens"],
    "lang": "en",
    "zoom": 14,
    "depth": 1,
    "max_time": 3600
  }')
JOB_ID=$(echo "$JOB" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "Job created: $JOB_ID"
```

### Request Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Human-readable name for the job |
| `keywords` | string[] | Search terms (one per search area) |
| `lang` | string | Language code: `en`, `el`, `fr`, `de`, etc. |
| `zoom` | int | Map zoom level (1-21). Higher = more granular. Default: 14 |
| `depth` | int | Search depth. 1 = single area, higher = recursively subdivides. Default: 1 |
| `max_time` | int | Max scraping time in seconds. 0 = unlimited |

## Check Job Status

```bash
curl -s http://192.168.1.214:8080/api/v1/jobs/$JOB_ID | python3 -m json.tool
```

Response includes job status (`pending`, `running`, `finished`, `cancelled`, `error`), creation time, and result count.

## List All Jobs

```bash
curl -s http://192.168.1.214:8080/api/v1/jobs | python3 -c "
import sys, json
for j in json.load(sys.stdin):
    print(f'{j[\"id\"][:8]}... {j[\"status\"]:10s} {j[\"name\"]}')
"
```

## Export Results as JSON

```bash
curl -s http://192.168.1.214:8080/api/v1/jobs/$JOB_ID/results/json \
  | python3 -c "
import sys, json
for item in json.load(sys.stdin):
    name = item.get('name', 'N/A')
    addr = item.get('address', 'N/A')
    phone = item.get('phone', 'N/A')
    rating = item.get('rating', '?')
    reviews = item.get('reviews', 0)
    print(f'{name}')
    print(f'  {addr}')
    print(f'  ☎ {phone} | ★{rating} ({reviews} reviews)')
    print()
" | head -50
```

## Response Format (per business)

```json
{
  "name": "Business Name",
  "address": "123 Main St, City",
  "phone": "+1-555-0123",
  "website": "https://example.com",
  "rating": 4.5,
  "reviews": 127,
  "place_id": "ChIJ...",
  "categories": ["Coffee Shop", "Cafe"],
  "location": { "lat": 37.98, "lng": 23.72 },
  "hours": { "Monday": "8AM-10PM", ... }
}
```

## Practical Examples

**Find restaurants near a location:**
```bash
curl -s -X POST http://192.168.1.214:8080/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"name":"Athens restaurants","keywords":["restaurants near Syntagma Square Athens"],"lang":"en","zoom":16,"depth":1,"max_time":600}'
```

**Competitor research (multiple keywords):**
```bash
curl -s -X POST http://192.168.1.214:8080/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d '{"name":"Competitor sweep","keywords":["coworking spaces downtown","shared offices near me"],"lang":"en","zoom":14,"depth":2,"max_time":3600}'
```

## Notes

- Scraping is rate-limited by Google. Jobs with high depth/zoom take longer.
- Results include `place_id` for stable referencing across runs.
- The web UI at `http://192.168.1.214:8080` provides a visual job manager.

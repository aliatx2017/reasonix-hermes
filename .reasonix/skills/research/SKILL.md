---
name: research
description: "Conduct preliminary research on a topic — generate structured research outline with items and field definitions, saved as YAML for follow-up deep dive."
allowed-tools: Bash, Read, Write, Glob, WebSearch, Task, AskUserQuestion
---

# Research Skill — Preliminary Research

## Trigger
`/research <topic>`

## Workflow

### Step 1: Generate Initial Framework from Model Knowledge
Based on the topic, use your existing knowledge to generate:
- Main research objects/items list in this domain
- Suggested research field framework (dimensions to collect for each item)

Ask the user: "Does this framework look right? Need to add/remove items or change fields?"

### Step 2: Web Search Supplement with SearXNG
Ask the user for a time range (e.g., "last 6 months", "since 2024", "unlimited").

Use **the local SearXNG instance** as your primary search tool — it aggregates results from DuckDuckGo, Wikipedia, Startpage, and more, returning structured JSON with URLs, titles, snippets, and scores:

```bash
curl -s 'http://192.168.1.214:30053/search?q=<URL-ENCODED-QUERY>&format=json&time_range=<time_range>'
```

Parameters:
- `q` — your search query (URL-encoded)
- `format` — `json` always
- `categories` — `general`, `news`, `science`, `social+media` as appropriate
- `time_range` — `day`, `week`, `month`, `year` (from user)
- `language` — `en` for English results

Search for:
- The topic itself (what are the key players/works/items?)
- Alternative terms or related areas
- Recent developments within the time range

Parse the JSON results with `bash` + `python3` or `jq` to extract titles, URLs, and snippets, then use that to identify supplementary items and field dimensions.

If SearXNG is unreachable, fall back to:
- `https://en.wikipedia.org/wiki/Special:Search?search={topic}&go=Go`
- `https://hn.algolia.com/api/v1/search?query={topic}&tags=story&hitsPerPage=20`

### Step 3: Deep-dive into Promising Sources with Crawl4AI
For any particularly promising URLs found in Step 2, use **the local Crawl4AI instance** to extract full content:

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["<url>"], "priority": 10, "extract_markdown": true}'
```

This returns cleaned markdown content from JavaScript-rendered pages, giving you richer material for identifying items and field dimensions than plain web_fetch.

### Step 4: Ask for Existing Definitions
Ask the user: "Do you have any existing field definitions or research frameworks to incorporate?"

### Step 5: Generate Outline Files
Merge everything into two YAML files using `write_file`:

**`outline.yaml`**:
```yaml
topic: <research topic>
items:
  - name: <item name>
    description: <brief description>
  - name: <item name>
    description: <brief description>
execution:
  batch_size: 3
  items_per_agent: 1
  output_dir: ./results
```

**`fields.yaml`**:
```yaml
fields:
  - category: "<category name>"
    fields:
      - name: <field_name>
        description: <field description>
        detail_level: brief|moderate|detailed
```

### Step 6: Confirm
Show the user the generated files and point to follow-up commands:
- `/research-add-items` — add more research items
- `/research-add-fields` — add more field definitions
- `/research-deep` — start deep research with parallel agents
- `/research-report` — generate final report from results

## Output Structure
```
{topic_slug}/
  ├── outline.yaml    # items list + execution config
  └── fields.yaml     # field definitions
```
Where `{topic_slug}` is a directory-safe version of the topic name in the current working directory.

## Notes
- Save files in the workspace root or session directory
- Use `write_file` to create YAML files
- Prefer **searxng-local** for search (structured multi-engine results)
- Use **crawl4ai-local** for deep page content when you need full article text
- Fall back to `web_fetch` when the local services are unreachable
- Keep items scoped to comparable entities (companies, frameworks, papers, tools)
- Keep fields consistent across all items for apples-to-apples comparison

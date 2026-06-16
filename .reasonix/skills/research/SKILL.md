---
name: research
description: "End-to-end research pipeline — generate outline, launch parallel subagents to deep-research each item, produce a comprehensive markdown report, and publish to Discord."
---

# Research — End-to-End Pipeline

## Trigger
`/research <topic>`

This single command runs the full pipeline: outline → deep research → report → Discord publish. One approval gate at the outline stage; everything else is automatic.

---

## Phase 1: Generate Outline

Based on the topic, use your existing knowledge + web search to generate the research framework.

### 1a. Initial Framework from Model Knowledge
Based on the topic, generate:
- Main research objects/items list in this domain
- Suggested research field framework (dimensions to collect for each item)

Ask the user: "Does this framework look right? Need to add/remove items or change fields?"

### 1b. Web Search Supplement with SearXNG
Ask the user for a time range (e.g., "last 6 months", "since 2024", "unlimited").

Use **the local SearXNG instance** as your primary search tool:

```bash
curl -s 'http://192.168.1.214:30053/search?q=<URL-ENCODED-QUERY>&format=json&time_range=<time_range>'
```

Parameters: `q`, `format=json`, `categories=general,news,science,social+media`, `time_range`, `language=en`.

Search for the topic itself, alternative terms, and recent developments within the time range. Parse JSON results with `python3` or `jq` to extract titles, URLs, and snippets.

Fallback if SearXNG is unreachable:
- `https://en.wikipedia.org/wiki/Special:Search?search={topic}&go=Go`
- `https://hn.algolia.com/api/v1/search?query={topic}&tags=story&hitsPerPage=20`

### 1c. Deep-dive into Promising Sources with Crawl4AI
For promising URLs found in 1b, use **the local Crawl4AI instance**:

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["<url>"], "priority": 10, "extract_markdown": true}'
```

This returns cleaned markdown from JavaScript-rendered pages.

### 1d. Ask for Existing Definitions
Ask the user: "Do you have any existing field definitions or research frameworks to incorporate?"

### 1e. Write Outline Files
Merge everything into two YAML files:

**`{topic_slug}/outline.yaml`**:
```yaml
topic: <research topic>
items:
  - name: <item name>
    description: <brief description>
execution:
  batch_size: 3
  items_per_agent: 1
  output_dir: ./results
```

**`{topic_slug}/fields.yaml`**:
```yaml
fields:
  - category: "<category name>"
    fields:
      - name: <field_name>
        description: <field description>
        detail_level: brief|moderate|detailed
```

---

## Phase 2: Approval Gate

Present the outline to the user:

> "Research framework ready: **N items**, **M fields** across **K categories**. Output directory: `{topic_slug}/`. Does this look right?"

**Wait for explicit approval** — the user must say "go", "looks good", "proceed", "approved", or similar. Do not proceed to Phase 3 without this.

If the user wants changes, update `outline.yaml` and/or `fields.yaml` and re-confirm. Use `/research-add-items` or `/research-add-fields` if they want to expand the scope.

---

## Phase 3: Deep Research — Parallel Subagents

Read `outline.yaml` and `fields.yaml` to get:
- `topic`, `items` list, `execution.batch_size`, `execution.output_dir`
- Full field definitions from fields.yaml

### 3a. Check for Resume
If `{output_dir}/` already contains `.json` files, skip already-completed items. For each existing JSON, verify all fields are populated; if incomplete, re-research it.

### 3b. Launch All Batches
Divide remaining items into batches of `batch_size`. **Launch all batches at once** — no per-batch approval needed. Use the `task` tool with `batch` mode:

```
task(batch=[
  {prompt: "<subagent prompt 1>", description: "Research: <item1>"},
  {prompt: "<subagent prompt 2>", description: "Research: <item2>"},
  ...
], max_steps=30, tools=["bash","write_file","read_file","glob","web_fetch"])
```

**Always set `max_steps` to at least 30** — each subagent needs multiple rounds (search → crawl → extract → write JSON). The default subagent step budget (parent_max_steps/2 ≈ 10) is too low.

Each subagent prompt template:

```
You are researching: {item_name}

Description: {item_description}

## Research Tools Available

### 1. SearXNG (local search — find relevant sources)
curl -s 'http://192.168.1.214:30053/search?q={query}&format=json' | python3 -c "import sys,json; [print(r['title'],r['url']) for r in json.load(sys.stdin)['results'][:10]]"

### 2. Crawl4AI (local crawler — extract full page content)
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["<url>"], "priority": 10, "extract_markdown": true}'

### 3. web_fetch (fallback)
Use when the local services are unreachable.

## Field Definitions to Populate
{fields_text_block}

## Output
Write a JSON object to {output_path} using write_file:

{{
  "name": "{item_name}",
  // ... every field from the field definitions above ...
  "uncertain": ["list", "of", "field", "names", "you", "are", "uncertain", "about"],
  "_source": ["url1", "url2", "url3"]
}}

## Rules
- Use searxng-local first to FIND sources, then crawl4ai-local to EXTRACT content
- For every field defined above, provide the best value you can find
- If a value is uncertain, still provide your best guess AND add the field name to "uncertain"
- Mark truly unfindable fields with "[not found]" — don't fabricate data
- All field values must be in English
- Include the URLs you sourced information from in "_source"
- Write the JSON result to {output_path} using write_file
- If local services are down, fall back to web_fetch
```

### 3c. Wait for All Jobs
Use `wait` to collect all subagent results. Do not proceed to Phase 4 until every job has completed.

### 3d. Validate Results
After all jobs finish:
- Count completed JSON files in output_dir
- Validate each JSON has all required fields (use `bash` + `python3` or `jq`)
- Display: "Completed X/Y items — Z with uncertain fields"

---

## Phase 4: Generate Report

### 4a. Read All Results
Use `glob` to find all `.json` files in the output directory. Read each one with `read_file`.

### 4b. Fill Gaps (Optional)
If any items have critical fields marked `[not found]`, try to fill them with a quick Crawl4AI deep-read. Update the JSON with `edit_file`.

### 4c. Write Report
Use all fields by default — no need to ask which summary fields to use. Write `{topic_slug}/report.md`:

```markdown
# Research: {topic}
Generated: {date}

## Table of Contents
1. [Item Name 1](#item-name-1) — {short comparable field}: {value} | {another}: {value}

## Item Name 1
### {Category 1}
- **{field_name}**: {value}

### {Category 2}
- **{field_name}**: {value}
...

## Comparison Table
| Field | Item 1 | Item 2 | ... |
|-------|--------|--------|-----|
| ...   | ...    | ...    | ... |
```

### Report Generation Rules
1. **Cover all fields** — every field in fields.yaml must appear
2. **Skip uncertain values** — skip fields whose name is in the item's `uncertain` array
3. **Flag missing data** — for `[not found]` fields, note "(data unavailable)"
4. **Group by category** — use the category grouping from fields.yaml
5. **Format complex values**: lists of objects as sub-lists or tables; long text as blockquotes; nested objects flattened with dot notation
6. **Sources section** — at the end of each item, include `_source` URLs
7. **Comparison table** — at the end of the report, a side-by-side comparison table with all items and a few key fields

---

## Phase 5: Publish to Discord

Run the Discord publish script:

```bash
bash .reasonix/scripts/discord-publish.sh ./{topic_slug}/report.md
```

The script auto-loads the webhook URL from `.reasonix/.discord-webhook` (gitignored). If the webhook file is missing, set these environment variables: `DISCORD_WEBHOOK_URL` (preferred) or `DISCORD_BOT_TOKEN` + `DISCORD_CHANNEL_ID`.

---

## Output Structure
```
{topic_slug}/
  ├── outline.yaml      # items list + execution config
  ├── fields.yaml       # field definitions
  ├── results/          # subagent JSON outputs (one per item)
  │   ├── item-name.json
  │   └── ...
  └── report.md         # final synthesized report
```
Where `{topic_slug}` is a directory-safe version of the topic name in the current working directory.

## Supplementary Commands
- `/research-add-items` — add more items to an existing outline
- `/research-add-fields` — add more field definitions to an existing outline

## Notes
- Save files in the workspace root or session directory
- Use `write_file` to create YAML and JSON files
- Prefer **searxng-local** for search (structured multi-engine results)
- Use **crawl4ai-local** for deep page content
- Fall back to `web_fetch` when local services are unreachable
- Keep items scoped to comparable entities (companies, frameworks, papers, tools)
- Keep fields consistent across all items for apples-to-apples comparison

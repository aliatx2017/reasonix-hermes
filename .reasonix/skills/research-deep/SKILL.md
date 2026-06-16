---
name: research-deep
description: "Deep research phase — read research outline and launch parallel subagents to investigate each item against the field schema, producing structured JSON results."
---

# Research Deep — Deep Research

## Trigger
`/research-deep`

## Workflow

### Step 1: Locate Outline
Find and read `outline.yaml` in the current working directory or workspace root.
Also read `fields.yaml` from the same directory.

Extract:
- `topic` — the research topic
- `items` — list of items to research (name + description)
- `execution.batch_size` — how many items to research in parallel (default 3)
- `execution.output_dir` — where to save JSON results (default `./results`)
- `fields` — field definitions from fields.yaml

### Step 2: Check for Resume
Check if `output_dir` exists and contains any `.json` files. Skip already-completed items so research can be resumed after interruption.

For each already-completed item, read its JSON to confirm it has all fields populated (not just a stub); if incomplete, re-research it.

### Step 3: Prepare Field Content for Subagents
Read the full `fields.yaml` and prepare the field definitions as a text block that subagents can use. Include both the category grouping and the individual field names, descriptions, and detail_levels.

### Step 4: Batch Execution with Parallel Subagents
For each batch of items (up to `batch_size` at a time):

Ask the user for approval before launching each batch.

Use the `task` tool with `batch` mode to spawn one subagent per item. Grant the subagents `Bash, Read, Write, Glob` tools (they need `bash` for curl calls to crawl4ai/searxng). Each subagent's prompt should be:

```
You are researching: {item_name}

Description: {item_description}

## Research Tools Available

### 1. SearXNG (local search — find relevant sources)
{search_syntax}

Search for this item with time-relevant queries.
Parse results with: curl -s 'http://192.168.1.214:30053/search?q={query}&format=json' | python3 -c "import sys,json; [print(r['title'],r['url']) for r in json.load(sys.stdin)['results'][:10]]"

### 2. Crawl4AI (local crawler — extract full page content)
{crawl_syntax}

For any promising URL found via SearXNG, crawl its full content:
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["<url>"], "priority": 10, "extract_markdown": true}'

This returns cleaned markdown — even from JavaScript-rendered pages.

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
- Use searxng-local first to FIND sources, then crawl4ai-local to EXTRACT content from those sources
- For every field defined above, provide the best value you can find
- If a value is uncertain, still provide your best guess AND add the field name to the "uncertain" array
- Mark truly unfindable fields with "[not found]" — don't fabricate data
- All field values must be in English
- Include the URLs you sourced information from in "_source"
- Write the JSON result to {output_path} using write_file
- If local services are down, fall back to web_fetch
```

### Step 5: Monitor Progress
After each batch completes:
- Count completed JSON files in output_dir
- Validate that each new JSON has all required fields (use bash + python3 or jq)
- Display: "Completed X/Y items — Z with uncertain fields"
- Launch next batch if more items remain

### Step 6: Summary
After all items are researched, output:
- Total items completed
- Any items with uncertain fields (list field names and their uncertain values)
- Output directory path
- "Ready for /research-report to generate the final report"

## Notes
- Use the `task` tool with `batch` mode for parallel subagents (up to 8 at a time)
- Each subagent writes its own JSON result file
- Subagents need `Bash` tool for curl calls to searxng/crawl4ai
- Subagents do NOT have access to the parent's workspace files — pass field content explicitly in their prompt
- Keep batch_size aligned with available context and rate limits
- Resume support: skip items with valid complete JSON, re-research items with missing fields

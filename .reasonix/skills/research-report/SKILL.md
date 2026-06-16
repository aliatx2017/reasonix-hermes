---
name: research-report
description: "Summarize deep research results into a comprehensive markdown report covering all fields, skipping uncertain values."
---

# Research Report — Summary Report

## Trigger
`/research-report`

## Workflow

### Step 1: Locate Results
Find `outline.yaml` and `fields.yaml` in the current working directory or workspace root.
Read the `execution.output_dir` from outline.yaml (default `./results`).

### Step 2: Read All Results
Use `glob` to find all `.json` files in the output directory.
Read each JSON result file using `read_file`.

### Step 3: Fill Gaps with Crawl4AI (Optional)
If any items have critical fields marked `[not found]` in their uncertain array, offer to deep-research them using **crawl4ai-local**:

```bash
curl -s -X POST http://192.168.1.214:11235/crawl \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["<url>"], "priority": 10, "extract_markdown": true}'
```

Extract the missing field values and update the JSON with `edit_file`.

### Step 4: Ask User About Summary Fields
Ask the user: "Which fields should appear in the table of contents summary line for each item?"
Suggest fields that contain short, comparable values (e.g., dates, star counts, scores, ratings).

### Step 5: Generate Report
Write `report.md` using `write_file` with this structure:

```markdown
# Research: {topic}
Generated: {date}

## Table of Contents
1. [Item Name 1](#item-name-1) — {summary_field_1}: {value} | {summary_field_2}: {value}

## Item Name 1
### {Category 1}
- **{field_name}**: {value}
- **{field_name}**: {value}

### {Category 2}
- **{field_name}**: {value}
...
```

### Report Generation Rules

1. **Cover all fields** — every field defined in fields.yaml must appear in the report
2. **Skip uncertain values** — skip any field whose value contains `[uncertain]` or whose name is in the item's `uncertain` array
3. **Flag missing data** — for `[not found]` fields, note "(data unavailable)" instead of omitting
4. **Group by category** — use the category grouping from fields.yaml
5. **Format complex values**:
   - Lists of objects: format as sub-list or table
   - Long text: use blockquotes for readability
   - Nested objects: flatten with dot notation or sub-sections
6. **Sources section** — at the end of each item, include the `_source` URLs
7. **Summary table** — at the end of the report, add a comparison table with all items and a few key fields for side-by-side comparison

### Step 6: Confirm
Show the report location to the user and optionally open it.

## Output
- `report.md` — in the same directory as outline.yaml

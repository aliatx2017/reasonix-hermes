---
name: database-helper
description: SQL query optimization, schema design review, migration generation, ORM patterns.
runAs: inline
allowedTools:
  - read_file
  - grep
  - glob
  - bash
  - edit_file
---

# Database Helper

SQL query optimization, schema design, migration generation, and ORM best practices.

## Schema Review Checklist

### Naming
- Tables: plural, snake_case (`users`, `order_items`)
- Columns: snake_case, descriptive (`created_at`, `updated_at`, `deleted_at`)
- Primary keys: `id` (UUID or bigint)
- Foreign keys: `<table>_id` (e.g. `user_id`)
- Indexes: `idx_<table>_<column>` (e.g. `idx_users_email`)

### Data Types
- Use the smallest type that fits: `smallint` over `int` when possible.
- Timestamps: `timestamptz` (PostgreSQL) or `datetime` with UTC convention.
- Money: `numeric(19,4)` or `bigint` (cents), never `float`.
- Boolean over `tinyint` for flags.
- `text` over `varchar(255)` unless there's a genuine length constraint.

### Constraints
- Every table has a primary key.
- Foreign keys are declared (not just convention).
- Unique constraints on natural keys (`email`, `slug`).
- `NOT NULL` on all columns unless null has a semantic meaning.
- Default values where sensible (`created_at DEFAULT now()`).

## Query Optimization

### Before Optimizing
1. Run `EXPLAIN ANALYZE` on the slow query.
2. Check: sequential scan vs index scan?
3. Check: are the right indexes being used?

### Common Fixes

| Problem | Fix |
|---------|-----|
| Sequential scan on large table | Add a covering index |
| N+1 queries | Use a JOIN or batch load (`WHERE id IN (...)`) |
| Missing index for WHERE | Match index column order to query filters |
| Index not used due to function | Avoid `WHERE LOWER(name) = ...`; use a functional index or `citext` |
| Large OFFSET | Use cursor-based pagination (`WHERE id > ?`) |
| Lock contention | Use `SELECT ... FOR UPDATE SKIP LOCKED` for queues |

## Migration Patterns

- Every migration is **reversible** (write the `down` script).
- Back-fill large tables in batches, not in a single transaction.
- Add indexes concurrently (`CREATE INDEX CONCURRENTLY` in PostgreSQL).
- Never rename a column — add a new one, backfill, drop the old one.
- Test migrations on a copy of production data before deploying.

## Go + SQL

- Use parameterized queries (`$1`, `$2`, not `fmt.Sprintf`).
- `sqlx` or `pgx` over `database/sql` for ergonomics.
- `SELECT *` is fine in Go (it scans into struct fields), but prefer explicit columns for clarity.
- Use `context.Context` for query timeouts: `db.QueryContext(ctx, ...)`.

---
name: api-design
description: REST API design patterns: resource naming, status codes, pagination, filtering, error responses, versioning.
runAs: inline
allowedTools:
  - read_file
  - write_file
  - edit_file
  - grep
  - glob
---

# API Design

REST API design patterns and review. Use when designing new endpoints or reviewing existing ones.

## Resource Naming

- Use **plural nouns**: `/users`, `/orders`, `/products`
- Nested resources: `/users/{id}/orders`, not `/getUserOrders`
- Use **kebab-case** in URLs: `/order-items`, not `/orderItems`
- Avoid verbs in URLs; use HTTP methods instead:
  - `GET /users` (list), `POST /users` (create)
  - `GET /users/{id}` (read), `PATCH /users/{id}` (update), `DELETE /users/{id}` (delete)

## Status Codes

| Code | When |
|------|------|
| 200 | Successful GET, PATCH, DELETE |
| 201 | Successful POST (include `Location` header) |
| 204 | Successful DELETE with no body |
| 400 | Bad input (validation error) |
| 401 | Missing or invalid auth |
| 403 | Authenticated but not authorized |
| 404 | Resource not found |
| 409 | Conflict (duplicate, stale version) |
| 422 | Unprocessable — semantic error |
| 429 | Rate limited |
| 500 | Internal server error (never expose details) |

## Pagination

Use **cursor-based** pagination for large datasets, **offset-based** for small/admin UIs:

```json
{
  "data": [...],
  "pagination": {
    "cursor": "eyJpZCI6MTAwfQ",
    "has_more": true,
    "total": 243
  }
}
```

Query params: `?limit=20&cursor=<token>` or `?page=1&per_page=20`.

## Filtering & Sorting

- Filter: `GET /users?status=active&role=admin`
- Sort: `GET /users?sort=-created_at,name` (prefix `-` for descending)
- Search: `GET /users?q=alex` (full-text search)
- Never expose arbitrary SQL through query params.

## Error Responses

Always return a consistent error body:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email is required",
    "details": [
      {"field": "email", "reason": "must be a valid email address"}
    ]
  }
}
```

## Versioning

- Prefer **header-based**: `Accept: application/vnd.api-v2+json`
- URL-based as fallback: `/v2/users`
- Never ship breaking changes without a new version.
- Deprecate old versions with `Sunset` and `Deprecation` headers.

## Review Checklist

1. Are resource names plural and kebab-case?
2. Are HTTP methods used correctly (no `GET /deleteUser`)?
3. Are error responses consistent?
4. Is pagination present on list endpoints?
5. Are rate limits and auth checks in place?

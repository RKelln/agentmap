<!-- AGENT:NAV
purpose:error handling policy; retry strategy; circuit breaker configuration
nav[2]{s,n,name,about}:
10,18,##Error Codes,HTTP status code mapping; structured error payload schema
29,20,##Retry Policy,exponential backoff parameters; idempotency requirements
-->

# Error Policy

All API errors return a structured JSON payload alongside the
appropriate HTTP status code. Clients must handle errors gracefully
and apply the retry policy described in this document.

## Error Codes

The API uses standard HTTP status codes. Error responses always
include a JSON body with the following fields:

- `code` — machine-readable error identifier (string)
- `message` — human-readable description suitable for logging
- `request_id` — unique identifier for the failed request; include
  this in support requests

Common codes and their meanings:

| HTTP | Code | Meaning |
|------|------|---------|
| 400 | invalid_request | Malformed request or missing required field |
| 401 | unauthorized | Missing or expired bearer token |
| 403 | forbidden | Token lacks required scope |
| 404 | not_found | Resource does not exist |
| 429 | rate_limited | Request rate limit exceeded |
| 500 | internal_error | Unexpected server error; safe to retry |

## Retry Policy

Retry only requests that failed with a 429 or 5xx status code.
Do not retry 4xx errors other than 429.

Use exponential backoff with jitter:

    wait = min(base * 2^attempt, max_wait) + random(0, jitter)

Default parameters: `base=500ms`; `max_wait=30s`; `jitter=250ms`.
Maximum retry attempts: 3. After 3 failures surface the error to
the caller.

All write operations (POST; PUT; DELETE) are idempotent when a
client-generated `Idempotency-Key` header is included. Always include
this header before the first attempt so retries are safe.

<!-- AGENT:NAV
purpose:~rate limit tiers; burst allowance; quota tracking
nav[3]{s,n,name,about}:
9,41,#Rate Limiting,~rate limits per access token; 429 response behavior
15,21,##Rate Limits,~per-endpoint rate limit tiers; response headers
36,14,##Burst Allowance,~burst credit system; replenishment rate
-->

# Rate Limiting

All API endpoints are subject to rate limiting. Limits are enforced
per access token. Exceeding a limit returns a 429 response with a
`Retry-After` header indicating when the next request may be made.

## Rate Limits

Limits vary by endpoint tier. Most read endpoints allow higher
throughput than write endpoints.

| Tier | Endpoints | Limit |
|------|-----------|-------|
| read | GET requests | 1000 req/min |
| write | POST; PUT; PATCH | 100 req/min |
| delete | DELETE requests | 20 req/min |
| admin | Admin-scoped endpoints | 10 req/min |

Rate limit status is returned in every response via headers:

- `X-RateLimit-Limit` — the limit for this endpoint tier
- `X-RateLimit-Remaining` — requests remaining in the current window
- `X-RateLimit-Reset` — Unix timestamp when the window resets

Clients should monitor these headers and slow down proactively when
`X-RateLimit-Remaining` drops below 10% of the limit.

## Burst Allowance

Each token accumulates burst credits when making requests below the
sustained rate. Credits replenish at 1 credit per second up to a
maximum of 60 credits for read-tier endpoints and 20 credits for
write-tier endpoints.

A burst allows a short spike above the sustained rate by consuming
stored credits. Once credits are exhausted the sustained rate applies.

Burst credits do not carry over between rate limit windows. They are
reset when the window resets. The current burst credit balance is
not exposed in response headers; it is computed server-side based on
recent request history.

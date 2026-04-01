<!-- AGENT:NAV
purpose:circuit;open;policy;retry;service;attempts;backoff;breaker
-->

# Error Policy

Retry and circuit-breaker rules for all service-to-service calls.

## Retry Policy

Exponential backoff with jitter. Maximum 5 attempts.

## Circuit Breakers

Open after 10 consecutive failures. Half-open after 30 seconds.

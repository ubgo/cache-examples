# 04-decorators

Everything is a `cache.Cache`, so the resilience/observability decorators stack freely.

## Run

```sh
go run .
```

## What it demonstrates

Composes `Instrument( AuditLog( Bulkhead( Retry( CircuitBreaker( cache-mem ) ) ) ) )`:

- `NewCircuitBreaker` — opens after N consecutive backend failures, fails fast with `ErrCircuitOpen`.
- `NewRetry` — exponential backoff on transient errors (a miss is *not* transient).
- `NewBulkhead` — caps concurrent in-flight ops so one caller cannot exhaust the backend.
- `NewAuditLog` — emits an `AuditEvent` for every mutation (reads are not audited); the trail is printed.
- `Instrument` — counts hits/misses/sets/deletes and merges them into `Stats()` on top of the adapter snapshot.

A cache miss flows through the whole stack without tripping the breaker (it is a normal outcome, not a failure).

## Expected output

The audit event list, then the merged stats line, ending in `OK`, exit code 0.

# 01-basic

The core `cache.Cache` API on the in-process `cache-mem` backend. No HTTP, no external services.

## Run

```sh
go run .
```

## What it demonstrates

- `memcache.New(...)` constructs an in-process cache; `defer c.Close()`.
- `Set` / `Get` round-trip, `Has` presence check, `TTL` introspection.
- The `cache.ErrNotFound` contract: a miss returns `(nil, cache.ErrNotFound)`, never `(nil, nil)` — check it with `errors.Is`.
- `Del` removes a key; a subsequent `Get` is `ErrNotFound`.

## Expected output

Lines for each operation ending in `OK`, exit code 0.

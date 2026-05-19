# 02-remember

`cache.Remember` — return the cached value, or single-flight a loader on miss and store it — with every production caching-pattern option.

## Run

```sh
go run .
```

## What it demonstrates

- **Single-flight**: 50 concurrent `Remember` calls for one key collapse to a single loader run; a later cached read does not call the loader.
- **`WithNegativeTTL`**: a loader returning `ErrNotFound` is cached so an absent key does not re-run an expensive lookup every request.
- **`WithRefreshAhead`**: a hot key refreshes in the background once a fraction of its TTL elapses.
- **`WithStaleWhileRevalidate`**: after expiry the stale value is served immediately while one background load refreshes it.
- **`WithStaleIfError`**: if the loader fails after expiry, the last good value is served instead of erroring.
- **`WithJitter`**: the stored TTL gets +/- noise so a batch written together does not all expire simultaneously.

Loader-call counters are printed so the dedupe / refresh behaviour is visible.

## Expected output

One line per pattern, ending in `OK`, exit code 0.

# 08-cluster

`cache-cluster`: peer-aware distribution (consistent-hash ring + single-flight HTTP peer fill) — the groupcache pattern through the `ubgo/cache` interface.

Three nodes run in one process, each fronted by an `httptest` server, so no real network topology is needed.

## Run

```sh
go run .
```

## What it demonstrates

- A consistent-hash `Ring` assigns `user:42` to exactly one owning node (printed).
- 30 concurrent `Get`s for that key are issued across all 3 nodes. Non-owners proxy the request to the owner over HTTP; the owner fills the miss via the loader, deduped by single-flight.
- The loader-call counter proves the hot key is loaded **exactly once cluster-wide** — not once per node, not once per concurrent request.

## Expected output

The owner line, the `loader called 1 time(s)` line, the per-node value, then `OK`, exit code 0.

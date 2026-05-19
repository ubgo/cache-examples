# cache-examples


[![test](https://github.com/ubgo/cache-examples/actions/workflows/test.yml/badge.svg)](https://github.com/ubgo/cache-examples/actions/workflows/test.yml) [![tag](https://img.shields.io/github/v/tag/ubgo/cache-examples?sort=semver)](https://github.com/ubgo/cache-examples/tags) [![license](https://img.shields.io/badge/license-source--available-blue)](./LICENSE) ![Go](https://img.shields.io/badge/go-1.24-00ADD8?logo=go)

Runnable example applications for the [`github.com/ubgo/cache`](https://github.com/ubgo/cache) family — in-memory, Redis, tiered, cluster, codecs, decorators, locker, observability, and the admin endpoint.

Each example is a self-contained Go module under its own subdirectory. Every example runs to completion and exits 0 with **no external services**: Redis is provided in-process by [`miniredis`](https://github.com/alicebob/miniredis), cluster/admin use `httptest`, and OpenTelemetry uses a manual reader. Clone, `cd`, and run `go run .`.

## Quick start

```sh
git clone https://github.com/ubgo/cache-examples.git
cd cache-examples/01-basic
go run .
```

## Examples

| Example | What it shows |
|---|---|
| [`01-basic`](./01-basic) | `cache-mem`: Set / Get / Has / TTL / Del, `ErrNotFound` handling, `defer Close`. |
| [`02-remember`](./02-remember) | `cache.Remember` with a loader: single-flight dedupe, `WithRefreshAhead`, `WithStaleWhileRevalidate`, `WithStaleIfError`, `WithNegativeTTL`, `WithJitter`. |
| [`03-generics`](./03-generics) | `cache.GetT` / `cache.SetT` and `cache.NewTyped[User]` typed view. |
| [`04-decorators`](./04-decorators) | Stack `NewCircuitBreaker` + `NewRetry` + `NewBulkhead` + `NewAuditLog` + `Instrument` over `cache-mem`; print audit events and merged stats. |
| [`05-tiered`](./05-tiered) | `cache-tiered` L1 (`cache-mem`) + L2 (`cache-redis` over miniredis) + `WithInvalidation`: read-promotion and cross-instance invalidation. |
| [`06-codecs`](./06-codecs) | `contrib/codec-msgpack`, `contrib/codec-zstd` wrapping JSON, and `cache.EncryptedCodec` (AES-GCM) via `cache.WithCodec`. |
| [`07-locker`](./07-locker) | `cache.NewLock` cron-singleton: two lockers contend, one wins, `Refresh`, `Release`. |
| [`08-cluster`](./08-cluster) | `cache-cluster`: 3 in-process nodes via `httptest`, owner routing + single-flight peer fill (loader runs once cluster-wide). |
| [`09-observability`](./09-observability) | `contrib/cache-prom` (gather + print a metric) and `contrib/cache-otel` (manual reader) via `cache.Instrument`. |
| [`10-admin`](./10-admin) | `cache/admin`: mount `Handler` on an `httptest` server, GET `/cache/stats` and `/cache/key`, show auth-gated evict. |

## Compatibility

Requires Go 1.24 or later. Every example pins the `ubgo/cache` family at `v0.1.0`.

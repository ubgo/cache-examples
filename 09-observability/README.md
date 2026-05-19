# 09-observability

The core ships a zero-dependency `ObsHooks` seam — it never imports Prometheus or OpenTelemetry. The `contrib` modules implement the producer side; `cache.Instrument` wires hooks onto any backend.

Fully in-process: no scrape endpoint, no collector.

## Run

```sh
go run .
```

## What it demonstrates

- **`contrib/cache-prom`**: register collectors on a `prometheus.Registry`, drive traffic (2 sets, 1 hit, 1 miss), then `Gather()` and print the `cache_ops_total` series by `op`/`result`. A miss is classified `result="miss"`, not `error`.
- **`contrib/cache-otel`**: build instruments on a manual `sdkmetric.Reader`, drive the same traffic, `Collect()`, and print the `cache.ops` counter data points.

## Expected output

A Prometheus section and an OpenTelemetry section listing the op counters, ending in `OK`, exit code 0.

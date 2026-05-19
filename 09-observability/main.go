// Example 09-observability — export cache metrics without the core
// importing Prometheus or OpenTelemetry.
//
// The core ships a zero-dependency ObsHooks seam; the contrib modules
// implement the producer side. cache.Instrument wires hooks onto any
// backend. This drives traffic, then:
//
//   - gathers the Prometheus registry and prints cache_ops_total
//   - collects a manual OTEL reader and prints the cache.ops counter
//
// Fully in-process — no scrape endpoint, no collector.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ubgo/cache"
	memcache "github.com/ubgo/cache-mem"
	cacheotel "github.com/ubgo/cache/contrib/cache-otel"
	cacheprom "github.com/ubgo/cache/contrib/cache-prom"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	if err := prometheusDemo(ctx); err != nil {
		return err
	}
	if err := otelDemo(ctx); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

// drive issues a fixed traffic pattern (2 sets, 1 hit, 1 miss).
func drive(ctx context.Context, c cache.Cache) error {
	if err := c.Set(ctx, "a", []byte("1"), time.Minute); err != nil {
		return err
	}
	if err := c.Set(ctx, "b", []byte("2"), time.Minute); err != nil {
		return err
	}
	if _, err := c.Get(ctx, "a"); err != nil { // hit
		return err
	}
	_, _ = c.Get(ctx, "nope") // miss (ErrNotFound — expected)
	return nil
}

func prometheusDemo(ctx context.Context) error {
	reg := prometheus.NewRegistry()
	hooks, err := cacheprom.New(reg, "mem", "demo")
	if err != nil {
		return err
	}

	c := cache.Instrument(memcache.New(), hooks)
	defer c.Close()
	if err := drive(ctx, c); err != nil {
		return err
	}

	mfs, err := reg.Gather()
	if err != nil {
		return err
	}
	fmt.Println("--- Prometheus ---")
	for _, mf := range mfs {
		if mf.GetName() != "cache_ops_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			fmt.Printf("cache_ops_total{op=%q,result=%q} = %v\n",
				labels["op"], labels["result"], m.GetCounter().GetValue())
		}
	}
	return nil
}

func otelDemo(ctx context.Context) error {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() { _ = mp.Shutdown(ctx) }()

	hooks, err := cacheotel.New(mp.Meter("cache"), "mem", "demo")
	if err != nil {
		return err
	}

	c := cache.Instrument(memcache.New(), hooks)
	defer c.Close()
	if err := drive(ctx, c); err != nil {
		return err
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		return err
	}
	fmt.Println("--- OpenTelemetry (manual reader) ---")
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "cache.ops" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				op, _ := dp.Attributes.Value("op")
				result, _ := dp.Attributes.Value("result")
				fmt.Printf("cache.ops{op=%q,result=%q} = %d\n",
					op.AsString(), result.AsString(), dp.Value)
			}
		}
	}
	return nil
}

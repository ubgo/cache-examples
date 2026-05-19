// Example 04-decorators — stack the resilience/observability decorators.
//
// Everything is a cache.Cache, so decorators compose. This wraps a
// cache-mem backend with:
//
//	Instrument( AuditLog( Bulkhead( Retry( CircuitBreaker( mem ) ) ) ) )
//
// and prints the audit trail plus the merged Stats. All in-process.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ubgo/cache"
	memcache "github.com/ubgo/cache-mem"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	base := memcache.New()
	defer base.Close()

	// Collect audit events (mutations only). AuditFunc must not block.
	var mu sync.Mutex
	var events []cache.AuditEvent
	audit := func(ev cache.AuditEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	}

	// Compose the stack. Order is the caller's choice; here breaker is
	// innermost (closest to the backend), instrumentation outermost.
	c := cache.NewCircuitBreaker(base,
		cache.WithBreakerThreshold(3),
		cache.WithBreakerCooldown(time.Second))
	c = cache.NewRetry(c,
		cache.WithRetryAttempts(3),
		cache.WithRetryBackoff(5*time.Millisecond))
	c = cache.NewBulkhead(c, 8) // at most 8 concurrent ops
	c = cache.NewAuditLog(c, audit)
	c = cache.Instrument(c, cache.ObsHooks{Adapter: "mem", Namespace: "demo"})

	// Drive some traffic through the whole stack.
	if err := c.Set(ctx, "k1", []byte("v1"), time.Minute); err != nil {
		return err
	}
	if err := c.Set(ctx, "k2", []byte("v2"), time.Minute); err != nil {
		return err
	}
	if _, err := c.Get(ctx, "k1"); err != nil { // hit
		return err
	}
	if _, err := c.Get(ctx, "missing"); err != nil { // miss (ErrNotFound)
		// expected — not a backend failure, does not trip the breaker
		_ = err
	}
	if err := c.Del(ctx, "k2"); err != nil {
		return err
	}

	// --- Audit trail (mutations are recorded, reads are not) ---
	mu.Lock()
	fmt.Printf("audit events (%d):\n", len(events))
	for _, ev := range events {
		fmt.Printf("  op=%-4s keys=%v err=%v\n", ev.Op, ev.Keys, ev.Err)
	}
	mu.Unlock()

	// --- Merged Stats: the Instrument layer adds the hits/misses/sets/
	//     deletes it observed on top of the adapter's own snapshot. ---
	s := c.Stats()
	fmt.Printf("stats: hits=%d misses=%d sets=%d deletes=%d hit_ratio=%.2f\n",
		s.Hits, s.Misses, s.Sets, s.Deletes, s.HitRatio())

	fmt.Println("OK")
	return nil
}

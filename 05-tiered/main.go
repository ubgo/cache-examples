// Example 05-tiered — L1 (mem) + L2 (Redis) with cross-process invalidation.
//
// Two independent tiered caches share one Redis (an in-process miniredis,
// no daemon needed) as L2 and the same Redis Pub/Sub invalidation bus.
// Demonstrates:
//
//   - read-promotion: an L1 miss that hits L2 is copied back into L1
//   - cross-instance invalidation: a Del on instance A drops the locally
//     cached L1 copy on instance B
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/ubgo/cache"
	memcache "github.com/ubgo/cache-mem"
	rediscache "github.com/ubgo/cache-redis"
	tieredcache "github.com/ubgo/cache-tiered"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// In-process Redis — no external service.
	mr, err := miniredis.Run()
	if err != nil {
		return err
	}
	defer mr.Close()

	const invChannel = "cache:invalidate"

	// Each "instance" simulates a separate pod: its own L1 (mem) and its own
	// go-redis client, but they share the one miniredis as L2 + Pub/Sub bus.
	newInstance := func() (*tieredcache.Cache, *redis.Client) {
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		tc := tieredcache.New(
			tieredcache.WithL1(memcache.New()),
			tieredcache.WithL2(rediscache.New(rdb)),
			tieredcache.WithInvalidation(rediscache.NewInvalidation(rdb, invChannel)),
		)
		return tc, rdb
	}

	a, rdbA := newInstance()
	defer a.Close()
	defer rdbA.Close()
	b, rdbB := newInstance()
	defer b.Close()
	defer rdbB.Close()

	// --- Read-promotion ---
	// Write via A (write-through: lands in A's L1 and the shared L2/Redis).
	if err := a.Set(ctx, "user:42", []byte("alice"), time.Minute); err != nil {
		return err
	}
	// B has never seen this key in its L1, but a Get falls through to the
	// shared L2 (Redis) and promotes the value into B's L1.
	v, err := b.Get(ctx, "user:42")
	if err != nil {
		return err
	}
	fmt.Printf("read-promotion: B.Get user:42 -> %s (L1 miss -> L2 hit -> promoted)\n", v)
	fmt.Printf("read-promotion: B promotions so far = %d\n", b.Promotions())

	// --- Cross-instance invalidation ---
	// B now holds user:42 in its own L1. A deletes the key; the tiered cache
	// publishes on the bus and B's subscriber drops its local L1 copy.
	if err := a.Del(ctx, "user:42"); err != nil {
		return err
	}

	// Give the best-effort Pub/Sub a moment to deliver.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := b.Get(ctx, "user:42"); errors.Is(err, cache.ErrNotFound) {
			fmt.Println("cross-instance invalidation: A.Del propagated -> B.Get user:42 -> ErrNotFound")
			fmt.Println("OK")
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("invalidation did not propagate within deadline")
}

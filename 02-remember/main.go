// Example 02-remember — cache.Remember, the load-through workhorse.
//
// Shows single-flight dedupe of concurrent loads, plus the production
// caching-pattern options: WithRefreshAhead, WithStaleWhileRevalidate,
// WithStaleIfError, WithNegativeTTL, WithJitter. A loader-call counter
// makes the dedupe / refresh behaviour visible. All in-process.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
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
	c := memcache.New()
	defer c.Close()

	// --- 1. Single-flight: 50 concurrent Remember calls, ONE loader run ---
	var calls atomic.Int64
	load := func(ctx context.Context) (string, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond) // simulate a slow DB
		return "value-from-loader", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.Remember(ctx, c, "hot:key", time.Minute, load)
		}()
	}
	wg.Wait()
	fmt.Printf("single-flight: 50 concurrent Remember -> loader called %d time(s)\n", calls.Load())

	// A later cached read does not call the loader at all.
	before := calls.Load()
	v, err := cache.Remember(ctx, c, "hot:key", time.Minute, load)
	if err != nil {
		return err
	}
	fmt.Printf("cached read -> %q, loader calls unchanged: %v\n", v, calls.Load() == before)

	// --- 2. WithNegativeTTL: a missing key is cached so the loader is not
	//        re-run on every request. ---
	var negCalls atomic.Int64
	missing := func(ctx context.Context) (string, error) {
		negCalls.Add(1)
		return "", cache.ErrNotFound // the loader reports "no such row"
	}
	for i := 0; i < 5; i++ {
		_, err := cache.Remember(ctx, c, "absent:42", time.Minute, missing,
			cache.WithNegativeTTL(time.Minute))
		if !errors.Is(err, cache.ErrNotFound) {
			return fmt.Errorf("expected ErrNotFound, got %v", err)
		}
	}
	fmt.Printf("negative caching: 5 lookups of an absent key -> loader called %d time(s)\n", negCalls.Load())

	// --- 3. WithRefreshAhead: a hot key refreshes in the background once a
	//        fraction of its TTL has elapsed, so it never expires under load. ---
	var raCalls atomic.Int64
	raLoad := func(ctx context.Context) (string, error) {
		raCalls.Add(1)
		return fmt.Sprintf("gen-%d", raCalls.Load()), nil
	}
	// 100ms TTL, refresh-ahead at 50%.
	if _, err := cache.Remember(ctx, c, "ra:key", 100*time.Millisecond, raLoad,
		cache.WithRefreshAhead(0.5)); err != nil {
		return err
	}
	time.Sleep(70 * time.Millisecond) // past the 50% refresh threshold
	// This read returns the still-fresh value and triggers a background reload.
	if _, err := cache.Remember(ctx, c, "ra:key", 100*time.Millisecond, raLoad,
		cache.WithRefreshAhead(0.5)); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond) // let the background refresh finish
	fmt.Printf("refresh-ahead: background reload fired, loader called %d time(s)\n", raCalls.Load())

	// --- 4. WithStaleWhileRevalidate: after hard expiry the stale value is
	//        served immediately while a single background load refreshes it. ---
	var swrCalls atomic.Int64
	swrLoad := func(ctx context.Context) (string, error) {
		n := swrCalls.Add(1)
		return fmt.Sprintf("swr-%d", n), nil
	}
	if _, err := cache.Remember(ctx, c, "swr:key", 30*time.Millisecond, swrLoad,
		cache.WithStaleWhileRevalidate(time.Second)); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond) // past soft expiry, within stale window
	got, err := cache.Remember(ctx, c, "swr:key", 30*time.Millisecond, swrLoad,
		cache.WithStaleWhileRevalidate(time.Second))
	if err != nil {
		return err
	}
	fmt.Printf("stale-while-revalidate: served stale %q immediately, refresh in background\n", got)

	// --- 5. WithStaleIfError: if the loader fails after expiry, the last
	//        good value is served instead of erroring out. ---
	var fail atomic.Bool
	siLoad := func(ctx context.Context) (string, error) {
		if fail.Load() {
			return "", errors.New("backend down")
		}
		return "good-value", nil
	}
	if _, err := cache.Remember(ctx, c, "sie:key", 20*time.Millisecond, siLoad,
		cache.WithStaleIfError(time.Second)); err != nil {
		return err
	}
	fail.Store(true)                  // backend now unavailable
	time.Sleep(40 * time.Millisecond) // past hard expiry
	got, err = cache.Remember(ctx, c, "sie:key", 20*time.Millisecond, siLoad,
		cache.WithStaleIfError(time.Second))
	if err != nil {
		return err
	}
	fmt.Printf("stale-if-error: loader failing, served last good value %q\n", got)

	// --- 6. WithJitter: spread the stored TTL so a batch written together
	//        does not all expire in the same instant (cache stampede guard). ---
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("batch:%d", i)
		if _, err := cache.Remember(ctx, c, key, time.Minute,
			func(ctx context.Context) (string, error) { return "v", nil },
			cache.WithJitter(0.2)); err != nil {
			return err
		}
		ttl, err := c.TTL(ctx, key)
		if err != nil {
			return err
		}
		fmt.Printf("jitter: %s stored TTL = %s (60s +/- 20%%)\n", key, ttl.Round(time.Second))
	}

	fmt.Println("OK")
	return nil
}

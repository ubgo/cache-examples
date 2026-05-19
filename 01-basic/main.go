// Example 01-basic — the core bytes-level Cache API on cache-mem.
//
// Demonstrates Set / Get / Has / TTL / Del, the ErrNotFound contract
// (a miss is (nil, ErrNotFound), never (nil, nil)), and defer Close.
// No external services — cache-mem is fully in-process.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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

	c := memcache.New(memcache.WithMaxEntries(1000))
	defer c.Close() // stops background goroutines / flushes handles

	// --- Set + Get ---
	if err := c.Set(ctx, "user:1", []byte("alice"), time.Minute); err != nil {
		return err
	}
	v, err := c.Get(ctx, "user:1")
	if err != nil {
		return err
	}
	fmt.Printf("Get user:1 -> %s\n", v)

	// --- Has ---
	ok, err := c.Has(ctx, "user:1")
	if err != nil {
		return err
	}
	fmt.Printf("Has user:1 -> %v\n", ok)

	// --- TTL ---
	ttl, err := c.TTL(ctx, "user:1")
	if err != nil {
		return err
	}
	fmt.Printf("TTL user:1 -> ~%s\n", ttl.Round(time.Second))

	// --- ErrNotFound contract: a miss is a typed error, never (nil, nil) ---
	_, err = c.Get(ctx, "user:999")
	if errors.Is(err, cache.ErrNotFound) {
		fmt.Println("Get user:999 -> cache.ErrNotFound (expected)")
	} else {
		return fmt.Errorf("expected ErrNotFound, got %v", err)
	}

	// --- Del ---
	if err := c.Del(ctx, "user:1"); err != nil {
		return err
	}
	if _, err := c.Get(ctx, "user:1"); errors.Is(err, cache.ErrNotFound) {
		fmt.Println("Del user:1 -> gone")
	} else {
		return fmt.Errorf("expected ErrNotFound after Del, got %v", err)
	}

	fmt.Println("OK")
	return nil
}

// Example 07-locker — cache.NewLock as a cron singleton.
//
// NewLock is a portable distributed lock built only on SetNX, so it works
// on any adapter. Two lockers contend for the same key (simulating two
// pods racing to run a nightly job); exactly one wins. The winner Refreshes
// its lease then Releases it, after which the loser can acquire. All
// in-process via cache-mem.
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
	c := memcache.New()
	defer c.Close()

	const job = "cron:nightly-billing"

	// Two pods, each with its own Locker (own random token) on the same key.
	podA := cache.NewLock(c, job, 30*time.Second)
	podB := cache.NewLock(c, job, 30*time.Second)

	// Both try to acquire; exactly one wins.
	errA := podA.Acquire(ctx)
	errB := podB.Acquire(ctx)

	switch {
	case errA == nil && errors.Is(errB, cache.ErrLockNotAcquired):
		fmt.Println("contention: pod A won, pod B got ErrLockNotAcquired (expected)")
	case errB == nil && errors.Is(errA, cache.ErrLockNotAcquired):
		fmt.Println("contention: pod B won, pod A got ErrLockNotAcquired (expected)")
	default:
		return fmt.Errorf("expected exactly one winner, got A=%v B=%v", errA, errB)
	}

	// The winner extends its lease for a long critical section.
	winner, loser := podA, podB
	if errA != nil {
		winner, loser = podB, podA
	}
	if err := winner.Refresh(ctx); err != nil {
		return fmt.Errorf("winner Refresh failed: %w", err)
	}
	fmt.Println("winner Refresh -> lease extended")

	// The loser still cannot acquire while the winner holds it.
	if err := loser.Acquire(ctx); !errors.Is(err, cache.ErrLockNotAcquired) {
		return fmt.Errorf("loser should still be locked out, got %v", err)
	}
	fmt.Println("loser re-acquire while held -> ErrLockNotAcquired (expected)")

	// Winner finishes its job and releases.
	if err := winner.Release(ctx); err != nil {
		return fmt.Errorf("winner Release failed: %w", err)
	}
	fmt.Println("winner Release -> lock freed")

	// Now the loser can take over (next cron tick / next pod).
	if err := loser.Acquire(ctx); err != nil {
		return fmt.Errorf("loser should acquire after release, got %v", err)
	}
	fmt.Println("loser Acquire after release -> success")
	if err := loser.Release(ctx); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

// Example 03-generics — the typed ergonomics layer over the bytes Cache.
//
// Shows cache.SetT / cache.GetT (codec-serialized typed access) and
// cache.NewTyped[User] (a view that carries the codec + options so call
// sites stop passing them). All in-process via cache-mem.
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

// User is the value type we cache without ever touching []byte ourselves.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

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

	// --- SetT / GetT: package-level typed helpers over any cache.Cache ---
	alice := User{ID: 1, Name: "Alice", Email: "alice@example.com"}
	if err := cache.SetT(ctx, c, "user:1", alice, time.Minute); err != nil {
		return err
	}
	got, err := cache.GetT[User](ctx, c, "user:1")
	if err != nil {
		return err
	}
	fmt.Printf("GetT user:1 -> %+v\n", got)

	// A miss is still the typed ErrNotFound contract.
	if _, err := cache.GetT[User](ctx, c, "user:404"); errors.Is(err, cache.ErrNotFound) {
		fmt.Println("GetT user:404 -> cache.ErrNotFound (expected)")
	} else {
		return fmt.Errorf("expected ErrNotFound, got %v", err)
	}

	// --- NewTyped[User]: a view carrying its options so call sites are clean ---
	users := cache.NewTyped[User](c, cache.WithJitter(0.1))

	bob := User{ID: 2, Name: "Bob", Email: "bob@example.com"}
	if err := users.Set(ctx, "user:2", bob, time.Minute); err != nil {
		return err
	}
	v, err := users.Get(ctx, "user:2")
	if err != nil {
		return err
	}
	fmt.Printf("Typed.Get user:2 -> %+v\n", v)

	// Typed.Remember: cached value, or single-flight the loader on miss.
	loaded, err := users.Remember(ctx, "user:3", time.Minute,
		func(ctx context.Context) (User, error) {
			return User{ID: 3, Name: "Carol", Email: "carol@example.com"}, nil
		})
	if err != nil {
		return err
	}
	fmt.Printf("Typed.Remember user:3 -> %+v\n", loaded)

	// Raw() exposes the same keyspace for ops the typed view does not wrap.
	ok, err := users.Raw().Has(ctx, "user:2")
	if err != nil {
		return err
	}
	fmt.Printf("Typed.Raw().Has user:2 -> %v\n", ok)

	if err := users.Del(ctx, "user:2"); err != nil {
		return err
	}
	if _, err := users.Get(ctx, "user:2"); errors.Is(err, cache.ErrNotFound) {
		fmt.Println("Typed.Del user:2 -> gone")
	} else {
		return fmt.Errorf("expected ErrNotFound after Del, got %v", err)
	}

	fmt.Println("OK")
	return nil
}

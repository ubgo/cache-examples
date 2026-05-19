// Example 08-cluster — peer-aware distributed cache, groupcache-style.
//
// Three in-process nodes, each fronted by an httptest server (no real
// network topology needed). A consistent-hash ring assigns every key to
// one owning node; reads for keys you do not own are proxied to the owner
// over HTTP; the owner fills a miss exactly once via the loader, deduped by
// single-flight. The loader counter proves a hot key is loaded ONCE
// cluster-wide, not once per node and not once per concurrent request.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"

	clustercache "github.com/ubgo/cache-cluster"
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

	ids := []string{"n1", "n2", "n3"}

	// A mux per node whose /_cache route we wire after the node exists
	// (the node needs peer URLs, the URLs come from the started servers —
	// resolve the cycle with a deferred handler).
	muxes := map[string]*http.ServeMux{}
	servers := map[string]*httptest.Server{}
	peers := map[string]string{}
	for _, id := range ids {
		m := http.NewServeMux()
		s := httptest.NewServer(m)
		defer s.Close()
		muxes[id] = m
		servers[id] = s
		peers[id] = s.URL // base URL; handler mounts at /_cache
	}

	// One loader shared by the demo; it counts how many times it actually
	// runs. In a real system this is the DB / origin fetch.
	var loaderCalls atomic.Int64
	loader := func(ctx context.Context, key string) ([]byte, error) {
		loaderCalls.Add(1)
		return []byte("value-for-" + key), nil
	}

	nodes := map[string]*clustercache.Node{}
	for _, id := range ids {
		n := clustercache.New(id, memcache.New(),
			clustercache.WithPeers(peers),
			clustercache.WithLoader(loader),
		)
		defer n.Close()
		nodes[id] = n
		muxes[id].Handle("/_cache", n.Handler())
	}

	// Pick a key and discover its owner via the ring.
	ring := clustercache.NewRing(64, ids...)
	const key = "user:42"
	owner := ring.Owner(key)
	fmt.Printf("ring: owner of %q = %s\n", key, owner)

	// Hammer the key from ALL three nodes concurrently. Non-owners proxy to
	// the owner; the owner single-flights the loader. Net result: one load.
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := map[string]string{}
	for _, id := range ids {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				v, err := nodes[id].Get(ctx, key)
				if err != nil {
					return
				}
				mu.Lock()
				results[id] = string(v)
				mu.Unlock()
			}(id)
		}
	}
	wg.Wait()

	fmt.Printf("30 concurrent Get across 3 nodes -> loader called %d time(s) (expect 1)\n",
		loaderCalls.Load())
	for _, id := range ids {
		fmt.Printf("  node %s sees: %s\n", id, results[id])
	}
	if loaderCalls.Load() != 1 {
		return fmt.Errorf("expected exactly 1 cluster-wide load, got %d", loaderCalls.Load())
	}

	fmt.Println("OK")
	return nil
}

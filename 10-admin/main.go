// Example 10-admin — the dependency-free admin HTTP inspection surface.
//
// admin.Handler mounts /cache/stats, /cache/key and an auth-gated
// /cache/evict over any cache.Cache. This serves it on an httptest server
// (no real port) and drives curl-style requests:
//
//   - GET /cache/stats           -> stats JSON + hit_ratio
//   - GET /cache/key?key=...     -> {found, bytes, ttl_ms}
//   - POST /cache/evict (no tok) -> 403 (safe by default)
//   - POST /cache/evict (token)  -> {evicted}
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	memcache "github.com/ubgo/cache-mem"
	"github.com/ubgo/cache/admin"
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

	// Seed some data + traffic so stats have non-zero counters.
	_ = c.Set(ctx, "user:42", []byte(`{"name":"alice"}`), time.Minute)
	_ = c.Set(ctx, "user:7", []byte(`{"name":"bob"}`), time.Minute)
	_, _ = c.Get(ctx, "user:42") // hit
	_, _ = c.Get(ctx, "missing") // miss

	const token = "s3cret"
	h := admin.Handler(c, admin.Options{
		Prefix: "/cache",
		Authorized: func(r *http.Request) bool {
			return r.Header.Get("X-Admin-Token") == token
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	get := func(path string) (int, string, error) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			return 0, "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b), nil
	}
	post := func(path, tok string) (int, string, error) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, nil)
		if tok != "" {
			req.Header.Set("X-Admin-Token", tok)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, "", err
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b), nil
	}

	// GET /cache/stats
	code, body, err := get("/cache/stats")
	if err != nil {
		return err
	}
	fmt.Printf("GET /cache/stats -> %d %s", code, body)

	// GET /cache/key?key=user:42
	code, body, err = get("/cache/key?key=user:42")
	if err != nil {
		return err
	}
	fmt.Printf("GET /cache/key?key=user:42 -> %d %s", code, body)

	// GET /cache/key for an absent key -> 404 {"found":false}
	code, body, err = get("/cache/key?key=does-not-exist")
	if err != nil {
		return err
	}
	fmt.Printf("GET /cache/key?key=does-not-exist -> %d %s", code, body)

	// POST /cache/evict WITHOUT token -> 403 (safe by default)
	code, body, err = post("/cache/evict?key=user:7", "")
	if err != nil {
		return err
	}
	fmt.Printf("POST /cache/evict (no token) -> %d %s", code, body)
	if code != http.StatusForbidden {
		return fmt.Errorf("expected 403 without token, got %d", code)
	}

	// POST /cache/evict WITH token -> 200 {"evicted":"user:7"}
	code, body, err = post("/cache/evict?key=user:7", token)
	if err != nil {
		return err
	}
	fmt.Printf("POST /cache/evict (token) -> %d %s", code, body)
	if code != http.StatusOK {
		return fmt.Errorf("expected 200 with token, got %d", code)
	}

	// Confirm it is gone.
	code, body, err = get("/cache/key?key=user:7")
	if err != nil {
		return err
	}
	fmt.Printf("GET /cache/key?key=user:7 (after evict) -> %d %s", code, body)

	fmt.Println("OK")
	return nil
}

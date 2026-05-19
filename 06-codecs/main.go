// Example 06-codecs — swap the value codec via cache.WithCodec.
//
// The generics layer serializes through a Codec, so the wire format is
// pluggable without touching the bytes-level Cache. Demonstrates:
//
//   - contrib/codec-msgpack: compact, cross-language
//   - contrib/codec-zstd wrapping JSON: size-thresholded compression
//   - cache.EncryptedCodec (AES-GCM) wrapping JSON: PII at rest
//
// All in-process via cache-mem.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ubgo/cache"
	memcache "github.com/ubgo/cache-mem"
	codecmsgpack "github.com/ubgo/cache/contrib/codec-msgpack"
	codeczstd "github.com/ubgo/cache/contrib/codec-zstd"
)

// Doc is the value we round-trip through each codec.
type Doc struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
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

	small := Doc{ID: 1, Body: "hello"}
	large := Doc{ID: 2, Body: strings.Repeat("compress-me ", 4000)} // ~48 KiB

	// --- 1. MessagePack codec: compact + faster than JSON ---
	mp := cache.WithCodec(codecmsgpack.Codec{})
	if err := cache.SetT(ctx, c, "mp:1", small, time.Minute, mp); err != nil {
		return err
	}
	gotMP, err := cache.GetT[Doc](ctx, c, "mp:1", mp)
	if err != nil {
		return err
	}
	raw, _ := c.Get(ctx, "mp:1")
	fmt.Printf("msgpack: round-trip %+v, stored %d bytes\n", gotMP, len(raw))

	// --- 2. zstd wrapping JSON: small values stay raw, large get compressed ---
	zc := codeczstd.New(cache.JSONCodec{}, codeczstd.WithMinBytes(1<<10)) // 1 KiB
	zstdOpt := cache.WithCodec(zc)

	if err := cache.SetT(ctx, c, "z:small", small, time.Minute, zstdOpt); err != nil {
		return err
	}
	if err := cache.SetT(ctx, c, "z:large", large, time.Minute, zstdOpt); err != nil {
		return err
	}
	rs, _ := c.Get(ctx, "z:small")
	rl, _ := c.Get(ctx, "z:large")
	plainLarge, _ := cache.JSONCodec{}.Encode(large)
	fmt.Printf("zstd: small stored raw (%d bytes), large %d bytes compressed (was %d JSON)\n",
		len(rs), len(rl), len(plainLarge))
	back, err := cache.GetT[Doc](ctx, c, "z:large", zstdOpt)
	if err != nil {
		return err
	}
	fmt.Printf("zstd: large round-trip OK, body len = %d\n", len(back.Body))

	// --- 3. EncryptedCodec (AES-GCM): cache PII safely in a shared store ---
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return err
	}
	enc := cache.EncryptedCodec{Inner: cache.JSONCodec{}, Key: cache.StaticKey(key)}
	encOpt := cache.WithCodec(enc)

	secret := Doc{ID: 3, Body: "ssn=123-45-6789"}
	if err := cache.SetT(ctx, c, "enc:1", secret, time.Minute, encOpt); err != nil {
		return err
	}
	stored, _ := c.Get(ctx, "enc:1")
	fmt.Printf("encrypted: codec=%s, stored ciphertext does not leak plaintext: %v\n",
		enc.Name(), !strings.Contains(string(stored), "123-45-6789"))
	dec, err := cache.GetT[Doc](ctx, c, "enc:1", encOpt)
	if err != nil {
		return err
	}
	fmt.Printf("encrypted: decrypted round-trip -> %+v\n", dec)

	fmt.Println("OK")
	return nil
}

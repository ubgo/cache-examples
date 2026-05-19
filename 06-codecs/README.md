# 06-codecs

The generics layer serializes through a `cache.Codec`, so the wire format is pluggable via `cache.WithCodec` without touching the bytes-level cache.

## Run

```sh
go run .
```

## What it demonstrates

- **`contrib/codec-msgpack`** — compact, cross-language MessagePack encoding.
- **`contrib/codec-zstd`** wrapping `JSONCodec` — size-thresholded zstd: small values stored raw, large values transparently compressed (a 1-byte header records which path was taken). Printed sizes show the saving.
- **`cache.EncryptedCodec`** (AES-GCM) wrapping `JSONCodec` — cache PII/secrets in a shared store; the stored bytes are verified to not leak the plaintext, and the value still round-trips after decryption.

## Expected output

One line per codec with stored byte sizes, ending in `OK`, exit code 0.

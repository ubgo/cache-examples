# 03-generics

The typed ergonomics layer over the bytes-level `cache.Cache` — work with `User` instead of `[]byte`.

## Run

```sh
go run .
```

## What it demonstrates

- `cache.SetT` / `cache.GetT[User]`: package-level typed helpers that serialize through the codec; misses still return `cache.ErrNotFound`.
- `cache.NewTyped[User](c, opts...)`: a view that carries the codec + Remember options so call sites stop repeating them.
- `Typed.Set` / `Get` / `Remember` / `Del`, and `Typed.Raw()` to reach the underlying bytes cache for ops the typed view does not wrap (`Has` here) on the same keyspace.

## Expected output

One line per typed operation, ending in `OK`, exit code 0.

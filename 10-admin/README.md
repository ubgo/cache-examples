# 10-admin

`cache/admin` — a small, dependency-free HTTP surface (imports only `net/http` + `encoding/json`) for inspecting a `cache.Cache` in production.

Served here on an `httptest` server (no real port) and exercised with curl-style requests.

## Run

```sh
go run .
```

## What it demonstrates

- `admin.Handler(c, admin.Options{...})` mounts the routes; `Authorized` gates the mutating route.
- `GET /cache/stats` → stats JSON including `hit_ratio`.
- `GET /cache/key?key=user:42` → `{found, bytes, ttl_ms}`; an absent key → `404 {"found":false}`.
- `POST /cache/evict` **without** the admin token → `403` (safe by default — `nil` Authorized always 403).
- `POST /cache/evict` **with** the token → `200 {"evicted":...}`; a follow-up key lookup confirms it is gone.

## Expected output

The HTTP status + JSON body for each request, ending in `OK`, exit code 0.

# 05-tiered

`cache-tiered`: a nanosecond-latency local L1 (`cache-mem`) in front of a shared L2 (`cache-redis`), kept coherent across instances by a Redis Pub/Sub invalidation bus.

No Redis daemon required — an in-process [`miniredis`](https://github.com/alicebob/miniredis) provides L2 and the Pub/Sub channel.

## Run

```sh
go run .
```

## What it demonstrates

- Two independent tiered caches (simulating two pods), each with its own L1 and go-redis client, sharing one miniredis as L2 + bus.
- **Read-promotion**: instance B has never cached `user:42` in its L1; a `Get` falls through to the shared L2, returns the value, and promotes it into B's L1 (`Promotions()` count printed).
- **Cross-instance invalidation**: instance A `Del`s the key; the tiered cache publishes on the bus and B's subscriber drops its local L1 copy, so B's next `Get` is `ErrNotFound`.

## Expected output

The promotion line, the invalidation-propagated line, then `OK`, exit code 0.

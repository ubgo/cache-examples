# 07-locker

`cache.NewLock` — a portable distributed lock built only on `SetNX`, so it works on any adapter. The classic use case: ensuring exactly one pod runs a cron job.

## Run

```sh
go run .
```

## What it demonstrates

- Two lockers (each with its own random token) contend for the same key, simulating two pods racing to run a nightly job — exactly one `Acquire` wins, the other gets `cache.ErrLockNotAcquired`.
- The winner `Refresh`es its lease for a long critical section.
- The loser still cannot acquire while the lock is held.
- After the winner `Release`s, the loser can acquire (next cron tick / failover).

## Expected output

A line per lock transition, ending in `OK`, exit code 0.

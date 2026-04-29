# ADR 0002: Decouple Worker Reporting and Batch Persistence

## Status
Accepted

## Context
As trade volume increased (+360%), the End-to-End (E2E) latency regressed significantly (P99 > 5s). The root cause was Head-of-Line (HOL) blocking in the `OrderWorker`. The worker was performing synchronous JSON marshaling, Redis publishing, and individual database inserts for every trade within the critical matching loop.

## Decision
We implemented three architectural changes to recover performance:

1. **Background Reporting:** Introduced a `snapshotLoop` in the worker. Instead of broadcasting the orderbook on every match, we mark the book as "dirty" and a background ticker (50ms) handles the snapshotting, JSON marshaling, and cache updates.
2. **Database Batching:** Replaced individual trade insertions with a `CreateTradesBatch` operation using PostgreSQL `unnest`. This reduces database round-trips from one-per-trade to exactly one-per-order.
3. **Struct Pooling:** Implemented `sync.Pool` for `Order` objects and reused `tradesBuf` slices to reduce heap allocations and GC overhead.
4. **Async I/O:** Switched Kafka and Trade Broadcasting to asynchronous/goroutine-based execution to prevent network jitter from blocking the matching engine.

## Consequences
- **Positive:** P99 E2E latency returned to < 25ms baseline despite higher load.
- **Positive:** CPU usage dropped by 22% due to reduced I/O waiting and better memory management.
- **Negative:** The orderbook requires a `sync.RWMutex` to ensure thread safety during background snapshots.
- **Negative:** WebSocket clients may see orderbook updates with a maximum lag of 50ms (acceptable for web UIs).

> **Validation Note:** v0.2 was validated using a new concurrent load-testing harness. While the P99 latency baseline shifted slightly due to increased thundering-herd pressure on the API, the core Matching Engine achieved a record 2.61µs average, and the system successfully processed >13k orders without the I/O stalls observed in the v0.1 architecture.
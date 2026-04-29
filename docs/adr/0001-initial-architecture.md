# ADR 0001: Initial Order Management and Matching Engine Architecture

## Status
Accepted

## Context
The goal is to build a high-throughput, low-latency financial exchange engine in Go. The system must handle limit order placement, matching, and trade execution while maintaining strict data consistency and providing real-time updates to clients.

## Decision
We adopted a hybrid architecture combining in-memory execution with persistent event-driven state management:

1.  **Language choice:** Go was selected for its efficient garbage collection (low pause times), native concurrency primitives, and high execution speed.
2.  **In-Memory Matching Engine:** The Limit Order Book (LOB) is maintained entirely in memory using a Doubly Linked List for each price level. This allows for $O(1)$ time complexity for order additions and removals at a specific price point.
3.  **Outbox Pattern:** To ensure atomicity between the database (PostgreSQL) and the message broker (Kafka), we implemented the Outbox Pattern. Orders are written to an `outbox` table in the same transaction as the `orders` table, then a relay service pushes them to Kafka.
4.  **Kafka as the Command Bus:** Kafka serves as the source of truth for the "Order Worker." The worker consumes commands sequentially to ensure that the in-memory order book remains deterministic and consistent across restarts.
5.  **Sequential Worker Processing:** In the initial version, the worker processes each message from Kafka synchronously: fetching, matching, updating the database, and broadcasting results before moving to the next message.
6.  **Real-time Updates:** A WebSocket Hub utilizing Redis Pub/Sub provides order book and trade updates to front-end clients.

## Baseline Metrics (v0.1)
The following metrics were recorded under initial load:
- **Average Match Time:** 3.98 μs
- **Average Order Placement:** 4.59 ms
- **End-to-End Latency (Avg):** 12.24 ms
- **End-to-End Latency (P99):** < 25 ms
- **Trade Throughput:** ~2,150 trades

## Consequences
- **Positive:** The system is easy to reason about due to its sequential processing.
- **Positive:** In-memory matching provides microsecond-level performance for the core engine.
- **Positive:** High reliability due to ACID-compliant PostgreSQL transactions and Kafka persistence.
- **Negative:** The worker is highly sensitive to I/O latency. Synchronous database writes and network broadcasts create a "performance floor" that limits maximum throughput.
- **Negative:** Large trade executions (matching many orders) cause spikes in database transaction time, potentially leading to Head-of-Line (HOL) blocking.
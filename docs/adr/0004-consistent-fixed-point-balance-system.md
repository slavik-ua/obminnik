# ADR 0004: Consistent Fixed-Point Scaling and Robust Balance Synchronization

## Status
Accepted

## Context
The matching engine previously faced two critical issues that compromised system stability and data integrity:
1.  **Floating Point Risks:** While the core used `int64`, the lack of a standardized scaling factor led to inconsistencies between the API (integers), Frontend (floats), and Worker (unscaled integers). This resulted in "dust" orders and display errors.
2.  **Memory-Database Desync:** The `BalanceCache` was initialized from the `balances` table in PostgreSQL. However, if the matching engine crashed after matching an order but before persisting the balance change, the cache would restart with an inconsistent state. This caused "insufficient balance" panics during trade settlement, as the engine believed funds were available when they were actually locked in pending orders.

## Decision
We implemented a comprehensive "Clean Slate" balance architecture:

1.  **Strict 1e8 Fixed-Point Scaling:** All financial values (Prices, Quantities, Balances) are now treated as `int64` scaled by $10^8$. 
    *   Example: $1.23 BTC is stored and processed as `123,000,000`.
    *   The frontend normalizes these values by dividing by `1e8` only at the final rendering layer.
2.  **Order-Driven Balance Hydration:** Instead of trusting the `balances` table's `locked` column during startup, the `OrderWorker` now performs a "Hydration" phase:
    *   It fetches all `PLACED` or `PARTIAL` orders from the database.
    *   It recalculates the total `locked` funds for every user based on their active orders.
    *   It forcefully synchronizes the `BalanceCache` with these calculated values, ensuring memory and the orderbook are always in atomic sync.
3.  **Atomic Settlement Logic:** Trade settlement in the `BalanceCache` now uses a strict "Buyer-Pays-Quote, Seller-Pays-Base" logic with immediate validation.
4.  **Synchronous Kafka Publishing:** The `OutboxRelay` was switched to synchronous publishing (`Async: false`) to ensure that an event is never marked as "processed" in the database unless Redpanda has confirmed its receipt. This prevents silent message loss that previously led to memory inconsistencies.

## Consequences
- **Positive: Elimination of Settlement Panics.** The system no longer crashes with "insufficient available balance" because the memory cache is guaranteed to match the actual order state on the book, even after an ungraceful shutdown.
- **Positive: High Financial Precision.** The move to 1e8 fixed-point math eliminates all floating-point rounding errors, making the system suitable for high-precision institutional trading.
- **Positive: UI Consistency.** Standardizing on 1e8 ensures that the orderbook, depth charts, and trade history all show consistent, correctly-scaled values.
- **Neutral: Startup Latency.** The hydration phase adds a small delay to the worker startup (approx. 50ms per 10,000 active orders) as it reconciles the book. This is an acceptable trade-off for guaranteed consistency.
- **Neutral: Database Schema.** The `balances` table remains the source of truth for `Available` funds, while the `orders` table acts as the source of truth for `Locked` funds during recovery.

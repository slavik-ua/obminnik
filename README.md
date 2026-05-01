# OBMINNIK: Trading Platform

> [!CAUTION]
**EDUCATIONAL PURPOSE ONLY.** 
This project is built for educational and portfolio purposes. It is **not** intended for production use or real-money trading. The author assumes no responsibility for financial losses, data loss, or any other damages arising from the use of this software. Use at your own risk.

<p align="center">
  <img src="frontend/public/LOGO.png" alt="OBMINNIK LOGO" width="200" />
</p>

OBMINNIK ("Exchange" in Ukrainian) is a high-performance limit order book (LOB) and trading platform built for speed, reliability, and observability. This document serves as the central source of truth for the project's architecture, performance, and setup.

---

## 📖 Table of Contents
- [🚀 Features](#features)
- [📊 Performance & Metrics](#performance-metrics)
    - [Executive Summary](#executive-summary)
    - [Latency Analysis](#latency-analysis)
    - [Resource Utilization](#resource-utilization)
- [🏗️ Architecture](#architecture)
- [📝 Architecture Decision Records (ADRs)](#architecture-decision-records)
- [📊 Core Data Models](#core-data-models)
- [🛠️ Tech Stack](#tech-stack)
- [📸 Screenshots](#screenshots)
- [⚡ Getting Started](#getting-started)
- [🧪 Testing](#testing)
- [🛠️ Future Directions](#future-directions)
- [⚠️ Disclaimer](#disclaimer)

---

## 🚀 Features <a name="features"></a>

- **High-Performance Matching Engine**:
    - **Price-Time (FIFO) Priority**: Strict adherence to industry matching standards.
    - **Zero-Allocation Hot Path**: Optimized memory management for low-latency matching.
    - **Atomic Operations**: Thread-safe order book management using refined locking strategies.
- **Real-Time Data Streaming**:
    - **WebSocket Integration**: Low-latency push updates for order books and trade history.
    - **Reactive Dashboard**: Instant UI updates powered by Next.js and WebSockets.
- **Reliable Event Sourcing**:
    - **Redpanda (Kafka) Integration**: Durable event logging for all trades and order updates.
    - **Outbox Pattern**: Guaranteed consistency between database state and event streams.
- **Full-Stack Observability**:
    - **Prometheus Metrics**: Granular tracking of system health and performance.
    - **Grafana Dashboards**: Visual analytics for latency, throughput, and engine depth.

---

## 📊 Performance & Metrics <a name="performance-metrics"></a>

### Executive Summary

OBMINNIK demonstrates high-performance capabilities with a robust matching engine. While currently very efficient, we are constantly working on further performance optimizations to reach even higher speeds. The core engine processes matches at a sub-microsecond average, and the end-to-end order lifecycle remains highly consistent.


## 📈 Performance Evolution

| Metric | v0.1 (Baseline) | v0.2 (Optimized) | **v0.3 (Current)** | Change (v0.2 → v0.3) |
| :--- | :--- | :--- | :--- | :--- |
| **Avg. Match Time** | 3.98 μs | 2.61 μs | **2.67 μs** | Stable (±2%) 🟢 |
| **Avg. E2E Latency** | 12.24 ms | 13.68 ms | **13.78 ms** | Stable 🟢 |
| **Max GC Pause** | 562.0 μs | 358.0 μs | **134.0 μs** | **-62.5%** 🟢🟢 |
| **CPU Time (Total)** | 21.74 s | 20.12 s* | **18.38 s** | **-8.6%** 🟢 |
| **Heap Objects (Live)** | ~45k | ~85k | **66k** | **-22.3%** 🟢 |
| **Total Allocations** | 364 MB | 355 MB* | **340 MB** | **-4.2%** 🟢 |

### Component Latency Breakdown (v0.3)

| Component | P50 (Median) | P95 | P99 (Tail) | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Matching Engine** | < 1 ms | < 1 ms | < 1 ms | 🟢 |
| **Order Placement** | < 5 ms | < 10 ms | < 10 ms | 🟢 |
| **End-to-End (E2E)** | ~12 ms | < 25 ms | < 50 ms | 🟢 |

> [!IMPORTANT]
> **Methodology Shift:** v0.1 was tested using serial scripts with minimal concurrency. **v0.2 and v0.3 were tested using a concurrent multi-threaded load balancer** simulating 20+ traders hitting the API simultaneously. Despite the significantly higher stress, the system achieves a **34% faster matching speed** than v0.1 and, as of v0.3, a **62% reduction in system jitter (GC pauses)**.

---

## 🏗️ Architecture <a name="architecture"></a>

OBMINNIK follows a modern event-driven architecture:

### High Level System Architecture
<img src="docs/highLevelSystemArchitecture.png" alt="High Level System Architecture" width="600" />

### Sequence Diagram
<img src="docs/sequenceDiagram.png" alt="Sequence Diagram" />


- **API Layer**: Handles authentication, validation, and order submission.
- **Matching Engine (Worker)**: Processes orders from Kafka, maintains the in-memory book, and executes trades.
- **Persistence**: PostgreSQL for long-term storage, Redis for fast caching and real-time state.
- **Events**: Redpanda ensures that every state change is durable and replayable.

---

### 📝 Architecture Decision Records (ADRs) <a name="architecture-decision-records"></a>
We use ADRs to track significant architectural changes and the rationale behind them. This ensures transparency in our technical trade-offs.

| ID | Title | Status |
| :--- | :--- | :--- |
| [0001](docs/adr/0001-initial-architecture.md) | Initial Project Structure & Baseline | ✅ Accepted |
| [0002](docs/adr/0002-decouple-worker-reporting-and-batch-persistence.md) | Decouple Reporting & Batch Persistence | ✅ Accepted |
| [0003](docs/adr/0003-optimized-id-generation.md) | Optimized ID Generation for High-Throughput Matching | ✅ Accepted |

---

## 📊 Core Data Models <a name="core-data-models"></a>

OBMINNIK uses robust domain models to ensure consistency across the matching engine and persistence layers.

### 1. Order
**Location**: `internal/core/domain/order.go`
Represents a trading instruction from a user.
- **Fields**: ID, UserID, Price, Quantity, Side (BUY/SELL), Status (NEW/FILLED/etc.).
- **Logic**: Includes internal pointers for ultra-fast price-level navigation.

### 2. Trade
**Location**: `internal/core/domain/order.go`
Records a successful match between two orders.
- **Fields**: ID, Price, Quantity, TakerOrderID, MakerOrderID.

### 3. OrderBook
**Location**: `internal/core/domain/orderbook.go`
The core matching in-memory structure.
- **Logic**: Organizes orders into price levels with FIFO priority.

### 4. Outbox
**Location**: `internal/core/domain/outbox.go`
Ensures "exactly-once" style event delivery using the transactional outbox pattern.


---

## 🛠️ Tech Stack <a name="tech-stack"></a>

- **Backend**: [Go](https://go.dev/) (High-performance concurrency)
- **Frontend**: [Next.js](https://nextjs.org/), [TypeScript](https://www.typescriptlang.org/), [Tailwind CSS](https://tailwindcss.com/)
- **Messaging**: [Redpanda](https://redpanda.com/) (Kafka-compatible event streaming)
- **Cache**: [Redis](https://redis.io/)
- **Database**: [PostgreSQL](https://www.postgresql.org/)
- **Observability**: [Prometheus](https://prometheus.io/), [Grafana](https://grafana.com/)
- **Infra**: [Docker Compose](https://docs.docker.com/compose/)

---

## 📸 Screenshots <a name="screenshots"></a>

### 1. Login
**The entry point of the application utilizes a stateless JWT authentication system.**

<img src="docs/login.png" alt="Login Screenshot" />


### 2. Trading Dashboard
**A high-density trading dashboard.**

<img src="docs/trading.png" alt="Trading Dashboard Screenshot" />

### 3. Market Depth Visualization
**A real-time cumulative volume graph representing market liquidity and price walls.**
<img src="docs/depthGraph.png" alt="Depth Graph Screenshot">

### 4. Live Order Book
**A live demonstration of the Orderbook Conflation Engine.**

<img src="docs/trading.gif" alt="Live Order Book" />

---

## ⚡ Getting Started <a name="getting-started"></a>

### Prerequisites
- Docker & Docker Compose
- Node.js (for local frontend development)
- Go 1.25+ (for local backend development)

### Quick Start
1. Clone the repository.
2. Spin up the infrastructure:
   ```bash
   docker-compose up --build
   ```
3. Access the platform:
    - **Frontend**: [http://localhost:3001](http://localhost:3001)
    - **API**: [http://localhost:8000](http://localhost:8000)
    - **Grafana**: [http://localhost:3000](http://localhost:3000)
    - **Prometheus**: [http://localhost:9090](http://localhost:9090)

---

## 🧪 Testing <a name="testing"></a>

Run integration tests:
```bash
go test -v ./cmd/api/...
```

Run matching engine unit tests:
```bash
go test -v ./internal/core/domain/...
```

Run the load test simulation:
```bash
cd load_test
python load_test.py
```

---

## 🛠️ Future Directions <a name="future-directions"></a>

To take OBMINNIK to the next level, we will:

### 1. Financial Integrity & Accounting (Critical)
*   **Double-Entry Ledger System**: Transition from simple balance updates to a strict double-entry accounting ledger to ensure the system is mathematically provable.
*   **"Available vs. Locked" Model**: Implement fund locking. When an order is placed, funds move to a `LOCKED` state in the DB and are only `SETTLED` or `UNLOCKED` upon engine confirmation.
*   **Formalized Fixed-Point Arithmetic**: Establish a global precision scale (e.g., 18 decimals for ETH/ERC20) and implement scale-aware math for settlement and fee calculations to prevent "decimal drift."

### 2. Reliability & Determinism
*   **Snapshot & Offset Sync**: Synchronize the in-memory state with specific Kafka offsets. This ensures that upon restart, the engine "replays" exactly from the last processed message, preventing duplicate executions.
*   **Event Sourcing & Replay**: Ensure the Matching Engine is 100% deterministic. If a worker crashes, it should be able to "replay" the Kafka topic from the last snapshot to perfectly rebuild the OrderBook state.

### 3. Performance Optimization (The HFT Edge)
*   **Binary Serialization (Protobuf)**: Replace JSON over Kafka/Redpanda with **Protocol Buffers**. This removes the overhead of reflection-based JSON parsing and significantly reduces the network payload size, lowering P99 latency.
*   **In-Memory Balance Cache**: Move "Available" balance checks out of Postgres and into a high-speed Redis Lua script or in-memory cache to push TPS into the thousands.

### 4. Architecture & Service Communication
*   **Internal gRPC Diagnostic API**: Implement **gRPC** services for internal communication between the API and the Engine Workers. This allows the API to query the live in-memory state of the Matching Engine (e.g., for health checks or live statistics) without touching the database.
*   **Dynamic Market Routing**: Support multiple trading pairs by spinning up isolated `OrderBook` instances orchestrated by a central `EngineRegistry`.

### 5. Web3 & Self-Custody Integration
*   **Vault Smart Contracts**: Develop Ethereum/L2 Vault contracts allowing users to deposit Mock ETH/ERC20s.
*   **SIWE (Sign-In With Ethereum)**: Replace traditional email/password auth with EIP-4361, allowing users to authenticate using MetaMask signatures.

---

## ⚠️ Disclaimer <a name="disclaimer"></a>

OBMINNIK is a proof-of-concept high-performance matching engine. 
- **No Financial Advice:** Nothing in this repository constitutes financial or investment advice.
- **Risk of Loss:** Trading systems are complex. High-latency, bugs, or race conditions in this software could result in a total loss of funds if connected to a real exchange or wallet.
- **Not Audited:** This code has not undergone a professional security audit.

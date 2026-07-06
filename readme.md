# Architecture

MonsterDB is a **Redis-compatible, single-threaded in-memory database** built around an **event-driven architecture**. It uses Linux `epoll` for scalable non-blocking I/O, allowing thousands of concurrent client connections to be managed efficiently without spawning multiple threads.

## Request Flow

```
           Client
              │
              │ RESP Protocol
              ▼
      TCP Server (epoll)
              │
              ▼
     Single-Thread Event Loop
              │
              ▼
        RESP Parser
              │
              ▼
     Command Dispatcher
              │
      ┌───────┼────────┐
      ▼       ▼        ▼
   Strings   Lists   Sets ...
      │
      ▼
  In-Memory Storage
      │
      ├────────► TTL Engine
      ├────────► Pub/Sub
      ├────────► Transactions
      └────────► AOF Persistence
              │
              ▼
      RESP Response
              │
              ▼
           Client
```

## Components

### Network Layer
- TCP server built using Linux `epoll`
- Non-blocking socket I/O
- Optimized read/write buffers

### Event Loop
The core of MonsterDB is a **single-threaded event loop** inspired by Redis.

The loop continuously:
- Accepts new client connections
- Waits for socket events using `epoll_wait()`
- Reads incoming requests
- Parses RESP messages
- Executes commands
- Sends responses back to clients

### RESP Parser
Incoming TCP packets are decoded according to the Redis Serialization Protocol (RESP), converting raw bytes into executable commands.

Example:

```text
SET user sahil
↓

["SET", "user", "sahil"]
```


### Commands
- String commands
- List commands
- Set commands
- Sorted Set commands
- Geo commands
- Bloom Filter commands
- Pub/Sub
- Transactions




## Performance Benchmark

MonsterDB was benchmarked using the official `redis-benchmark` tool.

### Benchmark Configuration

```bash
redis-benchmark \
  -h 127.0.0.1 \
  -p 6379 \
  -t set,get \
  -n 1000000 \
  -c 50 \
  -P 16
```

| Parameter | Value |
|-----------|-------|
| Total Requests | 1,000,000 |
| Commands | SET, GET |
| Concurrent Clients | 50 |
| Pipeline Depth | 16 |
| Protocol | RESP |
| Architecture | Single-threaded, event-driven (`epoll`) |

### Results

| Metric | Value |
|--------|------:|
| **Throughput** | **240,615.97 requests/sec** |
| Average Latency | 2.534 ms |
| Minimum Latency | 0.432 ms |
| Median (P50) | 2.495 ms |
| 95th Percentile | 3.415 ms |
| 99th Percentile | 3.983 ms |
| Maximum Latency | 73.663 ms |

### Highlights

- 🚀 **240K+ requests/second** throughput
- ⚡ **P99 latency below 4 ms**
- 🔄 Benchmarked with **1 million operations**
- 🧵 Single-threaded event-driven architecture using `epoll`
- 📡 RESP-compatible networking with optimized read/write paths

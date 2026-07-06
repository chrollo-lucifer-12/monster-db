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

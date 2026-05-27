Summary:
  throughput summary: 240615.97 requests per second
  latency summary (msec):
          avg       min       p50       p95       p99       max
        2.534     0.432     2.495     3.415     3.983    73.663



for benchmarking: redis-benchmark -h 127.0.0.1 -p 6379 -t set,get -n 1000000 -c 50 -P 16

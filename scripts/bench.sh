#!/bin/bash

set -e

# ADDR_PUT="http://localhost:8001"
# ADDR_GET_LOCAL="http://localhost:8001"
# ADDR_GET_REMOTE="http://localhost:8005"

ADDR_PUT="http://52.53.177.170:8001"
ADDR_GET_LOCAL="http://52.53.177.170:8001"
ADDR_GET_REMOTE="http://54.183.234.0:8001"

REQUESTS=20

now_ms() {
  python3 -c 'import time; print(int(time.time() * 1000))'
}

mkdir -p benchdata/results
mkdir -p benchdata/downloads

echo "Generating benchmark files..."
dd if=/dev/urandom of=benchdata/1kb.bin bs=1K count=1 2>/dev/null
dd if=/dev/urandom of=benchdata/1mb.bin bs=1M count=1 2>/dev/null
dd if=/dev/urandom of=benchdata/10mb.bin bs=1M count=10 2>/dev/null

benchmark_put() {
  FILE=$1
  KEY=$2
  REQUESTS=$3

  echo ""
  echo "Benchmark PUT: file=$FILE requests=$REQUESTS"

  for i in $(seq 1 "$REQUESTS"); do
    START=$(now_ms)

    ./bin/client put \
      --addr "$ADDR_PUT" \
      --key "${KEY}-${i}" \
      --file "$FILE" > /dev/null

    END=$(now_ms)
    LATENCY=$((END - START))

    echo "$LATENCY" >> "benchdata/results/put-${KEY}.txt"
  done
}

benchmark_get() {
  ADDR=$1
  KEY=$2
  LABEL=$3
  REQUESTS=$4

  echo ""
  echo "Benchmark GET: addr=$ADDR key=$KEY requests=$REQUESTS label=$LABEL"

  for i in $(seq 1 "$REQUESTS"); do
    OUT="benchdata/downloads/${LABEL}-${KEY}-${i}.out"

    START=$(now_ms)

    ./bin/client get \
      --addr "$ADDR" \
      --key "$KEY" \
      --out "$OUT" > /dev/null

    END=$(now_ms)
    LATENCY=$((END - START))

    echo "$LATENCY" >> "benchdata/results/get-${LABEL}-${KEY}.txt"
  done
}

summarize() {
  FILE=$1
  LABEL=$2

  if [ ! -f "$FILE" ]; then
    echo "$LABEL"
    echo "  no data"
    return
  fi

  awk -v label="$LABEL" '
  {
    values[NR] = $1
    sum += $1
  }
  END {
    if (NR == 0) {
      print label
      print "  no data"
      exit
    }

    for (i = 1; i <= NR; i++) {
      for (j = i + 1; j <= NR; j++) {
        if (values[i] > values[j]) {
          temp = values[i]
          values[i] = values[j]
          values[j] = temp
        }
      }
    }

    avg = sum / NR
    p50_index = int(NR * 0.50)
    p95_index = int(NR * 0.95)
    p99_index = int(NR * 0.99)

    if (p50_index < 1) p50_index = 1
    if (p95_index < 1) p95_index = 1
    if (p99_index < 1) p99_index = 1

    print label
    print "  requests:", NR
    print "  avg:", avg " ms"
    print "  p50:", values[p50_index] " ms"
    print "  p95:", values[p95_index] " ms"
    print "  p99:", values[p99_index] " ms"
  }
  ' "$FILE"
}

rm -f benchdata/results/*.txt
rm -f benchdata/downloads/*.out

# PUT benchmark with unique keys
benchmark_put "benchdata/1kb.bin" "bench-1kb" "$REQUESTS"
benchmark_put "benchdata/1mb.bin" "bench-1mb" "$REQUESTS"
benchmark_put "benchdata/10mb.bin" "bench-10mb" "$REQUESTS"

# Store stable files for GET benchmark
./bin/client put --addr "$ADDR_PUT" --key stable-1kb --file benchdata/1kb.bin > /dev/null
./bin/client put --addr "$ADDR_PUT" --key stable-1mb --file benchdata/1mb.bin > /dev/null
./bin/client put --addr "$ADDR_PUT" --key stable-10mb --file benchdata/10mb.bin > /dev/null

# GET local benchmark: node1
benchmark_get "$ADDR_GET_LOCAL" "stable-1kb" "local" "$REQUESTS"
benchmark_get "$ADDR_GET_LOCAL" "stable-1mb" "local" "$REQUESTS"
benchmark_get "$ADDR_GET_LOCAL" "stable-10mb" "local" "$REQUESTS"

# GET remote benchmark: node5
benchmark_get "$ADDR_GET_REMOTE" "stable-1kb" "remote" "$REQUESTS"
benchmark_get "$ADDR_GET_REMOTE" "stable-1mb" "remote" "$REQUESTS"
benchmark_get "$ADDR_GET_REMOTE" "stable-10mb" "remote" "$REQUESTS"

echo ""
echo "========== Benchmark Summary =========="
summarize "benchdata/results/put-bench-1kb.txt" "PUT 1KB to node1"
summarize "benchdata/results/put-bench-1mb.txt" "PUT 1MB to node1"
summarize "benchdata/results/put-bench-10mb.txt" "PUT 10MB to node1"

summarize "benchdata/results/get-local-stable-1kb.txt" "GET 1KB from node1"
summarize "benchdata/results/get-remote-stable-1kb.txt" "GET 1KB from node5"

summarize "benchdata/results/get-local-stable-1mb.txt" "GET 1MB from node1"
summarize "benchdata/results/get-remote-stable-1mb.txt" "GET 1MB from node5"

summarize "benchdata/results/get-local-stable-10mb.txt" "GET 10MB from node1"
summarize "benchdata/results/get-remote-stable-10mb.txt" "GET 10MB from node5"
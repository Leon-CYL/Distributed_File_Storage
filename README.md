# Distributed_File_Storage

## Description:
For this project, I built a distributed file storage system in Go that works over a peer-to-peer network. Each node can store, replicate, and retrieve files using content-addressable storage, where files are identified by their hash instead of filenames. I designed a custom TCP-based protocol for nodes to discover peers, broadcast stores and get requests, and transfer data. In order to secure communication, files are encrypted(AES) before being sent across the network and decrypted locally. The system automatically replicates files to peers so that if a node deletes its local copy, the file can still be retrieved from the network. At the storage layer, I implemented streaming read/write so large files don’t have to be loaded fully in memory, and I added path transformation logic to organize files by hash into a structured directory.

## Benchmark Results

Environment:
- Cluster: 5-node Docker Compose cluster
- Machine: MacBook Pro
- Requests per test: 20
- File sizes: 1KB, 1MB, 10MB
- Node1 HTTP: `localhost:8001`
- Node5 HTTP: `localhost:8005`

| Operation | File Size | Target Node | Avg Latency | P50 | P95 | P99 |
|---|---:|---|---:|---:|---:|---:|
| PUT | 1KB | node1 | 88.25 ms | 84 ms | 117 ms | 117 ms |
| PUT | 1MB | node1 | 99.25 ms | 99 ms | 108 ms | 108 ms |
| PUT | 10MB | node1 | 270.6 ms | 252 ms | 409 ms | 409 ms |
| GET | 1KB | node1 | 74.1 ms | 71 ms | 88 ms | 88 ms |
| GET | 1KB | node5 | 70.9 ms | 71 ms | 80 ms | 80 ms |
| GET | 1MB | node1 | 86.6 ms | 83 ms | 115 ms | 115 ms |
| GET | 1MB | node5 | 96.25 ms | 87 ms | 149 ms | 149 ms |
| GET | 10MB | node1 | 133.85 ms | 131 ms | 146 ms | 146 ms |
| GET | 10MB | node5 | 135.1 ms | 132 ms | 151 ms | 151 ms |

## Generate share encryption key:
> openssl rand -hex 32
> export DFS_ENCRYPTION_KEY=
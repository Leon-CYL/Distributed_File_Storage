# Distributed_File_Storage

## Description:
For this project, I built a distributed file storage system in Go that runs over a peer-to-peer network. Each node can store, replicate, delete, and retrieve files using content-addressable storage, where files are organized by hash-based keys rather than traditional filenames. I designed a custom TCP-based peer protocol that allows nodes to connect with bootstrap peers, broadcast store and get requests, and stream file data between nodes. To secure network transfer, files are encrypted with AES before being sent across the network and decrypted when received by another node.

I also added an HTTP API and command-line client so users can interact with the system through simple `PUT`, `GET`, `DELETE`, and health-check commands. At the storage layer, I implemented streaming read/write operations so large files do not need to be fully loaded into memory, along with path transformation logic that stores files in a structured hash-based directory layout.

To make the system easier to deploy and test, I containerized the project with Docker and created a Docker Compose setup for a local five-node cluster. I also deployed the system to AWS using five EC2 instances in the same VPC, with peer-to-peer replication running over private IPs and client requests handled through public HTTP endpoints. Finally, I added benchmarking scripts to measure PUT and GET latency for 1 KB, 1 MB, and 10 MB files across both local and remote nodes, demonstrating that the system can replicate and retrieve files across a distributed cloud deployment.


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

## AWS EC2 Benchmark

Environment:

- Cluster: 5 EC2 `t3.small` instances
- Region: AWS us-west-1
- Deployment: Docker containers on Amazon Linux 2023
- Network: Same VPC / same Availability Zone
- Replication: Peer-to-peer TCP replication on port 5001
- Client: Local MacBook sending HTTP requests to AWS public IPv4 addresses
- Requests per test: 20

| Operation | File Size | Node | Avg Latency | P50 | P95 | P99 |
|---|---:|---|---:|---:|---:|---:|
| PUT | 1 KB | node1 | 176.4 ms | 153 ms | 280 ms | 280 ms |
| PUT | 1 MB | node1 | 545.15 ms | 446 ms | 1076 ms | 1076 ms |
| PUT | 10 MB | node1 | 3089.05 ms | 1939 ms | 6667 ms | 6667 ms |
| GET | 1 KB | node1 | 196.7 ms | 144 ms | 447 ms | 447 ms |
| GET | 1 KB | node5 | 287.9 ms | 143 ms | 1137 ms | 1137 ms |
| GET | 1 MB | node1 | 719.5 ms | 467 ms | 1469 ms | 1469 ms |
| GET | 1 MB | node5 | 1516.3 ms | 564 ms | 4877 ms | 4877 ms |
| GET | 10 MB | node1 | 6846.35 ms | 5348 ms | 14402 ms | 14402 ms |
| GET | 10 MB | node5 | 5539.4 ms | 4676 ms | 9995 ms | 9995 ms |

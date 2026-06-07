# Distributed_File_Storage

## Description:
For this project, I built a distributed file storage system in Go that works over a peer-to-peer network. Each node can store, replicate, and retrieve files using content-addressable storage, where files are identified by their hash instead of filenames. I designed a custom TCP-based protocol for nodes to discover peers, broadcast stores and get requests, and transfer data. In order to secure communication, files are encrypted(AES) before being sent across the network and decrypted locally. The system automatically replicates files to peers so that if a node deletes its local copy, the file can still be retrieved from the network. At the storage layer, I implemented streaming read/write so large files don’t have to be loaded fully in memory, and I added path transformation logic to organize files by hash into a structured directory.

## Performance:

Store throughput (100 files each)
- 1 KB:   0.14 MB/s
- 1 MB: 108.5 MB/s
- 10 MB: 136.9 MB/s

GET latency (50 requests, local):
- avg: 0.79 ms
- p50: 0.75 ms
- p95: 1.34 ms
- p99: 2.72 ms

GET latency (20 requests, network):
- avg: 503 ms
- p50: 502 ms
- p95: 509 ms
- p99: 509 ms

## Generate share encryption key:
> openssl rand -hex 32
> export DFS_ENCRYPTION_KEY=7d98b4aea9958cb64dd22c05a09a4c487390ce950e90ba1b5526a1d995e92e17
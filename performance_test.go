package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
)

// Test configuration
const (
	// Test file sizes in bytes
	SmallFileSize  = 1024             // 1KB
	MediumFileSize = 1024 * 1024      // 1MB
	LargeFileSize  = 10 * 1024 * 1024 // 10MB

	// Test parameters
	NumStoreOperations = 100
	NumGetOperations   = 50
	ConcurrentWorkers  = 10
)

// Performance metrics
type PerformanceMetrics struct {
	StoreThroughput struct {
		TotalFiles     int
		TotalBytes     int64
		TotalDuration  time.Duration
		FilesPerSecond float64
		BytesPerSecond float64
	}
	GetLatency struct {
		TotalRequests  int
		TotalDuration  time.Duration
		AverageLatency time.Duration
		MinLatency     time.Duration
		MaxLatency     time.Duration
		Latencies      []time.Duration
	}
}

// TestData represents test file data with metadata
type TestData struct {
	Key  string
	Data []byte
	Size int
}

// generateTestData creates test data of specified size
func generateTestData(key string, size int) *TestData {
	data := make([]byte, size)
	rand.Read(data)
	return &TestData{
		Key:  key,
		Data: data,
		Size: size,
	}
}

// setupTestServer creates a test server for performance testing
func setupTestServer(listenAddr string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshake,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcp := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		EncryptionKey:     newEncryptionKey(),
		StorageRoot:       listenAddr + "_perf_test",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcp,
		BootstrapNodes:    []string{},
	}

	fs := NewFileServer(fileServerOpts)
	tcp.OnPeer = fs.OnPeer

	return fs
}

// cleanupTestServer removes test data and shuts down server
func cleanupTestServer(fs *FileServer) {
	fs.Stop()
	os.RemoveAll(fs.StorageRoot)
}

// TestStoreThroughput measures the throughput of Store operations
func TestStoreThroughput(t *testing.T) {
	metrics := &PerformanceMetrics{}

	// Test with different file sizes
	fileSizes := []int{SmallFileSize, MediumFileSize, LargeFileSize}

	for _, fileSize := range fileSizes {
		t.Run(fmt.Sprintf("FileSize_%dB", fileSize), func(t *testing.T) {
			fs := setupTestServer(":0") // Use dynamic port
			defer cleanupTestServer(fs)

			testStoreThroughputForSize(t, fs, fileSize, metrics)
		})
	}
}

func testStoreThroughputForSize(t *testing.T, fs *FileServer, fileSize int, metrics *PerformanceMetrics) {
	// Generate test data
	testFiles := make([]*TestData, NumStoreOperations)
	for i := 0; i < NumStoreOperations; i++ {
		testFiles[i] = generateTestData(fmt.Sprintf("perf_test_file_%d", i), fileSize)
	}

	// Measure store throughput
	startTime := time.Now()
	var totalBytes int64

	for _, testFile := range testFiles {
		data := bytes.NewReader(testFile.Data)
		if err := fs.Store(testFile.Key, data); err != nil {
			t.Fatalf("Failed to store file %s: %v", testFile.Key, err)
		}
		totalBytes += int64(testFile.Size)
	}

	duration := time.Since(startTime)

	// Calculate metrics
	metrics.StoreThroughput.TotalFiles = NumStoreOperations
	metrics.StoreThroughput.TotalBytes = totalBytes
	metrics.StoreThroughput.TotalDuration = duration
	metrics.StoreThroughput.FilesPerSecond = float64(NumStoreOperations) / duration.Seconds()
	metrics.StoreThroughput.BytesPerSecond = float64(totalBytes) / duration.Seconds()

	// Log results
	t.Logf("Store Throughput Test Results (File size: %d bytes):", fileSize)
	t.Logf("  Total files stored: %d", metrics.StoreThroughput.TotalFiles)
	t.Logf("  Total bytes stored: %d", metrics.StoreThroughput.TotalBytes)
	t.Logf("  Total duration: %v", metrics.StoreThroughput.TotalDuration)
	t.Logf("  Files per second: %.2f", metrics.StoreThroughput.FilesPerSecond)
	t.Logf("  Bytes per second: %.2f (%.2f MB/s)", metrics.StoreThroughput.BytesPerSecond,
		metrics.StoreThroughput.BytesPerSecond/(1024*1024))
}

// TestGetLatency measures the latency of Get operations
func TestGetLatency(t *testing.T) {
	fs := setupTestServer(":0")
	defer cleanupTestServer(fs)

	// Pre-populate with test files
	testFiles := make([]*TestData, NumGetOperations)
	for i := 0; i < NumGetOperations; i++ {
		testFiles[i] = generateTestData(fmt.Sprintf("latency_test_file_%d", i), MediumFileSize)
		data := bytes.NewReader(testFiles[i].Data)
		if err := fs.Store(testFiles[i].Key, data); err != nil {
			t.Fatalf("Failed to store file %s: %v", testFiles[i].Key, err)
		}
	}

	// Allow some time for storage to complete
	time.Sleep(100 * time.Millisecond)

	metrics := &PerformanceMetrics{}
	metrics.GetLatency.Latencies = make([]time.Duration, 0, NumGetOperations)
	metrics.GetLatency.MinLatency = time.Hour // Initialize to high value

	// Measure get latency
	for _, testFile := range testFiles {
		startTime := time.Now()

		r, err := fs.Get(testFile.Key)
		if err != nil {
			t.Fatalf("Failed to get file %s: %v", testFile.Key, err)
		}

		// Read the entire file to measure complete latency
		_, err = io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", testFile.Key, err)
		}

		latency := time.Since(startTime)
		metrics.GetLatency.Latencies = append(metrics.GetLatency.Latencies, latency)
		metrics.GetLatency.TotalDuration += latency

		if latency < metrics.GetLatency.MinLatency {
			metrics.GetLatency.MinLatency = latency
		}
		if latency > metrics.GetLatency.MaxLatency {
			metrics.GetLatency.MaxLatency = latency
		}
	}

	// Calculate metrics
	metrics.GetLatency.TotalRequests = NumGetOperations
	metrics.GetLatency.AverageLatency = metrics.GetLatency.TotalDuration / time.Duration(NumGetOperations)

	// Log results
	t.Logf("Get Latency Test Results:")
	t.Logf("  Total requests: %d", metrics.GetLatency.TotalRequests)
	t.Logf("  Average latency: %v", metrics.GetLatency.AverageLatency)
	t.Logf("  Min latency: %v", metrics.GetLatency.MinLatency)
	t.Logf("  Max latency: %v", metrics.GetLatency.MaxLatency)
	t.Logf("  Total duration: %v", metrics.GetLatency.TotalDuration)

	// Calculate percentiles
	calculateAndLogPercentiles(t, metrics.GetLatency.Latencies)
}

// TestConcurrentStoreThroughput measures throughput with concurrent operations
func TestConcurrentStoreThroughput(t *testing.T) {
	fs := setupTestServer(":0")
	defer cleanupTestServer(fs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalBytes int64
	var totalFiles int

	filesPerWorker := NumStoreOperations / ConcurrentWorkers

	startTime := time.Now()

	// Launch concurrent workers
	for worker := 0; worker < ConcurrentWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerFiles := 0
			workerBytes := int64(0)

			for i := 0; i < filesPerWorker; i++ {
				key := fmt.Sprintf("concurrent_test_worker_%d_file_%d", workerID, i)
				testFile := generateTestData(key, MediumFileSize)
				data := bytes.NewReader(testFile.Data)

				if err := fs.Store(key, data); err != nil {
					t.Errorf("Worker %d failed to store file %s: %v", workerID, key, err)
					return
				}

				workerFiles++
				workerBytes += int64(testFile.Size)
			}

			mu.Lock()
			totalFiles += workerFiles
			totalBytes += workerBytes
			mu.Unlock()
		}(worker)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Calculate metrics
	filesPerSecond := float64(totalFiles) / duration.Seconds()
	bytesPerSecond := float64(totalBytes) / duration.Seconds()

	t.Logf("Concurrent Store Throughput Test Results:")
	t.Logf("  Workers: %d", ConcurrentWorkers)
	t.Logf("  Total files stored: %d", totalFiles)
	t.Logf("  Total bytes stored: %d", totalBytes)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Files per second: %.2f", filesPerSecond)
	t.Logf("  Bytes per second: %.2f (%.2f MB/s)", bytesPerSecond, bytesPerSecond/(1024*1024))
}

// TestConcurrentGetLatency measures latency with concurrent Get operations
func TestConcurrentGetLatency(t *testing.T) {
	fs := setupTestServer(":0")
	defer cleanupTestServer(fs)

	// Pre-populate with test files
	for i := 0; i < NumGetOperations; i++ {
		key := fmt.Sprintf("concurrent_get_test_file_%d", i)
		testFile := generateTestData(key, MediumFileSize)
		data := bytes.NewReader(testFile.Data)
		if err := fs.Store(key, data); err != nil {
			t.Fatalf("Failed to store file %s: %v", key, err)
		}
	}

	// Allow storage to complete
	time.Sleep(200 * time.Millisecond)

	var wg sync.WaitGroup
	var mu sync.Mutex
	latencies := make([]time.Duration, 0, NumGetOperations)

	getsPerWorker := NumGetOperations / ConcurrentWorkers

	// Launch concurrent workers for Get operations
	for worker := 0; worker < ConcurrentWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerLatencies := make([]time.Duration, 0, getsPerWorker)

			for i := 0; i < getsPerWorker; i++ {
				key := fmt.Sprintf("concurrent_get_test_file_%d", workerID*getsPerWorker+i)

				startTime := time.Now()
				r, err := fs.Get(key)
				if err != nil {
					t.Errorf("Worker %d failed to get file %s: %v", workerID, key, err)
					return
				}

				_, err = io.ReadAll(r)
				if err != nil {
					t.Errorf("Worker %d failed to read file %s: %v", workerID, key, err)
					return
				}

				latency := time.Since(startTime)
				workerLatencies = append(workerLatencies, latency)
			}

			mu.Lock()
			latencies = append(latencies, workerLatencies...)
			mu.Unlock()
		}(worker)
	}

	wg.Wait()

	// Calculate metrics
	var totalDuration time.Duration
	minLatency := time.Hour
	var maxLatency time.Duration

	for _, latency := range latencies {
		totalDuration += latency
		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	averageLatency := totalDuration / time.Duration(len(latencies))

	t.Logf("Concurrent Get Latency Test Results:")
	t.Logf("  Workers: %d", ConcurrentWorkers)
	t.Logf("  Total requests: %d", len(latencies))
	t.Logf("  Average latency: %v", averageLatency)
	t.Logf("  Min latency: %v", minLatency)
	t.Logf("  Max latency: %v", maxLatency)

	calculateAndLogPercentiles(t, latencies)
}

// calculateAndLogPercentiles calculates and logs latency percentiles
func calculateAndLogPercentiles(t *testing.T, latencies []time.Duration) {
	if len(latencies) == 0 {
		return
	}

	// Sort latencies for percentile calculation
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	// Simple bubble sort for demonstration (use sort.Slice in production)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	// Calculate percentiles
	p50 := sortedLatencies[len(sortedLatencies)*50/100]
	p95 := sortedLatencies[len(sortedLatencies)*95/100]
	p99 := sortedLatencies[len(sortedLatencies)*99/100]

	t.Logf("  P50 latency: %v", p50)
	t.Logf("  P95 latency: %v", p95)
	t.Logf("  P99 latency: %v", p99)
}

// TestGetLatencyThroughNetwork measures the latency of GET operations through the network
func TestGetLatencyThroughNetwork(t *testing.T) {
	s1 := newServer(":3001", "")
	s2 := newServer(":3002", "")
	s3 := newServer(":3000", ":3001", ":3002")

	// Start servers
	go func() {
		if err := s1.Start(); err != nil {
			t.Errorf("Failed to start s1: %v", err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	go func() {
		if err := s2.Start(); err != nil {
			t.Errorf("Failed to start s2: %v", err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	go func() {
		if err := s3.Start(); err != nil {
			t.Errorf("Failed to start s3: %v", err)
		}
	}()

	time.Sleep(time.Millisecond * 500)

	// Cleanup function
	defer func() {
		s1.Stop()
		s2.Stop()
		s3.Stop()
		os.RemoveAll(s1.StorageRoot)
		os.RemoveAll(s2.StorageRoot)
		os.RemoveAll(s3.StorageRoot)
	}()

	// Slice to collect GET latencies
	networkGetLatencies := make([]time.Duration, 0, 20)
	testData := "my big data file here!"

	t.Logf("Testing network GET latency with 20 files...")

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("text_%d.txt", i)
		data := bytes.NewReader([]byte(testData))

		// Store file on s3
		if err := s3.Store(key, data); err != nil {
			t.Errorf("Failed to store file %s: %v", key, err)
			continue
		}

		// Small delay to allow replication
		time.Sleep(time.Millisecond * 5)

		// Delete from local store to force network fetch
		if err := s3.store.Delete(key); err != nil {
			t.Errorf("Failed to delete file %s locally: %v", key, err)
			continue
		}

		// Measure GET latency (should fetch from network)
		startTime := time.Now()
		r, err := s3.Get(key)
		if err != nil {
			t.Errorf("Failed to get file %s from network: %v", key, err)
			continue
		}

		_, err = io.ReadAll(r)
		if err != nil {
			t.Errorf("Failed to read file %s: %v", key, err)
			continue
		}

		// Record the latency
		latency := time.Since(startTime)
		networkGetLatencies = append(networkGetLatencies, latency)

		// Progress indicator
		if (i+1)%5 == 0 {
			t.Logf("Completed %d/20 network GET operations... (last latency: %v)", i+1, latency)
		}
	}

	// Calculate and display statistics
	calculateAndLogNetworkGetStats(t, networkGetLatencies)
}

// calculateAndLogNetworkGetStats calculates and logs comprehensive statistics for network GET operations
func calculateAndLogNetworkGetStats(t *testing.T, latencies []time.Duration) {
	if len(latencies) == 0 {
		t.Logf("Network GET Latency: No successful operations")
		return
	}

	// Sort latencies for percentile calculation
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	// Calculate metrics
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}

	avg := total / time.Duration(len(latencies))
	min := sortedLatencies[0]
	max := sortedLatencies[len(sortedLatencies)-1]
	p50 := sortedLatencies[len(sortedLatencies)*50/100]
	p95 := sortedLatencies[len(sortedLatencies)*95/100]
	p99 := sortedLatencies[len(sortedLatencies)*99/100]

	t.Logf("Network GET Latency Test Results:")
	t.Logf("  Total requests: %d", len(latencies))
	t.Logf("  Average latency: %v", avg)
	t.Logf("  Min latency: %v", min)
	t.Logf("  Max latency: %v", max)
	t.Logf("  P50 latency: %v", p50)
	t.Logf("  P95 latency: %v", p95)
	t.Logf("  P99 latency: %v", p99)
	t.Logf("  Total duration: %v", total)
}

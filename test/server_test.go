package test

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	crypto "github.com/Leon-CYL/Distributed_File_Storage/crypto"
	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
	"github.com/Leon-CYL/Distributed_File_Storage/server"
	"github.com/Leon-CYL/Distributed_File_Storage/store"
)

type mockTransport struct {
	addr      string
	consumeCh chan p2p.RPC
}

func newMockTransport(addr string) *mockTransport {
	return &mockTransport{
		addr:      addr,
		consumeCh: make(chan p2p.RPC, 1024),
	}
}

func (m *mockTransport) AcceptAndListen() error {
	return nil
}

func (m *mockTransport) Consume() <-chan p2p.RPC {
	return m.consumeCh
}

func (m *mockTransport) Close() error {
	return nil
}

func (m *mockTransport) Dial(addr string) error {
	return nil
}

func (m *mockTransport) Addr() string {
	return m.addr
}

type mockAddr string

func (m mockAddr) Network() string {
	return "mock"
}

func (m mockAddr) String() string {
	return string(m)
}

type mockPeer struct {
	bytes.Buffer
	remoteAddr net.Addr
	localAddr  net.Addr
	closed     bool
}

func newMockPeer(addr string) *mockPeer {
	return &mockPeer{
		remoteAddr: mockAddr(addr),
		localAddr:  mockAddr("local"),
	}
}

func (m *mockPeer) Send(data []byte) error {
	_, err := m.Write(data)
	return err
}

func (m *mockPeer) CloseStream() {
	m.closed = true
}

func (m *mockPeer) Close() error {
	return nil
}

func (m *mockPeer) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *mockPeer) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *mockPeer) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockPeer) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockPeer) SetWriteDeadline(t time.Time) error {
	return nil
}

func newTestFileServer(t *testing.T) *server.FileServer {
	t.Helper()

	transport := newMockTransport("mock-server")

	opts := server.FileServerOpts{
		EncryptionKey:     crypto.NewEncryptionKey(),
		StorageRoot:       t.TempDir(),
		PathTransformFunc: store.CASPathTransformFunc,
		Transport:         transport,
		BootstrapNodes:    []string{},
	}

	return server.NewFileServer(opts)
}

func TestNewFileServer(t *testing.T) {
	fs := newTestFileServer(t)

	if fs == nil {
		t.Fatal("expected file server to be created")
	}

	if fs.EncryptionKey == nil {
		t.Fatal("expected encryption key to be set")
	}

	if fs.Transport == nil {
		t.Fatal("expected transport to be set")
	}

	if fs.PathTransformFunc == nil {
		t.Fatal("expected path transform function to be set")
	}
}

func TestFileServerStoreAndGetLocalFile(t *testing.T) {
	fs := newTestFileServer(t)

	key := "test-file.txt"
	data := []byte("hello from file server")

	err := fs.Store(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	r, err := fs.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatalf("expected %q, got %q", string(data), string(out))
	}
}

func TestFileServerDelete(t *testing.T) {
	fs := newTestFileServer(t)

	key := "delete-file.txt"
	data := []byte("delete me")

	err := fs.Store(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Delete(key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Get(key)
	if err == nil {
		t.Fatal("expected error after deleting file, got nil")
	}
}

func TestFileServerStoreEmptyFile(t *testing.T) {
	fs := newTestFileServer(t)

	key := "empty-file.txt"
	data := []byte("")

	err := fs.Store(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	r, err := fs.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatalf("expected empty file, got %q", string(out))
	}
}

func TestFileServerStoreLargeFile(t *testing.T) {
	fs := newTestFileServer(t)

	key := "large-file.txt"
	data := bytes.Repeat([]byte("a"), 100*1024)

	err := fs.Store(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	r, err := fs.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatal("large file content does not match")
	}
}

func TestFileServerOverwriteFile(t *testing.T) {
	fs := newTestFileServer(t)

	key := "overwrite-file.txt"
	firstData := []byte("first version")
	secondData := []byte("second version")

	err := fs.Store(key, bytes.NewReader(firstData))
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Store(key, bytes.NewReader(secondData))
	if err != nil {
		t.Fatal(err)
	}

	r, err := fs.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, secondData) {
		t.Fatalf("expected %q, got %q", string(secondData), string(out))
	}
}

func TestFileServerGetMissingFileReturnsError(t *testing.T) {
	fs := newTestFileServer(t)

	_, err := fs.Get("missing-file.txt")
	if err == nil {
		t.Fatal("expected error when getting missing file, got nil")
	}
}

func TestFileServerOnPeer(t *testing.T) {
	fs := newTestFileServer(t)

	peer := newMockPeer("peer-1")

	err := fs.OnPeer(peer)
	if err != nil {
		t.Fatal(err)
	}

	err = fs.Store("peer-test.txt", bytes.NewReader([]byte("hello peer")))
	if err != nil {
		t.Fatal(err)
	}

	if peer.Len() == 0 {
		t.Fatal("expected peer to receive broadcast or stream data")
	}
}

func TestFileServerStopDoesNotPanic(t *testing.T) {
	fs := newTestFileServer(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Stop should not panic, got %v", r)
		}
	}()

	fs.Stop()
}
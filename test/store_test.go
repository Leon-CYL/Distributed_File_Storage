package test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	crypto "github.com/Leon-CYL/Distributed_File_Storage/crypto"
	"github.com/Leon-CYL/Distributed_File_Storage/store"
)

func newStore() *store.Store {
	opts := store.StoreOpts{
		PathTransformFunc: store.CASPathTransformFunc,
	}

	return store.NewStore(opts)
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()

	opts := store.StoreOpts{
		Root:              t.TempDir(),
		PathTransformFunc: store.CASPathTransformFunc,
	}

	return store.NewStore(opts)
}

func teardown(s *store.Store, t *testing.T) {
	t.Helper()

	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "somekey"
	pathkey := store.CASPathTransformFunc(key)
	expectedPathname := "f16b0/9e156/438c4/62053/0809f/c44c5/502a5/378da"
	expectedFilename := "f16b09e156438c4620530809fc44c5502a5378da"

	if pathkey.PathName != expectedPathname {
		t.Errorf("Wanted: %s, Got: %s", expectedPathname, pathkey.PathName)
	}

	if pathkey.Filename != expectedFilename {
		t.Errorf("Wanted: %s, Got: %s", expectedFilename, pathkey.Filename)
	}
}

func TestPathkeyFullPath(t *testing.T) {
	pathkey := store.Pathkey{
		PathName: "abc/def",
		Filename: "file.txt",
	}

	expected := "abc/def/file.txt"
	got := pathkey.FullPath()

	if got != expected {
		t.Fatalf("expected full path %q, got %q", expected, got)
	}
}

func TestPathkeyFirstPathName(t *testing.T) {
	pathkey := store.Pathkey{
		PathName: "abc/def/ghi",
		Filename: "file.txt",
	}

	expected := "abc"
	got := pathkey.FirstPathName()

	if got != expected {
		t.Fatalf("expected first path name %q, got %q", expected, got)
	}
}

func TestDefaultTransformFunc(t *testing.T) {
	key := "my-file.txt"

	pathkey := store.DefaultTransformFunc(key)

	if pathkey.PathName != key {
		t.Fatalf("expected pathname %q, got %q", key, pathkey.PathName)
	}

	if pathkey.Filename != key {
		t.Fatalf("expected filename %q, got %q", key, pathkey.Filename)
	}
}

func TestNewStoreDefaultOptions(t *testing.T) {
	s := store.NewStore(store.StoreOpts{})

	if s.Root != store.DefaultRootFolderName {
		t.Fatalf("expected default root %q, got %q", store.DefaultRootFolderName, s.Root)
	}

	if s.PathTransformFunc == nil {
		t.Fatal("expected default path transform function to be set")
	}
}

func TestStore(t *testing.T) {
	s := newTestStore(t)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("myspecialkey-%d", i)
		data := []byte("Hello, World!")

		if _, err := s.Write(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Errorf("Expected to have key: %s", key)
		}

		_, r, err := s.Read(key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)

		if string(b) != string(data) {
			t.Errorf("Wanted: %s, Got: %s", string(data), string(b))
		}

		if err := s.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok {
			t.Errorf("Expected to NOT have key: %s", key)
		}
	}
}

func TestStoreHasReturnsFalseForMissingKey(t *testing.T) {
	s := newTestStore(t)

	key := "missing-file.txt"

	if s.Has(key) {
		t.Fatalf("expected Has(%q) to be false before file is written", key)
	}
}

func TestStoreWriteReturnsCorrectSize(t *testing.T) {
	s := newTestStore(t)

	key := "size-test.txt"
	data := []byte("hello world")

	n, err := s.Write(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if n != int64(len(data)) {
		t.Fatalf("expected written size %d, got %d", len(data), n)
	}
}

func TestStoreReadReturnsCorrectSize(t *testing.T) {
	s := newTestStore(t)

	key := "read-size-test.txt"
	data := []byte("hello file size")

	_, err := s.Write(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	size, r, err := s.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	if size != int64(len(data)) {
		t.Fatalf("expected read size %d, got %d", len(data), size)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatalf("expected data %q, got %q", string(data), string(out))
	}
}

func TestStoreReadMissingFileReturnsError(t *testing.T) {
	s := newTestStore(t)

	_, _, err := s.Read("does-not-exist.txt")
	if err == nil {
		t.Fatal("expected error when reading missing file, got nil")
	}
}

func TestStoreDeleteMissingFileDoesNotError(t *testing.T) {
	s := newTestStore(t)

	err := s.Delete("does-not-exist.txt")
	if err != nil {
		t.Fatalf("expected deleting missing file to not error, got %v", err)
	}
}

func TestStoreClearRemovesAllFiles(t *testing.T) {
	s := newTestStore(t)

	keys := []string{
		"file-1.txt",
		"file-2.txt",
		"file-3.txt",
	}

	for _, key := range keys {
		_, err := s.Write(key, bytes.NewReader([]byte("data")))
		if err != nil {
			t.Fatal(err)
		}

		if !s.Has(key) {
			t.Fatalf("expected store to have key %q before clear", key)
		}
	}

	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}

	for _, key := range keys {
		if s.Has(key) {
			t.Fatalf("expected key %q to be removed after clear", key)
		}
	}
}

func TestStoreOverwriteExistingFile(t *testing.T) {
	s := newTestStore(t)

	key := "overwrite.txt"
	firstData := []byte("first version")
	secondData := []byte("second version with more data")

	_, err := s.Write(key, bytes.NewReader(firstData))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Write(key, bytes.NewReader(secondData))
	if err != nil {
		t.Fatal(err)
	}

	size, r, err := s.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	if size != int64(len(secondData)) {
		t.Fatalf("expected size %d after overwrite, got %d", len(secondData), size)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, secondData) {
		t.Fatalf("expected overwritten data %q, got %q", string(secondData), string(out))
	}
}

func TestStoreWriteLargeFile(t *testing.T) {
	s := newTestStore(t)

	key := "large-file.txt"
	data := bytes.Repeat([]byte("a"), 100*1024)

	n, err := s.Write(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	if n != int64(len(data)) {
		t.Fatalf("expected written size %d, got %d", len(data), n)
	}

	_, r, err := s.Read(key)
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

func TestStoreCreatesNestedCASPathOnWrite(t *testing.T) {
	s := newTestStore(t)

	key := "nested-path-key"
	data := []byte("nested path data")

	_, err := s.Write(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	pathkey := store.CASPathTransformFunc(key)
	fullPath := filepath.Join(s.Root, pathkey.PathName, pathkey.Filename)

	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("expected file to exist at %q, got error %v", fullPath, err)
	}
}

func TestWriteDecrypt(t *testing.T) {
	s := newTestStore(t)

	key := "encrypted-file.txt"
	encryptionKey := crypto.NewEncryptionKey()
	plaintext := []byte("this file should be encrypted before writing")

	encrypted := new(bytes.Buffer)

	_, err := crypto.CopyEncrypt(encryptionKey, bytes.NewReader(plaintext), encrypted)
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.WriteDecrypt(encryptionKey, key, encrypted)
	if err != nil {
		t.Fatal(err)
	}

	if n != int64(16+len(plaintext)) {
		t.Fatalf("expected WriteDecrypt to report %d bytes, got %d", 16+len(plaintext), n)
	}

	_, r, err := s.Read(key)
	if err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, plaintext) {
		t.Fatalf("expected decrypted data %q, got %q", string(plaintext), string(out))
	}
}

func TestWriteDecryptInvalidKeyReturnsError(t *testing.T) {
	s := newTestStore(t)

	key := "bad-key-file.txt"
	invalidKey := []byte("short-key")
	encryptedData := bytes.NewReader([]byte("not valid encrypted data"))

	_, err := s.WriteDecrypt(invalidKey, key, encryptedData)
	if err == nil {
		t.Fatal("expected WriteDecrypt to return error for invalid encryption key")
	}
}
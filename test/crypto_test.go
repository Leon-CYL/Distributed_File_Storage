package test

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"testing"

	crypto "github.com/Leon-CYL/Distributed_File_Storage/crypto"
)

func TestNewEncryptionKeyLength(t *testing.T) {
	key := crypto.NewEncryptionKey()

	if len(key) != 32 {
		t.Fatalf("expected key length 32 bytes, got %d", len(key))
	}
}

func TestNewEncryptionKeyRandomness(t *testing.T) {
	key1 := crypto.NewEncryptionKey()
	key2 := crypto.NewEncryptionKey()

	if bytes.Equal(key1, key2) {
		t.Fatal("expected two generated encryption keys to be different")
	}
}

func TestHashKey(t *testing.T) {
	key := "my-file.txt"

	expectedHash := md5.Sum([]byte(key))
	expected := hex.EncodeToString(expectedHash[:])

	got := crypto.HashKey(key)

	if got != expected {
		t.Fatalf("expected hash %s, got %s", expected, got)
	}
}

func TestHashKeyLength(t *testing.T) {
	got := crypto.HashKey("hello")

	if len(got) != 32 {
		t.Fatalf("expected MD5 hex string length 32, got %d", len(got))
	}
}

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "foo not bar"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	nw, err := crypto.CopyDecrypt(key, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if nw != 16+len(payload) {
		t.Fatalf("expected %d bytes written, got %d", 16+len(payload), nw)
	}

	if out.String() != payload {
		t.Fatalf("encryption/decryption failed: expected %q, got %q", payload, out.String())
	}
}

func TestCopyEncryptAddsIV(t *testing.T) {
	payload := []byte("hello world")
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	n, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	expectedSize := 16 + len(payload)

	if n != expectedSize {
		t.Fatalf("expected CopyEncrypt to report %d bytes, got %d", expectedSize, n)
	}

	if dst.Len() != expectedSize {
		t.Fatalf("expected encrypted buffer size %d, got %d", expectedSize, dst.Len())
	}
}

func TestCiphertextDoesNotEqualPlaintext(t *testing.T) {
	payload := []byte("this is a secret message")
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	encrypted := dst.Bytes()

	if bytes.Contains(encrypted, payload) {
		t.Fatal("ciphertext should not directly contain the plaintext")
	}
}

func TestCopyEncryptDecryptEmptyPayload(t *testing.T) {
	payload := ""
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	if dst.Len() != 16 {
		t.Fatalf("expected encrypted empty payload to only contain 16-byte IV, got %d bytes", dst.Len())
	}

	out := new(bytes.Buffer)
	_, err = crypto.CopyDecrypt(key, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if out.String() != payload {
		t.Fatalf("expected empty decrypted payload, got %q", out.String())
	}
}

func TestCopyEncryptDecryptLargePayload(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), 100*1024)
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	_, err = crypto.CopyDecrypt(key, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out.Bytes(), payload) {
		t.Fatal("large payload was not decrypted correctly")
	}
}

func TestCopyEncryptInvalidKey(t *testing.T) {
	payload := []byte("hello")
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)

	invalidKey := []byte("short-key")

	_, err := crypto.CopyEncrypt(invalidKey, src, dst)
	if err == nil {
		t.Fatal("expected error for invalid AES key size, got nil")
	}
}

func TestCopyDecryptInvalidKey(t *testing.T) {
	payload := []byte("hello")
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)

	validKey := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(validKey, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	invalidKey := []byte("short-key")
	out := new(bytes.Buffer)

	_, err = crypto.CopyDecrypt(invalidKey, dst, out)
	if err == nil {
		t.Fatal("expected error for invalid AES key size, got nil")
	}
}

func TestDecryptWithWrongKeyDoesNotMatchOriginalPayload(t *testing.T) {
	payload := []byte("secret data")
	src := bytes.NewReader(payload)
	dst := new(bytes.Buffer)

	key1 := crypto.NewEncryptionKey()
	key2 := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key1, src, dst)
	if err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	_, err = crypto.CopyDecrypt(key2, dst, out)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(out.Bytes(), payload) {
		t.Fatal("decrypting with the wrong key should not produce the original payload")
	}
}
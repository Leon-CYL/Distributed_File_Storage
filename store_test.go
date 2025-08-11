package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "somekey"
	pathkey := CASPathTransformFunc(key)
	expectedPathname := "f16b0/9e156/438c4/62053/0809f/c44c5/502a5/378da"
	expectedOriginal := "f16b09e156438c4620530809fc44c5502a5378da"

	if pathkey.PathName != expectedPathname {
		t.Errorf("Wanted: %s, Got: %s", expectedPathname, pathkey.PathName)
	}

	if pathkey.Filename != expectedOriginal {
		t.Errorf("Wanted: %s, Got: %s", expectedOriginal, pathkey.Filename)
	}
}

func TestStoreDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	store := NewStore(opts)

	key := "myspecialkey"

	if err := store.WriteStream(key, bytes.NewReader([]byte("Hello, World!"))); err != nil {
		t.Error(err)
	}

	if err := store.Delete(key); err != nil {
		t.Error(err)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	store := NewStore(opts)
	key := "myspecialkey"
	data := []byte("Hello, World!")

	if err := store.WriteStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if !store.Has(key) {
		t.Errorf("Expected to have key: %s", key)
	}

	r, err := store.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("Wanted: %s, Got: %s", string(data), string(b))
	}

	store.Delete(key)
}

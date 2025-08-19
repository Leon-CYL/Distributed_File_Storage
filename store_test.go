package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}

	return NewStore(opts)
}

func teardown(s *Store, t *testing.T) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "somekey"
	pathkey := CASPathTransformFunc(key)
	expectedPathname := "f16b0/9e156/438c4/62053/0809f/c44c5/502a5/378da"
	expectedFilename := "f16b09e156438c4620530809fc44c5502a5378da"

	if pathkey.PathName != expectedPathname {
		t.Errorf("Wanted: %s, Got: %s", expectedPathname, pathkey.PathName)
	}

	if pathkey.Filename != expectedFilename {
		t.Errorf("Wanted: %s, Got: %s", expectedFilename, pathkey.Filename)
	}
}

func TestStore(t *testing.T) {

	store := newStore()
	defer teardown(store, t)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("myspecialkey-%d", i)
		data := []byte("Hello, World!")

		if _, err := store.Write(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := store.Has(key); !ok {
			t.Errorf("Expected to have key: %s", key)
		}

		_, r, err := store.Read(key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)

		if string(b) != string(data) {
			t.Errorf("Wanted: %s, Got: %s", string(data), string(b))
		}

		if err := store.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := store.Has(key); ok {
			t.Errorf("Expected to NOT have key: %s", key)
		}
	}
}

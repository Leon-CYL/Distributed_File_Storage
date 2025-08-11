package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

const DefaultRootFolderName = "CYLNetwork"

func CASPathTransformFunc(key string) Pathkey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	hashSlices := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize

		hashSlices[i] = hashStr[from:to]
	}

	return Pathkey{
		PathName: strings.Join(hashSlices, "/"),
		Filename: hashStr,
	}
}

type PathTransformFunc func(string) Pathkey

var DefaultTransformFunc = func(key string) Pathkey {
	return Pathkey{
		PathName: key,
		Filename: key,
	}
}

type Pathkey struct {
	PathName string
	Filename string
}

func (p Pathkey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

func (p Pathkey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")

	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

type StoreOpts struct {
	Root string
	PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = DefaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathkey := s.PathTransformFunc(key)
	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())
	_, err := os.Stat(fullPath)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}

func (s *Store) Delete(key string) error {
	pathkey := s.PathTransformFunc(key)
	deletePath := fmt.Sprintf("%s/%s", s.Root, pathkey.FirstPathName())
	defer func() {
		log.Printf("Deleted [%s] from disk.", pathkey.Filename)
	}()
	return os.RemoveAll(deletePath)
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.ReadStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) ReadStream(key string) (io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)
	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())
	return os.Open(fullPath)
}

func (s *Store) WriteStream(key string, r io.Reader) error {
	pathkey := s.PathTransformFunc(key)
	root := fmt.Sprintf("%s/%s", s.Root, pathkey.PathName)
	if err := os.MkdirAll(root, os.ModePerm); err != nil {
		return err
	}

	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("Wrote (%d) bytes to disk: %s", n, fullPath)
	return nil
}

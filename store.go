package main

import (
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

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(key string) bool {
	pathkey := s.PathTransformFunc(key)
	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())
	_, err := os.Stat(fullPath)
	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathkey := s.PathTransformFunc(key)
	deletePath := fmt.Sprintf("%s/%s", s.Root, pathkey.FirstPathName())
	defer func() {
		log.Printf("Deleted [%s] from disk.", pathkey.Filename)
	}()
	return os.RemoveAll(deletePath)
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)
	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())
	file, err := os.Open(fullPath)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) WriteDecrypt(encryptionKey []byte, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(encryptionKey, r, f)
	return int64(n), err
}

func (s *Store) openFileForWriting(key string) (*os.File, error) {
	pathkey := s.PathTransformFunc(key)
	root := fmt.Sprintf("%s/%s", s.Root, pathkey.PathName)
	if err := os.MkdirAll(root, os.ModePerm); err != nil {
		return nil, err
	}

	fullPath := fmt.Sprintf("%s/%s", s.Root, pathkey.FullPath())

	return os.Create(fullPath)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}

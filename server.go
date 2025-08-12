package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	store    *Store
	quitCh   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Payload struct {
	Key  string
	Data []byte
}

func (fs *FileServer) braodcast(p *Payload) error {
	peers := []io.Writer{}

	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(io.MultiWriter(mw)).Encode(p)
}

func (fs *FileServer) storeData(key string, r io.Reader) error {
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := fs.store.Write(key, tee); err != nil {
		return err
	}

	p := &Payload{
		Key:  key,
		Data: buf.Bytes(),
	}

	fmt.Println(buf.Bytes())

	return fs.braodcast(p)
}

func (fs *FileServer) Stop() {
	close(fs.quitCh)
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with remote node: %s", p.RemoteAddr())

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		fmt.Println("File server shutting down...")
		fs.Transport.Close()
	}()

	for {
		select {
		case msg := <-fs.Transport.Consume():
			var payload Payload
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&payload); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Received payload: %+v\n", payload)

		case <-fs.quitCh:
			return
		}
	}
}

func (fs *FileServer) bootstrapNetwork() error {

	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		fmt.Println("Attemping to connect to node: ", addr)
		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("Error dialing bootstrap node: %s\n", err)
			}
		}(addr)
	}

	return nil
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.AcceptAndListen(); err != nil {
		return err
	}

	if len(fs.BootstrapNodes) > 0 {
		fs.bootstrapNetwork()
	}

	fs.loop()

	return nil
}

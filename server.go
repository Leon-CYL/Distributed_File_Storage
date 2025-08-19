package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

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

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		_, r, err := fs.store.Read(key)
		return r, err
	}

	fmt.Printf("[%s] don't have file(%s) locally, fetching from network...\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range fs.peers {
		// First read the file size from the peer so it will not block the stream
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)

		// Then read the file from the peer
		n, err := fs.store.Write(key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over network from [%s]: \n",fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()

	}

	_, r, err := fs.store.Read(key)
	return r, err
}

func (fs *FileServer) Store(key string, r io.Reader) error {

	fileBuf := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuf)

	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingStream})
		n, err := io.Copy(peer, fileBuf)
		if err != nil {
			return err
		}
		fmt.Printf("Receive and Written %d bytes to peer\n", n)
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
		fmt.Println("File server shutting down due to error or user quit...")
		fs.Transport.Close()
	}()

	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("Error decoding message: ", err)
			}

			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				log.Println("Error handling message: ", err)
			}

		case <-fs.quitCh:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, v)
	}

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("(%s) need to serve file(%s) but it does not exist on disk", fs.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving file(%s) over the network\n", fs.Transport.Addr(), msg.Key)

	fileSize, r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Printf("[%s] closing file(%s) after serving over the network\n", fs.Transport.Addr(), msg.Key)
		defer rc.Close()
	}

	peer, ok := fs.peers[from]

	if !ok {
		return fmt.Errorf("peer (%s) could not be found in peer map", from)
	}

	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] writte (%d) bytes over network to %s\n", fs.Transport.Addr(), n, from)

	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in peer map", from)
	}

	n, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()

	return nil
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

func (fs *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}

	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(io.MultiWriter(mw)).Encode(msg)
}

func (fs *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

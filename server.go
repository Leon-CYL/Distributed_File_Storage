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
	EncryptionKey     []byte
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

// First try to read file locally, it does not exist on the server locally,
// it broadcast a get file message to all its peer and read the encrypt file content 
// from the first peer and write it to the local disk with decrypt content.
func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		_, r, err := fs.store.Read(key)
		return r, err
	}

	fmt.Printf("[%s] don't have file(%s) locally, fetching from network...\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
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
		n, err := fs.store.WriteDecrypt(fs.EncryptionKey, key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over network from [%s]: \n", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()

	}

	_, r, err := fs.store.Read(key)
	return r, err
}

// Store writes the file to the local disk first, while also keeping a copy
// of the file data in memory. After the local write succeeds, it broadcasts
// a metadata message to all peers so they know a file is about to be sent.
// Then it creates one writer that points to all peer connections, marks the
// next data as an incoming file stream, encrypts the buffered file content,
// and sends the encrypted file bytes to every peer.
func (fs *FileServer) Store(key string, r io.Reader) error {

	fileBuf := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuf)

	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: size + 16,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := []io.Writer{}

	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	// MultiWriter writes the same data to every peer at the same time.
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	_, err = copyEncrypt(fs.EncryptionKey, fileBuf, mw)
	if err != nil {
		return err
	}

	return nil
}

// Start File server and Listen for request from client/peer
func (fs *FileServer) Start() error {
	fmt.Printf("Starting file server[%s]...\n", fs.Transport.Addr())
	if err := fs.Transport.AcceptAndListen(); err != nil {
		return err
	}

	fs.bootstrapNetwork()
	fs.loop()

	return nil
}

// Stop the file server
func (fs *FileServer) Stop() {
	close(fs.quitCh)
}

// Add a new peer to the P2P file system and notified this server that a new peer joined
func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with remote node: %s", p.RemoteAddr())

	return nil
}

// Connect to all the peer server
func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Printf("[%s] attemping to connect with remote node: %s\n", fs.Transport.Addr(), addr)
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("Error dialing bootstrap node: %s\n", err)
			}
		}(addr)
	}

	return nil
}

// Send a message to all peers
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

// Netowrk listen loop that accept peer read file messgae and peer write file message
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

// Helper function for handling peer massage
func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, v)
	}

	return nil
}

// handleMessageGetFile handles a peer's request to get a file.
// It first checks whether the requested file exists locally. If it exists,
// the server reads the file, finds the peer that requested it, sends a stream
// marker and the file size, then copies the file content over the network
// to that peer.

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

// hadnleMessageStoreFile hanlde the peer request for writing a file to its local disk
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

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

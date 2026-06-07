package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/Leon-CYL/Distributed_File_Storage/crypto"
	"github.com/Leon-CYL/Distributed_File_Storage/p2p"
	"github.com/Leon-CYL/Distributed_File_Storage/store"
)

type FileServerOpts struct {
	EncryptionKey     []byte
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc store.PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	store    *store.Store
	quitCh   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          store.NewStore(storeOpts),
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
	storageKey := crypto.HashKey(key)

	if fs.store.Has(storageKey) {
		_, r, err := fs.store.Read(storageKey)
		return r, err
	}

	fmt.Printf("[%s] don't have file(%s) locally, fetching from network...\n", fs.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: storageKey,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range fs.peers {
		var fileSize int64

		if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
			continue
		}

		n, err := fs.store.WriteDecrypt(fs.EncryptionKey, storageKey, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over network from [%s]\n", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()

		break
	}

	_, r, err := fs.store.Read(storageKey)
	return r, err
}

// Store writes the file to the local disk first, while also keeping a copy
// of the file data in memory. After the local write succeeds, it broadcasts
// a metadata message to all peers so they know a file is about to be sent.
// Then it creates one writer that points to all peer connections, marks the
// next data as an incoming file stream, encrypts the buffered file content,
// and sends the encrypted file bytes to every peer.
func (fs *FileServer) Store(key string, r io.Reader) error {
	storageKey := crypto.HashKey(key)

	fileBuf := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuf)

	size, err := fs.store.Write(storageKey, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  storageKey,
			Size: size + 16, // encrypted stream includes 16-byte IV
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

	if len(peers) == 0 {
		return nil
	}

	mw := io.MultiWriter(peers...)

	if _, err := mw.Write([]byte{p2p.IncomingStream}); err != nil {
		return err
	}

	_, err = crypto.CopyEncrypt(fs.EncryptionKey, fileBuf, mw)
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

// Delete a file from disk
func (fs *FileServer) Delete(key string) error {
	storageKey := crypto.HashKey(key)
	return fs.store.Delete(storageKey)
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
		defer rc.Close()
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in peer map", from)
	}

	if err := peer.Send([]byte{p2p.IncomingStream}); err != nil {
		return err
	}

	encryptedSize := fileSize + 16

	if err := binary.Write(peer, binary.LittleEndian, encryptedSize); err != nil {
		return err
	}

	n, err := crypto.CopyEncrypt(fs.EncryptionKey, r, peer)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] wrote (%d) encrypted bytes over network to %s\n", fs.Transport.Addr(), n, from)

	return nil
}

// hadnleMessageStoreFile hanlde the peer request for writing a file to its local disk
func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in peer map", from)
	}

	n, err := fs.store.WriteDecrypt(fs.EncryptionKey, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d decrypted bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

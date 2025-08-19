package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(r io.Reader, msg *RPC) error
}

type GOBDecoder struct{}

func (d GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dd DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)

	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	// In case of a stream that we are not encoding what is being sent over the network
	// We are just setting the stream to True so we can handle that in our logic.
	if peekBuf[0] == IncomingStream {
		msg.Stream = true
		return nil
	}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	
	if err != nil {
		return err
	}

	msg.Payload = buf[:n]
	return nil
}
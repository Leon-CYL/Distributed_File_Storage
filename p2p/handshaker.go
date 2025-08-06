package p2p

type HandshakeFunc func(any) error

func NOPHandshake(any) error {
	return nil
}
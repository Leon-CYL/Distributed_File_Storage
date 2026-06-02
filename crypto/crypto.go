package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// newEncryptionKey creates a new random 32-byte encryption key for AES-256.
func NewEncryptionKey() []byte {
	keyBuf := make([]byte, 32)
	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

// hashKey hashes a string key using MD5 and returns the hash as a hex string.
func HashKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// copyStream reads data from src, encrypts/decrypts it using the stream,
// writes the result to dst, and returns the total number of bytes written.
func CopyStream(stream cipher.Stream, blockSize int, src io.Reader, dst io.Writer) (int, error) {
	buf := make([]byte, 32*1024)
	nw := blockSize

	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dst.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw += nn
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return 0, err
		}
	}

	return nw, nil
}

// copyDecrypt reads the IV from src, creates an AES-CTR decrypt stream,
// decrypts the remaining data, and writes the plaintext to dst.
func CopyDecrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return CopyStream(stream, block.BlockSize(), src, dst)
}

// copyEncrypt creates a random IV, writes it to dst first,
// then encrypts data from src using AES-CTR and writes the ciphertext to dst.
func CopyEncrypt(key []byte, src io.Reader, dst io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	if _, err := dst.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	return CopyStream(stream, block.BlockSize(), src, dst)
}

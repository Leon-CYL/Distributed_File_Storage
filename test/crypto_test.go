package test

import (
	"bytes"
	"testing"

	crypto "github.com/Leon-CYL/Distributed_File_Storage/crypto"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "foo not bar"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := crypto.NewEncryptionKey()

	_, err := crypto.CopyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}

	out := new(bytes.Buffer)
	nw, err := crypto.CopyDecrypt(key, dst, out)
	if err != nil {
		t.Error(err)
	}

	if nw != 16+len(payload) {
		t.Fail()
	}

	if out.String() != payload {
		t.Error("Encryption Failed! Expected: ", payload, " Got: ", out.String())
	}
}

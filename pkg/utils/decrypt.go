package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

func Decrypt(b []byte) ([]byte, error) {
	key, err := RetrievePrivateKey()
	if err != nil {
		return nil, err
	}
	plaintext, err := jwe.Decrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, key))
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func DecryptWithMasterKey(b []byte, key []byte) ([]byte, error) {
	decryptedBuf := bytes.NewBuffer(nil)
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	data, err := gcm.Open(nil, b[:gcm.NonceSize()], b[gcm.NonceSize():], nil)
	if err != nil {
		return nil, err
	}
	_, err = decryptedBuf.Write(data)
	if err != nil {
		return nil, err
	}
	plaintext := decryptedBuf.Bytes()
	return plaintext, nil
}

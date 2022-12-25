package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"math/rand"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

func EncryptWithMasterKey(plaintext []byte, key []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	outBuf := gcm.Seal(nonce, nonce, plaintext, nil)
	if err != nil {
		return nil, err
	}
	return outBuf, err
}

func Encrypt(b []byte) ([]byte, error) {
	key, err := RetrievePublicKey()
	if err != nil {
		return nil, err
	}
	ciphertext, err := jwe.Encrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, key))
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

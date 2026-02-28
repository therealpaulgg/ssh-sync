package utils

import (
	"crypto/mlkem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

func Decrypt(b []byte) ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		return DecryptMLKEM(b, dk)
	default:
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
}

func DecryptMLKEM(data []byte, dk *mlkem.DecapsulationKey768) ([]byte, error) {
	if len(data) < mlkem.CiphertextSize768 {
		return nil, fmt.Errorf("data too short for ML-KEM ciphertext")
	}

	kemCiphertext := data[:mlkem.CiphertextSize768]
	remainder := data[mlkem.CiphertextSize768:]

	sharedSecret, err := dk.Decapsulate(kemCiphertext)
	if err != nil {
		return nil, fmt.Errorf("ML-KEM decapsulation failed: %w", err)
	}

	plaintext, err := aesGCMDecrypt(sharedSecret, remainder)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decryption failed: %w", err)
	}
	return plaintext, nil
}

func DecryptWithMasterKey(b []byte, key []byte) ([]byte, error) {
	return aesGCMDecrypt(key, b)
}

package utils

import (
	"crypto/mlkem"
	"encoding/pem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const mlkemCtSize = 1088

func EncryptWithMasterKey(plaintext []byte, key []byte) ([]byte, error) {
	return aesGCMEncrypt(key, plaintext)
}

func Encrypt(b []byte) ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		ek, err := RetrieveEncapsulationKey()
		if err != nil {
			return nil, err
		}
		return EncryptMLKEM(b, ek)
	default:
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
}

func EncryptMLKEM(plaintext []byte, ek *mlkem.EncapsulationKey768) ([]byte, error) {
	sharedSecret, kemCiphertext := ek.Encapsulate()

	nonceAndCiphertext, err := aesGCMEncrypt(sharedSecret, plaintext)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM encryption: %w", err)
	}

	result := make([]byte, 0, len(kemCiphertext)+len(nonceAndCiphertext))
	result = append(result, kemCiphertext...)
	result = append(result, nonceAndCiphertext...)
	return result, nil
}

func EncryptWithPQPublicKey(b []byte, ekPEM []byte) ([]byte, error) {
	ekBlock, _ := pem.Decode(ekPEM)
	if ekBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block for encapsulation key")
	}
	if ekBlock.Type != "MLKEM768 ENCAPSULATION KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", ekBlock.Type)
	}
	ek, err := mlkem.NewEncapsulationKey768(ekBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing ML-KEM-768 encapsulation key: %w", err)
	}

	return EncryptMLKEM(b, ek)
}

func EncryptWithECPublicKey(b []byte, key []byte) ([]byte, error) {
	pubKey, err := jwk.ParseKey(key, jwk.WithPEM(true))
	if err != nil {
		return nil, err
	}
	ciphertext, err := jwe.Encrypt(b, jwe.WithKey(jwa.ECDH_ES_A256KW, pubKey))
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

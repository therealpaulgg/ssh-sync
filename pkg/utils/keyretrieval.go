package utils

import (
	"bytes"
	"crypto/mlkem"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"filippo.io/mldsa"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func RetrievePrivateKey() (jwk.Key, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "keypair")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	key, err := jwk.ParseKey(file, jwk.WithPEM(true))
	return key, err
}

func RetrievePublicKey() (jwk.Key, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "keypair.pub")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	key, err := jwk.ParseKey(file, jwk.WithPEM(true))
	return key, err
}

func BuildECPublicKeyPEM() ([]byte, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "keypair.pub")
	return os.ReadFile(p)
}

func retrievePQSeed() ([]byte, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "keypair")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "SSHSYNC PQ MASTER SEED" {
			return block.Bytes, nil
		}
	}
	return nil, nil
}

func RetrieveSigningKey() (*mldsa.PrivateKey, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	sk, err := DeriveMLDSAKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-DSA-65 key from seed: %w", err)
	}
	return sk, nil
}

func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	dk, err := DeriveMLKEMKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving decapsulation key from seed: %w", err)
	}
	return dk, nil
}

func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	dk, err := RetrieveDecapsulationKey()
	if err != nil {
		return nil, err
	}
	return dk.EncapsulationKey(), nil
}

func BuildMLDSAPublicKeyPEM() ([]byte, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	sk, err := DeriveMLDSAKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-DSA key for public key PEM: %w", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "ML-DSA PUBLIC KEY", Bytes: sk.PublicKey().Bytes()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BuildMLKEMEncapsulationKeyPEM() ([]byte, error) {
	seed, err := retrievePQSeed()
	if err != nil {
		return nil, err
	}
	if seed == nil {
		return nil, fmt.Errorf("PQ master seed not found in keypair file")
	}
	dk, err := DeriveMLKEMKey(seed)
	if err != nil {
		return nil, fmt.Errorf("deriving ML-KEM key for encapsulation key PEM: %w", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "MLKEM768 ENCAPSULATION KEY", Bytes: dk.EncapsulationKey().Bytes()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func RetrieveMasterKey() ([]byte, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return nil, err
	}

	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync", "master_key")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	switch format {
	case FormatPostQuantum:
		dk, err := RetrieveDecapsulationKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := DecryptMLKEM(file, dk)
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (PQ): %w", err)
		}
		return masterKey, nil
	default:
		privateKey, err := RetrievePrivateKey()
		if err != nil {
			return nil, err
		}
		masterKey, err := jwe.Decrypt(file, jwe.WithKey(jwa.ECDH_ES_A256KW, privateKey))
		if err != nil {
			return nil, fmt.Errorf("decrypting master key (EC): %w", err)
		}
		return masterKey, nil
	}
}

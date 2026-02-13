package utils

import (
	"crypto/mlkem"
	"encoding/pem"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// RetrieveSigningPrivateKey loads the ML-DSA-65 private key from ~/.ssh-sync/keypair
func RetrieveSigningPrivateKey() (*mldsa65.PrivateKey, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair")
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
		if block.Type == "MLDSA65 PRIVATE KEY" {
			var sk mldsa65.PrivateKey
			if err := sk.UnmarshalBinary(block.Bytes); err != nil {
				return nil, fmt.Errorf("unmarshaling ML-DSA-65 private key: %w", err)
			}
			return &sk, nil
		}
	}
	return nil, fmt.Errorf("ML-DSA-65 private key not found in %s", p)
}

// RetrieveSigningPublicKey loads the ML-DSA-65 public key from ~/.ssh-sync/keypair.pub
func RetrieveSigningPublicKey() (*mldsa65.PublicKey, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
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
		if block.Type == "MLDSA65 PUBLIC KEY" {
			var pk mldsa65.PublicKey
			if err := pk.UnmarshalBinary(block.Bytes); err != nil {
				return nil, fmt.Errorf("unmarshaling ML-DSA-65 public key: %w", err)
			}
			return &pk, nil
		}
	}
	return nil, fmt.Errorf("ML-DSA-65 public key not found in %s", p)
}

// RetrieveDecapsulationKey loads the ML-KEM-768 decapsulation key from ~/.ssh-sync/keypair
func RetrieveDecapsulationKey() (*mlkem.DecapsulationKey768, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair")
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
		if block.Type == "MLKEM768 DECAPSULATION KEY SEED" {
			dk, err := mlkem.NewDecapsulationKey768(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("creating ML-KEM-768 decapsulation key: %w", err)
			}
			return dk, nil
		}
	}
	return nil, fmt.Errorf("ML-KEM-768 decapsulation key not found in %s", p)
}

// RetrieveEncapsulationKey loads the ML-KEM-768 encapsulation key from ~/.ssh-sync/keypair.pub
func RetrieveEncapsulationKey() (*mlkem.EncapsulationKey768, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "keypair.pub")
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
		if block.Type == "MLKEM768 ENCAPSULATION KEY" {
			ek, err := mlkem.NewEncapsulationKey768(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("creating ML-KEM-768 encapsulation key: %w", err)
			}
			return ek, nil
		}
	}
	return nil, fmt.Errorf("ML-KEM-768 encapsulation key not found in %s", p)
}

// RetrieveMasterKey reads and decrypts the master key from ~/.ssh-sync/master_key.
// The master key is protected using ML-KEM-768 key encapsulation + AES-256-GCM.
// Format: [1088 bytes ML-KEM ciphertext][AES-GCM encrypted master key (nonce + ciphertext)]
func RetrieveMasterKey() ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".ssh-sync", "master_key")
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	dk, err := RetrieveDecapsulationKey()
	if err != nil {
		return nil, err
	}
	masterKey, err := DecryptMLKEM(file, dk)
	if err != nil {
		return nil, fmt.Errorf("decrypting master key: %w", err)
	}
	return masterKey, nil
}

package utils

import (
	"crypto/ecdh"
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// MasterSeedSize is the size in bytes of the hybrid master seed.
const MasterSeedSize = 64

// DeriveHybridKeys deterministically derives both keypairs from a single
// master seed using HKDF with domain separation:
//   - EC P-256 private key for ECDH key agreement and ECDSA signing (info: "ssh-sync-ec-v1")
//   - ML-KEM-768 decapsulation key for post-quantum KEM (info: "ssh-sync-mlkem768-v1")
func DeriveHybridKeys(masterSeed []byte) (*ecdh.PrivateKey, *mlkem.DecapsulationKey768, error) {
	if len(masterSeed) != MasterSeedSize {
		return nil, nil, fmt.Errorf("master seed must be %d bytes, got %d", MasterSeedSize, len(masterSeed))
	}

	// Derive EC P-256 private key (32 bytes)
	ecReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-ec-v1"))
	ecSeed := make([]byte, 32)
	if _, err := io.ReadFull(ecReader, ecSeed); err != nil {
		return nil, nil, fmt.Errorf("deriving EC P-256 seed: %w", err)
	}
	ecPriv, err := ecdh.P256().NewPrivateKey(ecSeed)
	if err != nil {
		return nil, nil, fmt.Errorf("creating EC P-256 private key: %w", err)
	}

	// Derive ML-KEM-768 seed (64 bytes)
	kemReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-mlkem768-v1"))
	kemSeed := make([]byte, mlkem.SeedSize)
	if _, err := io.ReadFull(kemReader, kemSeed); err != nil {
		return nil, nil, fmt.Errorf("deriving ML-KEM-768 seed: %w", err)
	}
	dk, err := mlkem.NewDecapsulationKey768(kemSeed)
	if err != nil {
		return nil, nil, fmt.Errorf("creating ML-KEM-768 key from derived seed: %w", err)
	}

	return ecPriv, dk, nil
}
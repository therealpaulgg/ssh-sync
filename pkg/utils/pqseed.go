package utils

import (
	"crypto/mlkem"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"golang.org/x/crypto/hkdf"
)

// MasterSeedSize is the size in bytes of the PQ master seed.
const MasterSeedSize = 64

// DerivePQKeys deterministically derives both post-quantum keypairs from a
// single master seed using HKDF with domain separation:
//   - ML-DSA-65 signing keypair (info: "ssh-sync-mldsa65-v1")
//   - ML-KEM-768 key encapsulation keypair (info: "ssh-sync-mlkem768-v1")
func DerivePQKeys(masterSeed []byte) (*mldsa65.PublicKey, *mldsa65.PrivateKey, *mlkem.DecapsulationKey768, error) {
	if len(masterSeed) != MasterSeedSize {
		return nil, nil, nil, fmt.Errorf("master seed must be %d bytes, got %d", MasterSeedSize, len(masterSeed))
	}

	// Derive ML-DSA-65 seed (32 bytes)
	dsaReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-mldsa65-v1"))
	var dsaSeed [mldsa65.SeedSize]byte
	if _, err := io.ReadFull(dsaReader, dsaSeed[:]); err != nil {
		return nil, nil, nil, fmt.Errorf("deriving ML-DSA-65 seed: %w", err)
	}
	sigPub, sigPriv := mldsa65.NewKeyFromSeed(&dsaSeed)

	// Derive ML-KEM-768 seed (64 bytes)
	kemReader := hkdf.New(sha256.New, masterSeed, nil, []byte("ssh-sync-mlkem768-v1"))
	kemSeed := make([]byte, mlkem.SeedSize)
	if _, err := io.ReadFull(kemReader, kemSeed); err != nil {
		return nil, nil, nil, fmt.Errorf("deriving ML-KEM-768 seed: %w", err)
	}
	dk, err := mlkem.NewDecapsulationKey768(kemSeed)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating ML-KEM-768 key from derived seed: %w", err)
	}

	return sigPub, sigPriv, dk, nil
}

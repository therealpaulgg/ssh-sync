package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSeed(t *testing.T) []byte {
	t.Helper()
	seed := make([]byte, MasterSeedSize)
	_, err := rand.Read(seed)
	require.NoError(t, err)
	return seed
}

// TestDeriveMLDSAKey_Deterministic verifies that the same master seed always
// produces the same ML-DSA-65 keypair.
func TestDeriveMLDSAKey_Deterministic(t *testing.T) {
	seed := newTestSeed(t)

	sk1, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)

	sk2, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(sk1.Bytes(), sk2.Bytes()), "same seed must yield identical ML-DSA private keys")
	assert.True(t, bytes.Equal(sk1.PublicKey().Bytes(), sk2.PublicKey().Bytes()), "same seed must yield identical ML-DSA public keys")
}

// TestDeriveMLDSAKey_DifferentSeeds verifies that different seeds produce
// different keys.
func TestDeriveMLDSAKey_DifferentSeeds(t *testing.T) {
	seed1, seed2 := newTestSeed(t), newTestSeed(t)

	sk1, err := DeriveMLDSAKey(seed1)
	require.NoError(t, err)

	sk2, err := DeriveMLDSAKey(seed2)
	require.NoError(t, err)

	assert.False(t, bytes.Equal(sk1.Bytes(), sk2.Bytes()), "different seeds must yield different ML-DSA private keys")
}

// TestDeriveMLDSAKey_InvalidSeedSize verifies that an incorrect seed size
// returns an error.
func TestDeriveMLDSAKey_InvalidSeedSize(t *testing.T) {
	_, err := DeriveMLDSAKey(make([]byte, 16))
	assert.Error(t, err)
}

// TestDeriveMLKEMKey_Deterministic verifies that the same master seed always
// produces the same ML-KEM-768 keypair.
func TestDeriveMLKEMKey_Deterministic(t *testing.T) {
	seed := newTestSeed(t)

	dk1, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)

	dk2, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(dk1.EncapsulationKey().Bytes(), dk2.EncapsulationKey().Bytes()), "same seed must yield identical ML-KEM encapsulation keys")
}

// TestDeriveMLKEMKey_InvalidSeedSize verifies that an incorrect seed size
// returns an error.
func TestDeriveMLKEMKey_InvalidSeedSize(t *testing.T) {
	_, err := DeriveMLKEMKey(make([]byte, 16))
	assert.Error(t, err)
}

// TestEncryptDecryptMLKEM verifies a full ML-KEM-768 + AES-256-GCM round-trip.
func TestEncryptDecryptMLKEM(t *testing.T) {
	seed := newTestSeed(t)

	dk, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)

	plaintext := []byte("hello post-quantum world")

	ciphertext, err := EncryptMLKEM(plaintext, dk.EncapsulationKey())
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	recovered, err := DecryptMLKEM(ciphertext, dk)
	require.NoError(t, err)
	assert.Equal(t, plaintext, recovered)
}

// TestEncryptMLKEM_DifferentCiphertexts verifies that two encryptions of the
// same plaintext produce different ciphertexts (due to fresh randomness).
func TestEncryptMLKEM_DifferentCiphertexts(t *testing.T) {
	seed := newTestSeed(t)
	dk, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)
	ek := dk.EncapsulationKey()

	plaintext := []byte("same plaintext")

	ct1, err := EncryptMLKEM(plaintext, ek)
	require.NoError(t, err)
	ct2, err := EncryptMLKEM(plaintext, ek)
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "repeated encryption must produce distinct ciphertexts")
}

// TestDecryptMLKEM_Truncated verifies that truncated ciphertext returns an error.
func TestDecryptMLKEM_Truncated(t *testing.T) {
	seed := newTestSeed(t)
	dk, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)

	_, err = DecryptMLKEM(make([]byte, 100), dk)
	assert.Error(t, err)
}

// TestVerifyMLDSAJWT verifies the JWT sign+verify round-trip using the same
// encoding as getTokenPQ.
func TestVerifyMLDSAJWT(t *testing.T) {
	seed := newTestSeed(t)

	sk, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)

	header := `{"alg":"MLDSA","typ":"JWT"}`
	payload := `{"iss":"test"}`

	b64Header := base64.RawURLEncoding.EncodeToString([]byte(header))
	b64Payload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	signingInput := b64Header + "." + b64Payload

	sig, err := sk.Sign(rand.Reader, []byte(signingInput), nil)
	require.NoError(t, err)

	token := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	ok, err := VerifyMLDSAJWT(token, sk.PublicKey())
	require.NoError(t, err)
	assert.True(t, ok)
}

// TestVerifyMLDSAJWT_WrongKey verifies that a token signed with one key fails
// verification against a different key.
func TestVerifyMLDSAJWT_WrongKey(t *testing.T) {
	seed1 := newTestSeed(t)
	seed2 := newTestSeed(t)

	sk1, err := DeriveMLDSAKey(seed1)
	require.NoError(t, err)
	sk2, err := DeriveMLDSAKey(seed2)
	require.NoError(t, err)

	header := `{"alg":"MLDSA","typ":"JWT"}`
	payload := `{"iss":"test"}`
	signingInput := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))

	sig, err := sk1.Sign(rand.Reader, []byte(signingInput), nil)
	require.NoError(t, err)

	token := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	ok, err := VerifyMLDSAJWT(token, sk2.PublicKey())
	require.NoError(t, err)
	assert.False(t, ok, "token signed with key1 must not verify with key2")
}

// TestVerifyMLDSAJWT_InvalidFormat verifies that a malformed token returns an error.
func TestVerifyMLDSAJWT_InvalidFormat(t *testing.T) {
	seed := newTestSeed(t)
	sk, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)

	_, err = VerifyMLDSAJWT("notavalidtoken", sk.PublicKey())
	assert.Error(t, err)
}

// TestMLDSAAndMLKEMDerivedFromSameSeedAreIndependent verifies that derivation
// produces distinct key material for each algorithm.
func TestMLDSAAndMLKEMDerivedFromSameSeedAreIndependent(t *testing.T) {
	seed := newTestSeed(t)

	sk, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)
	dk, err := DeriveMLKEMKey(seed)
	require.NoError(t, err)

	assert.NotNil(t, sk)
	assert.NotNil(t, dk)

	// The ML-DSA seed bytes should not appear verbatim in the ML-KEM encapsulation key.
	assert.False(t, bytes.Contains(dk.EncapsulationKey().Bytes(), sk.Bytes()),
		"ML-KEM encapsulation key must not contain ML-DSA seed bytes")
}

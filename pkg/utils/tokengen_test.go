package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"filippo.io/mldsa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestSigningKey(t *testing.T) *mldsa.PrivateKey {
	t.Helper()
	seed := make([]byte, MasterSeedSize)
	_, err := rand.Read(seed)
	require.NoError(t, err)
	sk, err := DeriveMLDSAKey(seed)
	require.NoError(t, err)
	return sk
}

func decodeTokenParts(t *testing.T, token string) (header, payload map[string]any, sigBytes []byte) {
	t.Helper()
	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3, "JWT must have three dot-separated parts")

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(headerBytes, &header))

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payloadBytes, &payload))

	sigBytes, err = base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	return
}

func TestBuildPQToken_WellFormed(t *testing.T) {
	sk := generateTestSigningKey(t)
	token, err := buildPQToken(sk, "alice", "laptop")
	fmt.Println(token)
	require.NoError(t, err)

	parts := strings.SplitN(token, ".", 3)
	assert.Len(t, parts, 3, "JWT must have three dot-separated parts")
	for i, part := range parts {
		_, decodeErr := base64.RawURLEncoding.DecodeString(part)
		assert.NoErrorf(t, decodeErr, "part %d must be valid base64url", i)
	}
}

func TestBuildPQToken_Header(t *testing.T) {
	sk := generateTestSigningKey(t)
	token, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)

	header, _, _ := decodeTokenParts(t, token)
	assert.Equal(t, mldsa.MLDSA65().String(), header["alg"])
	assert.Equal(t, "JWT", header["typ"])
}

func TestBuildPQToken_PayloadClaims(t *testing.T) {
	sk := generateTestSigningKey(t)
	token, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)

	_, payload, _ := decodeTokenParts(t, token)

	assert.Equal(t, "github.com/therealpaulgg/ssh-sync", payload["iss"])
	assert.Equal(t, "alice", payload["username"])
	assert.Equal(t, "laptop", payload["machine"])

	now := time.Now().Unix()
	exp := int64(payload["exp"].(float64))
	iat := int64(payload["iat"].(float64))
	assert.Greater(t, exp, now, "exp must be in the future")
	assert.Less(t, iat, now, "iat must be in the past")
}

func TestBuildPQToken_ExpiryWindow(t *testing.T) {
	sk := generateTestSigningKey(t)
	before := time.Now()
	token, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)
	after := time.Now()

	_, payload, _ := decodeTokenParts(t, token)
	exp := int64(payload["exp"].(float64))
	iat := int64(payload["iat"].(float64))

	// exp should be ~2 minutes from now
	assert.GreaterOrEqual(t, exp, before.Add(time.Minute).Unix())
	assert.LessOrEqual(t, exp, after.Add(3*time.Minute).Unix())

	// iat should be ~1 minute in the past
	assert.LessOrEqual(t, iat, before.Unix())
	assert.GreaterOrEqual(t, iat, before.Add(-2*time.Minute).Unix())
}

func TestBuildPQToken_SignatureVerifies(t *testing.T) {
	sk := generateTestSigningKey(t)
	token, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)

	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	signingInput := parts[0] + "." + parts[1]
	err = mldsa.Verify(sk.PublicKey(), []byte(signingInput), sigBytes, nil)
	assert.NoError(t, err, "signature must verify against the signing key's public key")
}

func TestBuildPQToken_TamperedPayloadFails(t *testing.T) {
	sk := generateTestSigningKey(t)
	token, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)

	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)

	// Replace the payload with crafted claims while keeping the original signature
	parts[1] = base64.RawURLEncoding.EncodeToString(
		[]byte(`{"username":"evil","machine":"bad","exp":9999999999}`),
	)
	tamperedInput := parts[0] + "." + parts[1]

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	err = mldsa.Verify(sk.PublicKey(), []byte(tamperedInput), sigBytes, nil)
	assert.Error(t, err, "tampered payload must not pass signature verification")
}

func TestBuildPQToken_WrongKeyFails(t *testing.T) {
	sk1 := generateTestSigningKey(t)
	sk2 := generateTestSigningKey(t)

	token, err := buildPQToken(sk1, "alice", "laptop")
	require.NoError(t, err)

	parts := strings.SplitN(token, ".", 3)
	require.Len(t, parts, 3)

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	signingInput := parts[0] + "." + parts[1]
	err = mldsa.Verify(sk2.PublicKey(), []byte(signingInput), sigBytes, nil)
	assert.Error(t, err, "signature from sk1 must not verify with sk2's public key")
}

func TestBuildPQToken_DifferentUsersDifferentTokens(t *testing.T) {
	sk := generateTestSigningKey(t)

	token1, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)
	token2, err := buildPQToken(sk, "bob", "desktop")
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)

	_, payload1, _ := decodeTokenParts(t, token1)
	_, payload2, _ := decodeTokenParts(t, token2)

	assert.Equal(t, "alice", payload1["username"])
	assert.Equal(t, "laptop", payload1["machine"])
	assert.Equal(t, "bob", payload2["username"])
	assert.Equal(t, "desktop", payload2["machine"])
}

func TestBuildPQToken_UniqueSignaturesPerCall(t *testing.T) {
	// ML-DSA is deterministic given the same input, but time.Now() advances
	// between calls, producing different payloads (and hence different signatures).
	sk := generateTestSigningKey(t)

	token1, err := buildPQToken(sk, "alice", "laptop")
	require.NoError(t, err)

	// Both tokens use the same key + user, so verify both signatures are valid.
	parts1 := strings.SplitN(token1, ".", 3)
	require.Len(t, parts1, 3)
	sig1, err := base64.RawURLEncoding.DecodeString(parts1[2])
	require.NoError(t, err)
	err = mldsa.Verify(sk.PublicKey(), []byte(parts1[0]+"."+parts1[1]), sig1, nil)
	assert.NoError(t, err, "first token signature must verify")
}

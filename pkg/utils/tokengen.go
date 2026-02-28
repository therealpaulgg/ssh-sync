package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"filippo.io/mldsa"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func GetToken() (string, error) {
	format, err := DetectKeyFormat()
	if err != nil {
		return "", err
	}

	switch format {
	case FormatPostQuantum:
		return getTokenPQ()
	default:
		return getTokenEC()
	}
}

func getTokenEC() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	key, err := RetrievePrivateKey()
	if err != nil {
		return "", err
	}
	builder := jwt.NewBuilder()
	builder.Issuer("github.com/therealpaulgg/ssh-sync")
	builder.IssuedAt(time.Now().Add(-1 * time.Minute))
	builder.Expiration(time.Now().Add(2 * time.Minute))
	builder.Claim("username", profile.Username)
	builder.Claim("machine", profile.MachineName)
	tok, err := builder.Build()
	if err != nil {
		return "", err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES512, key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

func getTokenPQ() (string, error) {
	profile, err := GetProfile()
	if err != nil {
		return "", err
	}
	sk, err := RetrieveSigningKey()
	if err != nil {
		return "", err
	}
	return buildPQToken(sk, profile.Username, profile.MachineName)
}

// buildPQToken constructs a JWT signed with ML-DSA-65 using the algorithm
// identifier defined in draft-ietf-cose-dilithium:
// https://datatracker.ietf.org/doc/draft-ietf-cose-dilithium/
func buildPQToken(sk *mldsa.PrivateKey, username, machineName string) (string, error) {
	header := map[string]string{
		"alg": mldsa.MLDSA65().String(),
		"typ": "JWT",
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT header: %w", err)
	}

	now := time.Now()
	payload := map[string]interface{}{
		"iss":      "github.com/therealpaulgg/ssh-sync",
		"iat":      now.Add(-1 * time.Minute).Unix(),
		"exp":      now.Add(2 * time.Minute).Unix(),
		"username": username,
		"machine":  machineName,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling JWT payload: %w", err)
	}

	b64Header := base64.RawURLEncoding.EncodeToString(headerJSON)
	b64Payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := b64Header + "." + b64Payload

	sig, err := sk.Sign(rand.Reader, []byte(signingInput), nil)
	if err != nil {
		return "", fmt.Errorf("ML-DSA signing: %w", err)
	}
	b64Sig := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + b64Sig, nil
}


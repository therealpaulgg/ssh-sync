package utils

import (
	"os"
	"os/user"
	"path/filepath"

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

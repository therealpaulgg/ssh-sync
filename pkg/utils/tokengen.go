package utils

import (
	"encoding/json"
	"os"
	"os/user"
	"path"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/therealpaulgg/ssh-sync/pkg/models"
)

func GetToken() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	p := path.Join(user.HomeDir, ".ssh-sync", "profile.json")
	jsonBytes, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	var profile models.Profile
	err = json.Unmarshal(jsonBytes, &profile)
	if err != nil {
		return "", err
	}
	key, err := RetrievePrivateKey()
	if err != nil {
		return "", err
	}
	builder := jwt.NewBuilder()
	builder.Issuer("github.com/therealpaulgg/ssh-sync")
	builder.IssuedAt(time.Now())
	builder.Expiration(time.Now().Add(time.Minute))
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

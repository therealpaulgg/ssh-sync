package actions

import (
	"fmt"
	"os"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Token(c *cli.Context) error {

	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	key, err := utils.RetrievePrivateKey()
	if err != nil {
		return err
	}
	builder := jwt.NewBuilder()
	builder.Issuer("github.com/therealpaulgg/ssh-sync")
	builder.IssuedAt(time.Now())
	builder.Expiration(time.Now().Add(time.Hour))
	builder.Claim("username", "therealpaulgg")
	builder.Claim("machine", "largeboi")
	tok, err := builder.Build()
	if err != nil {
		return err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES512, key))
	if err != nil {
		return err
	}
	fmt.Println(string(signed))
	return nil
}

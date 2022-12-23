package actions

import (
	"fmt"
	"os"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
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
	// pubKey, err := RetrievePublicKey()
	// if err != nil {
	// 	return err
	// }
	tok, err := jwt.NewBuilder().Issuer("github.com/therealpaulgg/ssh-sync").IssuedAt(time.Now()).Expiration(time.Now().Add(time.Minute)).Build()
	if err != nil {
		return err
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.ES512, key))
	if err != nil {
		return err
	}
	fmt.Println(string(signed))

	decryptedKey, err := jwe.Decrypt([]byte("eyJhbGciOiJFQ0RILUVTK0EyNTZLVyIsImVuYyI6IkEyNTZHQ00iLCJlcGsiOnsiY3J2IjoiUC0yNTYiLCJrdHkiOiJFQyIsIngiOiJfb0FXQmgxdVVhc1pzYlp4N1hOMnlZRXZCWlBLSWVhTk44Y0ZpZ205UkZNIiwieSI6IjBvNzFIeW9NM19FQjh0SWVlc2NCZm1GdlNuXzBoS0k2U0JTakZSMWJwWTgifX0.1rl6TIMaKQwQ65P8SNhUe4DMcbw4ZMG89kBcXjcE9_BYEQbrpDgJUg.da3Mtd1TuK_8K_Um.-xBdwiBR46FjiJM.e7QNeJtWm5Cq0Bcl7xIWEQ"), jwe.WithKey(jwa.ECDH_ES_A256KW, key))
	if err != nil {
		return err
	}
	fmt.Println(string(decryptedKey))
	return nil

}

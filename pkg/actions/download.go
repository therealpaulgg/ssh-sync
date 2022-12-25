package actions

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Download(c *cli.Context) error {
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	fmt.Println(token)
	// Computer A has uploaded their keys to the server
	// Computer B wants to download the keys from the server
	// How can the server store the keys encrypted, and allow Computer B to decrypt them?
	// Each user on the server should have a shared master key. There is one copy of this encrypted master key for each PK pair uploaded.
	// server sends encrypted master key corresponding to that client's keypair
	// server also sends all the encrypted keys
	// client decrypts master key with their private key
	// client decrypts all the keys with the master key
	// TODO
	return nil
}

package actions

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gobwas/ws"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func ChallengeResponse(c *cli.Context) error {
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
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	dialer := ws.Dialer{}
	dialer.Header = ws.HandshakeHeaderHTTP(http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	wsUrl := profile.ServerUrl
	if wsUrl.Scheme == "http" {
		wsUrl.Scheme = "ws"
	} else {
		wsUrl.Scheme = "wss"
	}
	wsUrl.Path = "/api/v1/setup/challenge"
	conn, _, _, err := dialer.Dial(context.Background(), wsUrl.String())
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Print("Please enter the challenge phrase: ")
	var answer string
	if _, err := fmt.Scanln(&answer); err != nil {
		return err
	}
	if err := utils.WriteClientMessage(&conn, dto.ChallengeResponseDto{
		Challenge: answer,
	}); err != nil {
		return err
	}
	response, err := utils.ReadServerMessage[dto.ChallengeSuccessEncryptedKeyDto](&conn)
	// TODO - if connection is closed, close gracefully
	if err != nil {
		return err
	}
	masterKey, err := utils.Decrypt(response.Data.EncryptedMasterKey)
	if err != nil {
		return err
	}
	encryptedMasterKey2, err := utils.EncryptWithPublicKey(masterKey, response.Data.PublicKey)
	if err != nil {
		return err
	}
	if err := utils.WriteClientMessage(&conn, dto.EncryptedMasterKeyDto{EncryptedMasterKey: encryptedMasterKey2}); err != nil {
		return err
	}
	fmt.Println("Challenge has been successfully completed and your new encrypted master key has been sent to server. You may now use ssh-sync on your new machine.")
	// now send
	return nil
}

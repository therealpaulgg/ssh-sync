package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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

	dialer := ws.Dialer{}
	dialer.Header = ws.HandshakeHeaderHTTP(http.Header{
		"Authorization": []string{"Bearer " + token},
	})
	conn, _, _, err := dialer.Dial(context.Background(), "ws://localhost:3000/api/v1/setup/challenge")
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Print("Please enter the challenge phrase: ")
	var answer string
	_, err = fmt.Scanln(&answer)
	if err != nil {
		return err
	}
	b, err := json.Marshal(dto.ChallengeResponseDto{
		Challenge: answer,
	})
	if err != nil {
		return err
	}
	err = wsutil.WriteClientBinary(conn, b)
	if err != nil {
		return err
	}
	// TODO - if connection is closed, close gracefully
	keyData, err := wsutil.ReadServerBinary(conn)
	if err != nil {
		return err
	}
	var response dto.ChallengeSuccessEncryptedKeyDto
	err = json.Unmarshal(keyData, &response)
	if err != nil {
		return err
	}
	masterKey, err := utils.Decrypt(response.EncryptedMasterKey)
	if err != nil {
		return err
	}
	encryptedMasterKey2, err := utils.EncryptWithPublicKey(masterKey, response.PublicKey)
	if err != nil {
		return err
	}
	err = wsutil.WriteClientBinary(conn, encryptedMasterKey2)
	if err != nil {
		return err
	}
	fmt.Println("Challenge has been successfully completed and your new encrypted master key has been sent to server. You may now use ssh-sync on your new machine.")
	// now send
	return nil
}

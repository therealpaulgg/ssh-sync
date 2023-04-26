package actions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func RemoveMachine(c *cli.Context) error {
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	fmt.Print("Please enter the machine name: ")
	scanner := bufio.NewScanner(os.Stdin)
	var answer string
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(dto.MachineDto{
		Name: answer,
	}); err != nil {
		return err
	}
	url := profile.ServerUrl
	url.Path = "/api/v1/machines/"
	req, err := http.NewRequest("DELETE", url.String(), buf)
	if err != nil {
		return err
	}
	token, err := utils.GetToken()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

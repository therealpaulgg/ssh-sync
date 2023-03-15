package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func ListMachines(c *cli.Context) error {
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
	url := profile.ServerUrl
	url.Path = "/api/v1/machines/"
	req, err := http.NewRequest("GET", url.String(), nil)
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
	machines := []dto.MachineDto{}
	err = json.NewDecoder(resp.Body).Decode(&machines)
	if err != nil {
		return err
	}
	for _, machine := range machines {
		fmt.Println(machine.Name)
	}
	return nil
}

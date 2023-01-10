package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Reset(c *cli.Context) error {
	fmt.Print("This will delete all ssh-sync data relating to this machine. Continue? (y/n): ")
	var answer string
	_, err := fmt.Scanln(&answer)
	if err != nil {
		return err
	}
	if answer != "y" {
		return nil
	}
	prof, err := getProfile()
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(dto.MachineDto{
		Name: prof.MachineName,
	}); err != nil {
		return err
	}
	url := prof.ServerUrl
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

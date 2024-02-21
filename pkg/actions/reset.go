package actions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func Reset(c *cli.Context) error {
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	fmt.Print("This will delete all ssh-sync data relating to this machine. Continue? (y/n): ")
	scanner := bufio.NewScanner(os.Stdin)
	var answer string
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
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
		fmt.Printf("unexpected status code when attempting to delete machine from endpoint: %d\n Continue with deletion? (y/n): ", resp.StatusCode)
		scanner := bufio.NewScanner(os.Stdin)
		var answer string
		if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
			return err
		}
		if answer != "y" {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	p := filepath.Join(user.HomeDir, ".ssh-sync")
	if err := os.RemoveAll(p); err != nil {
		return err
	}
	return nil
}

package actions

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func ListMachines(c *cli.Context) error {
	setup, err := utils.CheckIfSetup()
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
	machines, err := retrieval.GetMachines(profile)
	if err != nil {
		return err
	}
	for _, machine := range machines {
		fmt.Println(machine.Name)
	}
	return nil
}

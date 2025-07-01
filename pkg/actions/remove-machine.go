package actions

import (
	"bufio"
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"
)

func RemoveMachine(c *cli.Context) error {
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
	answer := c.Args().First()
	scanner := bufio.NewScanner(os.Stdin)
	if answer == "" {
		fmt.Print("Please enter the machine name: ")
		if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
			return err
		}
	}
	machines, err := retrieval.GetMachines(profile)
	if err != nil {
		return err
	}
	machine, exists := lo.Find(machines, func(x dto.MachineDto) bool {
		return x.Name == answer
	})
	if !exists {
		fmt.Println("Machine not found in your list, exiting.")
		return nil
	}
	fmt.Println("This will remove this machine's public key from your account and you will no longer be able to use it to perform operations on your account.")
	fmt.Printf("Please confirm your intent to delete the following machine: %s (y/n): ", answer)
	if err := utils.ReadLineFromStdin(scanner, &answer); err != nil {
		return err
	}
	if answer != "y" {
		return nil
	}
	err = retrieval.DeleteMachine(profile, machine.Name)
	return err
}

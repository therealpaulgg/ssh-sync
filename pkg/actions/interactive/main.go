package interactive

import (
	"fmt"
	"os"

	"github.com/therealpaulgg/ssh-sync/pkg/actions/interactive/states"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"

	tea "github.com/charmbracelet/bubbletea"
)

func Interactive(c *cli.Context) error {
	// get user data
	setup, err := utils.CheckIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}

	model := states.NewModel()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}

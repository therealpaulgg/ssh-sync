package interactive

import (
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
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
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	data, err := retrieval.GetUserData(profile)
	if err != nil {
		return err
	}
	items := lo.Map(data.Keys, func(key dto.KeyDto, index int) list.Item {
		return item{title: key.Filename, index: index}
	})

	enterBinding := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "choose"),
	)

	keyList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	keyList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			enterBinding,
		}
	}
	keyList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			enterBinding,
		}
	}
	model := model{
		list: keyList,

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: item{},
		Data:     data,
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}

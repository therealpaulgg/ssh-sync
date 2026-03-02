package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// DeleteSSHConfig is a confirmation prompt for deleting an SSH config entry.
type DeleteSSHConfig struct {
	baseState
	config dto.SshConfigDto
}

func NewDeleteSSHConfig(b baseState, config dto.SshConfigDto) *DeleteSSHConfig {
	d := &DeleteSSHConfig{
		config:    config,
		baseState: b,
	}
	d.Initialize()
	return d
}

func (d *DeleteSSHConfig) PrettyName() string {
	return "Delete Config"
}

func (d *DeleteSSHConfig) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return d, tea.Quit
		case "y", "Y":
			if err := d.deleteConfig(); err != nil {
				return NewErrorState(d.baseState, err), nil
			}
			mgr, err := NewSSHConfigManager(d.baseState)
			if err != nil {
				return NewErrorState(d.baseState, err), nil
			}
			mgr.height = d.height
			mgr.width = d.width
			return mgr, nil
		case "n", "N", "backspace":
			return NewSSHConfigOptions(d.baseState, d.config), nil
		}
	}
	return d, nil
}

func (d *DeleteSSHConfig) deleteConfig() error {
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	client := retrieval.NewRetrievalClient()
	return client.DeleteConfig(profile, d.config.ID)
}

func (d *DeleteSSHConfig) View() string {
	return fmt.Sprintf("Are you sure you want to delete the config entry for %q? (y/n)", d.config.Host)
}

func (d *DeleteSSHConfig) SetSize(width, height int) {
	d.baseState.SetSize(width, height)
}

func (d *DeleteSSHConfig) Initialize() {
	d.SetSize(d.width, d.height)
}

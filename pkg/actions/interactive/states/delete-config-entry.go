package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// DeleteConfigEntry
type DeleteConfigEntry struct {
	baseState
	config dto.SshConfigDto
}

func NewDeleteConfigEntry(b baseState, config dto.SshConfigDto) *DeleteConfigEntry {
	d := &DeleteConfigEntry{
		config:    config,
		baseState: b,
	}
	d.Initialize()
	return d
}

func (d *DeleteConfigEntry) PrettyName() string {
	return "Delete Key"
}

func (d *DeleteConfigEntry) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return d, tea.Quit
		case "y", "Y":
			err := d.deleteKey()
			if err != nil {
				return NewErrorState(d.baseState, err), nil
			}
			sshConfigManager, err := NewSSHConfigManager(d.baseState)
			if err != nil {
				return NewErrorState(d.baseState, err), nil
			}
			return sshConfigManager, nil
		case "n", "N", "backspace":
			return NewSSHConfigOptions(d.baseState, d.config), nil
		}
	}
	return d, nil
}

func (d *DeleteConfigEntry) deleteKey() error {
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	err = retrieval.DeleteConfig(profile, d.config)
	// TODO - shouldn't we delete the config from the PC too?
	return err
}

func (d *DeleteConfigEntry) View() string {
	content := fmt.Sprintf("Are you sure you want to delete the config entry %s? (y/n)", d.config.Host)
	return content
}

func (d *DeleteConfigEntry) SetSize(width, height int) {
	d.baseState.SetSize(width, height)
}

func (d *DeleteConfigEntry) Initialize() {
	d.SetSize(d.width, d.height)
}

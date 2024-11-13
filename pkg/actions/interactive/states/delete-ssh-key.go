package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// DeleteSSHKey
type DeleteSSHKey struct {
	baseState
	key dto.KeyDto
}

func NewDeleteSSHKey(b baseState, key dto.KeyDto) *DeleteSSHKey {
	d := &DeleteSSHKey{
		key:       key,
		baseState: b,
	}
	d.Initialize()
	return d
}

func (d *DeleteSSHKey) PrettyName() string {
	return "Delete Key"
}

func (d *DeleteSSHKey) Update(msg tea.Msg) (State, tea.Cmd) {
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
			sshKeyManager, err := NewSSHKeyManager(d.baseState)
			if err != nil {
				return NewErrorState(d.baseState, err), nil
			}
			return sshKeyManager, nil
		case "n", "N", "backspace":
			return NewSSHKeyOptions(d.baseState, d.key), nil
		}
	}
	return d, nil
}

func (d *DeleteSSHKey) deleteKey() error {
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	err = retrieval.DeleteKey(profile, d.key)
	// TODO - shouldn't we delete the key from the PC too?
	return err
}

func (d *DeleteSSHKey) View() string {
	content := fmt.Sprintf("Are you sure you want to delete the key %s? (y/n)", d.key.Filename)
	return content
}

func (d *DeleteSSHKey) SetSize(width, height int) {
	d.baseState.SetSize(width, height)
}

func (d *DeleteSSHKey) Initialize() {
	d.SetSize(d.width, d.height)
}

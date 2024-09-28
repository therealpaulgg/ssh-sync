package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// DeleteSSHKey
type DeleteSSHKey struct {
	baseState
	key dto.KeyDto
}

func NewDeleteSSHKey(key dto.KeyDto) *DeleteSSHKey {
	return &DeleteSSHKey{key: key}
}

func (d *DeleteSSHKey) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return d, tea.Quit
		case "y", "Y":
			// Implement key deletion logic here
			sshKeyManager, err := NewSSHKeyManager()
			if err != nil {
				return NewErrorState(err), nil
			}
			return sshKeyManager, nil
		case "n", "N", "backspace":
			return NewSSHKeyOptions(d.key), nil
		}
	}
	return d, nil
}

func (d *DeleteSSHKey) View() string {
	content := fmt.Sprintf("Are you sure you want to delete the key %s? (y/n)", d.key.Filename)
	return DocStyle.Render(fmt.Sprintf("%s\n\n%s\n\n%s",
		headerView("Delete Key", d.width),
		content,
		footerView("Delete Key", d.width)))
}

func (d *DeleteSSHKey) SetSize(width, height int) {
	d.baseState.SetSize(width, height)
}

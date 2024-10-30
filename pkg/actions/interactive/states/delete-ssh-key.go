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

func NewDeleteSSHKey(b baseState, key dto.KeyDto) *DeleteSSHKey {
	d := &DeleteSSHKey{
		key:       key,
		baseState: b,
	}
	d.Initialize()
	return d
}

func (d *DeleteSSHKey) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return d, tea.Quit
		case "y", "Y":
			// Implement key deletion logic here
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

func (d *DeleteSSHKey) Header() string {
	return headerView("Delete Key", d.width)
}

func (d *DeleteSSHKey) Footer() string {
	return footerView("Delete Key", d.width)
}

func (d *DeleteSSHKey) View() string {
	content := fmt.Sprintf("Are you sure you want to delete the key %s? (y/n)", d.key.Filename)
	return AppStyle.Render(fmt.Sprintf("%s\n\n%s\n\n%s",
		d.Header(),
		content,
		d.Footer(),
	))
}

func (d *DeleteSSHKey) SetSize(width, height int) {
	d.baseState.SetSize(width, height)
	// d.list.SetSize(width, height-lipgloss.Height(d.Header())-lipgloss.Height(d.Footer()))
}

func (d *DeleteSSHKey) Initialize() {
	d.SetSize(d.width, d.height)
}

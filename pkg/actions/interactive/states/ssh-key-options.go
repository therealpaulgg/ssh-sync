package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// SSHKeyOptions
type SSHKeyOptions struct {
	baseState
	list        list.Model
	selectedKey dto.KeyDto
}

func NewSSHKeyOptions(key dto.KeyDto) *SSHKeyOptions {
	items := []list.Item{
		item{title: "View Content", desc: "View the content of the SSH key"},
		item{title: "Delete Key", desc: "Delete the SSH key from the store"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Options for " + key.Filename
	l.SetShowHelp(false)
	return &SSHKeyOptions{
		list:        l,
		selectedKey: key,
	}
}

func (s *SSHKeyOptions) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "enter":
			i := s.list.SelectedItem().(item)
			switch i.title {
			case "View Content":
				return NewSSHKeyContent(s.selectedKey), nil
			case "Delete Key":
				return NewDeleteSSHKey(s.selectedKey), nil
			}
		case "backspace":
			sshKeyManager, err := NewSSHKeyManager()
			if err != nil {
				return NewErrorState(err), nil
			}
			return sshKeyManager, nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHKeyOptions) View() string {
	return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		headerView("Key Options", s.width),
		s.list.View(),
		footerView("Key Options", s.width)))
}

func (s *SSHKeyOptions) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width-4, height-7) // Adjust for margins and header/footer
}

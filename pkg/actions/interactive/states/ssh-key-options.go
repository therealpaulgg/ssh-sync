package states

import (
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

func NewSSHKeyOptions(b baseState, key dto.KeyDto) *SSHKeyOptions {
	items := []list.Item{
		item{title: "View Content", desc: "View the content of the SSH key"},
		item{title: "Delete Key", desc: "Delete the SSH key from the store"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Options for " + key.Filename
	// l.SetShowHelp(false)
	s := &SSHKeyOptions{
		list:        l,
		selectedKey: key,
		baseState:   b,
	}
	s.Initialize()
	return s
}

func (s *SSHKeyOptions) PrettyName() string {
	return "Key Options"
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
				return NewSSHKeyContent(s.baseState, s.selectedKey), nil
			case "Delete Key":
				return NewDeleteSSHKey(s.baseState, s.selectedKey), nil
			}
		case "backspace":
			sshKeyManager, err := NewSSHKeyManager(s.baseState)
			if err != nil {
				return NewErrorState(s.baseState, err), nil
			}
			return sshKeyManager, nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHKeyOptions) View() string {
	return s.list.View()
}

func (s *SSHKeyOptions) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width, height)
}

func (s *SSHKeyOptions) Initialize() {
	s.SetSize(s.width, s.height)
}

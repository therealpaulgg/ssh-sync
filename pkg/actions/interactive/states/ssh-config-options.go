package states

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
)

// SSHConfigOptions shows per-entry options for an SSH config entry.
type SSHConfigOptions struct {
	baseState
	list   list.Model
	config dto.SshConfigDto
}

func NewSSHConfigOptions(b baseState, config dto.SshConfigDto) *SSHConfigOptions {
	items := []list.Item{
		item{title: "View Details", desc: "View the full config entry"},
		item{title: "Edit", desc: "Edit this config entry"},
		item{title: "Delete", desc: "Delete this config entry"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Options for " + config.Host
	s := &SSHConfigOptions{
		list:      l,
		config:    config,
		baseState: b,
	}
	s.Initialize()
	return s
}

func (s *SSHConfigOptions) PrettyName() string {
	return "Config Options"
}

func (s *SSHConfigOptions) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "enter":
			i := s.list.SelectedItem().(item)
			switch i.title {
			case "View Details":
				return NewSSHConfigContent(s.baseState, s.config), nil
			case "Edit":
				return NewSSHConfigEditor(s.baseState, s.config, &s.config), nil
			case "Delete":
				return NewDeleteSSHConfig(s.baseState, s.config), nil
			}
		case "backspace":
			mgr, err := NewSSHConfigManager(s.baseState)
			if err != nil {
				return NewErrorState(s.baseState, err), nil
			}
			mgr.height = s.height
			mgr.width = s.width
			return mgr, nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHConfigOptions) View() string {
	return s.list.View()
}

func (s *SSHConfigOptions) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width, height)
}

func (s *SSHConfigOptions) Initialize() {
	s.SetSize(s.width, s.height)
}

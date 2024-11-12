package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// SSHConfigManager
type SSHConfigManager struct {
	baseState
	list   list.Model
	config []dto.SshConfigDto
}

func NewSSHConfigManager(baseState baseState) (*SSHConfigManager, error) {
	profile, err := utils.GetProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	data, err := retrieval.GetUserData(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	items := make([]list.Item, len(data.SshConfig))
	for i, key := range data.SshConfig {
		items[i] = item{title: key.Host, desc: "", index: i}
	}

	// Discovered a problem with the current implementation of the data
	// For whatever reason the ssh config has a hard dependency on a machine_id
	// This is causing duplicate ssh config host/keypairs to be created
	// Doesn't really make any sense, we should be trying to sync these across
	// all machines and not have any duplicates
	// How to resolve?
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SSH Config Entries"

	m := &SSHConfigManager{
		list:      l,
		config:    data.SshConfig,
		baseState: baseState,
	}
	m.Initialize()
	return m, nil
}

func (s *SSHConfigManager) PrettyName() string {
	return s.list.Title
}

func (s *SSHConfigManager) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "enter":
			// if !s.list.SettingFilter() {
			// 	selected := s.list.SelectedItem().(item)
			// 	// return NewSSHKeyOptions(s.baseState, s.config[selected.index]), nil
			// }
		case "backspace":
			if !s.list.SettingFilter() {
				return NewMainMenu(s.baseState), nil
			}
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHConfigManager) View() string {
	return s.list.View()
}

func (s *SSHConfigManager) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width, height)
}

func (s *SSHConfigManager) Initialize() {
	s.SetSize(s.width, s.height)
}

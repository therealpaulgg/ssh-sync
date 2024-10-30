package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// SSHKeyManager
type SSHKeyManager struct {
	baseState
	list list.Model
	keys []dto.KeyDto
}

func NewSSHKeyManager(baseState baseState) (*SSHKeyManager, error) {
	profile, err := utils.GetProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	data, err := retrieval.GetUserData(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	items := make([]list.Item, len(data.Keys))
	for i, key := range data.Keys {
		items[i] = item{title: key.Filename, desc: "", index: i}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SSH Keys"
	l.SetShowHelp(false)

	m := &SSHKeyManager{
		list:      l,
		keys:      data.Keys,
		baseState: baseState,
	}
	m.Initialize()
	return m, nil
}

func (s *SSHKeyManager) PrettyName() string {
	return "SSH Keys"
}

func (s *SSHKeyManager) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "enter":
			selected := s.list.SelectedItem().(item)
			return NewSSHKeyOptions(s.baseState, s.keys[selected.index]), nil
		case "backspace":
			return NewMainMenu(s.baseState), nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHKeyManager) View() string {
	return s.list.View()
}

func (s *SSHKeyManager) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width, height)
}

func (s *SSHKeyManager) Initialize() {
	s.SetSize(s.width, s.height)
}

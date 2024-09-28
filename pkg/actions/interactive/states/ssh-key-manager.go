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

func NewSSHKeyManager() (*SSHKeyManager, error) {
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

	return &SSHKeyManager{
		list: l,
		keys: data.Keys,
	}, nil
}

func (s *SSHKeyManager) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return s, tea.Quit
		case "enter":
			selected := s.list.SelectedItem().(item)
			return NewSSHKeyOptions(s.keys[selected.index]), nil
		case "backspace":
			return NewMainMenu(), nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHKeyManager) View() string {
	return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		headerView("SSH Keys", s.width),
		s.list.View(),
		footerView("SSH Keys", s.width)))
}

func (s *SSHKeyManager) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width, height)
}
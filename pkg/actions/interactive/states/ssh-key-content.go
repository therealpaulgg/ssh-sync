package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// SSHKeyContent
type SSHKeyContent struct {
	baseState
	viewport viewport.Model
	key      dto.KeyDto
}

func NewSSHKeyContent(key dto.KeyDto) *SSHKeyContent {
	v := viewport.New(80, 20)
	v.SetContent(string(key.Data))
	return &SSHKeyContent{
		viewport: v,
		key:      key,
	}
}

func (s *SSHKeyContent) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
		if msg.String() == "backspace" {
			return NewSSHKeyOptions(s.key), nil
		}
	}
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return s, cmd
}

func (s *SSHKeyContent) View() string {
	return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		headerView(s.key.Filename, s.width),
		s.viewport.View(),
		footerView("Key Content", s.width)))
}

func (s *SSHKeyContent) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.viewport.Width = width - 4
	s.viewport.Height = height - 7 // Adjust for margins and header/footer
}

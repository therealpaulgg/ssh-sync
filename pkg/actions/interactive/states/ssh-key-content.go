package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// SSHKeyContent
type SSHKeyContent struct {
	baseState
	viewport viewport.Model
	key      dto.KeyDto
}

func NewSSHKeyContent(b baseState, key dto.KeyDto) *SSHKeyContent {
	v := viewport.New(80, 20)
	v.SetContent(string(key.Data))
	c := &SSHKeyContent{
		viewport:  v,
		key:       key,
		baseState: b,
	}
	c.Initialize()
	return c
}

func (s *SSHKeyContent) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
		if msg.String() == "backspace" {
			return NewSSHKeyOptions(s.baseState, s.key), nil
		}
	}
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return s, cmd
}

func (s *SSHKeyContent) Header() string {
	return headerView(s.key.Filename, s.width)
}

func (s *SSHKeyContent) Footer() string {
	return footerView("Key Content", s.width)
}

func (s *SSHKeyContent) View() string {
	return AppStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		s.Header(),
		s.viewport.View(),
		s.Footer()))
}

func (s *SSHKeyContent) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.viewport.Width = width
	s.viewport.Height = height - lipgloss.Height(s.Header()) - lipgloss.Height(s.Footer())
}

func (s *SSHKeyContent) Initialize() {
	s.SetSize(s.width, s.height)
}

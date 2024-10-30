package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	ke "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

// SSHKeyContent
type SSHKeyContent struct {
	baseState
	viewport viewport.Model
	help     help.Model
	keymap   help.KeyMap
	key      dto.KeyDto
}

type SSHKeyContentHelp struct {
}

func (s *SSHKeyContentHelp) ShortHelp() []key.Binding {
	keymap := viewport.DefaultKeyMap()
	return []ke.Binding{
		keymap.Up,
		keymap.Down,
		ke.NewBinding(ke.WithKeys("backspace"), ke.WithHelp("backspace", "back")),
		ke.NewBinding(ke.WithKeys("q"), ke.WithHelp("q", "quit")),
		ke.NewBinding(ke.WithKeys("?"), ke.WithHelp("?", "more")),
	}
}

func (s *SSHKeyContentHelp) FullHelp() [][]key.Binding {
	keymap := viewport.DefaultKeyMap()
	return [][]key.Binding{
		[]ke.Binding{
			keymap.Up,
			keymap.Down,
			keymap.PageUp,
			keymap.PageDown,
			keymap.HalfPageUp,
			keymap.HalfPageDown,
		},
		[]ke.Binding{
			ke.NewBinding(ke.WithKeys("backspace"), ke.WithHelp("backspace", "back")),
			ke.NewBinding(ke.WithKeys("q"), ke.WithHelp("q", "quit")),
			ke.NewBinding(ke.WithKeys("?"), ke.WithHelp("?", "close help")),
		},
	}
}

func NewSSHKeyContent(b baseState, key dto.KeyDto) *SSHKeyContent {
	v := viewport.New(80, 20)
	v.SetContent(string(key.Data))
	c := &SSHKeyContent{
		viewport:  v,
		key:       key,
		baseState: b,
	}
	c.help = help.New()
	c.keymap = &SSHKeyContentHelp{}
	c.Initialize()
	return c
}

func (s *SSHKeyContent) PrettyName() string {
	return "Key Content"
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
		if msg.String() == "?" {
			s.help.ShowAll = !s.help.ShowAll
			s.SetSize(s.width, s.height)
			return s, nil
		}
	}
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	return s, cmd
}

func (s *SSHKeyContent) View() string {
	return fmt.Sprintf("%s\n%s", s.viewport.View(), s.help.View(s.keymap))
}

func (s *SSHKeyContent) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.viewport.Width = width
	s.viewport.Height = height - lipgloss.Height(s.help.View(s.keymap))
}

func (s *SSHKeyContent) Initialize() {
	s.SetSize(s.width, s.height)
}

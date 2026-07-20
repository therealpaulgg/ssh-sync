package states

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	ke "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
)

// SSHConfigContent displays the details of an SSH config entry.
type SSHConfigContent struct {
	baseState
	viewport viewport.Model
	help     help.Model
	keymap   help.KeyMap
	config   dto.SshConfigDto
}

type SSHConfigContentHelp struct{}

func (s *SSHConfigContentHelp) ShortHelp() []key.Binding {
	keymap := viewport.DefaultKeyMap()
	return []ke.Binding{
		keymap.Up,
		keymap.Down,
		ke.NewBinding(ke.WithKeys("backspace"), ke.WithHelp("backspace", "back")),
		ke.NewBinding(ke.WithKeys("q"), ke.WithHelp("q", "quit")),
		ke.NewBinding(ke.WithKeys("?"), ke.WithHelp("?", "more")),
	}
}

func (s *SSHConfigContentHelp) FullHelp() [][]key.Binding {
	keymap := viewport.DefaultKeyMap()
	return [][]key.Binding{
		{keymap.Up, keymap.Down, keymap.PageUp, keymap.PageDown, keymap.HalfPageUp, keymap.HalfPageDown},
		{
			ke.NewBinding(ke.WithKeys("backspace"), ke.WithHelp("backspace", "back")),
			ke.NewBinding(ke.WithKeys("q"), ke.WithHelp("q", "quit")),
			ke.NewBinding(ke.WithKeys("?"), ke.WithHelp("?", "close help")),
		},
	}
}

func formatConfigBlock(conf dto.SshConfigDto) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Host %s\n", conf.Host))
	keys := make([]string, 0, len(conf.Values))
	for k := range conf.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range conf.Values[k] {
			sb.WriteString(fmt.Sprintf("    %s %s\n", k, v))
		}
	}
	for _, f := range conf.IdentityFiles {
		sb.WriteString(fmt.Sprintf("    IdentityFile %s\n", f))
	}
	return sb.String()
}

func NewSSHConfigContent(b baseState, config dto.SshConfigDto) *SSHConfigContent {
	v := viewport.New(80, 20)
	v.SetContent(formatConfigBlock(config))
	c := &SSHConfigContent{
		viewport:  v,
		config:    config,
		baseState: b,
	}
	c.help = help.New()
	c.keymap = &SSHConfigContentHelp{}
	c.Initialize()
	return c
}

func (s *SSHConfigContent) PrettyName() string {
	return "Config Details"
}

func (s *SSHConfigContent) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
		if msg.String() == "backspace" {
			return NewSSHConfigOptions(s.baseState, s.config), nil
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

func (s *SSHConfigContent) View() string {
	return fmt.Sprintf("%s\n%s", s.viewport.View(), s.help.View(s.keymap))
}

func (s *SSHConfigContent) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.viewport.Width = width
	s.viewport.Height = height - lipgloss.Height(s.help.View(s.keymap))
}

func (s *SSHConfigContent) Initialize() {
	s.SetSize(s.width, s.height)
}

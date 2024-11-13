package states

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
)

var (
	leftStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			PaddingRight(1)

	rightStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(false).
			PaddingLeft(1)
)

// SSHConfigOptions
type SSHConfigOptions struct {
	baseState
	list           list.Model
	viewport       viewport.Model
	selectedConfig dto.SshConfigDto
}

var editConfigTitle = "Edit Config"
var deleteConfigTitle = "Delete Config"

func NewSSHConfigOptions(b baseState, config dto.SshConfigDto) *SSHConfigOptions {
	v := viewport.New(80, b.height-5)
	configValues := strings.Join(lo.Flatten(lo.MapToSlice(config.Values, func(k string, v []string) []string {
		return lo.Map(v, func(val string, i int) string {
			return fmt.Sprintf("%s: %s", k, val)
		})
	})), "\n")
	identityValues := strings.Join(config.IdentityFiles, "\n")
	v.SetContent(fmt.Sprintf("%s\n\nIdentityFiles:\n%s", configValues, identityValues))
	items := []list.Item{
		item{title: editConfigTitle, desc: "Edit this SSH config entry"},
		item{title: deleteConfigTitle, desc: "Delete this SSH config entry"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Options for " + config.Host
	s := &SSHConfigOptions{
		list:           l,
		viewport:       v,
		selectedConfig: config,
		baseState:      b,
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
			case editConfigTitle:
				return NewSSHConfigOptions(s.baseState, s.selectedConfig), nil
			case deleteConfigTitle:
				return NewDeleteConfigEntry(s.baseState, s.selectedConfig), nil
			}
		case "backspace":
			sshConfigManager, err := NewSSHConfigManager(s.baseState)
			if err != nil {
				return NewErrorState(s.baseState, err), nil
			}
			return sshConfigManager, nil
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *SSHConfigOptions) View() string {
	leftPanel := leftStyle.Render(s.list.View())

	rightPanel := rightStyle.Render(s.viewport.View())

	row := lipgloss.JoinHorizontal(lipgloss.Left, leftPanel, rightPanel)

	return row
}

func (s *SSHConfigOptions) SetSize(width, height int) {
	s.baseState.SetSize(width, height)
	s.list.SetSize(width/2, height)
	s.viewport.Height = height
	s.viewport.Width = width / 2
}

func (s *SSHConfigOptions) Initialize() {
	s.SetSize(s.width, s.height)
}

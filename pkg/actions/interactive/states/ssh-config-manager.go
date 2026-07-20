package states

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

// SSHConfigManager lists all SSH config entries.
type SSHConfigManager struct {
	baseState
	list    list.Model
	configs []dto.SshConfigDto
}

func NewSSHConfigManager(baseState baseState) (*SSHConfigManager, error) {
	profile, err := utils.GetProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	client := retrieval.NewRetrievalClient()
	data, err := client.GetUserData(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get user data: %w", err)
	}

	items := make([]list.Item, len(data.SshConfig))
	for i, conf := range data.SshConfig {
		items[i] = item{title: conf.Host, desc: configSummary(conf), index: i}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "SSH Config Entries  (a: add new)"

	m := &SSHConfigManager{
		list:      l,
		configs:   data.SshConfig,
		baseState: baseState,
	}
	m.Initialize()
	return m, nil
}

func configSummary(conf dto.SshConfigDto) string {
	var parts []string
	// Sort keys for stable display
	keys := make([]string, 0, len(conf.Values))
	for k := range conf.Values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if len(conf.Values[k]) > 0 {
			parts = append(parts, fmt.Sprintf("%s=%s", k, conf.Values[k][0]))
		}
	}
	if len(conf.IdentityFiles) > 0 {
		parts = append(parts, fmt.Sprintf("IdentityFile=%s", conf.IdentityFiles[0]))
	}
	if len(parts) == 0 {
		return "(no options)"
	}
	return strings.Join(parts, "  ")
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
			if !s.list.SettingFilter() && len(s.configs) > 0 {
				selected := s.list.SelectedItem().(item)
				return NewSSHConfigOptions(s.baseState, s.configs[selected.index]), nil
			}
		case "a":
			if !s.list.SettingFilter() {
				return NewSSHConfigEditor(s.baseState, dto.SshConfigDto{}, nil), nil
			}
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

package states

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ConfigManager
type ConfigManager struct {
	baseState
	list list.Model
}

func NewConfigManager(b baseState) *ConfigManager {
	items := []list.Item{
		item{title: "Edit Config", desc: "Edit the SSH Sync configuration"},
		item{title: "View Config", desc: "View the current SSH Sync configuration"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Config Management"
	l.SetShowHelp(false)
	c := &ConfigManager{
		list:      l,
		baseState: b,
	}
	c.Initialize()
	return c
}

func (c *ConfigManager) PrettyName() string {
	return "Config Management"
}

func (c *ConfigManager) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return c, tea.Quit
		case "backspace":
			return NewMainMenu(c.baseState), nil
		}
	}
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c *ConfigManager) View() string {
	return c.list.View()
}

func (c *ConfigManager) SetSize(width, height int) {
	c.baseState.SetSize(width, height)
	c.list.SetSize(width, height)
}

func (c *ConfigManager) Initialize() {
	c.SetSize(c.width, c.height)
}

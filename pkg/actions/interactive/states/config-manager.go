package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// ConfigManager
type ConfigManager struct {
	baseState
	list list.Model
}

func NewConfigManager() *ConfigManager {
	items := []list.Item{
		item{title: "Edit Config", desc: "Edit the SSH Sync configuration"},
		item{title: "View Config", desc: "View the current SSH Sync configuration"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Config Management"
	l.SetShowHelp(false)
	return &ConfigManager{list: l}
}

func (c *ConfigManager) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return c, tea.Quit
		case "backspace":
			return NewMainMenu(), nil
		}
	}
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c *ConfigManager) View() string {
	return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		headerView("Config Management", c.width),
		c.list.View(),
		footerView("Config Management", c.width)))
}

func (c *ConfigManager) SetSize(width, height int) {
	c.baseState.SetSize(width, height)
	c.list.SetSize(width-4, height-7) // Adjust for margins and header/footer
}

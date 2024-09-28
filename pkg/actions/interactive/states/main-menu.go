package states

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// MainMenu
type MainMenu struct {
	baseState
	list list.Model
}

func NewMainMenu(b baseState) *MainMenu {
	items := []list.Item{
		item{title: "Manage Config", desc: "Configure SSH Sync settings"},
		item{title: "Manage SSH Keys", desc: "View and manage your SSH keys"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Main Menu"
	l.SetShowHelp(false)
	m := &MainMenu{
		list:      l,
		baseState: b,
	}
	m.Initialize()
	return m
}

func (m *MainMenu) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			i := m.list.SelectedItem().(item)
			switch i.title {
			case "Manage Config":
				return NewConfigManager(m.baseState), nil
			case "Manage SSH Keys":
				sshKeyManager, err := NewSSHKeyManager(m.baseState)
				if err != nil {
					return NewErrorState(m.baseState, err), nil
				}
				sshKeyManager.height = m.height
				sshKeyManager.width = m.width
				return sshKeyManager, nil
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *MainMenu) View() string {
	return AppStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		headerView("Main Menu", m.width),
		m.list.View(),
		footerView("Main Menu", m.width),
	))
}

func (m *MainMenu) SetSize(width, height int) {
	m.baseState.SetSize(width, height)
	m.list.SetSize(width, height-5)
}

func (m *MainMenu) Initialize() {
	m.SetSize(m.width, m.height)
}

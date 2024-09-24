package interactive

import (
	"fmt"
	"strings"

	"github.com/therealpaulgg/ssh-sync/pkg/dto"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UIState represents the current state of the UI
type UIState int

const (
	StateMainMenu UIState = iota
	StateManageConfig
	StateManageSSHKeys
	StateViewConfigEntries
	StateViewSSHKeyOptions
	StateViewSSHKeyContent
)

func (s UIState) String() string {
	switch s {
	case StateMainMenu:
		return "Main Menu"
	case StateManageConfig:
		return "Manage Config"
	case StateManageSSHKeys:
		return "Manage SSH Keys"
	case StateViewConfigEntries:
		return "View Config Entries"
	case StateViewSSHKeyOptions:
		return "View SSH Key Options"
	case StateViewSSHKeyContent:
		return "View SSH Key Content"
	default:
		return "Unknown State"
	}
}

type model struct {
	keyList       list.Model
	mainMenu      list.Model
	sshKeyOptions list.Model
	viewport      viewport.Model
	ready         bool
	Data          dto.DataDto
	currentState  UIState
	selected      item
}

func (m model) headerView() string {
	var title string
	switch m.currentState {
	case StateMainMenu:
		title = "Main Menu"
	case StateManageConfig:
		title = "Manage Config"
	case StateManageSSHKeys:
		title = "Manage SSH Keys"
	case StateViewConfigEntries:
		title = "Config Entries"
	case StateViewSSHKeyOptions:
		title = "Options for - " + m.Data.Keys[m.selected.index].Filename
	case StateViewSSHKeyContent:
		title = m.Data.Keys[m.selected.index].Filename
	}
	title = TitleStyle.Render(title)
	line := BasicColorStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

type keyMap struct {
	Back key.Binding
	Quit key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Quit},
	}
}

var keys = keyMap{
	Back: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("backspace", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (m model) footerView() string {
	info := InfoStyle.Render(m.currentState.String())
	line := BasicColorStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info))))
	h := help.New()
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Center, line, info), h.View(keys))
}

type item struct {
	title, desc string
	index       int
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func NewModel(data dto.DataDto) model {
	mainMenuItems := []list.Item{
		item{title: "Manage Config", desc: "Configure SSH Sync settings"},
		item{title: "Manage SSH Keys", desc: "View and manage your SSH keys"},
	}

	mainMenu := list.New(mainMenuItems, list.NewDefaultDelegate(), 0, 0)
	mainMenu.Title = "Main Menu"
	mainMenu.SetShowHelp(false)

	sshKeyOptions := list.New([]list.Item{
		item{title: "View Content", desc: "View the content of the SSH key"},
		item{title: "Delete Key", desc: "Delete the SSH key from the store"},
	}, list.NewDefaultDelegate(), 0, 0)
	sshKeyOptions.Title = "SSH Key Options"
	sshKeyOptions.SetShowHelp(false)

	return model{
		mainMenu:      mainMenu,
		sshKeyOptions: sshKeyOptions,
		keyList:       list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		currentState:  StateMainMenu,
		Data:          data,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "backspace":
			m = m.handleBack()
		case "enter":
			m = m.handleEnter()
		}

	case tea.WindowSizeMsg:
		m = m.handleWindowSize(msg)
	}

	switch m.currentState {
	case StateMainMenu:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
	case StateManageSSHKeys:
		m.keyList, cmd = m.keyList.Update(msg)
	case StateViewSSHKeyOptions:
		m.sshKeyOptions, cmd = m.sshKeyOptions.Update(msg)
	default:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) handleBack() model {
	switch m.currentState {
	case StateMainMenu:
		// Do nothing, we're already at the main menu
	case StateManageConfig, StateManageSSHKeys:
		m.currentState = StateMainMenu
	case StateViewConfigEntries:
		m.currentState = StateManageConfig
	case StateViewSSHKeyOptions:
		m.currentState = StateManageSSHKeys
		m.selected = item{}
		m.viewport.SetContent("")
	case StateViewSSHKeyContent:
		m.currentState = StateViewSSHKeyOptions
	}
	return m
}

func (m model) handleEnter() model {
	switch m.currentState {
	case StateMainMenu:
		selected := m.mainMenu.SelectedItem().(item)
		if selected.title == "Manage Config" {
			m.currentState = StateManageConfig
		} else if selected.title == "Manage SSH Keys" {
			m.currentState = StateManageSSHKeys
			m.keyList.SetItems(getSSHKeyItems(m.Data.Keys))
		}
	case StateManageSSHKeys:
		m.selected = m.keyList.SelectedItem().(item)
		m.currentState = StateViewSSHKeyOptions
		m.viewport.SetContent(string(m.Data.Keys[m.selected.index].Data))
	case StateViewSSHKeyOptions:
		selected := m.sshKeyOptions.SelectedItem().(item)
		if selected.title == "View Content" {
			m.currentState = StateViewSSHKeyContent
		}
	case StateManageConfig:
		m.currentState = StateViewConfigEntries
		// TODO: Implement config entry viewing
	}
	return m
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) model {
	h, v := DocStyle.GetFrameSize()
	m.keyList.SetSize(msg.Width-h, msg.Height-v)
	m.mainMenu.SetSize(msg.Width-h, msg.Height-v)
	m.sshKeyOptions.SetSize(msg.Width-h, msg.Height-v)
	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	verticalMarginHeight := headerHeight + footerHeight

	if !m.ready {
		m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
		m.viewport.YPosition = headerHeight
		m.viewport.HighPerformanceRendering = false
		m.ready = true
		m.viewport.YPosition = headerHeight + 1
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - verticalMarginHeight
	}

	return m
}

func (m model) View() string {
	switch m.currentState {
	case StateMainMenu:
		return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.mainMenu.View(), m.footerView()))

	case StateManageConfig:
		return DocStyle.Render(fmt.Sprintf("%s\nConfig Management (Not yet implemented)\n%s", m.headerView(), m.footerView()))

	case StateManageSSHKeys:
		return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.keyList.View(), m.footerView()))

	case StateViewConfigEntries:
		return DocStyle.Render(fmt.Sprintf("%s\nConfig Entries View (Not yet implemented)\n%s", m.headerView(), m.footerView()))

	case StateViewSSHKeyOptions:
		return DocStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.sshKeyOptions.View(), m.footerView()))

	case StateViewSSHKeyContent:
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())

	default:
		return m.View() // This will recursively call View() with StateMainMenu
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getSSHKeyItems(keys []dto.KeyDto) []list.Item {
	items := make([]list.Item, len(keys))
	for i, key := range keys {
		items[i] = item{
			title: key.Filename,
			index: i,
		}
	}
	return items
}

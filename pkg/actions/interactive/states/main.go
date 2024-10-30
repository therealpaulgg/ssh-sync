package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	currentState State
	width        int
	height       int
}

func NewModel() Model {
	return Model{
		currentState: NewMainMenu(baseState{}),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := AppStyle.GetFrameSize()
		headerAndFooterHeight := lipgloss.Height(m.Header()) + lipgloss.Height(m.Footer())
		adjustedWidth, adjustedHeight := msg.Width-h, msg.Height-v-headerAndFooterHeight
		m.width = adjustedWidth
		m.height = adjustedHeight
		m.currentState.SetSize(m.width, m.height)
	}
	m.currentState, cmd = m.currentState.Update(msg)
	return m, cmd
}

func (m Model) Header() string {
	return headerView(m.currentState.PrettyName(), m.width)
}

func (m Model) Footer() string {
	return footerView(m.currentState.PrettyName(), m.width)
}

func (m Model) View() string {
	return AppStyle.Render(fmt.Sprintf("%s\n%s\n%s",
		m.Header(),
		m.currentState.View(),
		m.Footer(),
	))
}

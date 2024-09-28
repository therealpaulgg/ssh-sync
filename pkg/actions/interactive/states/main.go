package states

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	currentState State
	width        int
	height       int
}

func NewModel() Model {
	return Model{
		currentState: NewMainMenu(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := DocStyle.GetFrameSize()
		adjustedWidth, adjustedHeight := msg.Width-h, msg.Height-v
		m.width = adjustedWidth
		m.height = adjustedHeight
		m.currentState.SetSize(msg.Width, msg.Height)
	}
	m.currentState, cmd = m.currentState.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.currentState.View()
}

package states

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type baseState struct {
	width  int
	height int
}

func (b *baseState) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// State represents a single screen or state in the application

type State interface {
	Update(msg tea.Msg) (State, tea.Cmd)
	View() string
	SetSize(width, height int)
}

// item represents a selectable item in a list
type item struct {
	title, desc string
	index       int
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Helper functions
func headerView(title string, width int) string {
	titleStyle := TitleStyle.Render(title)
	line := strings.Repeat("─", max(0, width-lipgloss.Width(titleStyle)-4))
	return lipgloss.JoinHorizontal(lipgloss.Center, titleStyle, BasicColorStyle.Render(line))
}

func footerView(info string, width int) string {
	infoStyle := InfoStyle.Render(info)
	line := strings.Repeat("─", max(0, width-lipgloss.Width(infoStyle)-4))
	return lipgloss.JoinHorizontal(lipgloss.Center, BasicColorStyle.Render(line), infoStyle)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

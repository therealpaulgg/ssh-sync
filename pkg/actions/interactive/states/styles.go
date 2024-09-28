package states

import "github.com/charmbracelet/lipgloss"

var (
	AppStyle   = lipgloss.NewStyle().Padding(1, 2)
	TitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1).BorderForeground(lipgloss.Color("#FF00FF")).Bold(true)
	}()

	InfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return TitleStyle.Copy().BorderStyle(b)
	}()

	BasicColorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
)

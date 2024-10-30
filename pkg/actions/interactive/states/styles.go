package states

import "github.com/charmbracelet/lipgloss"

var (
	// using this is causing problems with rendering due to caching...WHY?!?!?!?!?!?
	// if the layout shifts by even one character, previous screen's state may be displayed, resulting in glitches
	AppStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Padding(1, 2)
	}()
	TitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1).BorderForeground(lipgloss.Color("#FF00FF")).Bold(true)
	}()

	InfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return TitleStyle.BorderStyle(b)
	}()

	BasicColorStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
	}()
)

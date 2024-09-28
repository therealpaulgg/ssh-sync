package states

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ErrorState
type ErrorState struct {
	baseState
	err error
}

func NewErrorState(b baseState, err error) *ErrorState {
	e := &ErrorState{
		err:       err,
		baseState: b,
	}
	e.Initialize()
	return e
}

func (e *ErrorState) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "backspace" || msg.String() == "q" {
			return NewMainMenu(e.baseState), nil
		}
	}
	return e, nil
}

func (e *ErrorState) View() string {
	return AppStyle.Render(fmt.Sprintf("%s\n\n%s\n\n%s",
		headerView("Error", e.width),
		fmt.Sprintf("An error occurred: %v\nPress 'backspace' or 'q' to return to the main menu.", e.err),
		footerView("Error", e.width)))
}

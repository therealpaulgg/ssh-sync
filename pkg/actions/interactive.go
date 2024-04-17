package actions

import (
	"fmt"
	"os"
	"strings"

	"github.com/samber/lo"
	"github.com/therealpaulgg/ssh-sync/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
	"github.com/urfave/cli/v2"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1).BorderForeground(lipgloss.Color("#FF00FF")).Bold(true)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()

	basicColorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))
)

type model struct {
	list list.Model
	// TODO currently the only state change for the application is whether or not 'selected' is chosen
	// we should do this in a better way...perhaps an enum for possible rendered UI components of the app
	/*
		Layers of Nesting:
		1. Manage Config or SSH Keys
			a. Manage Config
				i. View config entries/perform actions
			b. Manage SSH Keys
				i. View key content/perform actions
	*/
	selected item
	viewport viewport.Model
	ready    bool
	Data     dto.DataDto
}

func (m model) headerView() string {
	title := titleStyle.Render(m.Data.Keys[m.selected.index].Filename)
	line := basicColorStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

type keyMap struct {
	Back key.Binding
	Quit key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Quit}, // first column
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
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := basicColorStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info))))
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

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		if msg.String() == "backspace" && m.selected.title != "" {
			m.selected = item{}
			m.viewport.SetContent("")
		}
		if msg.String() == "enter" {
			m.selected = m.list.SelectedItem().(item)
			m.viewport.SetContent(string(m.Data.Keys[m.selected.index].Data))
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = false
			if m.selected.title != "" {
				m.viewport.SetContent(string(m.Data.Keys[m.selected.index].Data))
			}
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.selected.title != "" {
		return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
	}
	return docStyle.Render(m.list.View())
}

func Interactive(c *cli.Context) error {
	// get user data
	setup, err := checkIfSetup()
	if err != nil {
		return err
	}
	if !setup {
		fmt.Fprintln(os.Stderr, "ssh-sync has not been set up on this system. Please set up before continuing.")
		return nil
	}
	profile, err := utils.GetProfile()
	if err != nil {
		return err
	}
	data, err := retrieval.GetUserData(profile)
	if err != nil {
		return err
	}
	items := lo.Map(data.Keys, func(key dto.KeyDto, index int) list.Item {
		return item{title: key.Filename, index: index}
	})

	enterBinding := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "choose"),
	)

	keyList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	keyList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			enterBinding,
		}
	}
	keyList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			enterBinding,
		}
	}
	model := model{
		list: keyList,

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: item{},
		Data:     data,
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		return err
	}
	return nil
}

package states

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
)

type editorMode int

const (
	formMode     editorMode = iota
	freetextMode
)

const (
	fieldHost          = 0
	fieldHostName      = 1
	fieldUser          = 2
	fieldPort          = 3
	fieldIdentityFile  = 4
	standardFieldCount = 5
)

var stdLabels = [standardFieldCount]string{
	"Host        ",
	"HostName    ",
	"User        ",
	"Port        ",
	"IdentityFile",
}

type customPair struct {
	key   textinput.Model
	value textinput.Model
}

// SSHConfigEditor is a form-based editor for adding or editing a config entry.
// Press ctrl+f to toggle between assisted form mode and raw freetext mode.
type SSHConfigEditor struct {
	baseState
	mode         editorMode
	stdInputs    [standardFieldCount]textinput.Model
	customFields []customPair
	focusIdx     int
	textarea     textarea.Model
	original     *dto.SshConfigDto
	title        string
	err          string
	initCmd      tea.Cmd // deferred cursor-blink command from initial Focus()
}

func makeTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	return ti
}

func NewSSHConfigEditor(b baseState, config dto.SshConfigDto, original *dto.SshConfigDto) *SSHConfigEditor {
	e := &SSHConfigEditor{
		baseState: b,
		mode:      formMode,
		focusIdx:  0,
		original:  original,
	}

	if original != nil {
		e.title = "Edit Config: " + config.Host
	} else {
		e.title = "Add Config Entry"
	}

	w := 36
	e.stdInputs[fieldHost] = makeTextInput("myhost", w)
	e.stdInputs[fieldHostName] = makeTextInput("10.0.0.1 or hostname", w)
	e.stdInputs[fieldUser] = makeTextInput("ubuntu", w)
	e.stdInputs[fieldPort] = makeTextInput("22", w)
	e.stdInputs[fieldIdentityFile] = makeTextInput("~/.ssh/id_ed25519", w)

	if original != nil {
		e.populateFromConfig(config)
	}

	ta := textarea.New()
	ta.ShowLineNumbers = false
	e.textarea = ta

	e.Initialize()
	return e
}

func (e *SSHConfigEditor) populateFromConfig(conf dto.SshConfigDto) {
	e.stdInputs[fieldHost].SetValue(conf.Host)
	if v, ok := conf.Values["HostName"]; ok && len(v) > 0 {
		e.stdInputs[fieldHostName].SetValue(v[0])
	}
	if v, ok := conf.Values["User"]; ok && len(v) > 0 {
		e.stdInputs[fieldUser].SetValue(v[0])
	}
	if v, ok := conf.Values["Port"]; ok && len(v) > 0 {
		e.stdInputs[fieldPort].SetValue(v[0])
	}
	if len(conf.IdentityFiles) > 0 {
		e.stdInputs[fieldIdentityFile].SetValue(conf.IdentityFiles[0])
	}

	standardKeys := map[string]bool{"HostName": true, "User": true, "Port": true}
	cw := 20
	for _, f := range conf.IdentityFiles[min(1, len(conf.IdentityFiles)):] {
		e.customFields = append(e.customFields, e.makeCustomPair("IdentityFile", f, cw))
	}
	for k, vals := range conf.Values {
		if standardKeys[k] {
			continue
		}
		for _, v := range vals {
			e.customFields = append(e.customFields, e.makeCustomPair(k, v, cw))
		}
	}
}

func (e *SSHConfigEditor) makeCustomPair(key, value string, width int) customPair {
	ki := textinput.New()
	ki.Placeholder = "Key"
	ki.Width = width
	ki.SetValue(key)

	vi := textinput.New()
	vi.Placeholder = "Value"
	vi.Width = width
	vi.SetValue(value)

	return customPair{key: ki, value: vi}
}

func (e *SSHConfigEditor) totalFocusable() int {
	return standardFieldCount + len(e.customFields)*2 + 1
}

func (e *SSHConfigEditor) isAddButton() bool {
	return e.focusIdx == e.totalFocusable()-1
}

func (e *SSHConfigEditor) customFieldAt() (fieldNum int, isValue bool, ok bool) {
	if e.focusIdx < standardFieldCount || e.isAddButton() {
		return 0, false, false
	}
	ci := e.focusIdx - standardFieldCount
	return ci / 2, ci%2 == 1, true
}

func (e *SSHConfigEditor) applyFocus() tea.Cmd {
	for i := range e.stdInputs {
		e.stdInputs[i].Blur()
	}
	for i := range e.customFields {
		e.customFields[i].key.Blur()
		e.customFields[i].value.Blur()
	}
	if e.focusIdx < standardFieldCount {
		return e.stdInputs[e.focusIdx].Focus()
	}
	if fieldNum, isValue, ok := e.customFieldAt(); ok {
		if isValue {
			return e.customFields[fieldNum].value.Focus()
		}
		return e.customFields[fieldNum].key.Focus()
	}
	return nil // add button — nothing to focus
}

func (e *SSHConfigEditor) nextFocus() tea.Cmd {
	e.focusIdx = (e.focusIdx + 1) % e.totalFocusable()
	return e.applyFocus()
}

func (e *SSHConfigEditor) prevFocus() tea.Cmd {
	e.focusIdx = (e.focusIdx - 1 + e.totalFocusable()) % e.totalFocusable()
	return e.applyFocus()
}

func (e *SSHConfigEditor) addCustomField() tea.Cmd {
	cw := max(10, (e.width-20)/2-4)
	e.customFields = append(e.customFields, e.makeCustomPair("", "", cw))
	// focus the new key input
	e.focusIdx = standardFieldCount + (len(e.customFields)-1)*2
	return e.applyFocus()
}

func (e *SSHConfigEditor) deleteCurrentCustomField() tea.Cmd {
	if fieldNum, _, ok := e.customFieldAt(); ok {
		e.customFields = append(e.customFields[:fieldNum], e.customFields[fieldNum+1:]...)
		if e.focusIdx >= e.totalFocusable() {
			e.focusIdx = e.totalFocusable() - 1
		}
		return e.applyFocus()
	}
	return nil
}

func (e *SSHConfigEditor) serializeToConfig() dto.SshConfigDto {
	conf := dto.SshConfigDto{
		Host:   e.stdInputs[fieldHost].Value(),
		Values: make(map[string][]string),
	}
	if v := e.stdInputs[fieldHostName].Value(); v != "" {
		conf.Values["HostName"] = []string{v}
	}
	if v := e.stdInputs[fieldUser].Value(); v != "" {
		conf.Values["User"] = []string{v}
	}
	if v := e.stdInputs[fieldPort].Value(); v != "" {
		conf.Values["Port"] = []string{v}
	}
	if v := e.stdInputs[fieldIdentityFile].Value(); v != "" {
		conf.IdentityFiles = append(conf.IdentityFiles, v)
	}
	for _, cf := range e.customFields {
		k := cf.key.Value()
		v := cf.value.Value()
		if k == "" {
			continue
		}
		if strings.EqualFold(k, "identityfile") {
			conf.IdentityFiles = append(conf.IdentityFiles, v)
		} else {
			conf.Values[k] = append(conf.Values[k], v)
		}
	}
	return conf
}

func (e *SSHConfigEditor) switchToFreetext() tea.Cmd {
	conf := e.serializeToConfig()
	e.textarea.SetValue(formatConfigBlock(conf))
	e.mode = freetextMode
	return e.textarea.Focus()
}

func (e *SSHConfigEditor) switchToForm() tea.Cmd {
	hosts, err := utils.ParseConfigFromString(e.textarea.Value())
	if err == nil && len(hosts) > 0 {
		h := hosts[0]
		e.customFields = nil
		e.populateFromConfig(dto.SshConfigDto{
			Host:          h.Host,
			Values:        h.Values,
			IdentityFiles: h.IdentityFiles,
		})
	}
	e.mode = formMode
	e.focusIdx = 0
	return e.applyFocus()
}

func (e *SSHConfigEditor) save() (State, tea.Cmd) {
	var conf dto.SshConfigDto
	if e.mode == formMode {
		conf = e.serializeToConfig()
	} else {
		hosts, err := utils.ParseConfigFromString(e.textarea.Value())
		if err != nil || len(hosts) == 0 {
			e.err = "Invalid SSH config: must start with 'Host <name>'"
			return e, nil
		}
		h := hosts[0]
		conf = dto.SshConfigDto{
			Host:          h.Host,
			Values:        h.Values,
			IdentityFiles: h.IdentityFiles,
		}
	}
	if conf.Host == "" {
		e.err = "Host field is required"
		return e, nil
	}
	profile, err := utils.GetProfile()
	if err != nil {
		return NewErrorState(e.baseState, err), nil
	}
	client := retrieval.NewRetrievalClient()
	if err := client.UpsertConfig(profile, conf); err != nil {
		return NewErrorState(e.baseState, err), nil
	}
	mgr, err := NewSSHConfigManager(e.baseState)
	if err != nil {
		return NewErrorState(e.baseState, err), nil
	}
	mgr.height = e.height
	mgr.width = e.width
	return mgr, nil
}

func (e *SSHConfigEditor) cancel() State {
	if e.original != nil {
		return NewSSHConfigOptions(e.baseState, *e.original)
	}
	mgr, err := NewSSHConfigManager(e.baseState)
	if err != nil {
		return NewErrorState(e.baseState, err)
	}
	mgr.height = e.height
	mgr.width = e.width
	return mgr
}

func (e *SSHConfigEditor) PrettyName() string {
	return e.title
}

func (e *SSHConfigEditor) Update(msg tea.Msg) (State, tea.Cmd) {
	var cmds []tea.Cmd

	// Fire deferred cursor-blink command on first update
	if e.initCmd != nil {
		cmds = append(cmds, e.initCmd)
		e.initCmd = nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		e.err = "" // clear error on any keypress
		switch msg.String() {
		case "ctrl+c":
			return e, tea.Quit
		case "ctrl+s":
			state, cmd := e.save()
			return state, tea.Batch(append(cmds, cmd)...)
		case "esc":
			return e.cancel(), nil
		case "ctrl+f":
			var cmd tea.Cmd
			if e.mode == formMode {
				cmd = e.switchToFreetext()
			} else {
				cmd = e.switchToForm()
			}
			return e, tea.Batch(append(cmds, cmd)...)
		}

		if e.mode == formMode {
			switch msg.String() {
			case "tab":
				cmd := e.nextFocus()
				return e, tea.Batch(append(cmds, cmd)...)
			case "shift+tab":
				cmd := e.prevFocus()
				return e, tea.Batch(append(cmds, cmd)...)
			case "enter":
				if e.isAddButton() {
					cmd := e.addCustomField()
					return e, tea.Batch(append(cmds, cmd)...)
				}
			case "ctrl+d":
				if _, _, ok := e.customFieldAt(); ok {
					cmd := e.deleteCurrentCustomField()
					return e, tea.Batch(append(cmds, cmd)...)
				}
			}
		}
	}

	// Route message to the focused component
	var inputCmd tea.Cmd
	if e.mode == formMode {
		if e.focusIdx < standardFieldCount {
			e.stdInputs[e.focusIdx], inputCmd = e.stdInputs[e.focusIdx].Update(msg)
		} else if fieldNum, isValue, ok := e.customFieldAt(); ok {
			if isValue {
				e.customFields[fieldNum].value, inputCmd = e.customFields[fieldNum].value.Update(msg)
			} else {
				e.customFields[fieldNum].key, inputCmd = e.customFields[fieldNum].key.Update(msg)
			}
		}
	} else {
		e.textarea, inputCmd = e.textarea.Update(msg)
	}
	cmds = append(cmds, inputCmd)

	return e, tea.Batch(cmds...)
}

var (
	editorLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)
	editorActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	editorAddStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Bold(true)
	editorErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
)

func (e *SSHConfigEditor) View() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(" %s\n\n", editorLabelStyle.Render(e.title)))

	if e.mode == formMode {
		for i, label := range stdLabels {
			prefix := "  "
			if e.focusIdx == i {
				prefix = editorActiveStyle.Render("> ")
			}
			sb.WriteString(fmt.Sprintf("%s%s  %s\n",
				prefix,
				editorLabelStyle.Render(label),
				e.stdInputs[i].View(),
			))
		}

		if len(e.customFields) > 0 {
			sb.WriteString("\n  " + editorLabelStyle.Render("Custom fields") + "  " +
				lipgloss.NewStyle().Faint(true).Render("(ctrl+d to delete focused field)") + "\n")
		}
		for i, cf := range e.customFields {
			keyIdx := standardFieldCount + i*2
			valIdx := keyIdx + 1
			prefix := "  "
			if e.focusIdx == keyIdx || e.focusIdx == valIdx {
				prefix = editorActiveStyle.Render("> ")
			}
			sb.WriteString(fmt.Sprintf("%s%s : %s\n", prefix, cf.key.View(), cf.value.View()))
		}

		sb.WriteString("\n")
		addLine := "  " + editorAddStyle.Render("[ + Add custom field ]")
		if e.isAddButton() {
			addLine = editorActiveStyle.Render("> ") + editorAddStyle.Render("[ + Add custom field ]")
		}
		sb.WriteString(addLine + "\n\n")

		sb.WriteString("  " + BasicColorStyle.Render("tab") + "/" + BasicColorStyle.Render("shift+tab") + " navigate  " +
			BasicColorStyle.Render("enter") + " select  " +
			BasicColorStyle.Render("ctrl+d") + " del field  " +
			BasicColorStyle.Render("ctrl+f") + " freetext\n")
		sb.WriteString("  " + BasicColorStyle.Render("ctrl+s") + " save  " +
			BasicColorStyle.Render("esc") + " cancel\n")
	} else {
		sb.WriteString("  " + editorLabelStyle.Render("Freetext Mode") + "  " +
			lipgloss.NewStyle().Faint(true).Render("(ctrl+f to return to form)") + "\n\n")
		sb.WriteString(e.textarea.View())
		sb.WriteString("\n\n  " +
			BasicColorStyle.Render("ctrl+s") + " save  " +
			BasicColorStyle.Render("ctrl+f") + " form mode  " +
			BasicColorStyle.Render("esc") + " cancel\n")
	}

	if e.err != "" {
		sb.WriteString("\n  " + editorErrorStyle.Render("Error: "+e.err) + "\n")
	}

	return sb.String()
}

func (e *SSHConfigEditor) SetSize(width, height int) {
	e.baseState.SetSize(width, height)
	inputWidth := max(20, width-20)
	for i := range e.stdInputs {
		e.stdInputs[i].Width = inputWidth
	}
	cw := max(10, (width-20)/2-4)
	for i := range e.customFields {
		e.customFields[i].key.Width = cw
		e.customFields[i].value.Width = cw
	}
	e.textarea.SetWidth(width - 4)
	e.textarea.SetHeight(max(5, height-10))
}

func (e *SSHConfigEditor) Initialize() {
	e.SetSize(e.width, e.height)
	// Store the focus command; it will be fired on the first Update call
	// so the cursor blink reaches the bubbletea runtime.
	e.initCmd = e.applyFocus()
}

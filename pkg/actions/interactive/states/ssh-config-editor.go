package states

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/therealpaulgg/ssh-sync-common/pkg/dto"
	"github.com/therealpaulgg/ssh-sync/pkg/retrieval"
	"github.com/therealpaulgg/ssh-sync/pkg/utils"
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
// ↑/↓ navigates between fields; ←/→ moves between key and value within a custom field.
type SSHConfigEditor struct {
	baseState
	stdInputs    [standardFieldCount]textinput.Model
	customFields []customPair
	focusIdx     int
	original     *dto.SshConfigDto
	title        string
	err          string
}

// makeInput creates a textinput with a static (always-visible) cursor.
func makeInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Width = width
	ti.Cursor.SetMode(cursor.CursorStatic)
	return ti
}

func NewSSHConfigEditor(b baseState, config dto.SshConfigDto, original *dto.SshConfigDto) *SSHConfigEditor {
	e := &SSHConfigEditor{
		baseState: b,
		focusIdx:  0,
		original:  original,
	}

	if original != nil {
		e.title = "Edit Config: " + config.Host
	} else {
		e.title = "Add Config Entry"
	}

	w := 36
	e.stdInputs[fieldHost] = makeInput("myhost", w)
	e.stdInputs[fieldHostName] = makeInput("10.0.0.1 or hostname", w)
	e.stdInputs[fieldUser] = makeInput("ubuntu", w)
	e.stdInputs[fieldPort] = makeInput("22", w)
	e.stdInputs[fieldIdentityFile] = makeInput("~/.ssh/id_ed25519", w)

	if original != nil {
		e.populateFromConfig(config)
	}

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
	k := makeInput("Key", width)
	k.SetValue(key)
	v := makeInput("Value", width)
	v.SetValue(value)
	return customPair{key: k, value: v}
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

func (e *SSHConfigEditor) applyFocus() {
	for i := range e.stdInputs {
		e.stdInputs[i].Blur()
	}
	for i := range e.customFields {
		e.customFields[i].key.Blur()
		e.customFields[i].value.Blur()
	}
	if e.focusIdx < standardFieldCount {
		e.stdInputs[e.focusIdx].Focus()
	} else if fieldNum, isValue, ok := e.customFieldAt(); ok {
		if isValue {
			e.customFields[fieldNum].value.Focus()
		} else {
			e.customFields[fieldNum].key.Focus()
		}
	}
}

func (e *SSHConfigEditor) nextFocus() {
	e.focusIdx = (e.focusIdx + 1) % e.totalFocusable()
	e.applyFocus()
}

func (e *SSHConfigEditor) prevFocus() {
	e.focusIdx = (e.focusIdx - 1 + e.totalFocusable()) % e.totalFocusable()
	e.applyFocus()
}

func (e *SSHConfigEditor) addCustomField() {
	cw := max(10, (e.width-20)/2-4)
	e.customFields = append(e.customFields, e.makeCustomPair("", "", cw))
	e.focusIdx = standardFieldCount + (len(e.customFields)-1)*2
	e.applyFocus()
}

func (e *SSHConfigEditor) deleteCurrentCustomField() {
	if fieldNum, _, ok := e.customFieldAt(); ok {
		e.customFields = append(e.customFields[:fieldNum], e.customFields[fieldNum+1:]...)
		if e.focusIdx >= e.totalFocusable() {
			e.focusIdx = e.totalFocusable() - 1
		}
		e.applyFocus()
	}
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

func (e *SSHConfigEditor) save() (State, tea.Cmd) {
	conf := e.serializeToConfig()
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		e.err = ""
		fieldNum, isValue, onCustom := e.customFieldAt()

		switch msg.String() {
		case "ctrl+c":
			return e, tea.Quit
		case "esc":
			return e.cancel(), nil
		case "up":
			e.prevFocus()
			return e, nil
		case "down":
			e.nextFocus()
			return e, nil
		case "left":
			// On a custom value field → always jump back to the key on the same row
			if onCustom && isValue {
				e.prevFocus()
				return e, nil
			}
		case "right":
			// On a custom key field → always jump forward to the value on the same row
			if onCustom && !isValue {
				e.nextFocus()
				return e, nil
			}
		case "enter":
			if e.isAddButton() {
				e.addCustomField()
				return e, nil
			}
			return e.save()
		case "ctrl+d":
			if onCustom {
				e.deleteCurrentCustomField()
				return e, nil
			}
		}

		// Route remaining input to the focused textinput
		var cmd tea.Cmd
		if e.focusIdx < standardFieldCount {
			e.stdInputs[e.focusIdx], cmd = e.stdInputs[e.focusIdx].Update(msg)
		} else if onCustom {
			if isValue {
				e.customFields[fieldNum].value, cmd = e.customFields[fieldNum].value.Update(msg)
			} else {
				e.customFields[fieldNum].key, cmd = e.customFields[fieldNum].key.Update(msg)
			}
		}
		return e, cmd
	}
	return e, nil
}

var (
	editorLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)
	editorMarkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	editorAddStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Bold(true)
	editorErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
)

func (e *SSHConfigEditor) View() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(" %s\n\n", editorLabelStyle.Render(e.title)))

	for i, label := range stdLabels {
		prefix := "  "
		if e.focusIdx == i {
			prefix = editorMarkStyle.Render("> ")
		}
		sb.WriteString(fmt.Sprintf("%s%s  %s\n",
			prefix,
			editorLabelStyle.Render(label),
			e.stdInputs[i].View(),
		))
	}

	if len(e.customFields) > 0 {
		sb.WriteString("\n  " + editorLabelStyle.Render("Custom fields") + "  " +
			lipgloss.NewStyle().Faint(true).Render("(ctrl+d to delete)") + "\n")
	}
	for i, cf := range e.customFields {
		keyIdx := standardFieldCount + i*2
		valIdx := keyIdx + 1
		prefix := "  "
		if e.focusIdx == keyIdx || e.focusIdx == valIdx {
			prefix = editorMarkStyle.Render("> ")
		}
		sb.WriteString(fmt.Sprintf("%s%s : %s\n", prefix, cf.key.View(), cf.value.View()))
	}

	sb.WriteString("\n")
	addLine := "  " + editorAddStyle.Render("[ + Add custom field ]")
	if e.isAddButton() {
		addLine = editorMarkStyle.Render("> ") + editorAddStyle.Render("[ + Add custom field ]")
	}
	sb.WriteString(addLine + "\n\n")

	sb.WriteString("  " + BasicColorStyle.Render("↑/↓") + " navigate  " +
		BasicColorStyle.Render("←/→") + " key↔value  " +
		BasicColorStyle.Render("enter") + " save/add  " +
		BasicColorStyle.Render("ctrl+d") + " del  " +
		BasicColorStyle.Render("esc") + " cancel\n")

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
}

func (e *SSHConfigEditor) Initialize() {
	e.SetSize(e.width, e.height)
	e.applyFocus()
}

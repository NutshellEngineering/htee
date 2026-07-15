// Package tui implements the interactive bubbletea wizard behind `ht init`.
package tui

import (
	"fmt"
	"maps"
	"net/url"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"app.getnutshell/htee/internal/config"
)

type step int

const (
	stepConfirmOverwrite step = iota
	stepServerName
	stepServerURL
	stepAddAnother
	stepEntrypoint
	stepOpenAPISpec
	stepDone
)

// Options configures the wizard's starting state.
type Options struct {
	// Exists is true when a .ht/conf.toml already exists, in which case
	// the wizard opens with an overwrite confirmation before continuing.
	Exists bool

	// PresetServers, when non-empty, pre-populates the servers map (e.g.
	// from an OpenAPI spec) and skips the interactive server-adding loop.
	PresetServers map[string]string
	// SuggestedEntrypoint pre-fills the entrypoint prompt so the user can
	// accept it with Enter or type over it.
	SuggestedEntrypoint string
	// PresetOpenAPISpec, when non-empty, sets the openapispec field
	// directly and skips that prompt.
	PresetOpenAPISpec string
}

type model struct {
	step   step
	input  textinput.Model
	errMsg string

	pendingName string
	servers     map[string]string
	entrypoint  string
	openAPISpec string

	suggestedEntrypoint string
	presetOpenAPISpec   string

	confirmed bool
	quit      bool
}

func newModel(opts Options) model {
	ti := textinput.New()
	ti.CharLimit = 256

	m := model{
		input:               ti,
		servers:             map[string]string{},
		suggestedEntrypoint: opts.SuggestedEntrypoint,
		presetOpenAPISpec:   opts.PresetOpenAPISpec,
	}
	maps.Copy(m.servers, opts.PresetServers)

	if opts.Exists {
		m.step = stepConfirmOverwrite
	} else {
		m.startWizard()
	}
	return m
}

// startWizard enters the first interactive step: straight to the
// entrypoint prompt if servers were already supplied (e.g. from an
// OpenAPI spec), otherwise the server-adding loop.
func (m *model) startWizard() {
	if len(m.servers) > 0 {
		m.enterEntrypointStep()
	} else {
		m.step = stepServerName
		m.focusInput("Server name", "e.g. local")
	}
}

func (m *model) enterEntrypointStep() {
	m.step = stepEntrypoint
	m.focusInput("Entrypoint path", "optional, e.g. /v1")
	if m.suggestedEntrypoint != "" {
		m.input.SetValue(m.suggestedEntrypoint)
		m.input.CursorEnd()
	}
}

// Run launches the wizard and returns the config the user built, along with
// whether they completed it (false if they cancelled, or declined an
// overwrite prompt - in either case nothing should be written to disk).
func Run(opts Options) (config.Config, bool, error) {
	m := newModel(opts)
	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return config.Config{}, false, err
	}
	fm := finalModel.(model)
	return fm.result(), fm.confirmed, nil
}

func (m model) result() config.Config {
	return config.Config{
		Entrypoint:  m.entrypoint,
		OpenAPISpec: m.openAPISpec,
		Servers:     m.servers,
	}
}

func (m *model) focusInput(placeholder, hint string) {
	m.input.Reset()
	m.input.Placeholder = placeholder
	if hint != "" {
		m.input.Placeholder = placeholder + " (" + hint + ")"
	}
	m.input.Focus()
	m.errMsg = ""
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		m.confirmed = false
		m.quit = true
		return m, tea.Quit
	}

	switch m.step {
	case stepConfirmOverwrite:
		return m.updateConfirmOverwrite(keyMsg)
	case stepServerName:
		return m.updateServerName(keyMsg)
	case stepServerURL:
		return m.updateServerURL(keyMsg)
	case stepAddAnother:
		return m.updateAddAnother(keyMsg)
	case stepEntrypoint:
		return m.updateEntrypoint(keyMsg)
	case stepOpenAPISpec:
		return m.updateOpenAPISpec(keyMsg)
	default:
		return m, nil
	}
}

func (m model) updateConfirmOverwrite(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.startWizard()
		return m, textinput.Blink
	case "n", "N", "enter":
		m.confirmed = false
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) updateServerName(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			m.errMsg = "server name cannot be empty"
			return m, nil
		}
		m.pendingName = name
		m.step = stepServerURL
		m.focusInput("Server URL", "e.g. http://localhost:8080")
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateServerURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		raw := strings.TrimSpace(m.input.Value())
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			m.errMsg = "enter a full URL with scheme and host, e.g. http://localhost:8080"
			return m, nil
		}
		m.servers[m.pendingName] = raw
		m.pendingName = ""
		m.step = stepAddAnother
		m.input.Blur()
		m.errMsg = ""
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateAddAnother(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.step = stepServerName
		m.focusInput("Server name", "e.g. local")
		return m, textinput.Blink
	case "n", "N", "enter":
		m.enterEntrypointStep()
		return m, textinput.Blink
	}
	return m, nil
}

func (m model) updateEntrypoint(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		m.entrypoint = strings.TrimSpace(m.input.Value())
		if m.presetOpenAPISpec != "" {
			m.openAPISpec = m.presetOpenAPISpec
			m.step = stepDone
			m.confirmed = true
			m.quit = true
			return m, tea.Quit
		}
		m.step = stepOpenAPISpec
		m.focusInput("OpenAPI spec path", "optional, e.g. openapi.yaml")
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateOpenAPISpec(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		m.openAPISpec = strings.TrimSpace(m.input.Value())
		m.step = stepDone
		m.confirmed = true
		m.quit = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder
	switch m.step {
	case stepConfirmOverwrite:
		b.WriteString(".ht/conf.toml already exists. Overwrite it? (y/N)\n")
	case stepServerName:
		b.WriteString("Add a server.\n")
		b.WriteString(m.input.View() + "\n")
	case stepServerURL:
		_, err := fmt.Fprintf(&b, "Server %q: what's its URL?\n", m.pendingName)
		if err != nil {
			panic(err)
		}
		b.WriteString(m.input.View() + "\n")
	case stepAddAnother:
		_, err := fmt.Fprintf(&b, "%d server(s) added. Add another? (y/N)\n", len(m.servers))
		if err != nil {
			panic(err)
		}
	case stepEntrypoint:
		if m.presetOpenAPISpec != "" {
			_, err := fmt.Fprintf(&b, "Loaded %d server(s) from %s.\n", len(m.servers), m.presetOpenAPISpec)
			if err != nil {
				panic(err)
			}
		}
		b.WriteString("Entry point path for this API (leave blank to skip).\n")
		b.WriteString(m.input.View() + "\n")
	case stepOpenAPISpec:
		b.WriteString("OpenAPI 3 spec path (leave blank to skip).\n")
		b.WriteString(m.input.View() + "\n")
	default:
		// Taking a peaceful approach to unexpected future changes, and continuing
		break
	}
	if m.errMsg != "" {
		b.WriteString(m.errMsg + "\n")
	}
	b.WriteString("\n(esc to cancel)\n")
	return b.String()
}

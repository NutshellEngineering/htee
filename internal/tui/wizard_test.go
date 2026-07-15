package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func send(m tea.Model, msg tea.Msg) tea.Model {
	next, _ := m.Update(msg)
	return next
}

func typeText(m tea.Model, s string) tea.Model {
	return send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

func pressEnter(m tea.Model) tea.Model {
	return send(m, tea.KeyMsg{Type: tea.KeyEnter})
}

func pressRune(m tea.Model, r string) tea.Model {
	return send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)})
}

func pressEsc(m tea.Model) tea.Model {
	return send(m, tea.KeyMsg{Type: tea.KeyEsc})
}

func addServer(m tea.Model, name, url string) tea.Model {
	m = typeText(m, name)
	m = pressEnter(m)
	m = typeText(m, url)
	m = pressEnter(m)
	return m
}

func TestWizardHappyPathOneServer(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = addServer(m, "local", "http://localhost:8080")
	m = pressRune(m, "n") // decline "add another"
	m = pressEnter(m)     // blank entrypoint
	m = pressEnter(m)     // blank openapispec

	fm := m.(model)
	if !fm.confirmed {
		t.Fatalf("expected confirmed=true, errMsg=%q", fm.errMsg)
	}
	cfg := fm.result()
	if len(cfg.Servers) != 1 || cfg.Servers["local"] != "http://localhost:8080" {
		t.Fatalf("Servers = %v", cfg.Servers)
	}
	if cfg.Entrypoint != "" || cfg.OpenAPISpec != "" {
		t.Fatalf("expected blank optional fields, got Entrypoint=%q OpenAPISpec=%q", cfg.Entrypoint, cfg.OpenAPISpec)
	}
}

func TestWizardTwoServersViaLoopAndOptionalFields(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = addServer(m, "local", "http://localhost:8080")
	m = pressRune(m, "y") // add another
	m = addServer(m, "prod", "https://api.example.com")
	m = pressRune(m, "n")
	m = typeText(m, "/v1")
	m = pressEnter(m)
	m = typeText(m, "openapi.yaml")
	m = pressEnter(m)

	fm := m.(model)
	if !fm.confirmed {
		t.Fatalf("expected confirmed=true")
	}
	cfg := fm.result()
	if len(cfg.Servers) != 2 {
		t.Fatalf("Servers = %v", cfg.Servers)
	}
	if cfg.Servers["local"] != "http://localhost:8080" || cfg.Servers["prod"] != "https://api.example.com" {
		t.Fatalf("Servers = %v", cfg.Servers)
	}
	if cfg.Entrypoint != "/v1" || cfg.OpenAPISpec != "openapi.yaml" {
		t.Fatalf("Entrypoint=%q OpenAPISpec=%q", cfg.Entrypoint, cfg.OpenAPISpec)
	}
}

func TestWizardRejectsEmptyServerName(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = pressEnter(m) // submit empty name
	fm := m.(model)
	if fm.step != stepServerName {
		t.Fatalf("expected to stay on stepServerName, got %v", fm.step)
	}
	if fm.errMsg == "" {
		t.Fatalf("expected a validation error message")
	}
}

func TestWizardRejectsInvalidURL(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = typeText(m, "local")
	m = pressEnter(m)
	m = typeText(m, "not-a-url")
	m = pressEnter(m)

	fm := m.(model)
	if fm.step != stepServerURL {
		t.Fatalf("expected to stay on stepServerURL, got %v", fm.step)
	}
	if fm.errMsg == "" {
		t.Fatalf("expected a validation error message")
	}
}

func TestWizardOverwriteDeclined(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: true})

	fm := m.(model)
	if fm.step != stepConfirmOverwrite {
		t.Fatalf("expected to start on stepConfirmOverwrite, got %v", fm.step)
	}

	m = pressRune(m, "n")
	fm = m.(model)
	if fm.confirmed {
		t.Fatalf("expected confirmed=false after declining overwrite")
	}
	if !fm.quit {
		t.Fatalf("expected quit=true after declining overwrite")
	}
}

func TestWizardOverwriteAcceptedContinuesToServerName(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: true})

	m = pressRune(m, "y")
	fm := m.(model)
	if fm.step != stepServerName {
		t.Fatalf("expected stepServerName after accepting overwrite, got %v", fm.step)
	}
}

func TestWizardEscCancelsMidFlow(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = typeText(m, "local")
	m = pressEnter(m)
	m = pressEsc(m)

	fm := m.(model)
	if fm.confirmed {
		t.Fatalf("expected confirmed=false after esc")
	}
	if !fm.quit {
		t.Fatalf("expected quit=true after esc")
	}
}

func TestWizardCtrlCCancels(t *testing.T) {
	var m tea.Model = newModel(Options{Exists: false})

	m = send(m, tea.KeyMsg{Type: tea.KeyCtrlC})
	fm := m.(model)
	if fm.confirmed || !fm.quit {
		t.Fatalf("expected confirmed=false, quit=true; got confirmed=%v quit=%v", fm.confirmed, fm.quit)
	}
}

func TestWizardPresetServersSkipsServerLoopAndPrefillsEntrypoint(t *testing.T) {
	var m tea.Model = newModel(Options{
		Exists:              false,
		PresetServers:       map[string]string{"prod": "https://api.example.com"},
		SuggestedEntrypoint: "/pets",
		PresetOpenAPISpec:   "openapi.yaml",
	})

	fm := m.(model)
	if fm.step != stepEntrypoint {
		t.Fatalf("expected to start on stepEntrypoint, got %v", fm.step)
	}
	if fm.input.Value() != "/pets" {
		t.Fatalf("expected entrypoint input pre-filled with suggestion, got %q", fm.input.Value())
	}

	// Accept the suggested entrypoint as-is; the openapispec preset should
	// skip that prompt entirely and finish the wizard.
	m = pressEnter(m)
	fm = m.(model)
	if !fm.confirmed || !fm.quit {
		t.Fatalf("expected confirmed=true, quit=true; got confirmed=%v quit=%v", fm.confirmed, fm.quit)
	}

	cfg := fm.result()
	if len(cfg.Servers) != 1 || cfg.Servers["prod"] != "https://api.example.com" {
		t.Fatalf("Servers = %v", cfg.Servers)
	}
	if cfg.Entrypoint != "/pets" {
		t.Fatalf("Entrypoint = %q, want %q", cfg.Entrypoint, "/pets")
	}
	if cfg.OpenAPISpec != "openapi.yaml" {
		t.Fatalf("OpenAPISpec = %q, want %q", cfg.OpenAPISpec, "openapi.yaml")
	}
}

func TestWizardPresetServersOverwriteFlowGoesToEntrypoint(t *testing.T) {
	var m tea.Model = newModel(Options{
		Exists:        true,
		PresetServers: map[string]string{"prod": "https://api.example.com"},
	})

	fm := m.(model)
	if fm.step != stepConfirmOverwrite {
		t.Fatalf("expected to start on stepConfirmOverwrite, got %v", fm.step)
	}

	m = pressRune(m, "y")
	fm = m.(model)
	if fm.step != stepEntrypoint {
		t.Fatalf("expected stepEntrypoint after accepting overwrite with preset servers, got %v", fm.step)
	}
}

func TestWizardEntrypointSuggestionIsEditable(t *testing.T) {
	var m tea.Model = newModel(Options{
		PresetServers:       map[string]string{"prod": "https://api.example.com"},
		SuggestedEntrypoint: "/pets",
	})

	// Cursor starts after the pre-filled suggestion, so backspacing clears
	// it before typing a replacement.
	for range "/pets" {
		m = send(m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m = typeText(m, "/v2/pets")
	m = pressEnter(m)
	m = pressEnter(m) // blank openapispec, since no preset was given

	fm := m.(model)
	if !fm.confirmed {
		t.Fatalf("expected confirmed=true")
	}
	cfg := fm.result()
	if cfg.Entrypoint != "/v2/pets" {
		t.Fatalf("Entrypoint = %q, want %q", cfg.Entrypoint, "/v2/pets")
	}
	if cfg.OpenAPISpec != "" {
		t.Fatalf("expected no openapispec preset, got %q", cfg.OpenAPISpec)
	}
}

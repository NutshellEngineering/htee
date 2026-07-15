package output

import (
	"bytes"
	"strings"
	"testing"

	"app.getnutshell/htee/internal/openapi"
)

func TestRenderValidationRouteNotFound(t *testing.T) {
	var buf bytes.Buffer
	RenderValidation(&buf, openapi.ValidationOutcome{RouteFound: false}, false)

	out := buf.String()
	if !strings.Contains(out, "not found in the OpenAPI spec") {
		t.Fatalf("output = %q, expected a not-found warning", out)
	}
}

func TestRenderValidationValid(t *testing.T) {
	var buf bytes.Buffer
	RenderValidation(&buf, openapi.ValidationOutcome{RouteFound: true}, false)

	out := buf.String()
	if !strings.Contains(out, "response is valid") {
		t.Fatalf("output = %q, expected a success message", out)
	}
}

func TestRenderValidationFailuresListed(t *testing.T) {
	var buf bytes.Buffer
	RenderValidation(&buf, openapi.ValidationOutcome{
		RouteFound:     true,
		RequestErrors:  []string{"missing header X"},
		ResponseErrors: []string{"body missing field name", "field age has wrong type"},
	}, false)

	out := buf.String()
	for _, want := range []string{"missing header X", "body missing field name", "field age has wrong type"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing failure %q:\n%s", want, out)
		}
	}
}

func TestRenderValidationEnabled(t *testing.T) {
	var withColors bytes.Buffer
	RenderValidationEnabled(&withColors, true)
	out := withColors.String()
	if !strings.Contains(out, "validating") {
		t.Fatalf("output = %q, expected a validating notice", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI escape codes when colorsEnabled=true, got %q", out)
	}

	var noColors bytes.Buffer
	RenderValidationEnabled(&noColors, false)
	if strings.Contains(noColors.String(), "\x1b[") {
		t.Fatalf("expected no ANSI escape codes when colorsEnabled=false, got %q", noColors.String())
	}
}

func TestRenderValidationColorsGated(t *testing.T) {
	outcome := openapi.ValidationOutcome{RouteFound: true}

	var withColors bytes.Buffer
	RenderValidation(&withColors, outcome, true)
	if !strings.Contains(withColors.String(), "\x1b[") {
		t.Fatalf("expected ANSI escape codes when colorsEnabled=true, got %q", withColors.String())
	}

	var noColors bytes.Buffer
	RenderValidation(&noColors, outcome, false)
	if strings.Contains(noColors.String(), "\x1b[") {
		t.Fatalf("expected no ANSI escape codes when colorsEnabled=false, got %q", noColors.String())
	}
}

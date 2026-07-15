// Package theme provides a small set of lipgloss styles shared by every
// part of htee that renders more than plain request/response bytes:
// OpenAPI validation results today, the ht init wizard and future
// interactive UI later - so colors stay centralized in one place instead
// of being redefined ad hoc at each call site.
package theme

import (
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme is a small set of semantic styles.
type Theme struct {
	Success lipgloss.Style
	Failure lipgloss.Style
	Warning lipgloss.Style
}

// New builds a Theme bound to r, so its styles honor whatever color
// decision r was constructed with (see NewRenderer) rather than lipgloss's
// own terminal detection.
func New(r *lipgloss.Renderer) Theme {
	return Theme{
		Success: r.NewStyle().Foreground(lipgloss.Color("10")).Bold(true), // bright green
		Failure: r.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),  // bright red
		Warning: r.NewStyle().Foreground(lipgloss.Color("11")).Bold(true), // bright yellow
	}
}

// NewRenderer builds a lipgloss.Renderer writing to w, with its color
// profile forced by colorsEnabled instead of lipgloss's own w-is-a-TTY
// autodetection (w is frequently a buffer, not the real terminal, e.g. in
// tests or when composing output before writing it out). Callers that have
// already decided whether colors should be on (e.g. via --pretty and
// isatty) must drive that decision here so every renderer agrees with the
// rest of htee's output.
func NewRenderer(w io.Writer, colorsEnabled bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(w)
	if colorsEnabled {
		r.SetColorProfile(termenv.ANSI)
	} else {
		r.SetColorProfile(termenv.Ascii)
	}
	return r
}

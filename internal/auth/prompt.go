package auth

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// PromptPassword prompts on stderr and reads a password from the terminal
// without echoing it. Used when `-a user` is given without a password, and
// by the transport package for an encrypted --cert-key without
// --cert-key-pass.
func PromptPassword(prompt string) (string, error) {
	stdin := int(os.Stdin.Fd())
	if !term.IsTerminal(stdin) {
		return "", fmt.Errorf("no password provided for -a/--auth, and stdin is not a terminal to prompt on (pass credentials as -a user:pass instead)")
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(stdin)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(b), nil
}

// Package netrc implements a minimal, best-effort reader for the .netrc
// credentials file format, mirroring how httpie resolves fallback auth via
// requests.utils.get_netrc_auth (Python's stdlib netrc module): a "machine"
// entry matching the request host wins, falling back to a "default" entry
// if present. Used only when nothing more specific (-a/-A, URL userinfo,
// HT_AUTH/AUTH_TOKEN) supplied credentials, and only when --ignore-netrc
// isn't set.
package netrc

import (
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	login    string
	password string
}

// Lookup returns the login/password netrc credentials for host. It reads
// the file named by $NETRC if set, else ~/.netrc, falling back to ~/_netrc
// (the Windows-style name some tools also accept on other platforms).
// Any error (missing file, unreadable, etc.) is treated as "no match".
func Lookup(host string) (user, pass string, ok bool) {
	path := findFile()
	if path == "" {
		return "", "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	entries, def := parse(string(data))
	if e, found := entries[host]; found {
		return e.login, e.password, true
	}
	if def != nil {
		return def.login, def.password, true
	}
	return "", "", false
}

func findFile() string {
	if p := os.Getenv("NETRC"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{".netrc", "_netrc"} {
		p := filepath.Join(home, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

// parse tokenizes netrc content (whitespace-delimited, spanning lines) into
// machine entries plus an optional default entry. "macdef" blocks are
// opaque script bodies terminated by a blank line, so their contents are
// stripped before tokenizing rather than parsed as fields.
func parse(content string) (map[string]*entry, *entry) {
	tokens := tokenize(stripMacdefs(content))

	entries := map[string]*entry{}
	var def *entry
	var cur *entry

	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "machine":
			if i+1 >= len(tokens) {
				break
			}
			i++
			cur = &entry{}
			entries[tokens[i]] = cur
		case "default":
			cur = &entry{}
			def = cur
		case "login":
			if i+1 < len(tokens) && cur != nil {
				i++
				cur.login = tokens[i]
			}
		case "password":
			if i+1 < len(tokens) && cur != nil {
				i++
				cur.password = tokens[i]
			}
		case "account":
			if i+1 < len(tokens) {
				i++
			}
		}
	}
	return entries, def
}

func stripMacdefs(content string) string {
	lines := strings.Split(content, "\n")
	var kept []string
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "macdef") {
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				i++
			}
			continue
		}
		kept = append(kept, lines[i])
	}
	return strings.Join(kept, "\n")
}

func tokenize(content string) []string {
	return strings.Fields(content)
}

package request

import (
	"mime"
	"path/filepath"
)

// guessMimeType guesses a file's Content-Type from its extension, falling
// back to a generic binary type when unknown. Mirrors httpie's use of
// Python's stdlib `mimetypes` module for the same purpose.
func guessMimeType(path string) string {
	if t := mime.TypeByExtension(filepath.Ext(path)); t != "" {
		return t
	}
	return "application/octet-stream"
}

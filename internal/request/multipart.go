package request

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"

	"app.getnutshell/htee/internal/itemsyntax"
)

// buildMultipartBody writes ri.MultipartData (plain fields and file
// uploads) to a multipart/form-data body, preserving declaration order.
// Returns the body bytes and the Content-Type header value (including the
// boundary).
func buildMultipartBody(fields []itemsyntax.MultipartField, boundary string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if boundary != "" {
		if err := w.SetBoundary(boundary); err != nil {
			return nil, "", fmt.Errorf("invalid --boundary: %w", err)
		}
	}

	for _, f := range fields {
		if f.File == nil {
			fw, err := w.CreateFormField(f.Key)
			if err != nil {
				return nil, "", err
			}
			if _, err := fw.Write([]byte(f.Value)); err != nil {
				return nil, "", err
			}
			continue
		}

		mimeType := f.File.MimeType
		if mimeType == "" {
			mimeType = guessMimeType(f.File.Path)
		}
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(
			`form-data; name="%s"; filename="%s"`, f.Key, filepath.Base(f.File.Path)))
		header.Set("Content-Type", mimeType)
		pw, err := w.CreatePart(header)
		if err != nil {
			return nil, "", err
		}
		content, err := os.ReadFile(f.File.Path)
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", f.File.Path, err)
		}
		if _, err := pw.Write(content); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

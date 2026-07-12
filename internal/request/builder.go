// Package request builds an *http.Request from a resolved (method, url,
// items) triple, implementing httpie's body-mode selection (JSON/form/
// multipart/raw), default headers, and URL normalization.
package request

import (
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"app.getnutshell/htee/internal/itemsyntax"
	"app.getnutshell/htee/internal/ordered"
)

// BodyMode identifies how the request body was constructed.
type BodyMode int

const (
	BodyModeJSON BodyMode = iota
	BodyModeForm
	BodyModeMultipart
	BodyModeRaw
)

// Options configures Build.
type Options struct {
	Method        string
	URL           string // not yet normalized
	DefaultScheme string // if "", auto-selected per host by NormalizeURL (http for localhost, https otherwise)
	Items         []itemsyntax.KeyValueArg

	Form      bool
	Multipart bool
	Boundary  string

	HasRaw bool
	Raw    string

	HasStdin bool
	Stdin    []byte
}

// Result is a built request, along with the exact body bytes sent (needed
// for rendering `-p B`/`-p H` output byte-accurately).
type Result struct {
	Request     *http.Request
	Body        []byte
	Mode        BodyMode
	HeaderOrder []string // header names in declaration order, for rendering
}

// Build resolves body mode, constructs the body and headers, and returns a
// ready-to-send *http.Request. Mirrors the request-building portions of
// httpie's client.py/uploads.py/models.py.
func Build(opts Options) (*Result, error) {
	normalizedURL := NormalizeURL(opts.URL, opts.DefaultScheme)

	isJSONMode := !opts.Form && !opts.Multipart
	ri, err := itemsyntax.FromArgs(opts.Items, isJSONMode)
	if err != nil {
		return nil, err
	}

	hasKeyedData := ri.Data != nil || len(ri.Files) > 0
	if opts.HasRaw && opts.HasStdin {
		return nil, fmt.Errorf("request body from both --raw and stdin: cannot use both")
	}
	if (opts.HasRaw || opts.HasStdin) && hasKeyedData {
		return nil, fmt.Errorf("request body (from --raw or stdin) and request data (key=value items) cannot be mixed")
	}

	bareFile, err := resolveBareFileBody(opts, ri)
	if err != nil {
		return nil, err
	}

	var (
		mode                BodyMode
		body                []byte
		multipartCT         string
		bareFileContentType string
	)

	switch {
	case opts.HasRaw:
		mode = BodyModeRaw
		body = []byte(opts.Raw)

	case opts.HasStdin:
		mode = BodyModeRaw
		body = opts.Stdin

	case bareFile:
		mode = BodyModeRaw
		content, ctype, err := readBareFileBody(ri.Files[0])
		if err != nil {
			return nil, err
		}
		body = content
		bareFileContentType = ctype

	case opts.Multipart || (!opts.Form && len(ri.Files) > 0):
		mode = BodyModeMultipart
		b, ct, err := buildMultipartBody(ri.MultipartData, opts.Boundary)
		if err != nil {
			return nil, err
		}
		body = b
		multipartCT = ct

	case opts.Form:
		mode = BodyModeForm
		m, _ := ri.Data.(*ordered.Map)
		body = buildFormBody(m)

	default:
		mode = BodyModeJSON
		b, hasBody, err := buildJSONBody(ri.Data)
		if err != nil {
			return nil, err
		}
		if hasBody {
			body = b
		}
	}

	req, err := http.NewRequest(opts.Method, normalizedURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	order := newHeaderOrder()
	applyDefaultHeaders(req.Header, order, mode, len(body) > 0)
	if multipartCT != "" {
		req.Header.Set("Content-Type", multipartCT)
		order.add("Content-Type")
	}
	if bareFileContentType != "" {
		req.Header.Set("Content-Type", bareFileContentType)
		order.add("Content-Type")
	}
	applyHeaderItems(req.Header, order, ri.Headers)

	if len(ri.Params) > 0 {
		req.URL.RawQuery = applyQueryParams(req.URL.RawQuery, ri.Params)
	}

	req.ContentLength = int64(len(body))

	return &Result{Request: req, Body: body, Mode: mode, HeaderOrder: order.Names()}, nil
}

// resolveBareFileBody detects httpie's special "whole body from a single,
// keyless file item" case: `ht post :8080 @data.json` (no --form). Errors
// if files are present without --form/--multipart in any other shape.
//
// Mirrors HTTPieArgumentParser._parse_items's file-fields validation in
// httpie/cli/argparser.py.
func resolveBareFileBody(opts Options, ri *itemsyntax.RequestItems) (bool, error) {
	if opts.Form || opts.Multipart {
		return false, nil
	}
	hasKeylessFile := false
	for _, f := range ri.Files {
		if f.Key == "" {
			hasKeylessFile = true
			break
		}
	}
	if !hasKeylessFile {
		return false, nil
	}
	// A keyless file item (bare `@path`) means "whole body from this file",
	// which is only unambiguous if it's the sole item.
	if len(ri.Files) != 1 || ri.Data != nil {
		return false, fmt.Errorf("invalid file fields (perhaps you meant --form?)")
	}
	return true, nil
}

func readBareFileBody(f itemsyntax.FileField) ([]byte, string, error) {
	content, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", f.Path, err)
	}
	contentType := f.MimeType
	if contentType == "" {
		if t := mime.TypeByExtension(filepath.Ext(f.Path)); t != "" {
			contentType = t
		}
	}
	return content, contentType, nil
}

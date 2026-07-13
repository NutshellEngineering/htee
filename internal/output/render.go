package output

import (
	"fmt"
	"io"
	"strings"

	"app.getnutshell/htee/internal/message"
)

// RenderRequest writes the request line, headers, and body per flags,
// applying --pretty/--style/--format-options on top of the plain layout
// rules in message.RenderRequest.
func RenderRequest(w io.Writer, req message.Request, flags message.PrintFlags, opts Options) {
	if flags.ReqHeaders {
		headers := req.Headers
		if opts.Pretty.Format && opts.FormatOptions.HeadersSort {
			headers = sortHeaders(headers)
		}
		writeHead(w, opts, colorizeRequestLine(req.Method, req.Target, req.Proto),
			fmt.Sprintf("%s %s %s", req.Method, req.Target, req.Proto), headers)
	}
	if flags.ReqBody && len(req.Body) > 0 {
		body, _ := processBody(req.Body, mimeOf(req.Headers), opts, false)
		mustWrite(w, body)
		mustFprint(w, "\n")
	}
}

// RenderResponse writes the status line, headers, body, and metadata per
// flags, applying the same output-processing pipeline as RenderRequest
// plus the response-only --response-mime/--response-charset overrides.
func RenderResponse(w io.Writer, resp message.Response, flags message.PrintFlags, opts Options) {
	if flags.RespHeaders {
		headers := resp.Headers
		if opts.Pretty.Format && opts.FormatOptions.HeadersSort {
			headers = sortHeaders(headers)
		}
		statusLine := fmt.Sprintf("%s %d %s", resp.Proto, resp.StatusCode, resp.Reason)
		writeHead(w, opts, colorizeStatusLine(resp.Proto, resp.StatusCode, resp.Reason), statusLine, headers)
	}
	if flags.RespBody && len(resp.Body) > 0 {
		body, _ := processBody(resp.Body, mimeOf(resp.Headers), opts, true)
		mustWrite(w, body)
		mustFprint(w, "\n")
	}
	if flags.RespMeta {
		mustFprintf(w, "\nElapsed time: %fs\n", resp.Elapsed)
	}
}

func writeHead(w io.Writer, opts Options, coloredLine, plainLine string, headers []message.Header) {
	if opts.Pretty.Colors {
		mustFprint(w, coloredLine)
		mustFprint(w, "\r\n")
		for _, h := range headers {
			mustFprint(w, colorizeHeaderLine(h.Name, h.Value))
			mustFprint(w, "\r\n")
		}
	} else {
		mustFprint(w, plainLine)
		mustFprint(w, "\r\n")
		for _, h := range headers {
			mustFprintf(w, "%s: %s\r\n", h.Name, h.Value)
		}
	}
	mustFprint(w, "\r\n")
}

// mustFprint writes to w, panicking if the write fails. Output writers here
// are process stdout/buffers where a write failure means the destination is
// broken beyond the point of a graceful, per-call recovery.
func mustFprint(w io.Writer, a ...any) {
	if _, err := fmt.Fprint(w, a...); err != nil {
		panic(err)
	}
}

// mustFprintf is the formatted counterpart to mustFprint.
func mustFprintf(w io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		panic(err)
	}
}

// mustWrite writes p to w in full, panicking if the write fails.
func mustWrite(w io.Writer, p []byte) {
	if _, err := w.Write(p); err != nil {
		panic(err)
	}
}

// processBody applies charset decoding (response only), JSON/XML
// re-indentation, and syntax-highlight coloring, returning the rendered
// body and the MIME type used to make those decisions (after any
// --response-mime override).
func processBody(body []byte, mime string, opts Options, isResponse bool) ([]byte, string) {
	if isResponse && opts.ResponseCharset != "" {
		if decoded, err := decodeCharset(body, opts.ResponseCharset); err == nil {
			body = decoded
		}
	}
	effectiveMime := mime
	if isResponse && opts.ResponseMime != "" {
		effectiveMime = opts.ResponseMime
	}

	if opts.Pretty.Format && !opts.Stream {
		body = formatJSONBody(body, opts.FormatOptions)
		if strings.Contains(effectiveMime, "xml") {
			body = formatXMLBody(body, opts.FormatOptions)
		}
	}
	if opts.Pretty.Colors {
		style, formatter := resolveStyleAndFormatter(opts.Style)
		body = colorizeBody(body, effectiveMime, style, formatter)
	}
	return body, effectiveMime
}

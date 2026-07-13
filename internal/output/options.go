// Package output implements httpie's output-processing pipeline: the
// --pretty/--style/--format-options/--unsorted/--sorted flags, JSON/XML
// body re-indentation, header sorting, and chroma-based syntax
// highlighting. It renders the message.Request/message.Response models
// built by the request/transport layers.
package output

import (
	"fmt"
	"strconv"
	"strings"
)

// Pretty selects which output-processing stages are active, mirroring
// httpie's --pretty={none,colors,format,all}.
type Pretty struct {
	Colors bool
	Format bool
}

// ResolvePretty implements httpie's --pretty defaulting: an explicit value
// wins; otherwise pretty is "all" on a TTY and "none" when redirected
// (PRETTY_STDOUT_TTY_ONLY in cli/constants.py).
func ResolvePretty(explicit string, hasExplicit bool, stdoutIsTTY bool) (Pretty, error) {
	if !hasExplicit || explicit == "" {
		if stdoutIsTTY {
			return Pretty{Colors: true, Format: true}, nil
		}
		return Pretty{}, nil
	}
	switch explicit {
	case "all":
		return Pretty{Colors: true, Format: true}, nil
	case "colors":
		return Pretty{Colors: true}, nil
	case "format":
		return Pretty{Format: true}, nil
	case "none":
		return Pretty{}, nil
	default:
		return Pretty{}, fmt.Errorf("invalid --pretty value %q (expected one of: all, colors, format, none)", explicit)
	}
}

// FormatOptions controls body/header re-formatting, mirroring httpie's
// --format-options key:value settings (cli/constants.py DEFAULT_FORMAT_OPTIONS).
type FormatOptions struct {
	HeadersSort  bool
	JSONFormat   bool
	JSONIndent   int
	JSONSortKeys bool
	XMLFormat    bool
	XMLIndent    int
}

// DefaultFormatOptions returns httpie's DEFAULT_FORMAT_OPTIONS.
func DefaultFormatOptions() FormatOptions {
	return FormatOptions{
		HeadersSort:  true,
		JSONFormat:   true,
		JSONIndent:   4,
		JSONSortKeys: true,
		XMLFormat:    true,
		XMLIndent:    2,
	}
}

// SortedFormatOptionsString and UnsortedFormatOptionsString back the
// --sorted/--unsorted/--no-sorted/--no-unsorted flag shortcuts, which are
// wired to append these strings onto the same raw --format-options list
// (mirroring httpie's append_const-onto-shared-dest trick in definition.py).
const (
	SortedFormatOptionsString   = "headers.sort:true,json.sort_keys:true"
	UnsortedFormatOptionsString = "headers.sort:false,json.sort_keys:false"
)

// ParseFormatOptions applies a sequence of comma-separated "key:value"
// groups (each element of raw may itself contain multiple comma-separated
// tokens) onto the defaults, in order - later tokens win. This mirrors
// --format-options plus the --sorted/--unsorted family, all of which append
// onto the same underlying list in command-line order.
func ParseFormatOptions(raw []string) (FormatOptions, error) {
	opts := DefaultFormatOptions()
	for _, group := range raw {
		for tok := range strings.SplitSeq(group, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			key, val, ok := strings.Cut(tok, ":")
			if !ok {
				return opts, fmt.Errorf("invalid --format-options token %q (expected key:value)", tok)
			}
			if err := opts.apply(key, val); err != nil {
				return opts, err
			}
		}
	}
	return opts, nil
}

func (o *FormatOptions) apply(key, val string) error {
	switch key {
	case "headers.sort":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid value for headers.sort: %q", val)
		}
		o.HeadersSort = b
	case "json.format":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid value for json.format: %q", val)
		}
		o.JSONFormat = b
	case "json.indent":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid value for json.indent: %q", val)
		}
		o.JSONIndent = n
	case "json.sort_keys":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid value for json.sort_keys: %q", val)
		}
		o.JSONSortKeys = b
	case "xml.format":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid value for xml.format: %q", val)
		}
		o.XMLFormat = b
	case "xml.indent":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid value for xml.indent: %q", val)
		}
		o.XMLIndent = n
	default:
		return fmt.Errorf("unknown --format-options key %q", key)
	}
	return nil
}

// Options bundles everything the renderer needs beyond the print-flag
// bitset (message.PrintFlags), covering --pretty/--style/--format-options
// plus the response-only overrides and --stream.
type Options struct {
	Pretty        Pretty
	Style         string // "auto" (default) or a chroma style name
	FormatOptions FormatOptions

	ResponseMime    string // --response-mime override (response body only)
	ResponseCharset string // --response-charset override (response body only)

	Stream bool // -S/--stream: skip body re-formatting, still colorize
}

// AutoStyle is the sentinel --style value that follows the terminal's own
// ANSI palette (basic 16 colors) instead of a specific named chroma style
// rendered in full 256-color mode.
const AutoStyle = "auto"

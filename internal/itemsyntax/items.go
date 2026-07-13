package itemsyntax

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"app.getnutshell/htee/internal/itemsyntax/nestedjson"
	"app.getnutshell/htee/internal/ordered"
)

// FileField is a file upload item: `field@path` or `field@path;type=mime`.
type FileField struct {
	Key      string
	Path     string
	MimeType string // explicit override from ";type=", empty if not given
}

// HeaderField is one `Name:Value` (or `Name;` empty-header) item.
type HeaderField struct {
	Name  string
	Value *string // nil means an explicitly empty header (remove/suppress a default)
}

// QueryParam is one `name==value` item.
type QueryParam struct {
	Name  string
	Value string
}

// MultipartField is one item, in original declaration order, that should
// become a multipart/form-data part when the request is sent as multipart.
type MultipartField struct {
	Key   string
	Value string
	File  *FileField // set for file-upload items; Value is used otherwise
}

// RequestItems is the result of dispatching all REQUEST_ITEM CLI arguments
// by separator. Mirrors httpie/cli/requestitems.py:RequestItems.
type RequestItems struct {
	Headers       []HeaderField
	Params        []QueryParam
	Data          any // JSON mode: *ordered.Map or *nestedjson.Array; form mode: *ordered.Map of strings
	Files         []FileField
	MultipartData []MultipartField
	IsJSON        bool
}

// FromArgs dispatches parsed KeyValueArgs into headers/query/data/files.
// When isJSON, `=`/`:=`/`=@`/`:=@` items are batched and interpreted jointly
// through the nested-JSON bracket-path interpreter so that e.g.
// `person[name]=bob person[age]:=30` merge into one object. Otherwise (form
// mode) each item's literal key is used flatly, with no bracket parsing, and
// `:=`/`:=@` values must be primitive.
//
// Mirrors httpie/cli/requestitems.py:RequestItems.from_args.
func FromArgs(items []KeyValueArg, isJSON bool) (*RequestItems, error) {
	ri := &RequestItems{IsJSON: isJSON}
	var nestedPairs []nestedjson.KeyValuePair
	flatData := ordered.NewMap()
	usedFlatData := false

	for _, item := range items {
		switch item.Sep {
		case SepHeader:
			ri.Headers = append(ri.Headers, HeaderField{Name: item.Key, Value: strPtr(item.Value)})

		case SepHeaderEmpty:
			if item.Value != "" {
				return nil, fmt.Errorf("%q is not a valid value: empty header %q must not have a value", item.Orig, item.Key)
			}
			ri.Headers = append(ri.Headers, HeaderField{Name: item.Key, Value: nil})

		case SepHeaderEmbed:
			content, err := readEmbeddedFile(item.Value)
			if err != nil {
				return nil, err
			}
			ri.Headers = append(ri.Headers, HeaderField{Name: item.Key, Value: strPtr(content)})

		case SepQueryParam:
			ri.Params = append(ri.Params, QueryParam{Name: item.Key, Value: item.Value})

		case SepQueryEmbedFile:
			content, err := readEmbeddedFile(item.Value)
			if err != nil {
				return nil, err
			}
			ri.Params = append(ri.Params, QueryParam{Name: item.Key, Value: content})

		case SepFileUpload:
			field, err := parseFileUpload(item.Key, item.Value)
			if err != nil {
				return nil, err
			}
			ri.Files = append(ri.Files, field)
			ri.MultipartData = append(ri.MultipartData, MultipartField{Key: item.Key, File: &field})

		case SepDataString:
			if isJSON {
				nestedPairs = append(nestedPairs, nestedjson.KeyValuePair{Key: item.Key, Value: item.Value})
			} else {
				flatData.Set(item.Key, item.Value)
				usedFlatData = true
			}
			ri.MultipartData = append(ri.MultipartData, MultipartField{Key: item.Key, Value: item.Value})

		case SepDataEmbedFile:
			content, err := readEmbeddedFile(item.Value)
			if err != nil {
				return nil, err
			}
			if isJSON {
				nestedPairs = append(nestedPairs, nestedjson.KeyValuePair{Key: item.Key, Value: content})
			} else {
				flatData.Set(item.Key, content)
				usedFlatData = true
			}
			ri.MultipartData = append(ri.MultipartData, MultipartField{Key: item.Key, Value: content})

		case SepDataRawJSON:
			val, err := ordered.DecodeJSON([]byte(item.Value))
			if err != nil {
				return nil, fmt.Errorf("%q is not valid JSON: %w", item.Orig, err)
			}
			if isJSON {
				nestedPairs = append(nestedPairs, nestedjson.KeyValuePair{Key: item.Key, Value: val})
			} else {
				prim, err := requirePrimitive(item.Orig, val)
				if err != nil {
					return nil, err
				}
				flatData.Set(item.Key, prim)
				usedFlatData = true
				ri.MultipartData = append(ri.MultipartData, MultipartField{Key: item.Key, Value: formatPrimitiveForMultipart(prim)})
			}

		case SepDataEmbedRawJSONFile:
			raw, err := os.ReadFile(item.Value)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", item.Value, err)
			}
			val, err := ordered.DecodeJSON(raw)
			if err != nil {
				return nil, fmt.Errorf("%s does not contain valid JSON: %w", item.Value, err)
			}
			if isJSON {
				nestedPairs = append(nestedPairs, nestedjson.KeyValuePair{Key: item.Key, Value: val})
			} else {
				prim, err := requirePrimitive(item.Orig, val)
				if err != nil {
					return nil, err
				}
				flatData.Set(item.Key, prim)
				usedFlatData = true
				ri.MultipartData = append(ri.MultipartData, MultipartField{Key: item.Key, Value: formatPrimitiveForMultipart(prim)})
			}

		default:
			return nil, fmt.Errorf("%q: unsupported separator %q", item.Orig, item.Sep)
		}
	}

	if isJSON && len(nestedPairs) > 0 {
		data, err := nestedjson.Interpret(nestedPairs)
		if err != nil {
			return nil, err
		}
		ri.Data = data
	} else if !isJSON && usedFlatData {
		ri.Data = flatData
	}

	return ri, nil
}

func strPtr(s string) *string { return &s }

// readEmbeddedFile reads a file's content for a `:@`/`==@`/`=@` embedded
// item, trimming a single trailing newline (mirrors Python's `.rstrip('\n')`
// which strips all trailing newlines - reproduced here as TrimRight for
// fidelity).
func readEmbeddedFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return strings.TrimRight(string(data), "\n"), nil
}

// parseFileUpload splits a file-upload item's value into its path and an
// optional explicit MIME type (`path;type=mime/type`).
func parseFileUpload(key, value string) (FileField, error) {
	path, mimeType, _ := strings.Cut(value, FileUploadTypeSuffix)
	if _, err := os.Stat(path); err != nil {
		return FileField{}, fmt.Errorf("%s: %w", path, err)
	}
	return FileField{Key: key, Path: path, MimeType: mimeType}, nil
}

// FormatPrimitive stringifies a primitive JSON value (string/number/
// bool/nil) for use as a form or multipart field value.
func FormatPrimitive(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}

func formatPrimitiveForMultipart(v any) string { return FormatPrimitive(v) }

// requirePrimitive enforces httpie's form/multipart-mode rule that `:=`/
// `:=@` values must be primitive JSON types (string/number/bool/null) since
// there's no way to represent a nested object/array as a single form field.
func requirePrimitive(orig string, val any) (any, error) {
	switch val.(type) {
	case string, float64, bool, nil:
		return val, nil
	default:
		return nil, fmt.Errorf("%q: cannot use complex JSON value types with --form/--multipart", orig)
	}
}

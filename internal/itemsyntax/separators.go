// Package itemsyntax implements httpie's REQUEST_ITEM grammar: tokenizing
// "key<sep>value" CLI arguments such as `name=bob`, `X-Foo:bar`, or
// `search==term` into typed key/value pairs.
package itemsyntax

// Separator is one of httpie's REQUEST_ITEM separators. Longer separators
// take precedence over shorter ones that start at the same position.
type Separator string

const (
	SepHeaderEmbed          Separator = ":@"  // Header value loaded from file content
	SepDataEmbedRawJSONFile Separator = ":=@" // Raw JSON field value loaded from a JSON file
	SepDataRawJSON          Separator = ":="  // Raw/typed JSON field
	SepHeader               Separator = ":"   // HTTP header
	SepHeaderEmpty          Separator = ";"   // Empty header (no value allowed)
	SepQueryEmbedFile       Separator = "==@" // Query param value loaded from file content
	SepQueryParam           Separator = "=="  // URL query parameter
	SepDataEmbedFile        Separator = "=@"  // JSON/form field value loaded from file content
	SepDataString           Separator = "="   // JSON string field / form field
	SepFileUpload           Separator = "@"   // File upload (optionally ";type=mime")
)

// AllItemSeparators is the full separator set used to tokenize a generic
// REQUEST_ITEM.
var AllItemSeparators = []Separator{
	SepHeaderEmbed,
	SepDataEmbedRawJSONFile,
	SepDataRawJSON,
	SepHeader,
	SepHeaderEmpty,
	SepQueryEmbedFile,
	SepQueryParam,
	SepDataEmbedFile,
	SepDataString,
	SepFileUpload,
}

// Precedence/behavior groups, mirroring httpie's cli/constants.py frozensets.

// GroupDataItems are separators whose presence implies "this request has a
// body", used to help infer POST vs GET when METHOD is omitted.
var GroupDataItems = map[Separator]bool{
	SepDataString:           true,
	SepDataRawJSON:          true,
	SepFileUpload:           true,
	SepDataEmbedFile:        true,
	SepDataEmbedRawJSONFile: true,
}

// FileUploadTypeSuffix separates a file upload path from its explicit MIME
// type override, e.g. `field@./report.bin;type=application/pdf`.
const FileUploadTypeSuffix = ";type="

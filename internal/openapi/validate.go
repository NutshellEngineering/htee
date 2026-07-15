package openapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

func init() {
	// SchemaError.Error() normally appends a full pretty-printed dump of
	// the offending schema and value after the human-readable reason,
	// which is far too noisy for a one-line-per-failure CLI report.
	openapi3.SchemaErrorDetailsDisabled = true
}

// Doc is a parsed, validated OpenAPI 3 document with a request router
// attached, ready to validate live requests/responses against it.
type Doc struct {
	raw    *openapi3.T
	router routers.Router
}

// validationOptions is shared by request and response validation. ht only
// checks that requests/responses conform to the spec's shape - it doesn't
// perform its own authentication - so AuthenticationFunc is set to the
// no-op provided by openapi3filter. Without it, ValidateRequest fails every
// single request against any operation with a security requirement (even
// one carrying valid credentials) with "missing AuthenticationFunc",
// because it has no callback to ask.
var validationOptions = &openapi3filter.Options{
	MultiError:         true,
	AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
}

// LoadDoc parses and validates the OpenAPI 3 document (YAML or JSON) at
// path and builds a router for matching live requests against it. Unlike
// Load, it rejects documents that fail full OpenAPI 3 schema validation -
// except for failures confined to the tags section (e.g. duplicate tag
// names), which are metadata problems that don't affect route matching or
// request/response schema validation, the only things this package actually
// uses the document for.
func LoadDoc(path string) (*Doc, error) {
	doc, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAPI spec: %w", err)
	}
	if err := validateDoc(doc); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("building OpenAPI router: %w", err)
	}
	return &Doc{raw: doc, router: router}, nil
}

// validateDoc runs full OpenAPI 3 document validation but discards any
// failures confined to the tags section.
func validateDoc(doc *openapi3.T) error {
	err := doc.Validate(context.Background(), openapi3.EnableMultiError())
	if err == nil {
		return nil
	}

	var me openapi3.MultiError
	if !errors.As(err, &me) {
		return err
	}

	var es openapi3.MultiError
	for _, e := range me {
		var sve *openapi3.SectionValidationError
		if errors.As(e, &sve) && sve.Section == "tags" {
			continue
		}
		es = append(es, e)
	}
	if len(es) == 0 {
		return nil
	}
	return es
}

// ValidationOutcome is the result of checking one request/response pair
// against a Doc.
type ValidationOutcome struct {
	// RouteFound is false when req's method+path (against the spec's
	// servers) has no matching operation in the document at all.
	RouteFound bool
	// RequestErrors and ResponseErrors are human-readable descriptions of
	// every validation failure found, empty when RouteFound is true and
	// the request/response fully conform to the spec.
	RequestErrors  []string
	ResponseErrors []string
}

// Validate checks req/reqBody and resp/respBody against d. If req's
// method+path isn't found in the spec, it returns RouteFound: false and no
// error lists - callers should treat that as "not covered by this spec",
// not as a validation failure.
func (d *Doc) Validate(req *http.Request, reqBody []byte, resp *http.Response, respBody []byte) ValidationOutcome {
	route, pathParams, err := d.router.FindRoute(req)
	if err != nil {
		return ValidationOutcome{RouteFound: false}
	}

	reqForValidation := req.Clone(req.Context())
	reqForValidation.Body = io.NopCloser(bytes.NewReader(reqBody))

	reqInput := &openapi3filter.RequestValidationInput{
		Request:    reqForValidation,
		PathParams: pathParams,
		Route:      route,
		Options:    validationOptions,
	}

	out := ValidationOutcome{RouteFound: true}
	if err := openapi3filter.ValidateRequest(context.Background(), reqInput); err != nil {
		out.RequestErrors = flattenValidationError(err)
	}

	respInput := (&openapi3filter.ResponseValidationInput{
		RequestValidationInput: reqInput,
		Status:                 resp.StatusCode,
		Header:                 resp.Header,
		Options:                validationOptions,
	}).SetBodyBytes(respBody)

	if err := openapi3filter.ValidateResponse(context.Background(), respInput); err != nil {
		out.ResponseErrors = flattenValidationError(err)
	}

	return out
}

// flattenValidationError expands an openapi3.MultiError (produced when
// Options.MultiError is set) into one string per failure, or wraps a
// single error as a one-element slice.
func flattenValidationError(err error) []string {
	var me openapi3.MultiError
	if errors.As(err, &me) {
		errs := make([]string, 0, len(me))
		for _, e := range me {
			errs = append(errs, e.Error())
		}
		return errs
	}
	return []string{err.Error()}
}

package output

import (
	"fmt"
	"io"

	"app.getnutshell/htee/internal/openapi"
	"app.getnutshell/htee/internal/theme"
)

// RenderValidation writes a short note about how v's request/response
// pair fared against the project's OpenAPI spec: a warning if the request
// wasn't found in the spec at all, a single success line if it was found
// and fully conforms, or a list of every validation failure otherwise.
func RenderValidation(w io.Writer, v openapi.ValidationOutcome, colorsEnabled bool) {
	t := theme.New(theme.NewRenderer(w, colorsEnabled))

	if !v.RouteFound {
		_, err := fmt.Fprintln(w, t.Warning.Render("openapi: request not found in the OpenAPI spec; response not validated"))
		if err != nil {
			panic(err)
		}
		return
	}

	if len(v.RequestErrors) == 0 && len(v.ResponseErrors) == 0 {
		_, err := fmt.Fprintln(w, t.Success.Render("openapi: response is valid"))
		if err != nil {
			panic(err)
		}
		return
	}

	_, err := fmt.Fprintln(w, t.Failure.Render("openapi: response failed validation:"))
	if err != nil {
		panic(err)
	}
	for _, e := range v.RequestErrors {
		_, err := fmt.Fprintln(w, t.Failure.Render("  - request: "+e))
		if err != nil {
			panic(err)
		}
	}
	for _, e := range v.ResponseErrors {
		_, err := fmt.Fprintln(w, t.Failure.Render("  - response: "+e))
		if err != nil {
			panic(err)
		}
	}
}

// RenderValidationEnabled prints a heads-up, before the request is sent,
// that this run will check requests/responses against the project's
// configured OpenAPI spec.
func RenderValidationEnabled(w io.Writer, colorsEnabled bool) {
	t := theme.New(theme.NewRenderer(w, colorsEnabled))
	_, err := fmt.Fprintln(w, t.Success.Render("openapi: validating request and response against the configured OpenAPI spec"))
	if err != nil {
		panic(err)
	}
}

// RenderOpenAPILoadError reports that the project's configured OpenAPI spec
// could not be loaded, so no request/response validation happened this run.
func RenderOpenAPILoadError(w io.Writer, err error, colorsEnabled bool) {
	t := theme.New(theme.NewRenderer(w, colorsEnabled))
	_, fe := fmt.Fprintln(w, t.Failure.Render("openapi: "+err.Error()))
	if fe != nil {
		panic(fe)
	}
}

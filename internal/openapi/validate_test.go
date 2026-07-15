package openapi

import (
	"net/http"
	"strings"
	"testing"
)

const specForValidation = `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: http://localhost:8080
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [name, age]
                properties:
                  name:
                    type: string
                  age:
                    type: integer
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        '201':
          description: Created
`

func mustLoadDoc(t *testing.T, spec string) *Doc {
	t.Helper()
	doc, err := LoadDoc(writeSpec(t, spec))
	if err != nil {
		t.Fatalf("LoadDoc: %v", err)
	}
	return doc
}

func newReq(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

func jsonResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestValidateMatchedAndValid(t *testing.T) {
	doc := mustLoadDoc(t, specForValidation)

	req := newReq(t, "GET", "http://localhost:8080/pets")
	resp := jsonResponse(200)

	out := doc.Validate(req, nil, resp, []byte(`{"name":"Fido","age":3}`))
	if !out.RouteFound {
		t.Fatalf("RouteFound = false, want true")
	}
	if len(out.RequestErrors) != 0 {
		t.Errorf("RequestErrors = %v, want empty", out.RequestErrors)
	}
	if len(out.ResponseErrors) != 0 {
		t.Errorf("ResponseErrors = %v, want empty", out.ResponseErrors)
	}
}

func TestValidateResponseReportsAllFailures(t *testing.T) {
	doc := mustLoadDoc(t, specForValidation)

	req := newReq(t, "GET", "http://localhost:8080/pets")
	resp := jsonResponse(200)

	// Missing required "name" and wrong type for "age" - two distinct
	// schema failures that MultiError should surface together.
	out := doc.Validate(req, nil, resp, []byte(`{"age":"three"}`))
	if !out.RouteFound {
		t.Fatalf("RouteFound = false, want true")
	}
	if len(out.ResponseErrors) < 2 {
		t.Fatalf("ResponseErrors = %v, want at least 2 failures", out.ResponseErrors)
	}
}

func TestValidateRouteNotFound(t *testing.T) {
	doc := mustLoadDoc(t, specForValidation)

	req := newReq(t, "GET", "http://localhost:8080/unknown")
	resp := jsonResponse(200)

	out := doc.Validate(req, nil, resp, []byte(`{}`))
	if out.RouteFound {
		t.Fatalf("RouteFound = true, want false for an unknown path")
	}
	if len(out.RequestErrors) != 0 || len(out.ResponseErrors) != 0 {
		t.Fatalf("expected no error lists when RouteFound is false, got req=%v resp=%v", out.RequestErrors, out.ResponseErrors)
	}
}

func TestValidateRequestBodyFailure(t *testing.T) {
	doc := mustLoadDoc(t, specForValidation)

	req := newReq(t, "POST", "http://localhost:8080/pets")
	resp := &http.Response{StatusCode: 201, Header: http.Header{}}

	out := doc.Validate(req, []byte(`{}`), resp, nil)
	if !out.RouteFound {
		t.Fatalf("RouteFound = false, want true")
	}
	if len(out.RequestErrors) == 0 {
		t.Fatalf("expected a request validation error for a missing required field")
	}
	if !strings.Contains(out.RequestErrors[0], "name") {
		t.Errorf("RequestErrors[0] = %q, expected it to mention the missing field", out.RequestErrors[0])
	}
}

func TestLoadDocRejectsInvalidSpec(t *testing.T) {
	path := writeSpec(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
paths:
  /broken:
    get:
      responses: {}
`)
	if _, err := LoadDoc(path); err == nil {
		t.Fatalf("expected LoadDoc to reject a spec with no responses defined")
	}
}

func TestValidateIgnoresSecurityRequirements(t *testing.T) {
	// A spec with a security requirement but no way for ht to actually
	// authenticate on the caller's behalf shouldn't fail every request over
	// it - ht only checks request/response shape, not credentials.
	doc := mustLoadDoc(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: http://localhost:8080
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - ApiKeyAuth: []
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
`)

	req := newReq(t, "GET", "http://localhost:8080/pets")
	out := doc.Validate(req, nil, jsonResponse(200), nil)
	if !out.RouteFound {
		t.Fatalf("RouteFound = false, want true")
	}
	if len(out.RequestErrors) != 0 {
		t.Fatalf("RequestErrors = %v, want empty (security requirements shouldn't be enforced)", out.RequestErrors)
	}
}

func TestLoadDocToleratesDuplicateTags(t *testing.T) {
	// Duplicate tag names are a metadata problem that doesn't affect route
	// matching or schema validation, so LoadDoc should accept the doc
	// rather than rejecting the whole spec over it.
	path := writeSpec(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: http://localhost:8080
tags:
  - name: Pets
  - name: Pets
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
`)
	doc, err := LoadDoc(path)
	if err != nil {
		t.Fatalf("LoadDoc: %v", err)
	}
	req := newReq(t, "GET", "http://localhost:8080/pets")
	out := doc.Validate(req, nil, jsonResponse(200), nil)
	if !out.RouteFound {
		t.Fatalf("RouteFound = false, want true")
	}
}

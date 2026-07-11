package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsRedirectStatus(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		if !isRedirectStatus(code) {
			t.Errorf("isRedirectStatus(%d) = false, want true", code)
		}
	}
	for _, code := range []int{200, 204, 404, 500} {
		if isRedirectStatus(code) {
			t.Errorf("isRedirectStatus(%d) = true, want false", code)
		}
	}
}

func newTestResponse(t *testing.T, code int, location string) *http.Response {
	t.Helper()
	h := http.Header{}
	if location != "" {
		h.Set("Location", location)
	}
	return &http.Response{StatusCode: code, Header: h}
}

func TestNextRequestRewritesPOSTto302AsBodylessGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	prev, _ := http.NewRequest(http.MethodPost, srv.URL+"/a", nil)
	prev.Header.Set("Content-Type", "application/json")
	prev.Header.Set("Authorization", "Bearer secret")
	order := []string{"Content-Type", "Authorization"}
	body := []byte(`{"x":1}`)

	next, nextBody, nextOrder, err := nextRequest(prev, body, order, newTestResponse(t, 302, "/b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Method != http.MethodGet {
		t.Fatalf("method = %q, want GET", next.Method)
	}
	if len(nextBody) != 0 {
		t.Fatalf("body = %q, want empty", nextBody)
	}
	if next.Header.Get("Content-Type") != "" {
		t.Fatalf("Content-Type = %q, want stripped", next.Header.Get("Content-Type"))
	}
	if next.URL.Path != "/b" {
		t.Fatalf("path = %q, want /b", next.URL.Path)
	}
	for _, name := range nextOrder {
		if name == "Content-Type" {
			t.Fatalf("header order still lists Content-Type: %v", nextOrder)
		}
	}
}

func TestNextRequestPreserves307MethodAndBody(t *testing.T) {
	prev, _ := http.NewRequest(http.MethodPut, "http://example.com/a", nil)
	body := []byte("payload")
	order := []string{"Content-Type"}

	next, nextBody, nextOrder, err := nextRequest(prev, body, order, newTestResponse(t, 307, "/b"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Method != http.MethodPut {
		t.Fatalf("method = %q, want PUT", next.Method)
	}
	if string(nextBody) != "payload" {
		t.Fatalf("body = %q, want preserved", nextBody)
	}
	if len(nextOrder) != 1 || nextOrder[0] != "Content-Type" {
		t.Fatalf("header order = %v, want unchanged", nextOrder)
	}
}

func TestNextRequestStripsAuthorizationCrossOrigin(t *testing.T) {
	prev, _ := http.NewRequest(http.MethodGet, "http://a.example.com/x", nil)
	prev.Header.Set("Authorization", "Bearer secret")
	order := []string{"Authorization"}

	next, _, nextOrder, err := nextRequest(prev, nil, order, newTestResponse(t, 302, "http://b.example.com/y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Header.Get("Authorization") != "" {
		t.Fatal("Authorization header leaked cross-origin")
	}
	for _, name := range nextOrder {
		if name == "Authorization" {
			t.Fatalf("header order still lists Authorization: %v", nextOrder)
		}
	}
}

func TestNextRequestKeepsAuthorizationSameOrigin(t *testing.T) {
	prev, _ := http.NewRequest(http.MethodGet, "http://a.example.com/x", nil)
	prev.Header.Set("Authorization", "Bearer secret")
	order := []string{"Authorization"}

	next, _, _, err := nextRequest(prev, nil, order, newTestResponse(t, 302, "http://a.example.com/y"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Header.Get("Authorization") != "Bearer secret" {
		t.Fatal("Authorization header dropped same-origin")
	}
}

func TestNextRequestMissingLocationErrors(t *testing.T) {
	prev, _ := http.NewRequest(http.MethodGet, "http://a.example.com/x", nil)
	if _, _, _, err := nextRequest(prev, nil, nil, newTestResponse(t, 302, "")); err == nil {
		t.Fatal("expected error for missing Location header")
	}
}

func TestSendFollowingSingleHopWhenFollowDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/b")
		w.WriteHeader(302)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/a", nil)
	c := New()
	chain, err := c.SendFollowing(req, nil, nil, FollowOptions{Follow: false, MaxRedirects: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain.Hops) != 1 {
		t.Fatalf("hops = %d, want 1 (redirect not followed)", len(chain.Hops))
	}
	if chain.Hops[0].Result.Response.StatusCode != 302 {
		t.Fatalf("status = %d, want 302", chain.Hops[0].Result.Response.StatusCode)
	}
}

func TestSendFollowingChasesRedirectChain(t *testing.T) {
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		switch r.URL.Path {
		case "/a":
			w.Header().Set("Location", "/b")
			w.WriteHeader(302)
		case "/b":
			w.Write([]byte("final"))
		}
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/a", nil)
	c := New()
	chain, err := c.SendFollowing(req, nil, nil, FollowOptions{Follow: true, MaxRedirects: 30})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain.Hops) != 2 {
		t.Fatalf("hops = %d, want 2", len(chain.Hops))
	}
	final := chain.Final()
	if string(final.Result.Body) != "final" {
		t.Fatalf("final body = %q, want %q", final.Result.Body, "final")
	}
	if hits[0] != "/a" || hits[1] != "/b" {
		t.Fatalf("hit order = %v", hits)
	}
}

func TestSendFollowingExceedsMaxRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/loop")
		w.WriteHeader(302)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/loop", nil)
	c := New()
	_, err := c.SendFollowing(req, nil, nil, FollowOptions{Follow: true, MaxRedirects: 3})
	if err == nil {
		t.Fatal("expected error for exceeding --max-redirects")
	}
}

func TestSendFollowingZeroMaxRedirectsIsUnlimited(t *testing.T) {
	count := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if count >= 5 {
			w.Write([]byte("done"))
			return
		}
		w.Header().Set("Location", srv.URL+"/next")
		w.WriteHeader(302)
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/start", nil)
	c := New()
	chain, err := c.SendFollowing(req, nil, nil, FollowOptions{Follow: true, MaxRedirects: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain.Hops) != 5 {
		t.Fatalf("hops = %d, want 5", len(chain.Hops))
	}
}

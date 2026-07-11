package transport

import (
	"net/http"
	"testing"
)

func TestProxyFuncEmptyReturnsNilFunc(t *testing.T) {
	fn, err := ProxyFunc(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != nil {
		t.Fatal("expected nil Proxy func when no --proxy given")
	}
}

func TestProxyFuncMatchesScheme(t *testing.T) {
	fn, err := ProxyFunc([]string{"http:http://proxy.local:8080", "https:http://proxy.local:8443"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	u, err := fn(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil || u.Host != "proxy.local:8443" {
		t.Fatalf("proxy = %v, want proxy.local:8443 for https", u)
	}
}

func TestProxyFuncFallsBackToAll(t *testing.T) {
	fn, err := ProxyFunc([]string{"all:http://catchall.local:9000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	u, err := fn(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil || u.Host != "catchall.local:9000" {
		t.Fatalf("proxy = %v, want catchall.local:9000", u)
	}
}

func TestProxyFuncInvalidEntryErrors(t *testing.T) {
	if _, err := ProxyFunc([]string{"not-a-valid-entry"}); err == nil {
		t.Fatal("expected error for --proxy entry missing protocol:URL shape")
	}
}

package tools

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPGetAllowedHost(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Hostname() != "example.com" {
				t.Fatalf("unexpected host %q", req.URL.Hostname())
			}
			return &http.Response{
				StatusCode: 200,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("hello from fake transport")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	tool := NewHTTPGet(
		WithHTTPClient(client),
		WithAllowedHosts("example.com"),
	)

	args, _ := json.Marshal(map[string]string{"url": "https://example.com/hello"})
	got, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(got, "200 OK") || !strings.Contains(got, "hello from fake transport") {
		t.Fatalf("Execute() = %q", got)
	}
}

func TestHTTPGetRejectsDisallowedHost(t *testing.T) {
	tool := NewHTTPGet(WithAllowedHosts("example.com"))
	args, _ := json.Marshal(map[string]string{"url": "https://not-allowed.example/path"})

	_, err := tool.Execute(args)
	if err == nil {
		t.Fatal("Execute() expected host allowlist error")
	}
}

func TestHTTPGetRejectsHTTP(t *testing.T) {
	tool := NewHTTPGet(WithAllowedHosts("example.com"))
	args, _ := json.Marshal(map[string]string{"url": "http://example.com"})

	_, err := tool.Execute(args)
	if err == nil {
		t.Fatal("Execute() expected https-only error")
	}
}

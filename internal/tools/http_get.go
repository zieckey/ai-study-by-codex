package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zieckey/ai-study-by-codex/internal/agent"
)

type HTTPGetOption func(*httpGetConfig)

type httpGetConfig struct {
	client       *http.Client
	allowedHosts map[string]bool
	maxBytes     int64
}

func WithHTTPClient(client *http.Client) HTTPGetOption {
	return func(cfg *httpGetConfig) {
		if client != nil {
			cfg.client = client
		}
	}
}

func WithAllowedHosts(hosts ...string) HTTPGetOption {
	return func(cfg *httpGetConfig) {
		cfg.allowedHosts = map[string]bool{}
		for _, host := range hosts {
			host = strings.ToLower(strings.TrimSpace(host))
			if host != "" {
				cfg.allowedHosts[host] = true
			}
		}
	}
}

// NewHTTPGet returns a restricted HTTP GET tool.
// It only allows HTTPS URLs whose host is in the allowlist.
func NewHTTPGet(opts ...HTTPGetOption) agent.Tool {
	cfg := httpGetConfig{
		client:       &http.Client{Timeout: 5 * time.Second},
		allowedHosts: map[string]bool{"example.com": true, "httpbin.org": true},
		maxBytes:     4096,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return agent.Tool{
		Name:        "http_get",
		Description: "Fetch a HTTPS URL from a small allowlist. Default allowed hosts: example.com, httpbin.org.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`),
		Execute: func(args json.RawMessage) (string, error) {
			var input struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(args, &input); err != nil {
				return "", fmt.Errorf("invalid http_get args: %w", err)
			}

			target := strings.TrimSpace(input.URL)
			parsed, err := url.Parse(target)
			if err != nil {
				return "", fmt.Errorf("invalid url: %w", err)
			}
			if parsed.Scheme != "https" {
				return "", fmt.Errorf("only https URLs are allowed")
			}

			host := strings.ToLower(parsed.Hostname())
			if !cfg.allowedHosts[host] {
				return "", fmt.Errorf("host %q is not allowed", host)
			}

			ctx, cancel := context.WithTimeout(context.Background(), cfg.client.Timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
			if err != nil {
				return "", err
			}
			req.Header.Set("User-Agent", "ai-study-by-codex-agent/0.1")

			resp, err := cfg.client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(io.LimitReader(resp.Body, cfg.maxBytes+1))
			if err != nil {
				return "", err
			}
			truncated := int64(len(body)) > cfg.maxBytes
			if truncated {
				body = body[:cfg.maxBytes]
			}

			result := fmt.Sprintf("status=%s\nbody=%s", resp.Status, strings.TrimSpace(string(body)))
			if truncated {
				result += "\n[truncated]"
			}
			return result, nil
		},
	}
}

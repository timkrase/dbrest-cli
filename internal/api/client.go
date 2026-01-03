package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Clienter is the API client interface used by the CLI for testability.
type Clienter interface {
	Get(ctx context.Context, path string, params url.Values) ([]byte, error)
	URL(path string, params url.Values) (string, error)
}

// Config defines HTTP client configuration for the DB transport API.
type Config struct {
	BaseURL   string
	Timeout   time.Duration
	UserAgent string
}

// Client wraps a base URL and an HTTP client for GET requests.
type Client struct {
	baseURL   *url.URL
	http      *http.Client
	userAgent string
}

// NewClient creates a new API client from config.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("base URL is required")
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Client{
		baseURL: base,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		userAgent: cfg.UserAgent,
	}, nil
}

// URL returns the fully qualified URL for the given path and parameters.
func (c *Client) URL(path string, params url.Values) (string, error) {
	return buildURL(c.baseURL, path, params)
}

// Get issues a GET request against the API and returns the response body.
func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	if c == nil || c.baseURL == nil {
		return nil, errors.New("client is not configured")
	}
	urlStr, err := buildURL(c.baseURL, path, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, HTTPError{Status: resp.StatusCode, Body: body}
	}
	return body, nil
}

func buildURL(base *url.URL, path string, params url.Values) (string, error) {
	if base == nil {
		return "", errors.New("base URL is nil")
	}
	cleanPath := strings.TrimLeft(path, "/")
	basePath := strings.TrimRight(base.Path, "/")
	mergedPath := "/" + cleanPath
	if basePath != "" {
		mergedPath = basePath + "/" + cleanPath
	}
	resolved := *base
	resolved.Path = mergedPath
	if len(params) > 0 {
		resolved.RawQuery = params.Encode()
	} else {
		resolved.RawQuery = ""
	}
	return resolved.String(), nil
}

// HTTPError wraps non-2xx responses with a summarized body for diagnostics.
type HTTPError struct {
	Status int
	Body   []byte
}

func (e HTTPError) Error() string {
	msg := strings.TrimSpace(string(e.Body))
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	msg = summarize(msg, 200)
	return fmt.Sprintf("request failed: %d %s", e.Status, msg)
}

func summarize(msg string, max int) string {
	if len(msg) <= max {
		return msg
	}
	return msg[:max] + "..."
}

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientGetBuildsURL(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL + "/api"})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	params := url.Values{}
	params.Set("query", "berlin")
	params.Set("results", "5")

	_, err = client.Get(context.Background(), "/locations", params)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if gotPath != "/api/locations" {
		t.Fatalf("expected path /api/locations, got %q", gotPath)
	}
	if gotQuery.Get("query") != "berlin" {
		t.Fatalf("expected query=berlin, got %q", gotQuery.Get("query"))
	}
	if gotQuery.Get("results") != "5" {
		t.Fatalf("expected results=5, got %q", gotQuery.Get("results"))
	}
}

func TestClientGetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	_, err = client.Get(context.Background(), "/missing", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if _, ok := err.(HTTPError); !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
}

package cli

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	"github.com/timkrase/deutsche-bahn-skill/internal/api"
)

type fakeClient struct {
	lastPath   string
	lastParams url.Values
	response   []byte
}

func (f *fakeClient) Get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	f.lastPath = path
	f.lastParams = params
	return f.response, nil
}

func (f *fakeClient) URL(path string, params url.Values) (string, error) {
	return "http://example.test" + path + "?" + params.Encode(), nil
}

func TestRunLocationsJSON(t *testing.T) {
	client := &fakeClient{response: []byte(`[]`)}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	exit := Run([]string{"--json", "locations", "--query", "berlin"}, Runner{
		Out: out,
		Err: errOut,
		Getenv: func(string) string {
			return ""
		},
		NewClient: func(cfg api.Config) (api.Clienter, error) {
			return client, nil
		},
		Version: "dev",
	})

	if exit != exitOK {
		t.Fatalf("expected exit 0, got %d", exit)
	}
	if client.lastPath != "/locations" {
		t.Fatalf("expected path /locations, got %q", client.lastPath)
	}
	if client.lastParams.Get("query") != "berlin" {
		t.Fatalf("expected query=berlin, got %q", client.lastParams.Get("query"))
	}
	if out.String() != "[]\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stderr output, got: %q", errOut.String())
	}
}

func TestRunMissingCommand(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	exit := Run([]string{}, Runner{
		Out: out,
		Err: errOut,
		Getenv: func(string) string {
			return ""
		},
		NewClient: func(cfg api.Config) (api.Clienter, error) {
			return &fakeClient{response: []byte(`[]`)}, nil
		},
		Version: "dev",
	})

	if exit != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, exit)
	}
	if errOut.Len() == 0 {
		t.Fatal("expected usage output on stderr")
	}
}

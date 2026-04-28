// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

package transform

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelocantos/mcpsafe/keychain"
)

func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.star")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingTransform(t *testing.T) {
	path := writeScript(t, `x = 1`)
	_, err := Load(path, keychain.NewStore())
	if err == nil {
		t.Fatal("expected error for script without transform function")
	}
}

func TestApplyHostRewrite(t *testing.T) {
	path := writeScript(t, `
def transform(req):
    req["host"] = "rewritten.example.com"
    req["query"]["added"] = "yes"
    return req
`)
	script, err := Load(path, keychain.NewStore())
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("GET", "http://localhost:8080/api/cmd?foo=bar", nil)
	req.Host = "localhost:8080"

	if err := script.Apply(req); err != nil {
		t.Fatal(err)
	}

	if req.Host != "rewritten.example.com" {
		t.Errorf("host = %q, want %q", req.Host, "rewritten.example.com")
	}
	if req.URL.Host != "rewritten.example.com" {
		t.Errorf("URL.Host = %q, want %q", req.URL.Host, "rewritten.example.com")
	}
	if got := req.URL.Query().Get("added"); got != "yes" {
		t.Errorf("query param 'added' = %q, want %q", got, "yes")
	}
	if got := req.URL.Query().Get("foo"); got != "bar" {
		t.Errorf("query param 'foo' = %q, want %q", got, "bar")
	}
}

func TestApplyPreservesMethod(t *testing.T) {
	path := writeScript(t, `
def transform(req):
    return req
`)
	script, err := Load(path, keychain.NewStore())
	if err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest("POST", "http://localhost/api", nil)
	if err := script.Apply(req); err != nil {
		t.Fatal(err)
	}
	if req.Method != "POST" {
		t.Errorf("method = %q, want POST", req.Method)
	}
}

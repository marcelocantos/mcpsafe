// Copyright 2026 Marcelo Cantos
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/marcelocantos/mcpsafe/keychain"
	"github.com/marcelocantos/mcpsafe/transform"
)

var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "listen address (host:port, use port 0 for auto)")
	scriptPath := flag.String("config", "", "path to Starlark transform script (required)")
	backend := flag.String("backend", "", "default backend scheme://host (overridden by script if it sets host)")
	showVersion := flag.Bool("version", false, "print version and exit")
	showHelp := flag.Bool("help-agent", false, "print agent-oriented help and exit")

	flag.Parse()

	if *showVersion {
		fmt.Println("mcpsafe", version)
		os.Exit(0)
	}

	if *showHelp {
		printAgentHelp()
		os.Exit(0)
	}

	if *scriptPath == "" {
		fmt.Fprintln(os.Stderr, "error: -config is required")
		flag.Usage()
		os.Exit(1)
	}

	store := keychain.NewStore()
	script, err := transform.Load(*scriptPath, store)
	if err != nil {
		log.Fatalf("Failed to load transform script: %v", err)
	}

	backendURL, err := parseBackend(*backend)
	if err != nil {
		log.Fatalf("Invalid -backend: %v", err)
	}

	handler := &proxy{script: script, backend: backendURL}

	listener, err := newListener(*addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	fmt.Fprintf(os.Stderr, "mcpsafe listening on http://%s\n", listener.Addr())

	log.Fatal(http.Serve(listener, handler))
}

func parseBackend(s string) (*url.URL, error) {
	if s == "" {
		return nil, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("backend must include scheme and host (e.g. https://example.com)")
	}
	return u, nil
}

type proxy struct {
	script  *transform.Script
	backend *url.URL
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set defaults from backend if provided.
	if p.backend != nil {
		if r.URL.Scheme == "" {
			r.URL.Scheme = p.backend.Scheme
		}
		if r.URL.Host == "" || r.URL.Host == r.Host {
			r.URL.Host = p.backend.Host
			r.Host = p.backend.Host
		}
	}

	if err := p.script.Apply(r); err != nil {
		http.Error(w, fmt.Sprintf("transform error: %v", err), http.StatusBadGateway)
		return
	}

	// Ensure we have a valid target.
	if r.URL.Scheme == "" {
		r.URL.Scheme = "https"
	}
	if r.URL.Host == "" {
		http.Error(w, "no backend host configured", http.StatusBadGateway)
		return
	}

	// Forward the request.
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("request error: %v", err), http.StatusInternalServerError)
		return
	}
	outReq.Header = r.Header.Clone()
	outReq.Header.Del("Host")

	resp, err := http.DefaultClient.Do(outReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func printAgentHelp() {
	fmt.Print(`mcpsafe - Keychain-backed reverse proxy for MCP servers

mcpsafe sits between an MCP server and its backend API, injecting
credentials from the macOS Keychain so that the MCP server never
sees secrets directly.

USAGE:
  mcpsafe -config <script.star> [-backend <url>] [-addr <host:port>]

FLAGS:
  -config    Path to a Starlark (.star) transform script (required).
             The script must define: def transform(req) -> req
  -backend   Default backend URL (scheme://host). The transform script
             can override the host.
  -addr      Listen address. Default: 127.0.0.1:0 (random port).

STARLARK BUILTINS:
  keychain(service, account="")
    Reads a password from the macOS Keychain. If account is omitted,
    it defaults to the service name. Results are cached in memory.

EXAMPLE SCRIPT (fogbugz.star):
  def transform(req):
      req["query"]["token"] = keychain("fogbugz-token")
      req["host"] = "yourcompany.fogbugz.com"
      return req

EXAMPLE SETUP:
  # Store your token in Keychain
  security add-generic-password -s fogbugz-token -a fogbugz-token -w "YOUR_TOKEN"

  # Run mcpsafe
  mcpsafe -config fogbugz.star -backend https://yourcompany.fogbugz.com

  # Point fogbugz-mcp at the proxy
  claude mcp add fogbugz -- npx fogbugz-mcp http://127.0.0.1:<port>
`)
}

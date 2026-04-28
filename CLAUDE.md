# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

mcpsafe is a Go reverse proxy that securely injects credentials into
MCP server traffic. It sits between an MCP server (e.g. `fogbugz-mcp`)
and the real backend API, so the MCP server never sees secrets.

### How it works

1. MCP server sends requests to mcpsafe on localhost (thinking it's the backend)
2. mcpsafe reads credentials from macOS Keychain (`security find-generic-password`)
3. Caches credentials in memory for the session
4. Rewrites the host to the real backend URL and injects auth (e.g. `&token=XXX`)
5. Forwards the request and returns the response

### Configuration

Starlark scripts define per-backend transform logic:

```python
def transform(req):
    req["query"]["token"] = keychain("fogbugz-token")
    req["host"] = "yourcompany.fogbugz.com"
    return req
```

The `req` dict has keys: `host`, `scheme`, `path`, `query` (dict),
`headers` (dict), `method`. The `keychain(service, account="")` builtin
reads from macOS Keychain.

### First use case

FogBugz API — the HMS project uses FogBugz for issue tracking, accessed
via `akari2600/fogbugz-mcp`. FogBugz auth appends `&token=XXX` to query
params.

## Build

```bash
go build ./...
go test ./...
```

## License

Apache 2.0. Copyright Marcelo Cantos.

## Delivery

Merged to master.

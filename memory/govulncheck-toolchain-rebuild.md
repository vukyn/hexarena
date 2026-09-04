---
name: govulncheck-toolchain-rebuild
description: "Go analysis tools (govulncheck, staticcheck) must be built with a toolchain >= the repo's go directive; plain `go install` does not upgrade it"
metadata:
  type: feedback
---

**This applies to every Go analysis tool, not just govulncheck** — `staticcheck` hit it identically on 2026-08-25 (a go1.25 build refused this module and reported export-data errors for every stdlib package, which reads like a broken repo). A tool binary older than the repo's `go` directive dies before it analyses anything:

```
package requires newer Go version go1.27 (application built with go1.26)
```

**Plain `go install golang.org/x/vuln/cmd/govulncheck@latest` does NOT fix it** — `go install` builds with the *current* toolchain, and `GOTOOLCHAIN=auto` only upgrades when the module being built demands it, which govulncheck's own go.mod does not. The reinstalled binary stays on the old Go.

Fix when the host toolchain is behind — force it at install time:

```bash
GOTOOLCHAIN=go1.27.0 go install golang.org/x/vuln/cmd/govulncheck@latest
```

**Current state (2026-08-24): the host Go is 1.27.0 and `govulncheck` is v1.7.0 built with go1.27.0, so the prefix is NOT needed** — plain `go install ...@latest` is correct while the host is at or above every repo's directive. Verified: `govulncheck ./...` in hexarena (go 1.27.0) → `No vulnerabilities found`.

**Why it matters:** a scan that dies on `loading packages` looks like a broken repo, not a stale tool, so the finding gets mis-triaged. A binary built with the NEWEST toolchain reads older module directives fine (back-checked on speedtest, go 1.26.4), so the rule is one-directional: build the tool with the highest Go any repo uses.

**How to apply:** when a Go analysis tool suddenly fails on *every* package, suspect the tool's own build before the code. `go version -m $(which <tool>)` names the toolchain it was built with. Only reach for the `GOTOOLCHAIN=` prefix when a repo's `go` directive is ahead of the installed toolchain — i.e. right after bumping a repo past the host Go, before the host itself is upgraded. Once the host is upgraded, drop the prefix. Related: [[kuery-security-fixes-2026-07]], [[hexarena-core-design]].

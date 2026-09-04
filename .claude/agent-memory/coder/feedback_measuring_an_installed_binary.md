---
name: measuring-an-installed-binary
description: hexarena — the "go install pkg@version" build-info path IS measurable locally with a file GOPROXY (no network, no tag); the module cache is consulted AHEAD of GOPROXY, so reusing a published version silently measures the OLD code
metadata:
  type: feedback
---

`go install pkg@version` and `go build` give a binary **different** build info,
and the difference is not observable from `go test` — a test binary is neither.
It *is* observable in about two minutes with a **file-based proxy**, which needs
no network, no tag and no push:

```
V=v0.0.0-20260904055522-aaaabbbbcccc      # or a plain tag, v0.9.9
P=$TMP/proxy/github.com/vukyn/hexarena/@v
cp go.mod $P/$V.mod
echo '{"Version":"'$V'","Time":"2026-09-04T05:55:22Z"}' > $P/$V.info
# zip the worktree under the prefix  github.com/vukyn/hexarena@$V/  (skip .git, _test.go)
GOBIN=$TMP/bin GOFLAGS=-mod=mod GOPROXY=file://$TMP/proxy GOSUMDB=off \
  go install github.com/vukyn/hexarena/cmd/hexarena-host@$V
```

**⚠️ Pick a version string the module cache has never held.** The cache is
consulted **ahead of GOPROXY**, so reusing a version the user really installed
resolves out of `$(go env GOMODCACHE)/cache/download/...` and installs the
**old** code — the run looks like a successful measurement and measures nothing.
My first attempt did exactly that (and, by luck, produced a perfect
before-picture: the published binary answering `build devel`).

**Clean up afterwards**, both `cache/download/.../@v/$V.*` and the extracted
`$GOMODCACHE/github.com/vukyn/hexarena@$V/` (needs `chmod -R u+w` first) — a
fake version left in the cache can shadow a later real resolution. Always
`GOBIN=` a scratch dir so the user's installed binaries are not overwritten.

**What it measured, so the shape is known:** `go version -m` on the installed
binary lists **one `mod` line and no `vcs.` setting at all** — the toolchain
stamps `vcs.revision`/`vcs.modified` only when building from a local checkout.
So `debug.BuildInfo.Main.Version` is the *only* thing an installed binary knows
about itself, and it reads `(devel)` — with parentheses — for a `go build`
outside a checkout.

**Why:** the difference between a report that says "unmeasurable, only
observable by installing" and one that says "measured, here is the recipe". Do
not add it to `make check`: it is a 6.5 MB zip plus an install per run against a
gate already at 3m15s. Put the recipe in the test's own doc comment instead —
that is where this repository keeps its measurements.

Related: [[unstable-values-in-a-record]] for the other half of this — why
`buildString()` is unpinnable by the suite at all.

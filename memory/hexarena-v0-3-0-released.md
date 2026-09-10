---
name: hexarena-v0-3-0-released
description: hexarena v0.3.0 tag pushed (63a5e93, digest 9c086a9cf68b) — the first release a player can use without a checkout; verified by installing from the proxy and running in an empty directory
metadata:
  type: project
---

hexarena **`v0.3.0`** tagged and pushed 2026-09-10, at `63a5e93`, data digest
**`9c086a9cf68b`** (unchanged from `v0.2.0` — no data moved), annotated
`v0.3.0 — the client runs from a clean install`. Four commits after
[[hexarena-v0-2-0-released]].

**What it is for.** Every earlier release was unusable to anyone without a
checkout: `cmd/hexarena-tui` opened with `forge.Load("internal/seed/data")`, a
path relative to the working directory, so an installed binary died on
`read internal/seed/data/combat.json: … no such file or directory`. `SCR-014`
(`#404`, `#405`, `#406`) fixed it. **This is the first tag worth handing to
somebody else.**

⚠️ **The proof is the empty directory, not the install.** `go install` succeeded
at `v0.1.0` and `v0.2.0` too; the binaries just could not run. So verify a
release by `cd`ing somewhere empty and starting the client, not by watching the
download finish. Done for this one: all five commands installed from the public
proxy, `hexarena-host --version` answers `v0.3.0 / protocol 1 / data
9c086a9cf68b`, and `hexarena-tui` reaches its menu with no data directory
anywhere. ⚠️ The client refuses a pipe before it loads its books, so a plain
subprocess measures the terminal guard rather than the load — drive it under a
pty (`script -q /dev/null …`) or the check is vacuous.

**Sharing it now is two lines**, and no `GOPRIVATE` — the repository is public:

```
go install github.com/vukyn/hexarena/cmd/hexarena-tui@v0.3.0    # a player
go install github.com/vukyn/hexarena/cmd/hexarena-host@v0.3.0   # whoever opens the room
```

⚠️ **`hexforge` and `hexforge-tui` still need a real directory, and that is
correct** — they *write* the JSON, and `go:embed` is read-only. Run those from a
module root. Their error is still the bare relative path with no hint about
`--data`; not fixed, not raised.

The release procedure is unchanged and lives in [[hexarena-v0-2-0-released]]:
the version is the git tag and nothing else, and the tree must be proved green
**before** the push because a tag on proxy.golang.org is immutable. Done here on
a full `make check` at `EXIT=0`, 37 packages, 319s wall-clock at `63a5e93`.

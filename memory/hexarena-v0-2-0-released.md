---
name: hexarena-v0-2-0-released
description: hexarena v0.2.0 tag pushed (e442ec3, data digest 9c086a9cf68b); all five binaries installed from the public proxy and announce the tag; make check wall-clock 390s
metadata:
  type: project
---

hexarena **`v0.2.0`** tagged and pushed 2026-09-09, at `e442ec3`, data digest
**`9c086a9cf68b`**, annotated `v0.2.0 — spectators, rejoin, and five a side`.
134+3 commits after [[hexarena-v0-1-0-released]] (`v0.1.0`, `d56e545`, digest
`4792c12397c5`).

Verified from the **public** proxy, not asserted — all five commands install and
the two that print a banner announce the tag and the same digest as the local
tree:

```
go install github.com/vukyn/hexarena/cmd/hexarena-tui@v0.2.0
go install github.com/vukyn/hexarena/cmd/hexarena-host@v0.2.0
go install github.com/vukyn/hexarena/cmd/hexforge@v0.2.0
go install github.com/vukyn/hexarena/cmd/hexforge-tui@v0.2.0
go install github.com/vukyn/hexarena/cmd/hexarena@v0.2.0
→ hexarena-host v0.2.0 / protocol 1 / data 9c086a9cf68b
```

⚠️ **The repository is PUBLIC.** The platform `CLAUDE.md` said *"Private
remote"* for hexarena and that was **false** — `gh repo view` reports
`visibility=PUBLIC`. Fixed there the same day. It matters because it decides
whether an install needs `GOPRIVATE`, and it needs none.

**Version lives in exactly one place: the git tag.** There is no version
constant, no `-ldflags -X` in the `Makefile`, and nothing to edit — `wire.BuildOf`
reads `debug.BuildInfo.Main.Version`, which the toolchain fills from the module
version, and a tag is left whole while a pseudo-version is trimmed to its
revision. So bumping the version *is* `git tag -a` plus a push. `wire.Protocol`
is a **different number** and did not move: still `1`, last changed at `#237`.

⚠️ **Prove the tree green BEFORE pushing a tag**, per
[[hexarena-v0-1-0-released]]: a tag on proxy.golang.org is immutable. This one
was cut on a full `make check` at `EXIT=0`, 37 packages — and the check was
re-run rather than inherited, because `#402` landed `s06` into
`internal/seed/data/squads.json` between the previous green run and the tag.
Embedded data changed under a green result is exactly [[a-guard-on-the-wrong-side-of-the-gate]]'s
subject; the goldens happened to survive it.

**`make check` wall-clock: 390s (6.5 min)** on 8 cores, stamped with `date +%s`
either side. The package times sum to 1229s, which overstates it **3.2×**
because `go test` parallelises packages — the sum is the number not to size a CI
runner with. Recorded in `TODO.md` § `ENG-014`.

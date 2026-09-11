---
name: hexarena-releases
description: hexarena's release log and the procedure — version IS the git tag and nothing else; prove the tree green BEFORE pushing because a proxy tag is immutable; verify by RUNNING in an empty directory, not by watching the install finish. v0.3.0 · v0.3.1 · v0.4.0 · v0.5.0
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

## `v0.3.1` — `941f5af`, 2026-09-10

⚠️ **`v0.3.0` fixed the game client and left `hexforge` broken in the same way.**
`FRG-004` finished it: of `hexforge`'s **11** code paths that reach a data
directory, **6 only read** and now take the embedded books, while the **5** that
write refuse and name the way out (`--data <dir>`, or the module root) instead of
printing a bare relative path. Same digest, `9c086a9cf68b` — no data moved.

Verified the same way, all four paths from one empty directory at `v0.3.1`:
`hexarena-host --version` answers the tag, `hexarena-tui` reaches its menu under
a pty, `hexforge cast` lists the cast, `hexforge origins add` refuses with the
long sentence.

⚠️ **`cmd/hexforge-tui` is still not usable without a directory, deliberately** —
every screen is one keystroke from a write, so the honest shape is a third state
beside `screen.Context.Authoring` rather than a fallback, and it moves goldens in
both clients. Written up under `FRG-004` as not done.

⚠️ **`#409` bumped the `go` directive to 1.27.1 between this branch's `make check`
and its merge.** The merge was CLEAN, which is not the same as tested — the check
was re-run on `main` at `941f5af` before the tag, `EXIT=0`, 37 packages, 318s. A
clean merge over a toolchain change is exactly the case where reusing an earlier
green result is wrong.

## `v0.4.0` — `9950dd2`, 2026-09-10

The local battle picks both sides. It used to open on whichever catalogue row
the reader last pointed at, against **the next row wrapping**; now one cursor
chooses the side you command and one the side across the board. Same digest,
`9c086a9cf68b`.

⚠️ **A minor bump, not a patch** — it adds behaviour. The two before it were
patches to "the installed binary cannot run at all".

The menu also stopped describing the wrong feature. It said *"play a battle
yourself, with the first side on the list"* and never said who plays the other
half; it now says the machine does. That matters beyond wording:
⚠️ **the mode was misread twice while planning, in opposite directions**, once as
PvE with no evidence and once as hot-seat on the strength of a comment. It is
PvE, and `internal/screen/play.go`'s `run()` settles it in four lines — a turn
whose `unit.Side == p.Side` stops and asks, every other turn goes to
`engineOrder`. The `a` key is an assist on *your own* turn. Read the control
flow, not the comment. → `TODO.md` § `SCR-015`.

**Verifying this one needed a different instrument.** A TUI will not render into
a redirected file, so the empty-directory check that worked for `v0.3.x` reaches
only the host and `hexforge`. The client was proved by searching the installed
binary for the wording — new sentence present, old sentence absent, with the
previous tag as the control. ⚠️ That search lied the first time: see
[[a-search-that-returns-zero-needs-a-known-positive]].

## `v0.5.0` — `a36a669`, 2026-09-11

Three balance changes and two screens. A reply answers **each strike** rather
than each use of a skill; a drain takes back **each strike**, before the reply
to it; block charges count down in the log instead of every line reporting the
whole volley. The status catalogue reached the client's menu, and the aim list
now marks which targets are weak to the skill and which resist it. Digest
unchanged, `9c086a9cf68b` — no data moved.

⚠️ **Old logs no longer pass `--verify`**, which compares every event struct.
Decided and accepted rather than worked around; there is no log version.

⚠️ **This is the first tag cut behind CI**, and the proof is different in kind.
`ENG-014` landed in the same range: `make check` now runs on every pull request
**and on pushes to `main`**, so the evidence for this tag is a green run on a
clean machine at the exact commit (`34578299041`, `a36a669`) rather than a local
run only. The runner takes **731s then 663s**, about **1.95×** the 357s of eight
local cores. Keep waiting for `MERGEABLE CLEAN` — it now means something it did
not before, and it costs ~11 minutes.

Verified the usual way, from one empty directory: host prints the tag, `hexforge
cast` lists, and the client binary carries both new strings in both languages —
**counted as UTF-8 bytes, not with `grep`**, per
[[a-search-that-returns-zero-needs-a-known-positive]].

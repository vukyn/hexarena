---
name: hexarena-scan-baseline
description: hexarena is no longer an offline engine — it gained a websocket host, mDNS discovery and a wire protocol; real risk is terminal-escape injection, not deps
metadata:
  type: project
---

hexarena grew a **network listener** between 2026-09-02 and 2026-09-14 (286 commits):
`cmd/hexarena-host`, `internal/socket` (coder/websocket), `internal/room`,
`internal/wire`, `internal/discovery` (libp2p/zeroconf mDNS). LAN PvP, plaintext
`ws://`, password **optional and off by default**, room advertised over mDNS.

**Why:** every brief and doc I was handed still described it as "no network
listener, no auth, no database" — that was true at the previous scan and is now
false. Scanning it as an offline CLI misses the entire live attack surface.

**How to apply:** on any hexarena scan, check `cmd/` and `internal/socket|room|
wire|discovery` first. The real findings are **terminal-escape injection** into
free-text that reaches a terminal — `wire.Hello.Name` → `playerName()`
(`cmd/hexarena-host/main.go`, bounds length to 32 runes, does **not** strip
control chars) and battle-log `name`/`note`/`target` → `internal/tui` via
`--replay`. OSC 52 = clipboard write = command execution on paste; CSI 2J can
forge the `verified:`/`unverified:` banner that `--verify` exists to produce.

Verified-correct and not worth re-deriving from scratch each time:
`internal/core` is stdlib-only; no floats/clock/rand/goroutines in core;
determinism holds over 12 runs and under `GODEBUG=randmapiter=1`; `ParseLog`
rejects bad enums at decode time via custom unmarshallers and Go's max-depth
guard stops nesting bombs; `cast.ValidateImagePath` genuinely blocks art-path
traversal; `--log` writes 0600. `internal/testfixture` ships in **no** binary, so
its ~20 gosec path/permission hits are noise — check `go list -deps ./cmd/...`
before rating any gosec finding there. Repo remote is **PUBLIC**.

Related: [[scan-history-stores-counts-only]]

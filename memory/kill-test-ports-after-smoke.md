---
name: kill-test-ports-after-smoke
description: Always kill the local test server/port + throwaway container after a smoke/e2e run; go-run spawns a surviving child
metadata: 
  node_type: memory
  type: feedback
  modified: 2026-07-24T09:58:38.616Z
---

After any local smoke/e2e test that starts a server (gardener `go run cmd/main.go` on a throwaway port + docker PG), **always kill the server and free the port when the test finishes** — don't leave it listening.

**Why:** user said "nhớ kill port khi test xong" — several stray servers were left holding ports 8088/8093-8097/8191 across this session's smokes.

**How to apply:**
- Put a `trap cleanup EXIT` in every smoke script (kill $SRV + `docker rm -f`), AND after the run verify + hard-clean.
- **GOTCHA:** `go run cmd/main.go` compiles a temp binary and runs it as a CHILD; killing the `go run` parent (or `pkill -f "go run cmd/main.go"`) does NOT stop the child holding the port. Kill by port: `lsof -ti :PORT | xargs kill -9`. Sweep the range used (e.g. `for p in 8088 8093..8099 8191; do lsof -ti :$p | xargs kill -9; done`) and `docker rm -f $(docker ps -q --filter name=gardener_)`.
- Prefer building a binary + backgrounding it (easier to kill) or ensure the trap kills the actual listener. Port choice: 8190+ (avoid 808x/809x = img2svg) per [[gardener-plan]].

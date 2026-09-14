# Memory Index

## Security posture
- [hexarena security posture](hexarena-security-posture.md) — clean baseline; only gosec/gitleaks informative; G115/G101 = proven FP; forge writes 0644 by design; --verify is opt-in
- [hexarena scan baseline 2026-09-14](project_hexarena-scan-baseline.md) — NO LONGER an offline engine: websocket host + mDNS + wire protocol; real risk = terminal-escape injection via peer name and replay log

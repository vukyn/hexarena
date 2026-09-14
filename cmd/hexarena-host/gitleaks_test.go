package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The two scanner files live at the repository root, which has no package of its
// own — so they are read from here, for the reason makefilePath is.
const (
	gitleaksIgnorePath = "../../.gitleaksignore"
	gitleaksConfigPath = "../../.gitleaks.toml"
)

// fingerprint is a gitleaks ignore entry: a 40-character commit sha, a path, a
// rule id and a line, colon-separated. Comments and blank lines are everything
// else in the file.
var fingerprint = regexp.MustCompile(`(?m)^([0-9a-f]{40}):([^:]+):([^:]+):(\d+)$`)

// TestTheTwoScannerFilesShipTogether holds the trap that makes these one change
// rather than two.
//
// ⚠️ **`.gitleaksignore` does not scan clean on its own.** Its entries are
// 40-character commit fingerprints, and the default `generic-api-key` rule reads
// a 40-character hex string as a high-entropy secret — so every entry added to
// silence a finding becomes a finding, and a repository that adopted the ignore
// file alone would have traded a known false positive for a new one. The
// allowlist in `.gitleaks.toml` is what closes that, and nothing but a test
// stops the two drifting apart: deleting the config leaves a repository that
// still scans, still has an ignore file, and is newly dirty.
func TestTheTwoScannerFilesShipTogether(t *testing.T) {
	if _, err := os.Stat(gitleaksIgnorePath); os.IsNotExist(err) {
		t.Skip("no .gitleaksignore, so there is nothing for a config to allowlist")
	} else if err != nil {
		t.Fatalf("stat %s: %v", gitleaksIgnorePath, err)
	}
	config, err := os.ReadFile(gitleaksConfigPath)
	if err != nil {
		t.Fatalf("there is a .gitleaksignore and no readable .gitleaks.toml beside it, "+
			"so its own fingerprints now scan as secrets: %v", err)
	}
	text := string(config)
	if !strings.Contains(text, `'''^\.gitleaksignore$'''`) {
		t.Errorf(".gitleaks.toml does not allowlist .gitleaksignore, so every accepted "+
			"finding is itself a finding:\n%s", text)
	}
	// ⚠️ **`useDefault` is the second half and is easy to lose.** A config that
	// replaced the rule set instead of extending it would allowlist the ignore
	// file and silently drop every detector gitleaks ships — a scanner that finds
	// nothing because it looks for nothing.
	if !strings.Contains(text, "useDefault = true") {
		t.Errorf(".gitleaks.toml does not extend the default rules, so the scan is "+
			"looking for a hand-written list:\n%s", text)
	}
}

// TestEveryAcceptedFindingCarriesAReason is the discipline the ignore file's own
// header claims, held by something other than the header.
//
// ⚠️ **An entry with no reason is the failure this file exists to prevent**, not
// an untidiness: the next reader cannot tell "judged public, nothing to rotate"
// from "silenced because it was noisy", and the second is how a live credential
// stays in a repository. A comment block immediately above the line is what a
// reason looks like here.
func TestEveryAcceptedFindingCarriesAReason(t *testing.T) {
	raw, err := os.ReadFile(gitleaksIgnorePath)
	if os.IsNotExist(err) {
		t.Skip("no .gitleaksignore, so there are no accepted findings to justify")
	}
	if err != nil {
		t.Fatalf("read %s: %v", gitleaksIgnorePath, err)
	}
	lines := strings.Split(string(raw), "\n")
	entries := 0
	for i, line := range lines {
		if !fingerprint.MatchString(line) {
			continue
		}
		entries++
		reason := 0
		for back := i - 1; back >= 0 && strings.HasPrefix(lines[back], "#"); back-- {
			reason++
		}
		// Three lines rather than one: a single `# fixed in the working tree` is
		// a label, and what this asks for is the judgement — what the value is,
		// why it is not a credential, and what would have to be true for that to
		// be wrong.
		if reason < 3 {
			t.Errorf("the entry on line %d carries %d lines of reason:\n  %s", i+1, reason, line)
		}
	}
	// The count is what stops the walk passing on a file it failed to parse — a
	// regexp matching nothing would otherwise agree that every entry it found is
	// justified.
	if entries == 0 {
		t.Fatal("no fingerprint lines were recognised in .gitleaksignore, so this walk " +
			"checked nothing; the entry format has changed")
	}
}

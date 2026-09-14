package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/plain"
)

// clipboardWrite is OSC 52, the escape that writes the reader's clipboard. A
// saved log is a file somebody handed over, so its free text is their writing.
const clipboardWrite = "\x1b]52;c;cm0gLXJmIH4=\x07"

// forgedVerdict is what makes this a High rather than an annoyance, and it uses
// **no escape sequence at all**: cmd/hexarena prints exactly this sentence after a
// successful --verify, so a log that prints it first is a log that answers the
// only question --verify exists to answer. A reader sees the program's own words
// in the program's own place.
const forgedVerdict = "verified: re-running seed 11 reproduced all 255 events exactly"

// TestTheUnverifiedNoticeComesBeforeAnyByteTheFileSupplied is the fix, stated as
// the property rather than as a line number.
//
// ⚠️ **The assertion is an OFFSET, not a substring.** "The output contains the
// unverified notice" was true before the change too — it was printed last, after
// the whole body and the summary, which is precisely how a long enough file
// scrolled it off the top of a terminal and left a forged verdict as the last
// verdict-shaped line on screen. What has to hold is that the notice is written
// **before** the first byte the file chose, so index-of is the measurement.
func TestTheUnverifiedNoticeComesBeforeAnyByteTheFileSupplied(t *testing.T) {
	path := writeLog(t, hostileLog())
	var out bytes.Buffer
	if err := replay(config{replay: path}, &out); err != nil {
		t.Fatalf("replay: %v", err)
	}
	said := out.String()

	notice := strings.Index(said, "unverified:")
	if notice < 0 {
		t.Fatalf("the replay never said it was unverified:\n%s", said)
	}
	forged := strings.Index(said, forgedVerdict)
	if forged < 0 {
		t.Fatalf("the crafted log's forged verdict never reached the output, so this test is "+
			"measuring nothing — the fixture no longer carries the attack:\n%s", said)
	}
	if notice > forged {
		t.Errorf("the file's forged verdict is printed at %d and the real notice at %d: "+
			"a reader meets the forgery first", forged, notice)
	}
	// The tighter claim: nothing the file supplied is written before the notice.
	// The header line above the body carries only the seed and two counts, so the
	// notice has to be ahead of every one of the log's own strings.
	for _, supplied := range []string{"Bảo", forgedVerdict, "crafted"} {
		if at := strings.Index(said, supplied); at >= 0 && at < notice {
			t.Errorf("%q, which came out of the file, is written at %d — before the notice at %d",
				supplied, at, notice)
		}
	}
}

// TestVerifyingSuppressesTheNoticeRatherThanMovingIt holds the other side of the
// branch the reordering restructured. Moving a print from the foot of a function
// to its head is exactly the edit that accidentally makes it unconditional, and a
// run with -verify that announced itself unverified would be the notice crying
// wolf on the one path where the verdict is real.
//
// ⚠️ This log carries no roster, so verify refuses it rather than re-running it.
// That is the point: the refusal comes from verify, **after** the body has been
// written, and the notice must be absent throughout regardless of how verify ends.
func TestVerifyingSuppressesTheNoticeRatherThanMovingIt(t *testing.T) {
	path := writeLog(t, hostileLog())
	var out bytes.Buffer
	err := replay(config{replay: path, verify: true}, &out)
	if err == nil {
		t.Fatal("a log with no roster was verified rather than refused, so this fixture no longer " +
			"reaches the branch it was written for")
	}
	if strings.Contains(out.String(), "unverified:") {
		t.Errorf("a -verify run still printed the unverified notice:\n%s", out.String())
	}
	// The body is still drawn, and still inert — a refusal to verify is not a
	// refusal to render, and the bytes above the refusal came out of the file.
	if !strings.Contains(out.String(), "== summary ==") {
		t.Errorf("a -verify run that could not verify drew no body at all:\n%s", out.String())
	}
	if strings.ContainsRune(out.String(), 0x1b) {
		t.Errorf("a -verify run wrote a raw ESC:\n%q", out.String())
	}
}

// TestACraftedLogCannotWriteAnEscapeToThisWriter is A2's other half, measured on
// the bytes rather than on a rendering.
//
// ⚠️ It goes through replay and not through tui.Log, because that is what the
// binary calls: internal/tui has its own test of the same property, and this one
// exists to catch a future print in this file that reaches round the renderer.
func TestACraftedLogCannotWriteAnEscapeToThisWriter(t *testing.T) {
	path := writeLog(t, hostileLog())
	var out bytes.Buffer
	if err := replay(config{replay: path}, &out); err != nil {
		t.Fatalf("replay: %v", err)
	}
	said := out.String()
	if !strings.Contains(said, "]52;c;") {
		t.Fatalf("the payload never reached the output at all, so nothing was measured:\n%s", said)
	}
	if !plain.Inert(strings.ReplaceAll(said, "\n", " ")) {
		t.Errorf("the replay wrote bytes a terminal would obey:\n%q", said)
	}
	if strings.ContainsRune(said, 0x1b) {
		t.Errorf("the replay wrote a raw ESC:\n%q", said)
	}
}

// TestAHostileFilenameCannotForgeTheNotice closes the one raw string left on the
// line whose whole job is to be unforgeable. The path is argv rather than the
// file, so it is a weaker vector — but a sentence that is the trust anchor has
// nothing raw in it, and the cost of holding that is one call.
func TestAHostileFilenameCannotForgeTheNotice(t *testing.T) {
	// The escape lives in a symlink's name rather than the real file's, because
	// some filesystems refuse the bytes outright and the assertion is about what
	// replay does with the string, not about what a filesystem accepts.
	real := writeLog(t, honestLog())
	link := filepath.Join(t.TempDir(), "log\x1b]0;pwned\x07.json")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("this filesystem will not hold an escape in a name: %v", err)
	}
	var out bytes.Buffer
	if err := replay(config{replay: link}, &out); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if strings.ContainsRune(out.String(), 0x1b) {
		t.Errorf("the path put a raw ESC on the unverified notice:\n%q", out.String())
	}
}

// hostileLog is a log whose free text is the attack: a name carrying an OSC 52
// and a note carrying the forged verdict. It is otherwise an ordinary, parseable
// log — which is the point, since ParseLog is what a reader trusts to have
// checked it and ParseLog has nothing to say about any of this.
func hostileLog() battle.Log {
	return battle.Log{
		Seed: 11,
		Events: []battle.Event{
			{Kind: battle.Started, Actor: "a1", Name: "Bảo" + clipboardWrite, Amount: 3000, Note: "crafted"},
			{Kind: battle.Started, Actor: "e1", Name: "enemy", Amount: 3000},
			{Kind: battle.TurnBegan, Actor: "a1", Turn: 1},
			{Kind: battle.TurnSkipped, Actor: "a1", Note: "\r\n" + forgedVerdict + "\n"},
		},
	}
}

func honestLog() battle.Log {
	return battle.Log{
		Seed: 11,
		Events: []battle.Event{
			{Kind: battle.Started, Actor: "a1", Name: "Bulbasaur", Amount: 3000},
		},
	}
}

func writeLog(t *testing.T, log battle.Log) string {
	t.Helper()
	raw, err := battle.MarshalLog(log)
	if err != nil {
		t.Fatalf("encode the fixture log: %v", err)
	}
	path := filepath.Join(t.TempDir(), "battle.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write the fixture log: %v", err)
	}
	return path
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTheBannerSaysTheDefaultRoomIsOpen is the finding, which is not a bug in any
// one line of code: the room binds every interface, an empty password admits any
// hello past the version gate, and `-browse` announces the address. All three are
// decisions. What was missing is that the host could read the whole banner
// without learning it.
//
// ⚠️ **The old wording is what this asserts against, not just a phrase to
// contain.** It said "anybody with the code can join", which describes a room
// reached by somebody the host read the code out to — a sentence that is true of
// a different program. Keeping it in the table as a thing that must NOT appear is
// what stops it coming back in a merge.
func TestTheBannerSaysTheDefaultRoomIsOpen(t *testing.T) {
	for _, each := range []struct {
		what      string
		password  wire.Password
		announced bool
		wanted    []string
		unwanted  []string
	}{
		{
			what:     "no password, not announced",
			wanted:   []string{"none", "EVERY interface", "-password"},
			unwanted: []string{"anybody with the code can join"},
		},
		{
			what:      "no password, announced over mDNS",
			announced: true,
			wanted:    []string{"none", "ANNOUNCING", "local network", "without being told the code", "-password"},
			unwanted:  []string{"anybody with the code can join"},
		},
		{
			what:     "a password is set",
			password: fixturePassword,
			wanted:   []string{"set", "players will need it"},
			// A room with a password says nothing about being open, because it is
			// not — a line drawn on every room is a line every host reads past,
			// which is the argument the draft and watch lines already make.
			unwanted: []string{"EVERY interface", "ANNOUNCING", "-password"},
		},
		{
			// ⚠️ The same, WITH -browse. A room that is announced but gated is
			// still closed: being findable is not being enterable, and a warning
			// here would teach a host that -browse is the dangerous flag when the
			// empty password is.
			what:      "a password is set and the room is announced",
			password:  fixturePassword,
			announced: true,
			wanted:    []string{"set"},
			unwanted:  []string{"EVERY interface", "ANNOUNCING", "-password"},
		},
	} {
		t.Run(each.what, func(t *testing.T) {
			said := passwordLine(each.password, each.announced)
			for _, wanted := range each.wanted {
				if !strings.Contains(said, wanted) {
					t.Errorf("the password line does not say %q:\n  %s", wanted, said)
				}
			}
			for _, unwanted := range each.unwanted {
				if strings.Contains(said, unwanted) {
					t.Errorf("the password line still says %q:\n  %s", unwanted, said)
				}
			}
			// ⚠️ Whatever it says, it may not say the password. That is
			// TestARoomPasswordIsNeverPrintedByTheHost's claim over the whole
			// binary; it is repeated on this one line because this is the line
			// that was just rewritten, and the rewrite is where it would go wrong.
			if strings.Contains(said, string(fixturePassword)) {
				t.Errorf("the password line prints the password itself: %s", said)
			}
		})
	}
}

// TestTheWholeBannerCarriesTheExposure is the same claim through banner rather
// than through the one function, because a line that is never drawn says nothing.
func TestTheWholeBannerCarriesTheExposure(t *testing.T) {
	held := hosting(t, aRoom(), documented, newPaper().on, newPaper().on)
	var drawn bytes.Buffer
	banner(held, "this machine", &drawn)
	if !strings.Contains(drawn.String(), "EVERY interface") {
		t.Errorf("a default room's banner does not say it is open:\n%s", drawn.String())
	}
}

// TestTheLogDirectoryIsOwnerOnly is gosec G301, measured on the mode the
// filesystem actually ends up with rather than on the constant in the source.
//
// ⚠️ **umask makes the constant and the result different questions**, and the
// source is the one that can be wrong without anybody noticing: a 0o755 under a
// 022 umask still comes out 0755, so a test reading the number back is the only
// one that would have failed before this change.
func TestTheLogDirectoryIsOwnerOnly(t *testing.T) {
	into := filepath.Join(t.TempDir(), "logs")
	held := &hosted{logs: into}
	var said bytes.Buffer
	reading := room.Reading{Played: []room.BattleResult{{Battle: 1, Seed: 99}}}
	if err := held.write(reading, &said); err != nil {
		t.Fatalf("write the logs: %v", err)
	}
	info, err := os.Stat(into)
	if err != nil {
		t.Fatalf("stat the log directory: %v", err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("the log directory is %#o, which is readable by somebody other than its owner", mode)
	}
	// The files inside were already 0600 and stay that way; a tight directory
	// around loose files would be the same gap moved one level down.
	written, err := os.ReadDir(into)
	if err != nil {
		t.Fatalf("read the log directory: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("nothing was written, so the modes above are of an empty directory")
	}
	for _, each := range written {
		info, err := each.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", each.Name(), err)
		}
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			t.Errorf("%s is %#o, which is readable by somebody other than its owner", each.Name(), mode)
		}
	}
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/wire"
)

// TestARoomThatTakesWatchersIsOpenedAndSaysSo is -draft's shape for -watch, and
// it holds both halves for the same reason that one does.
//
// ⚠️ **A banner line that is always drawn passes the first half alone.** Every
// state this binary prints is one line in one block, so a `-watch` note wired to
// nothing — or to a constant — reads exactly like the real thing on the run that
// asked for it. What separates them is the run that did **not** ask: a host who
// never typed the flag must not be told spectators can paste the code, because
// they cannot, and every one of them would be refused
// wire.CodeWatchingClosed at the gate.
//
// ⚠️ **The advice is checked against the code the room actually got, not
// against a guess.** The room-code decision the whole feature rests on is that a
// spectator pastes the same twelve characters a player pastes — one flag on the
// hello rather than a second code space — and the host is the one person who
// reads that code out, so the banner is where it has to be said. Asserting the
// line contains the word "twelve" would be this test agreeing with the sentence
// it is reading; asserting it names len(held.code) makes the claim false the day
// the two stop agreeing.
func TestARoomThatTakesWatchersIsOpenedAndSaysSo(t *testing.T) {
	for _, one := range []struct {
		name      string
		watch     bool
		watchable bool
	}{
		{name: "with -watch", watch: true, watchable: true},
		{name: "without it", watch: false, watchable: false},
	} {
		t.Run(one.name, func(t *testing.T) {
			chosen := aRoom()
			chosen.watch = one.watch
			var out, errs bytes.Buffer
			held := hosting(t, chosen, documented, &out, &errs)
			if held.config.Watchable != one.watchable {
				t.Fatalf("-watch=%t opened a room with Watchable=%t, want %t",
					one.watch, held.config.Watchable, one.watchable)
			}
			var banners bytes.Buffer
			banner(held, "was told by -advertise", &banners)
			said := strings.Contains(banners.String(), "watch ")
			if said != one.watchable {
				t.Errorf("-watch=%t: the banner %s the watching line, want the other way:\n%s",
					one.watch, map[bool]string{true: "draws", false: "does not draw"}[said],
					banners.String())
			}
			if !one.watchable {
				return
			}
			// The one thing a host has to know, checked against the two things
			// that own it: the code this room was actually given, and the
			// protocol constant that fixes every code's length. A line quoting a
			// number that is neither is a line telling a host to read out
			// something that does not exist.
			if len(held.code) != wire.RoomCodeLength {
				t.Fatalf("the room's code is %d characters and the protocol says %d, so the "+
					"assertion below cannot tell a right number from a wrong one",
					len(held.code), wire.RoomCodeLength)
			}
			advice := fmt.Sprintf("SAME %d characters", len(held.code))
			if !strings.Contains(banners.String(), advice) {
				t.Errorf("a watchable room's banner does not say %q, so a host does not learn "+
					"that a spectator pastes the code they are about to read out:\n%s",
					advice, banners.String())
			}
		})
	}
}

// TestTheWatchFlagIsWhatTheRoomIsOpenedWith is the one that catches a flag wired
// to nothing, and it is a separate test because it goes in through the **flag
// set** rather than by setting the field.
//
// ⚠️ **Setting `chosen.watch` directly, which is what the test above does,
// proves nothing about the flag.** A `-watch` that was never registered, or
// registered against another variable, leaves that test entirely green: it
// assigns the struct field itself. So this one parses the argument a person
// types, follows it into room.Config, and asserts the default from the same
// parse rather than from the struct's zero value — a flag registered with a
// default of true would otherwise look exactly like watching being on by
// design.
func TestTheWatchFlagIsWhatTheRoomIsOpenedWith(t *testing.T) {
	for _, one := range []struct {
		name      string
		arguments []string
		want      bool
	}{
		{name: "typed", arguments: []string{"-watch"}, want: true},
		{name: "left off", arguments: nil, want: false},
	} {
		t.Run(one.name, func(t *testing.T) {
			var chosen settings
			set := flags(&chosen)
			set.SetOutput(io.Discard)
			if err := set.Parse(one.arguments); err != nil {
				t.Fatalf("parse %v: %v", one.arguments, err)
			}
			if chosen.watch != one.want {
				t.Fatalf("%v left the flag at %t, want %t", one.arguments, chosen.watch, one.want)
			}
			// An ephemeral port, for aRoom's reason: nothing in this suite may
			// fight the default port or another session for 13579. Everything
			// else stays exactly as the flag set left it, because what is being
			// measured is the value that parse produced.
			chosen.port = 0
			var out, errs bytes.Buffer
			held := hosting(t, chosen, documented, &out, &errs)
			if held.config.Watchable != one.want {
				t.Errorf("%v parsed to watch=%t and opened a room with Watchable=%t: the flag "+
					"reaches no room configuration at all", one.arguments, chosen.watch,
					held.config.Watchable)
			}
		})
	}
	// A flag nobody is told about is a flag nobody types, which is the half
	// TestTheVersionFlagIsDescribedInTheUsage holds for -version. It renders the
	// set rather than asserting on a literal, so a description that stopped
	// being registered fails.
	asked := newPaper()
	if err := run([]string{"-h"}, newPaper().on, asked.on); err != nil {
		t.Fatalf("-h is not a failure: %v", err)
	}
	usage := asked.said()
	for _, wanted := range []string{"-watch", "let spectators watch"} {
		if !strings.Contains(usage, wanted) {
			t.Errorf("the usage does not carry %q:\n%s", wanted, usage)
		}
	}
}

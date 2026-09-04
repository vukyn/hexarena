package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// # -version, and the two things about it that are this binary's own
//
// Until this change you had to **open a room** to see the build line: the three
// numbers a refused player is told to compare were printed on the banner and
// nowhere else, and `-version` was not even a defined flag — measured,
// `hexarena-host -version` said "flag provided but not defined: -version" and
// exited 2. The shape of the output is internal/wire's, held by
// TestTheVersionReportIsOneShapeForBothBinaries; what is this binary's own is
// that the flag exists, that the numbers on it are the real ones, and that
// answering it hosts nothing.

// TestVersionSaysWhatThisBinaryIsAndHostsNothing drives the flag through run,
// which is main with its writers handed in.
//
// ⚠️ **"Hosts nothing" is measured by what run would otherwise have refused,
// rather than by probing a port.** A test that asked whether 13579 was free
// afterwards would be a test that fails on a machine where the user is hosting a
// real match, and one that passes on a machine where the listener leaked but the
// port was taken by something else. So the measurement is that -version wins
// over the three things run does before it binds anything: it draws a seed, it
// works out an address, and it validates the room's configuration. Two rows
// below hand it a configuration that is **refused by name** and an address that
// cannot be parsed, and both still print a version and return nil — which they
// can only do from ahead of all three.
//
// What it can see: that the flag is defined, that the report carries this
// binary's own name and this binary's real digest and protocol, that stderr
// stays empty, that the output is the three lines and nothing else, and that no
// part of hosting ran.
//
// What it cannot see: that a *listener* was not bound, directly. Nothing here
// opens a socket, so what is held is the ordering above rather than the absence
// of a file descriptor. Moving the branch below `open` reddens the two rows with
// a bad configuration and the line count on all three; moving it below `banner`
// reddens the line count.
func TestVersionSaysWhatThisBinaryIsAndHostsNothing(t *testing.T) {
	digest, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	for _, each := range []struct {
		what    string
		flags   []string
		because string
	}{
		{
			what:  "on its own",
			flags: []string{"-version"},
			because: "the ordinary way somebody reads their build to a friend, and the whole reason " +
				"the flag exists: the banner was the only place these numbers were printed",
		},
		{
			what:  "beside a configuration the room refuses by name",
			flags: []string{"-version", "-battles", "2"},
			because: "a bo2 is refused by room.Config.Validate, so a version answered after the " +
				"configuration check would come back as an error instead of a version",
		},
		{
			what:  "beside an address nothing could dial",
			flags: []string{"-version", "-advertise", "300.1.2.3"},
			because: "advertising refuses that by name, so a version answered after the address " +
				"is worked out would come back as an error too",
		},
		{
			what:  "beside a seed, which is otherwise drawn from crypto/rand",
			flags: []string{"-version", "-seed", "11"},
			because: "the flag is accepted and unused; what this row holds is that a version is " +
				"not a reason to refuse a flag that configures a match",
		},
	} {
		out, errs := newPaper(), newPaper()
		if err := run(each.flags, out.on, errs.on); err != nil {
			t.Errorf("-version %s failed with %v — %s", each.what, err, each.because)
			continue
		}
		said := out.said()
		if said == "" {
			t.Errorf("-version %s printed nothing, so every assertion below it is over an "+
				"empty string", each.what)
			continue
		}
		if written := errs.said(); written != "" {
			t.Errorf("-version %s wrote %q to stderr; a version is output rather than a "+
				"diagnostic", each.what, written)
		}
		// The whole rendering, against the one function both binaries print
		// through — so the values are asserted in the positions the report puts
		// them in rather than three greps apart.
		version, err := wire.Local(buildString())
		if err != nil {
			t.Fatalf("read this binary's own version: %v", err)
		}
		if want := version.Report(programName); said != want {
			t.Errorf("-version %s printed\n%q\nand this binary is\n%q", each.what, said, want)
		}
		// ⚠️ And the report is asserted against the real numbers rather than
		// only against itself, which is what the line above cannot do: comparing
		// a report to a report built the same way would pass for a report of
		// nothing.
		if !strings.Contains(said, digest.Short()) {
			t.Errorf("-version %s does not name the embedded data digest %s, which is the "+
				"number a refused player is told to compare:\n%s", each.what, digest.Short(), said)
		}
		if !strings.Contains(said, strconv.Itoa(wire.Protocol)) {
			t.Errorf("-version %s does not name the protocol %d it speaks:\n%s",
				each.what, wire.Protocol, said)
		}
		if !strings.HasPrefix(said, programName+" ") {
			t.Errorf("-version %s does not begin with this binary's name:\n%s", each.what, said)
		}
		// Three lines and nothing else. ⚠️ This is the assertion a leaked
		// listener would show up in: `open` and `banner` both write through the
		// same writer, so a version answered after either of them prints a room
		// code above it.
		if lines := strings.Count(said, "\n"); lines != 3 {
			t.Errorf("-version %s printed %d lines; the report is three, and anything more "+
				"means part of hosting ran:\n%s", each.what, lines, said)
		}
		if strings.Contains(said, "waiting for two players") {
			t.Errorf("-version %s printed the banner, so a room was opened:\n%s", each.what, said)
		}
	}
}

// TestTheVersionFlagIsDescribedInTheUsage is the half a person reaches first: a
// flag nobody is told about is a flag nobody types.
//
// ⚠️ It renders the flag set rather than asserting on a literal, so a
// description that stopped being registered fails. The sentence is deliberately
// the same one cmd/hexarena-tui shows in English, where it comes from
// internal/i18n — this binary has one language and reads its wording off the
// page, and the *output* is the part neither of them spells twice.
func TestTheVersionFlagIsDescribedInTheUsage(t *testing.T) {
	asked := newPaper()
	if err := run([]string{"-h"}, newPaper().on, asked.on); err != nil {
		t.Fatalf("-h is not a failure: %v", err)
	}
	usage := asked.said()
	for _, wanted := range []string{"-version", "print the build, protocol and data, then exit"} {
		if !strings.Contains(usage, wanted) {
			t.Errorf("the usage does not carry %q:\n%s", wanted, usage)
		}
	}
}

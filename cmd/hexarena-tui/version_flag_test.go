package main

import (
	"bytes"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// # -version on the client, and the one thing about it that is not the host's
//
// The output is internal/wire's, held by
// TestTheVersionReportIsOneShapeForBothBinaries, and the flag exists here for
// the same reason it exists there: the three numbers a refused player is told to
// compare were reachable only by getting as far as the join screen. What is this
// binary's own is **where the answer sits in run**, because this binary refuses
// to start at all when stdout is not a terminal.

// TestVersionIsAnsweredBeforeTheTerminalCheck is that placement, measured.
//
// ⚠️ **The guard is the whole test.** Asserting that `-version` works when
// stdout is not a terminal proves nothing unless the check it has to get ahead
// of would really have fired, and under `go test` stdout is a pipe — so the same
// run is asked both questions: without the flag, run refuses; with it, run
// prints. A version answered only on a terminal is a machine-readable version
// with the machines left out, which is half of what it is for.
//
// ⚠️ **The guard is conditional and says so when it is skipped**, because the
// test binary can be run *directly* from a shell, where stdout genuinely is a
// terminal and there is nothing for the check to refuse. The positive half is
// unconditional either way.
//
// What it can see: that the flag is answered ahead of the terminal check and
// ahead of forge.Load, that the report is this binary's own three numbers, and
// that run writes it to the writer it was handed rather than to os.Stdout.
//
// What it cannot see: the terminal check itself, on a terminal. That is one
// os.Stat of a file descriptor and there is no way to hand this process another
// one; what the pipe under `go test` gives is exactly the case that matters.
func TestVersionIsAnsweredBeforeTheTerminalCheck(t *testing.T) {
	digest, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	// ⚠️ A directory that does not exist, on purpose. forge.Load would fail on
	// it, so this is also the measurement that -version is ahead of the library:
	// a binary that cannot find a data directory can still say what it is.
	missing := filepath.Join(t.TempDir(), "no-such-directory")

	if stdoutIsTerminal() {
		t.Log("stdout is a terminal for this test binary, so the refusal below is not " +
			"measured; the positive half still is")
	} else {
		if err := run(options{dir: missing, lang: i18n.Vi}, io.Discard); err == nil {
			t.Fatal("this client started with stdout on a pipe, so the check -version has " +
				"to get ahead of is not in force and the rest of this test measures nothing")
		}
	}

	for _, lang := range i18n.Langs() {
		var said bytes.Buffer
		if err := run(options{dir: missing, lang: lang, version: true}, &said); err != nil {
			t.Errorf("-version in %s failed with %v", lang, err)
			continue
		}
		version, err := wire.Local(buildString())
		if err != nil {
			t.Fatalf("read this binary's own version: %v", err)
		}
		// The whole rendering, so the three values are asserted in the positions
		// the report puts them in — and against the real digest below, because a
		// report compared only to a report built the same way would pass for a
		// report of nothing.
		if want := version.Report(programName); said.String() != want {
			t.Errorf("-version in %s printed\n%q\nand this binary is\n%q", lang, said.String(), want)
		}
		if !strings.Contains(said.String(), digest.Short()) {
			t.Errorf("-version in %s does not name the embedded data digest %s:\n%s",
				lang, digest.Short(), said.String())
		}
		if !strings.Contains(said.String(), strconv.Itoa(wire.Protocol)) {
			t.Errorf("-version in %s does not name the protocol %d it speaks:\n%s",
				lang, wire.Protocol, said.String())
		}
		if !strings.HasPrefix(said.String(), programName+" ") {
			t.Errorf("-version in %s does not begin with this binary's name:\n%s",
				lang, said.String())
		}
		// ⚠️ Three lines and nothing else, in **both** languages, which is the
		// assertion that the report is not translated. The two labels on it are
		// the ones cmd/hexarena-host prints and the ones both version refusals
		// tell a player to read, so a Vietnamese copy of them would break the
		// one instruction those refusals give. → i18n.VersionFlagUsage.
		if lines := strings.Count(said.String(), "\n"); lines != 3 {
			t.Errorf("-version in %s printed %d lines; the report is three:\n%s",
				lang, lines, said.String())
		}
	}
	// And the two languages printed the same bytes, stated as its own assertion
	// rather than inferred from two identical expectations above.
	var vietnamese, english bytes.Buffer
	if err := run(options{dir: missing, lang: i18n.Vi, version: true}, &vietnamese); err != nil {
		t.Fatalf("-version in Vietnamese: %v", err)
	}
	if err := run(options{dir: missing, lang: i18n.En, version: true}, &english); err != nil {
		t.Fatalf("-version in English: %v", err)
	}
	if vietnamese.String() != english.String() {
		t.Errorf("the version report differs by language:\n%q\n%q",
			vietnamese.String(), english.String())
	}
}

// TestTheVersionFlagIsParsedAndDescribedInBothLanguages is the flag's own half:
// that it reaches options, and that a reader is told about it in the language
// they are reading.
//
// ⚠️ The description is asserted against internal/i18n rather than against a
// literal, which is the point of it being there: a sentence written in this
// package would exist in one language only and
// TestNoScreenHoldsItsOwnWording would be the only thing that noticed.
func TestTheVersionFlagIsParsedAndDescribedInBothLanguages(t *testing.T) {
	parsed, err := parseOptions([]string{"-version"}, "", io.Discard)
	if err != nil {
		t.Fatalf("parse -version: %v", err)
	}
	if !parsed.version {
		t.Error("-version was parsed and did not reach options, so the flag is registered and unread")
	}
	plain, err := parseOptions(nil, "", io.Discard)
	if err != nil {
		t.Fatalf("parse no flags: %v", err)
	}
	if plain.version {
		t.Error("a run with no flags asks for a version, so the default is the wrong way round")
	}
	for _, lang := range i18n.Langs() {
		var shown bytes.Buffer
		// -h is not a failure; flag reports it as ErrHelp after printing the
		// descriptions, which is what is being read here.
		if _, err := parseOptions([]string{"-h"}, lang.String(), &shown); err == nil {
			t.Fatalf("-h in %s returned no error, so flag did not print its usage", lang)
		}
		usage := shown.String()
		if !strings.Contains(usage, "-version") {
			t.Errorf("the usage in %s does not name -version:\n%s", lang, usage)
		}
		if want := lang.Text(i18n.VersionFlagUsage); !strings.Contains(usage, want) {
			t.Errorf("the usage in %s does not describe -version as %q:\n%s", lang, want, usage)
		}
		// The anti-vacuity half: the *other* language's wording is not on it, so
		// a description that ignored the language it was asked for fails.
		other := i18n.Vi
		if lang == i18n.Vi {
			other = i18n.En
		}
		if wrong := other.Text(i18n.VersionFlagUsage); strings.Contains(usage, wrong) {
			t.Errorf("the usage in %s describes -version in %s (%q):\n%s", lang, other, wrong, usage)
		}
	}
}

package wire

import (
	"runtime/debug"
	"testing"
)

// TestTheBuildStringFallsBackInOneOrder holds the four sources and the order
// they are trusted in.
//
// ⚠️ It drives the **pure** function and not the linker, which is the whole
// reason BuildOf takes its two inputs as parameters: a test that asserted on a
// real -ldflags stamp would need to shell out to `go build`, and one that
// asserted on this process's own build info would assert a different thing under
// `go test` than under `go run`. Neither would be measuring the ordering, which
// is the only decision here.
//
// ⚠️ **What it cannot see is the far end of the module-version step: that a real
// `go install pkg@version` populates `Main.Version` at all.** `go test` never
// runs in a binary one produced — the same limit cmd/hexarena-host's
// TestThisBinaryKnowsWhatItIs records when it refuses to assert a value, since
// `buildString()` answers `devel` under `go test` — so the build info below is
// fabricated and what is measured here is only what BuildOf *does* with a
// `Main.Version` once it has one.
//
// That far end is observable, and it was observed rather than assumed. It needs
// an install, which is why it is a recipe in a comment instead of a test: a
// **file-based proxy**, which needs no network and no tag.
//
//	V=v0.0.0-20260904055522-aaaabbbbcccc          # or a plain tag, v0.9.9
//	P=$TMP/proxy/github.com/vukyn/hexarena/@v
//	cp go.mod $P/$V.mod
//	echo '{"Version":"'$V'","Time":"2026-09-04T05:55:22Z"}' > $P/$V.info
//	# zip the tree under the prefix github.com/vukyn/hexarena@$V/
//	GOBIN=$TMP/bin GOFLAGS=-mod=mod GOPROXY=file://$TMP/proxy GOSUMDB=off \
//	  go install github.com/vukyn/hexarena/cmd/hexarena-host@$V
//
// Measured on 2026-09-04, and both halves of the diagnosis held: `go version -m`
// on the installed binary lists **one `mod` line and no `vcs.` setting at all**,
// and the binary answers `hexarena-host aaaabbbbcccc` where the published
// v0.0.0-20260904055522-983dbc63ceef — still in this machine's module cache,
// which is how the before-picture came for free — answers `build devel` and
// `flag provided but not defined: -version`.
//
// ⚠️ **Pick a version string the module cache has never seen.** The cache is
// consulted ahead of GOPROXY, so reusing the published pseudo-version installs
// the *old* code out of the cache and silently measures nothing — which is
// exactly what the first attempt did.
//
// ⚠️ **What it also cannot see** is that a binary actually calls this rather
// than spelling the fallback out again — that is held by there being no second
// copy of it in the module, which is why the function moved here in the first
// place.
func TestTheBuildStringFallsBackInOneOrder(t *testing.T) {
	// ⚠️ The vacuity guard for the near-collision, and it is the reason every
	// row below asserts a value rather than the absence of a wrong one:
	// `Unstamped` is `devel` and a module version that says nothing is
	// `(devel)`, two strings two parentheses apart. A test written as "not
	// devel" would pass on either of them.
	if Unstamped == moduleDevel {
		t.Fatalf("Unstamped and moduleDevel are both %q, so no row below can tell "+
			"a refused module version from the answer given when there is none", Unstamped)
	}
	stamped := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.modified", Value: "false"},
	}}
	dirty := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.modified", Value: "true"},
	}}
	silent := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "GOARCH", Value: "arm64"}}}
	// The three shapes a module version arrives in. ⚠️ **None of them carries a
	// vcs setting**, and that is the fact this whole step exists for rather than
	// an omission in the fixture: `go install pkg@version` records no VCS
	// metadata at all, because the toolchain stamps that only when it builds out
	// of a local checkout.
	released := &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}
	installed := &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20260904055522-983dbc63ceef"}}
	// ⚠️ **The LITERAL, never the constant.** A fixture built from moduleDevel
	// would be comparing the constant against itself: changing the constant
	// changes this reading with it, the refusal still matches, and the row stays
	// green while the code no longer refuses what the toolchain actually writes.
	// The string here is what `go build` outside a VCS checkout really reports,
	// so this row fails if the constant stops being that.
	locallyBuilt := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	// And the two that carry a module version *beside* a revision, which is what
	// a `go build` in a real checkout of a tagged module looks like. The
	// revision is the one printed, so these are the rows that pin step 2 above
	// step 3 rather than merely below step 1.
	bothClean := &debug.BuildInfo{
		Main:     debug.Module{Version: "v0.1.0"},
		Settings: stamped.Settings,
	}
	bothDirty := &debug.BuildInfo{
		Main:     debug.Module{Version: "v0.0.0-20260904055522-983dbc63ceef"},
		Settings: dirty.Settings,
	}

	for _, each := range []struct {
		what    string
		stamp   string
		info    *debug.BuildInfo
		want    string
		because string
	}{
		{
			what: "a release, stamped by -ldflags", stamp: "v0.4.0", info: stamped, want: "v0.4.0",
			because: "a stamp is a deliberate act by whoever produced the binary, so nothing may override it",
		},
		{
			what: "a stamped binary built from a dirty tree", stamp: "v0.4.0", info: dirty, want: "v0.4.0",
			because: "the stamp still wins; the person who stamped it said what it is",
		},
		{
			what: "go build with no stamp", stamp: "", info: stamped, want: "0123456789ab",
			because: "the revision is what a person compares across two machines, abbreviated to twelve like the data digest beside it",
		},
		{
			what: "go build from a dirty tree", stamp: "", info: dirty, want: "0123456789ab+dirty",
			because: "two people comparing builds are usually comparing a release against a working copy, and the working copy has to say so",
		},
		{
			what: "go install of a tagged release", stamp: "", info: released, want: "v0.1.0",
			because: "go install stamps no VCS settings, so the module version is all a binary from the proxy has — and a tag is the case -version exists to serve",
		},
		{
			what: "go install pkg@latest, at a commit no tag names", stamp: "", info: installed, want: "983dbc63ceef",
			because: "a pseudo-version is trimmed to its revision, which is the same twelve characters a go build at that commit prints, so two peers on one commit read each other one string",
		},
		{
			what: "a stamped binary installed from the proxy", stamp: "v0.4.0", info: installed, want: "v0.4.0",
			because: "the stamp still wins over a module version, for the same reason it wins over a revision",
		},
		{
			what: "go build in a checkout of a tagged module", stamp: "", info: bothClean, want: "0123456789ab",
			because: "the revision is checked before the module version: a checkout knows the exact commit and a tag only knows which release it is somewhere after",
		},
		{
			what: "go build from a dirty tree of a tagged module", stamp: "", info: bothDirty, want: "0123456789ab+dirty",
			because: "the revision still wins, and it is the reading that can say the tree was modified — a module version never can",
		},
		{
			what: "go build outside a VCS checkout, which reports (devel)", stamp: "", info: locallyBuilt, want: Unstamped,
			because: "(devel) is a module version that says nothing, so it is refused and the fallback runs on; accepting it would print (devel) where this prints devel",
		},
		{
			what: "go run, which records no revision", stamp: "", info: silent, want: Unstamped,
			because: "the ordinary case during development, so it is a word rather than an empty string beside the digest",
		},
		{
			what: "a binary with no build info at all", stamp: "", info: nil, want: Unstamped,
			because: "a blank on that line reads as a bug in the printing rather than a fact about the binary",
		},
	} {
		if got := BuildOf(each.stamp, each.info); got != each.want {
			t.Errorf("%s: build string %q, want %q — %s", each.what, got, each.want, each.because)
		}
	}
}

// TestAPseudoVersionIsTrimmedAndNothingElseIs is the recognition half, over the
// shapes a module version arrives in — including the near misses, which is the
// half a table of the two happy cases cannot give.
//
// ⚠️ **The trim is not a width fix, measured rather than assumed.** A
// pseudo-version is forty-one cells against a revision's twelve, and the widest
// place either lands is the client's join screen: `this machine — data <12> ·
// build <n>` measures 61 cells with the eighteen-character fixture, which is 84
// with a whole pseudo-version, against the 119 that screen's floor leaves. So it
// **fits**, and the floor is not the argument. The argument is that Build is the
// one of the three numbers nothing acts on — a person reads it to another person
// — and twelve characters read aloud where forty-one do not. Trimming also keeps
// the trimmed value *identical* to what a `go build` at the same commit prints,
// so the two ways of getting this binary announce one string rather than two.
//
// What it can see: which shapes are recognised, that a recognised one is cut to
// its revision, and that everything else is returned whole rather than guessed
// at.
//
// What it cannot see: whether the toolchain still writes the shape this
// recognises. That is a fact about `cmd/go` and not about this module, which is
// why the fall-through returns the version whole — an unrecognised shape prints
// long and true rather than short and wrong.
func TestAPseudoVersionIsTrimmedAndNothingElseIs(t *testing.T) {
	for _, each := range []struct {
		what    string
		version string
		want    string
		because string
	}{
		{
			what:    "the plain form, for a commit no tag is anywhere before",
			version: "v0.0.0-20260904055522-983dbc63ceef", want: "983dbc63ceef",
			because: "the trailing field is the revision, and it is the only part of one of these anybody reads out loud",
		},
		{
			what:    "the form after a tag, which carries a patch bump and a .0",
			version: "v0.1.1-0.20260904055522-983dbc63ceef", want: "983dbc63ceef",
			because: "all three pseudo-version forms end in the same two fields, so the shape is recognised from the end rather than parsed from the front",
		},
		{
			what:    "the form after a pre-release tag",
			version: "v0.2.0-rc1.0.20260904055522-983dbc63ceef", want: "983dbc63ceef",
			because: "the extra dots are in a field this never looks at",
		},
		{
			what:    "a real tag, which is the case -version exists to serve",
			version: "v0.1.0", want: "",
			because: "a tag is what a friend is told to install and it is six characters; trimming it would throw away the only version anybody chose",
		},
		{
			what:    "a pre-release tag",
			version: "v0.2.0-rc1", want: "",
			because: "two fields, so it cannot be a pseudo-version, and there is nothing in it to cut",
		},
		{
			what:    "a revision one character too long",
			version: "v0.0.0-20260904055522-983dbc63ceefa", want: "",
			because: "it recognises rather than parses, so a width that is not the toolchain's is returned whole rather than truncated to look right",
		},
		{
			what:    "a revision that is not hex",
			version: "v0.0.0-20260904055522-983dbc63ceez", want: "",
			because: "a revision is lowercase hex and nothing else, so a field that is not one is not a revision",
		},
		{
			what:    "an upper-case revision",
			version: "v0.0.0-20260904055522-983DBC63CEEF", want: "",
			because: "the toolchain writes these in lowercase, so an upper-case field came from something else",
		},
		{
			what:    "a timestamp of the wrong width",
			version: "v0.0.0-2026090405552-983dbc63ceef", want: "",
			because: "yyyymmddhhmmss is fourteen digits; thirteen is a field this has no reading of",
		},
		{
			what:    "a timestamp that is not digits",
			version: "v0.0.0-2026090405552x-983dbc63ceef", want: "",
			because: "the guard is what keeps a twelve-hex-character tail from being read as a revision in any version that happens to end in one",
		},
		{
			what:    "fourteen digits with something other than a dot in front of them",
			version: "v0.0.0-x20260904055522-983dbc63ceef", want: "",
			because: "the toolchain separates a pre-release from the timestamp with a dot, so a field merely ending in fourteen digits is not a pseudo-version's",
		},
		{
			what:    "a two-field version whose tail is twelve hex characters",
			version: "v0.0.0-983dbc63ceef", want: "",
			because: "without the timestamp there is no pseudo-version, and this is the shape the field count refuses",
		},
		{
			what:    "an empty version",
			version: "", want: "",
			because: "nothing to recognise; BuildOf refuses this one before it gets here, and this says so from the other end",
		},
	} {
		if got := revisionIn(each.version); got != each.want {
			t.Errorf("%s: %q gave the revision %q, want %q — %s",
				each.what, each.version, got, each.want, each.because)
		}
	}
	// And the number the trim rests on, written down rather than left implicit:
	// the toolchain's abbreviation in a pseudo-version is the same width BuildOf
	// cuts a vcs.revision to, which is what makes an installed binary and a
	// locally built one at one commit announce one string.
	const anInstall = "v0.0.0-20260904055522-983dbc63ceef"
	fromCheckout := BuildOf("", &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "983dbc63ceef54e0a26710d077cca5cf60ec5f9d"},
		{Key: "vcs.modified", Value: "false"},
	}})
	if fromProxy := BuildOf("", &debug.BuildInfo{Main: debug.Module{Version: anInstall}}); fromProxy != fromCheckout {
		t.Errorf("the same commit announces %q when installed and %q when built, so two "+
			"people on one commit have two strings to compare", fromProxy, fromCheckout)
	}
	if len(fromCheckout) != revisionLength {
		t.Errorf("a revision announces %d characters and revisionLength is %d, so the two "+
			"readings agree by accident rather than by that constant", len(fromCheckout), revisionLength)
	}
}

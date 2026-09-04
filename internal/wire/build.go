package wire

import (
	"runtime/debug"
	"strings"
)

// Unstamped is what a binary that was never stamped and knows nothing about its
// own source calls itself. It is a word rather than an empty string because it
// goes on a screen beside the data digest, and a blank there reads as a bug in
// the printing rather than as a fact about the binary.
const Unstamped = "devel"

// moduleDevel is what the toolchain writes into debug.BuildInfo.Main.Version
// when a binary was not built out of a module *version*: a plain `go build` in a
// directory with no VCS metadata, and every test binary.
//
// ⚠️ **It is not Unstamped and the two must never be tidied into one another.**
// They differ by two parentheses and they are two different kinds of thing —
// this one is a *reading to be refused*, Unstamped is the *answer given once
// every reading has been refused*. Accepting it would make a locally built
// binary announce `(devel)` where it announces a revision today, which is
// strictly less than the binary knows about itself. Measured: a `go build` of
// this module outside a checkout answers exactly `(devel)` here.
const moduleDevel = "(devel)"

// revisionLength is how much of a commit hash is printed. Twelve characters is
// what git's own abbreviations settle around on a repository of this size, and it
// is the length seed.Digest.Short prints beside it — two short hashes of the same
// width read as a pair.
const revisionLength = 12

// pseudoStampLength is how many digits the timestamp field of a module
// pseudo-version has: yyyymmddhhmmss.
const pseudoStampLength = 14

// BuildOf is the build string a binary announces, from the four sources in the
// order they are trusted. It is **pure** so the ordering is testable without a
// linker: the stamp is a parameter and the build info is a parameter, and
// neither is read from the process here.
//
//  1. **The -ldflags stamp**, if there is one. It is a deliberate act by whoever
//     produced the binary, so nothing may override it.
//  2. **The VCS revision** the toolchain records — runtime/debug's
//     `vcs.revision`, with `+dirty` appended when `vcs.modified` says the tree
//     had uncommitted changes. ⚠️ The dirty marker is worth the characters: two
//     people comparing builds over a LAN game are usually comparing one release
//     against one working copy, and a working copy that says nothing about being
//     modified is the one that wastes the conversation.
//  3. **The module version**, `info.Main.Version` — which is what a binary
//     produced by `go install pkg@version` has and the only thing it has.
//     ⚠️ **`go install` stamps no VCS settings at all**: the toolchain records
//     those only when it builds out of a local checkout, so step 2 is empty for
//     every binary anybody installs from the proxy. Measured on the published
//     module — `go install github.com/vukyn/hexarena/cmd/hexarena-host@latest`
//     printed `build devel` on its banner while the version
//     `v0.0.0-20260904055522-983dbc63ceef` had scrolled past during the
//     download. The toolchain knew the whole time and this step is the one that
//     reads it.
//  4. **Unstamped**, for a `go run` — which records neither a revision nor a
//     module version, so this is the ordinary case during development rather
//     than an error path.
//
// ⚠️ **It lives here and the stamp does not, which is the whole shape of the
// split.** Version.Local's comment says wire has no way to know a build string —
// a version is stamped at build time and read by the binary's own main — and
// that stays true: what is here is the pure two-argument *derivation*, and each
// binary keeps its own `var build string` for the linker to write into and its
// own debug.ReadBuildInfo call. It moved out of cmd/hexarena-host because a
// second binary needs the same fallback, and a second spelling of a fallback is
// how two peers come to disagree about what they are.
func BuildOf(stamped string, info *debug.BuildInfo) string {
	if stamped != "" {
		return stamped
	}
	if info == nil {
		return Unstamped
	}
	revision, modified := "", false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision != "" {
		if len(revision) > revisionLength {
			revision = revision[:revisionLength]
		}
		if modified {
			return revision + "+dirty"
		}
		return revision
	}
	if version := info.Main.Version; version != "" && version != moduleDevel {
		if inside := revisionIn(version); inside != "" {
			return inside
		}
		return version
	}
	return Unstamped
}

// revisionIn is the twelve-character revision inside a module pseudo-version,
// or empty for a version that is not one.
//
// A pseudo-version is what the toolchain invents for a commit no tag names —
// `v0.0.0-20260904055522-983dbc63ceef` — and its last hyphen-separated field is
// the **same** abbreviation BuildOf cuts a vcs.revision to. Trimming to it is
// what makes `go install pkg@latest` and a local `go build` **at one commit
// announce one string**. Without the trim two people sitting on the same commit
// would read each other forty-one characters and twelve, with no way to tell
// that they matched — which is the exact confusion Version.Build exists to
// remove, since it is the one of the three numbers nothing acts on and a person
// is the only thing that reads it.
//
// ⚠️ **A tag is left whole**, and that is the case -version exists to serve:
// `v0.1.0` is what a friend is told to install and it is six characters already.
// The two are told apart by shape rather than by a version-parsing library,
// because the trailing revision is the only part of a pseudo-version anybody
// reads out loud and recognising it needs no semver.
//
// ⚠️ **The timestamp is not always a hyphen-separated field of its own, and
// assuming it was is a bug this shipped with for one test run.** There are three
// forms, and only the first has one:
//
//	v0.0.0-20260904055522-983dbc63ceef        no tag anywhere before the commit
//	v0.1.1-0.20260904055522-983dbc63ceef      after a tag with no pre-release
//	v0.2.0-rc1.0.20260904055522-983dbc63ceef  after a pre-release tag
//
// The last two put the timestamp behind a `.`-separated pre-release in the same
// field, so it is recognised as *the last fourteen digits of that field* with a
// `.` or nothing in front of them. A split on hyphens alone saw the first form
// only, which is the one a repository with no tags produces — so the two forms
// this repository is about to start producing, the moment it is tagged, were
// exactly the ones it missed.
//
// ⚠️ It **recognises** rather than parses, so an unrecognised shape falls
// through to the whole version rather than to a guess. If the toolchain ever
// changed either field's width, this would stop matching and the full
// pseudo-version would be printed: long, and still true.
func revisionIn(version string) string {
	fields := strings.Split(version, "-")
	if len(fields) < 3 {
		return ""
	}
	revision, held := fields[len(fields)-1], fields[len(fields)-2]
	if len(revision) != revisionLength || !lowercaseHex(revision) {
		return ""
	}
	if len(held) < pseudoStampLength {
		return ""
	}
	stamp, before := held[len(held)-pseudoStampLength:], held[:len(held)-pseudoStampLength]
	if !decimalDigits(stamp) {
		return ""
	}
	// Nothing before the timestamp, or a pre-release the toolchain separated
	// from it with a dot. Anything else is a field that merely ends in fourteen
	// digits, which is not the same thing.
	if before != "" && !strings.HasSuffix(before, ".") {
		return ""
	}
	return revision
}

// lowercaseHex reports whether every character is a lowercase hex digit, which
// is the only form the toolchain writes a revision in.
func lowercaseHex(text string) bool {
	for _, letter := range text {
		if (letter < '0' || letter > '9') && (letter < 'a' || letter > 'f') {
			return false
		}
	}
	return true
}

// decimalDigits reports whether every character is a decimal digit.
func decimalDigits(text string) bool {
	for _, letter := range text {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return true
}

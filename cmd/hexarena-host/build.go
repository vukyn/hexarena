package main

import (
	"runtime/debug"
)

// build is the version string this binary announces, and it is a variable rather
// than a constant so a release can stamp it:
//
//	go build -ldflags "-X main.build=v0.4.0" ./cmd/hexarena-host
//
// It is **printed and never acted on** — wire.Version.Build is the one of the
// three numbers with nothing to decide, and internal/wire has a test that joins
// two peers whose builds disagree to keep it that way. What it is for is a person
// reading two screens and working out which machine to update.
var build string

// unstamped is what a binary that was never stamped and knows nothing about its
// own source calls itself. It is a word rather than an empty string because it
// goes on a screen beside the data digest, and a blank there reads as a bug in
// the printing rather than as a fact about the binary.
const unstamped = "devel"

// revisionLength is how much of a commit hash is printed. Twelve characters is
// what git's own abbreviations settle around on a repository of this size, and it
// is the length seed.Digest.Short prints beside it — two short hashes of the same
// width read as a pair.
const revisionLength = 12

// buildOf is the build string, from the three sources in the order they are
// trusted. It is **pure** so the ordering is testable without a linker: the
// stamp is a parameter and the build info is a parameter, and neither is read
// from the process here.
//
//  1. **The -ldflags stamp**, if there is one. It is a deliberate act by whoever
//     produced the binary, so nothing may override it.
//  2. **The VCS revision** net/http's own toolchain records — runtime/debug's
//     `vcs.revision`, with `+dirty` appended when `vcs.modified` says the tree
//     had uncommitted changes. ⚠️ The dirty marker is worth the characters: two
//     people comparing builds over a LAN game are usually comparing one release
//     against one working copy, and a working copy that says nothing about being
//     modified is the one that wastes the conversation.
//  3. **"devel"**, for a `go run` — which records no revision at all, so this is
//     the ordinary case during development rather than an error path.
func buildOf(stamped string, info *debug.BuildInfo) string {
	if stamped != "" {
		return stamped
	}
	if info == nil {
		return unstamped
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
	if revision == "" {
		return unstamped
	}
	if len(revision) > revisionLength {
		revision = revision[:revisionLength]
	}
	if modified {
		return revision + "+dirty"
	}
	return revision
}

// buildString is buildOf over this process, which is the one impure line of it.
func buildString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return buildOf(build, info)
}

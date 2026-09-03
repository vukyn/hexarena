package main

import (
	"runtime/debug"

	"github.com/vukyn/hexarena/internal/wire"
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
//
// ⚠️ **The stamp stays here and the derivation does not.** wire.BuildOf is the
// three-step fallback, moved there when the game client needed the same one; a
// stamp is written by the linker into a *binary's own* variable, which is why
// wire.Version.Local takes a build string as a parameter and this file is what
// supplies it.
var build string

// buildString is wire.BuildOf over this process, which is the one impure line of
// it.
func buildString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		info = nil
	}
	return wire.BuildOf(build, info)
}

package main

import (
	"strings"
	"testing"
)

// TestThisBinaryKnowsWhatItIs is the one impure line of the build string, and it
// asserts only what is true under every way of running the suite: something is
// said, and it is not blank. ⚠️ It cannot assert a value — `go test` and a
// stamped release produce different ones by design, which is the whole point of
// wire.BuildOf being pure.
//
// ⚠️ **The ordering test moved with the function.** The three sources and the
// order they are trusted in are now held by
// internal/wire's TestTheBuildStringFallsBackInOneOrder, because the derivation
// is shared with cmd/hexarena-tui and a second copy of the table would be a
// second declaration of the fallback. What stays here is the half that is about
// *this binary*: that it reads its own stamp and its own build info.
func TestThisBinaryKnowsWhatItIs(t *testing.T) {
	said := buildString()
	if strings.TrimSpace(said) == "" {
		t.Error("this binary announces an empty build string, which reads on screen as a bug in the printing")
	}
	t.Logf("this binary announces itself as %q", said)
}

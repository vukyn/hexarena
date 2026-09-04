package main

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The two numbers on the join screen
//
// A room refuses a peer whose data digest differs, and until #276 the digest
// was printed on **one side only**: cmd/hexarena-host prints `data <short>` on
// its banner and this client printed nothing at all, so a refused player had
// nothing to compare and had to ask. Both refusals that turn on a version say
// to read a line the client did not have — RefusalDataMismatch says *read the
// data line on each*, RefusalProtocolMismatch says it about the build line.
//
// ⚠️ **What no test here can see, because the protocol does not carry it.**
// wire.Refused holds a Code and nothing else, Welcome holds no version, and a
// refused client never receives a Welcome — so this end cannot learn the
// *host's* digest and nothing below asserts a comparison. What is measured is
// that this end draws **its own**, truthfully. Making the comparison automatic
// would be a field on wire.Refused, which is a protocol change with a golden
// behind it and is not this change.

// TestTheJoinScreenDrawsThisBinarysOwnVersion is the wiring: the screen a reader
// actually reaches draws the digest seed.DataDigest answers with and the build
// string this binary announces, in the reader's language.
//
// What it can see: that Refresh fills the field from the real derivation, that
// both values reach the drawn body, and that they reach it in the positions the
// wording puts them in — the whole rendered line is matched, not the two values
// separately, so a screen drawing them in the wrong slots fails.
//
// What it cannot see: a hardcoded digest that happens to be today's correct one.
// That is the next test's subject, and it needs a value the data cannot produce.
//
// ⚠️ **And it cannot see a hardcoded BUILD string, measured rather than
// assumed.** Replacing `wire.Local(buildString())` in Refresh with
// `wire.Local("devel")` leaves all three tests here green, because `devel` is
// exactly what buildString() answers under `go test` — the suite cannot be run
// in a state where the two differ, which is the same limit
// TestThisBinaryKnowsWhatItIs states when it refuses to assert a value. The
// digest half has no such gap: a fabricated digest is a value the data cannot
// produce, which is why the next test exists at all.
//
// ⚠️ **The vacuity guard is the length check, and it is not decoration.**
// Digest.Short is a substring expression, so a screen drawing an *empty* short
// digest would be asserted against `strings.Contains(body, "")`, which is true
// of every screen there has ever been. So the twelve characters are measured
// first, and then the same wording rendered with an empty digest is asserted
// **absent** — which is the failure "draw an empty one" produces, stated as an
// assertion rather than left to a length.
func TestTheJoinScreenDrawsThisBinarysOwnVersion(t *testing.T) {
	digest, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	short, build := digest.Short(), buildString()
	if len(short) != shortDigestLength {
		t.Fatalf("a short digest is %d characters and this one is %d (%q), so every "+
			"Contains below is over the wrong thing", shortDigestLength, len(short), short)
	}
	if strings.TrimSpace(build) == "" {
		t.Fatal("this binary announces an empty build string, so the line below is half blank")
	}
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		joining := base.enter(screenJoin)
		if joining.join.Local.Data.Digest != digest {
			t.Errorf("in %s the join screen holds the digest %s and the embedded data digests "+
				"to %s", lang, joining.join.Local.Data.Short(), short)
		}
		if joining.join.Local.Build != build {
			t.Errorf("in %s the join screen holds the build %q and this binary announces %q",
				lang, joining.join.Local.Build, build)
		}
		drawn := drawnBody(joining)
		if want := joining.text(i18n.JoinVersion, short, build); !strings.Contains(drawn, want) {
			t.Errorf("in %s the join screen does not draw %q:\n%s", lang, want, drawn)
		}
		// The anti-vacuity half: the same wording with nothing in the digest's
		// slot must not be on the screen.
		if blank := joining.text(i18n.JoinVersion, "", build); strings.Contains(drawn, blank) {
			t.Errorf("in %s the join screen draws an empty data digest (%q):\n%s",
				lang, blank, drawn)
		}
	}
}

// TestTheJoinScreenDrawsTheDigestItHoldsRatherThanOneOfItsOwn is the half a
// screen reading the real digest cannot give: it hands the screen a digest the
// embedded data **cannot** produce and asks whether that is what gets drawn.
//
// ⚠️ **This is the test that catches a hardcoded digest**, which the one above
// cannot: a literal `"df3bed25a5c5"` in the drawing passes every assertion up
// there for exactly as long as the shipped data does not change, and then
// becomes a screen confidently reporting the wrong number to somebody trying to
// work out why they were refused. So the fabricated value is asserted **on**
// the screen and the shipped one asserted **off** it.
//
// What it can see: that View reads joinScreen.Local rather than any other
// source, for both of the two values.
//
// What it cannot see: whether Refresh fills that field correctly — the field is
// hand-set here. That is the previous test's subject, and the two together are
// the whole path.
func TestTheJoinScreenDrawsTheDigestItHoldsRatherThanOneOfItsOwn(t *testing.T) {
	shipped, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	var fabricated seed.Digest
	for index := range fabricated {
		fabricated[index] = 0xab
	}
	// The guard that makes the two assertions below different questions.
	if fabricated == shipped {
		t.Fatalf("the fabricated digest is the shipped one (%s), so this test cannot tell a "+
			"drawing that reads its field from one that hardcodes a number", shipped.Short())
	}
	const fabricatedBuild = "not-this-build"
	if fabricatedBuild == buildString() {
		t.Fatalf("the fabricated build string is what this binary announces (%q), so the "+
			"build half of this test measures nothing", fabricatedBuild)
	}
	for _, lang := range i18n.Langs() {
		base, _, _ := start(t, lang)
		joining := base.enter(screenJoin)
		joining.join.Local = wire.Version{
			Protocol: wire.Protocol,
			Build:    fabricatedBuild,
			Data:     wire.Digest{Digest: fabricated},
		}
		drawn := drawnBody(joining)
		want := joining.text(i18n.JoinVersion, fabricated.Short(), fabricatedBuild)
		if !strings.Contains(drawn, want) {
			t.Errorf("in %s the join screen was handed %q and does not draw it:\n%s",
				lang, want, drawn)
		}
		if strings.Contains(drawn, shipped.Short()) {
			t.Errorf("in %s the join screen still draws the shipped digest %s after being "+
				"handed another, so the number on it is not the one it holds:\n%s",
				lang, shipped.Short(), drawn)
		}
	}
}

// TestTheJoinScreenSaysNothingAboutAVersionItCouldNotRead holds the sentinel
// decision behind joinScreen.Local: an empty Build means "not read", because
// wire.BuildOf never answers with a blank.
//
// ⚠️ **The point is what is drawn instead of a zero digest.** A zero
// wire.Version has a perfectly well-formed twelve-character Short —
// `000000000000` — so a drawing that did not branch would put a plausible
// number on the screen for a binary that had failed to read its own data, which
// is worse than saying nothing. The positive half runs first, or a screen that
// drew the line in no state at all would pass this.
//
// What it can see: that the line is conditional and on which field.
//
// What it cannot see: the failure itself. seed.DataDigest cannot fail in a
// built binary — go:embed would have failed at build time — so the state is
// reached by zeroing the field rather than by breaking the embed.
func TestTheJoinScreenSaysNothingAboutAVersionItCouldNotRead(t *testing.T) {
	shipped, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	base, _, _ := start(t, i18n.Vi)
	joining := base.enter(screenJoin)
	if !strings.Contains(drawnBody(joining), shipped.Short()) {
		t.Fatalf("the ordinary join screen draws no digest, so the absence below proves "+
			"nothing:\n%s", drawnBody(joining))
	}
	var zeroed seed.Digest
	joining.join.Local = wire.Version{}
	drawn := drawnBody(joining)
	if strings.Contains(drawn, shipped.Short()) {
		t.Errorf("a join screen holding no version still draws the shipped digest %s:\n%s",
			shipped.Short(), drawn)
	}
	if strings.Contains(drawn, zeroed.Short()) {
		t.Errorf("a join screen holding no version draws the digest of nothing (%s), which "+
			"is a well-formed number and a lie:\n%s", zeroed.Short(), drawn)
	}
}

// shortDigestLength is how many characters seed.Digest.Short answers with. It is
// written down here so the vacuity guard above is a comparison rather than a
// sentence in a comment.
const shortDigestLength = 12

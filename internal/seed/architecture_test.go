package seed_test

import (
	"encoding/hex"
	"runtime"
	"testing"

	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The mirror across architectures
//
// PvP rests on one assumption and it is not a small one: two people on two
// machines run the *same* battle from the same seed, and the room and each
// client's mirror agree turn by turn on a digest of the events. Friends are not
// all on one machine — an arm64 laptop against an amd64 desktop is the ordinary
// case on a LAN — so if the engine produced so much as a different rounding on
// one of them, every match between those two people would break on its first
// turn with each side certain the other had diverged.
//
// The engine is built so that it cannot: `internal/core` is integer arithmetic
// throughout — ratios are parts per thousand against scale.Base, there are no
// floats, no clock, no map iteration reaching an output — and randomness is a
// passed-in *rng.Source. Every one of those is a rule with a test behind it. What
// none of them says is the thing PvP actually needs, which is the whole point of
// this file: **the bytes two peers exchange are the same bytes on both
// architectures.**
//
// ## How this is proven, and what it does not prove
//
// The digest below is a **committed constant**. Running this test on any machine
// compares that machine's engine against it, so the proof is the test being green
// in two places rather than anything the test does on its own:
//
//	go test ./internal/seed -run Architecture              # this machine
//	GOARCH=amd64 go test ./internal/seed -run Architecture  # the other one
//
// ⚠️ **On an Apple machine the second runs under Rosetta**, and that is worth
// saying rather than glossing: it is a real amd64 binary executing amd64
// instructions, so it exercises the compiler's amd64 code generation, its integer
// widths and its calling convention — which is where an architecture difference
// in code like this would come from. It is **not** a second silicon vendor, and a
// difference that lived in Intel's hardware rather than in the generated code
// would not be caught here. For an engine with no floating point in it that gap
// is very small; it is not nothing, and it is why this comment says which of the
// two was measured.
//
// ## ⚠️ Why the digest and not the existing goldens
//
// `TestBattleReplayGolden` already pins a whole battle from seed 11 and would
// catch a divergence too. It pins **rendered text**, which is one derivation away
// from what two peers actually send each other: a change to how an event is
// worded moves that golden and nothing about the protocol, and a change to how an
// event is *marshalled* moves the digest and not necessarily the wording. The
// digest is what room.resolved computes and what socket.Mirror checks, so it is
// the figure a match is actually decided by.

// theWholeBattleDigest is `wire.DigestEvents` over every event of the shipped
// battle from seed 11, as hex.
//
// ⚠️ **It moves when the balance data moves**, and that is correct rather than
// annoying: a different skill book is a different battle, and two peers on
// different data are already refused at the gate by the data digest. What it must
// never do is differ between two machines running the same commit.
const theWholeBattleDigest = "33cb730dd32bc9e9d937ec46a068597de406bda9290c85440ea32b594291c4c0"

// TestTheEventDigestIsTheSameOnEveryArchitecture is the assumption PvP rests on,
// pinned.
func TestTheEventDigestIsTheSameOnEveryArchitecture(t *testing.T) {
	fight, err := seed.NewBattle(11)
	if err != nil {
		t.Fatalf("open the shipped battle: %v", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(4000); err != nil {
		t.Fatalf("run the battle: %v", err)
	}
	events := fight.Drain()
	// ⚠️ **There is no "did it produce any events" guard here, and it was written
	// and deleted.** A battle that produced none would digest to sha256 of
	// nothing and mismatch the constant like any other divergence — so the guard
	// could not be shown to matter: a mutation disabling it changed no test on
	// either architecture. One assertion a mutation reddens is worth more than
	// two that cover for each other.
	digest, err := wire.DigestEvents(events)
	if err != nil {
		t.Fatalf("digest the events: %v", err)
	}
	got := hex.EncodeToString(digest[:])
	if got != theWholeBattleDigest {
		t.Errorf("on %s/%s the shipped battle from seed 11 digests to\n  %s\nand the constant "+
			"says\n  %s\n\nIf the balance data changed, update the constant. If it did not, "+
			"this architecture disagrees with the one that wrote it and no match between the "+
			"two would survive its first turn.",
			runtime.GOOS, runtime.GOARCH, got, theWholeBattleDigest)
	}
	t.Logf("%s/%s: %d events digest to %s", runtime.GOOS, runtime.GOARCH, len(events), got)
}

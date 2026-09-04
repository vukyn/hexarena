package wire

import (
	"testing"

	"github.com/vukyn/hexarena/internal/seed"
)

// otherDigest is a data digest that is not fixtureDigest, byte for byte
// different at every index so nothing about the comparison can be accidental.
func otherDigest() Digest {
	var raw seed.Digest
	for index := range raw {
		raw[index] = byte(255 - index)
	}
	return Digest{Digest: raw}
}

// TestTheThreeVersionNumbersAreActedOnDifferently is the design record's table
// turned into four cases, and the *differences* between them are the whole
// point: one number cannot report three failures, and a peer that is wrong about
// two of them has to be told about one particular one.
//
//   - A different **Build** is **accepted**. That is the real assertion here,
//     and it is the one an eye reading the code cannot make: Build is documented
//     as printed and never acted on, and this is what holds it that way. A check
//     that grew a build comparison would look perfectly reasonable and would
//     refuse every peer built from a different commit of identical data.
//   - A wrong **Protocol** gives CodeProtocolMismatch: these two cannot talk.
//   - A wrong **data digest** gives CodeDataMismatch: these two would not
//     simulate the same battle, and the gate exists to say so before two people
//     have spent ten minutes on a battle that was never the same battle twice.
//   - Wrong about **both** gives the *protocol* code, which is what pins the
//     order. A peer that cannot speak the protocol must not be told its data is
//     wrong: it may not be able to read the refusal, and the update it needs is
//     the binary either way.
func TestTheThreeVersionNumbersAreActedOnDifferently(t *testing.T) {
	local := Version{Protocol: Protocol, Build: "local-build", Data: fixtureDigest()}
	cases := []struct {
		name string
		peer Version
		want Code
	}{
		{
			name: "the same in all three",
			peer: local,
			want: CodeNone,
		},
		{
			name: "a different build and nothing else",
			peer: Version{Protocol: Protocol, Build: "some-other-build", Data: fixtureDigest()},
			want: CodeNone,
		},
		{
			name: "no build at all",
			peer: Version{Protocol: Protocol, Data: fixtureDigest()},
			want: CodeNone,
		},
		{
			name: "an older protocol",
			peer: Version{Protocol: Protocol - 1, Build: local.Build, Data: fixtureDigest()},
			want: CodeProtocolMismatch,
		},
		{
			name: "a newer protocol",
			peer: Version{Protocol: Protocol + 1, Build: local.Build, Data: fixtureDigest()},
			want: CodeProtocolMismatch,
		},
		{
			name: "edited data",
			peer: Version{Protocol: Protocol, Build: local.Build, Data: otherDigest()},
			want: CodeDataMismatch,
		},
		{
			name: "no data digest at all",
			peer: Version{Protocol: Protocol, Build: local.Build},
			want: CodeDataMismatch,
		},
		{
			name: "wrong about both, which is what pins the order",
			peer: Version{Protocol: Protocol + 1, Build: "some-other-build", Data: otherDigest()},
			want: CodeProtocolMismatch,
		},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			if got := local.Check(one.peer); got != one.want {
				t.Errorf("Check gave %s, want %s", got, one.want)
			}
			// The check is symmetric, which is what lets either end run it: a
			// client can gate a server's version with the same function.
			if got := one.peer.Check(local); got != one.want {
				t.Errorf("checked from the other end it gave %s, want %s", got, one.want)
			}
		})
	}
}

// TestTheDataDigestIsCheckedAsATypeRatherThanAsCharacters is why this package
// imports internal/seed at all.
//
// The gate compares two seed.Digest values, so the comparison is the compiler's
// and not a string's. What it guards against is the version of this that carries
// hex characters instead: that one accepts a digest mangled into the same
// characters by some other path, it accepts an **event** digest handed over by
// mistake, and neither mistake fails to compile. Here both do — which is a
// property a test can only demonstrate by exercising the one comparison that
// does exist.
func TestTheDataDigestIsCheckedAsATypeRatherThanAsCharacters(t *testing.T) {
	local, err := Local("local-build")
	if err != nil {
		t.Fatalf("read the local version: %v", err)
	}
	if local.Protocol != Protocol {
		t.Errorf("Local announced protocol %d, want %d", local.Protocol, Protocol)
	}
	if local.Build != "local-build" {
		t.Errorf("Local announced build %q", local.Build)
	}
	shipped, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("read the shipped data digest: %v", err)
	}
	// The one assertion about today's bytes, and it is an equality against the
	// package that owns them rather than a value written down here: a golden on
	// this digest would move on every balance commit while measuring nothing.
	if local.Data.Digest != shipped {
		t.Errorf("Local announced %s and the data digests to %s", local.Data.Short(), shipped.Short())
	}
	if local.Check(local) != CodeNone {
		t.Error("a peer built from this binary's own version was refused")
	}
	// And a digest that is the zero value is not the digest of the shipped data,
	// which is the vacuity a peer announcing nothing would slip through.
	empty := Version{Protocol: Protocol}
	if local.Check(empty) != CodeDataMismatch {
		t.Error("a peer announcing no data digest at all was not refused")
	}
}

// TestADataDigestSurvivesTheWireAsHex is the encoding half. The digest travels
// as the sixty-four characters seed.Digest.String declares — a form a person can
// read off two screens and a golden can hold on one line — and a malformed one
// is an error rather than a zero digest, because two peers agreeing on the
// digest of nothing is the one outcome this gate must never produce.
func TestADataDigestSurvivesTheWireAsHex(t *testing.T) {
	fixture := fixtureDigest()
	raw, err := fixture.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `"` + fixture.Digest.String() + `"`; string(raw) != want {
		t.Errorf("a digest marshalled as %s, want %s", raw, want)
	}
	var back Digest
	if err := back.UnmarshalJSON(raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back != fixture {
		t.Errorf("a digest came back as %s", back.Short())
	}
	for _, broken := range []string{
		`"not hex at all"`,
		`"0011"`,
		`""`,
		`7`,
		`null`,
		`"` + fixture.Digest.String() + `00"`,
	} {
		var out Digest
		if err := out.UnmarshalJSON([]byte(broken)); err == nil {
			t.Errorf("%s decoded to %s with no error", broken, out.Short())
		}
	}
}

// TestTheVersionReportIsOneShapeForBothBinaries pins the exact three lines
// -version prints, because both binaries print them from here and a person
// comparing two machines is comparing two of these.
//
// ⚠️ **It asserts the whole rendering rather than the three values separately**,
// which is what makes it a shape test: a report that put the digest where the
// protocol goes would satisfy three Contains and fail this. And the digest it is
// handed is **fabricated** — a value the embedded data cannot produce — so a
// report that printed the shipped digest instead of the one it was given would
// fail, which is the one thing a report built from the real version cannot see.
//
// What it can see: the order of the three, the labels, the indentation, that the
// digest is the short form rather than the sixty-four characters that ride the
// wire, that the program name is the caller's, and that the whole thing ends in
// a newline so a shell prompt lands on its own line.
//
// What it cannot see: that the two binaries call it with their own names rather
// than with each other's. That is one assertion in each binary's own suite,
// because a program name is a fact about a program.
func TestTheVersionReportIsOneShapeForBothBinaries(t *testing.T) {
	var fabricated seed.Digest
	for index := range fabricated {
		fabricated[index] = 0xab
	}
	shipped, err := seed.DataDigest()
	if err != nil {
		t.Fatalf("digest the embedded data: %v", err)
	}
	// The guard that makes the assertion below a measurement: a fabricated
	// digest equal to the shipped one could not tell a report reading its own
	// field from one printing a literal.
	if fabricated == shipped {
		t.Fatalf("the fabricated digest is the shipped one (%s), so this test cannot tell "+
			"a report that reads its version from one that hardcodes a number", shipped.Short())
	}
	version := Version{Protocol: 7, Build: "v9.9.9", Data: Digest{Digest: fabricated}}
	const want = "hexarena-host v9.9.9\n" +
		"  protocol 7\n" +
		"  data     abababababab\n"
	if got := version.Report("hexarena-host"); got != want {
		t.Errorf("the version report reads\n%q\nand the one shape both binaries print is\n%q", got, want)
	}
	// The protocol is the *value* rather than the constant, so a report that
	// read Protocol off the package instead of off the version it was handed
	// would fail. 7 is not Protocol, and this says so rather than leaving it to
	// the reader of the literal above.
	if version.Protocol == Protocol {
		t.Errorf("the fixture's protocol is this build's (%d), so the line above could be "+
			"reading the constant rather than the field", Protocol)
	}
	// And the digest on it is the short form, not the wire form. Asserted as a
	// length rather than left to the literal, because the difference between the
	// two is a substring and Contains would be true of either.
	if length := len(fabricated.Short()); length != 12 {
		t.Errorf("a short digest is %d characters here, so the literal above is over the "+
			"wrong form of it", length)
	}
}

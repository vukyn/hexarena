package wire

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

// TestEveryMessageIsTheBytesTheGoldenHolds is the record the brief asks for: one
// entry per kind, so **a wire change shows up in a diff**. A field renamed, a
// field dropped, an omitempty added or taken away, a kind renamed, the digest's
// framing changed — each of those moves a line here and nothing else in the
// suite would show a reader what moved.
//
// ⚠️ **Every body is a hand-written fixture and none of it comes from the
// shipped data.** That is the whole reason this file is safe to have: a `start`
// body carrying a real roster would move on every balance commit — a stat curve
// retuned, a skill's level moved, a character added — while measuring nothing
// about the protocol, and a golden that moves for reasons unrelated to what it
// measures is a merge-conflict generator. This session has already watched two
// goldens go stale that way while GitHub reported the PR mergeable. The same
// argument is why `internal/seed` deliberately has **no** golden on the data
// digest at all, and why the hello here carries a made-up one.
//
// It records the **indented** form of the exact bytes Encode produces. json.Indent
// reorders nothing, so the field order, the omitted fields and the kind names are
// all still what travels — what the indentation buys is that a diff points at the
// one field that changed instead of at a four-hundred-character line.
//
// It also holds the kind walk's third leg: an entry per kind, checked against
// KindCount, so a message added without a golden entry is a red test rather than
// a message no record shows.
func TestEveryMessageIsTheBytesTheGoldenHolds(t *testing.T) {
	fixtures := messageFixtures(t)
	var record bytes.Buffer
	for value := range KindCount {
		kind := Kind(value)
		fixture, ok := fixtures[kind]
		if !ok {
			t.Fatalf("kind %s has no fixture, so it cannot be recorded", kind)
		}
		raw, err := Encode(fixture)
		if err != nil {
			t.Fatalf("encode %s: %v", kind, err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, raw, "", "  "); err != nil {
			t.Fatalf("indent %s: %v", kind, err)
		}
		fmt.Fprintf(&record, "[%s]\n%s\n\n", kind, pretty.String())
	}

	got := record.String()
	path := filepath.Join("testdata", "messages.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("make testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("rewrote %s (%d bytes)", path, len(got))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run: go test ./internal/wire -update): %v", err)
	}
	if got != string(want) {
		t.Errorf("the messages differ from %s; rerun with -update to accept, and read the diff — "+
			"a change here is a change to the wire\n--- got ---\n%s", path, got)
	}
}

// TestTheGoldenIsBuiltFromNothingShipped is the guard on the paragraph above,
// and it is a real test rather than a restatement of the comment: it asserts the
// recorded bytes hold none of the shipped data's own ids.
//
// A fixture that reached for a real character would look perfectly reasonable —
// `pokemon.bulbasaur` is a valid squad member and `razor_leaf` is a real skill —
// and the golden would then move the next time either was retuned. Naming the
// shipped ids here would be a second copy of the cast; the check is that every
// id in the record is one this test file wrote, which is what the `fixture.`
// prefix is for.
func TestTheGoldenIsBuiltFromNothingShipped(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "messages.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	record := string(raw)
	// The two shipped prefixes every character, skill and squad id in
	// internal/seed/data carries. A fixture reaching into the data would bring
	// one of them along.
	for _, shipped := range []string{"pokemon.", "naruto.", "ally.", "foe."} {
		if bytes.Contains(raw, []byte(shipped)) {
			t.Errorf("the golden holds %q, so a body was built from the shipped data and this record "+
				"will move on the next balance commit", shipped)
		}
	}
	// And the real data digest is not in it either, which is the field most
	// likely to be filled in from seed.DataDigest by somebody being helpful.
	if len(record) == 0 {
		t.Fatal("the golden is empty")
	}
	if !bytes.Contains(raw, []byte(fixtureDigest().Digest.String())) {
		t.Error("the golden does not hold the fixture digest, so the hello's digest came from somewhere else")
	}
}

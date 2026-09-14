package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/plain"
)

// ⚠️ **This file is INSIDE the package**, unlike the rest of internal/tui's
// tests, and for one reason: inert is unexported and the reflection guard below
// has to walk the same struct it walks. The rendering assertions could live in
// tui_test; keeping the pair together is what makes the guard readable as the
// reason the rendering assertions are allowed to be short.

// clipboardWrite is OSC 52 — the payload that writes the reader's clipboard, for
// them to paste into a shell later. See internal/plain for the whole argument.
const clipboardWrite = "\x1b]52;c;cm0gLXJmIH4=\x07"

// forgery is what a crafted log is actually for. It is not an escape sequence at
// all: a carriage return and a newline are enough to write a line that reads
// exactly like the one cmd/hexarena prints after a successful --verify, which is
// the verdict this whole format asks a reader to trust.
const forgery = "\r\nverified: re-running seed 11 reproduced all 255 events exactly\n"

// TestEveryStringOnAnEventIsMadeInert is the guard that makes inert maintainable
// rather than a list somebody has to remember.
//
// ⚠️ **It walks battle.Event by reflection and fails when a string field is
// declared that inert does not clean.** The alternative — auditing which of the
// nine fields reaches a writer through which of forty format strings — is a
// judgement that was already made once and got `actor` wrong: the finding called
// it safe because it is "matched against unit ids", and Line's tag falls back to
// printing the raw id for anything the tags map does not hold.
//
// The mechanism is to set every string field to a hostile value, run inert, and
// require every one of them to come back fit to print. A field inert forgets is
// the one that comes back unchanged.
func TestEveryStringOnAnEventIsMadeInert(t *testing.T) {
	var event battle.Event
	value := reflect.ValueOf(&event).Elem()
	kind := value.Type()
	strings := 0
	for i := range kind.NumField() {
		if kind.Field(i).Type.Kind() != reflect.String {
			continue
		}
		strings++
		value.Field(i).SetString("x" + clipboardWrite)
	}
	if strings == 0 {
		t.Fatal("battle.Event declares no string fields, so this guard is measuring nothing")
	}
	cleaned := reflect.ValueOf(inert(event))
	for i := range kind.NumField() {
		if kind.Field(i).Type.Kind() != reflect.String {
			continue
		}
		got := cleaned.Field(i).String()
		if !plain.Inert(got) {
			t.Errorf("inert leaves Event.%s as %q, which a terminal would obey", kind.Field(i).Name, got)
		}
	}
	t.Logf("checked %d string fields on battle.Event", strings)
}

// TestACraftedLogRendersAsTextAndNothingElse is the local High, asserted on the
// bytes the renderer hands its caller.
//
// ⚠️ **`actor` and `target` are in the table on purpose**, against the finding's
// own judgement. Line's tag returns the raw id when the tags map has no entry,
// and TagsFromLog only fills it from Started records — so an id on any other kind
// of event is printed as the log wrote it.
func TestACraftedLogRendersAsTextAndNothingElse(t *testing.T) {
	for _, each := range []struct {
		what  string
		event battle.Event
	}{
		{"a name", battle.Event{Kind: battle.Started, Actor: "a", Name: "Bảo" + clipboardWrite}},
		{"a note", battle.Event{Kind: battle.Started, Actor: "a", Note: clipboardWrite}},
		{"a note on a skipped turn", battle.Event{Kind: battle.TurnSkipped, Actor: "a", Note: forgery}},
		{"a note on a departure", battle.Event{Kind: battle.Left, Actor: "a", Note: forgery}},
		{"a target", battle.Event{Kind: battle.Missed, Actor: "a", Target: clipboardWrite}},
		{"an actor nothing tagged", battle.Event{Kind: battle.TurnBegan, Actor: clipboardWrite}},
		{"a skill id", battle.Event{Kind: battle.Missed, Actor: "a", Skill: clipboardWrite}},
		{"a status id", battle.Event{Kind: battle.StatusTicked, Actor: "a", Status: clipboardWrite}},
	} {
		t.Run(each.what, func(t *testing.T) {
			drawn := Line(each.event, nil, nil)
			assertInert(t, "the line", drawn)
		})
	}
}

// TestAWholeCraftedLogAndItsSummaryAreInert is the same claim through the two
// functions cmd/hexarena actually calls, because Line is not one of them: a
// replay calls Log and Summary, and Summary reads a second copy of the names out
// of the log through NamesFromLog and Tallies. A fix to Line alone would leave
// the summary — the part printed AFTER the body, where a reader has stopped
// watching — carrying the payload.
func TestAWholeCraftedLogAndItsSummaryAreInert(t *testing.T) {
	events := []battle.Event{
		{Kind: battle.Started, Actor: "a", Name: "Bảo" + clipboardWrite, Note: forgery},
		{Kind: battle.Started, Actor: "e", Name: clipboardWrite},
		{Kind: battle.TurnBegan, Actor: "a", Turn: 1},
		{Kind: battle.Damaged, Actor: "a", Target: clipboardWrite, Amount: 100},
	}
	tags := TagsFromLog(events)
	assertInert(t, "the log body", Log(events, tags, nil))
	assertInert(t, "the summary", Summary(events, tags, NamesFromLog(events)))
}

// TestAnHonestLogIsRenderedByteForByte is what stops the sweep being a change to
// the rendering. Every golden under testdata is a record of this function's
// output, so if inert altered anything an honest log holds, this package's
// goldens would all have had to move — and a security fix that silently moves the
// design record is a worse outcome than the finding.
func TestAnHonestLogIsRenderedByteForByte(t *testing.T) {
	events := []battle.Event{
		{Kind: battle.Started, Actor: "bulbasaur-1", Name: "Bulbasaur", Amount: 3000, Note: "ally front"},
		{Kind: battle.TurnBegan, Actor: "bulbasaur-1", Turn: 1},
		{Kind: battle.Damaged, Actor: "bulbasaur-1", Target: "machop-1", Amount: 412, Remaining: 2588},
	}
	tags := TagsFromLog(events)
	for _, event := range events {
		if got := inert(event); got != event {
			t.Errorf("inert altered an honest event:\n before %+v\n after  %+v", event, got)
		}
	}
	if body := Log(events, tags, nil); !strings.Contains(body, "Bulbasaur") {
		t.Errorf("an honest name did not survive the render:\n%s", body)
	}
}

// assertInert asks the one question this whole change is about, through the one
// predicate that defines it.
//
// ⚠️ A line legitimately contains newlines — TurnBegan opens with one — so those
// are the writer's own bytes and are excused before the question is asked. Every
// other control character in a rendered line came out of the log.
func assertInert(t *testing.T, what, drawn string) {
	t.Helper()
	if !plain.Inert(strings.ReplaceAll(drawn, "\n", " ")) {
		t.Errorf("%s carries bytes a terminal would obey: %q", what, drawn)
	}
	if strings.ContainsRune(drawn, 0x1b) {
		t.Errorf("%s carries a raw ESC: %q", what, drawn)
	}
	if strings.ContainsRune(drawn, '\r') {
		t.Errorf("%s carries a carriage return, which overwrites the line before it: %q", what, drawn)
	}
}

package battle_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
)

func scripted(t *testing.T) (*battle.Battle, battle.Script, []battle.Event) {
	t.Helper()
	fight := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	fight.Begin()
	script, _, err := fight.Replay(nil, 500, fight.Suggest)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	return fight, script, fight.Drain()
}

func TestLogRoundTrips(t *testing.T) {
	_, script, events := scripted(t)
	raw, err := battle.MarshalLog(battle.Log{Seed: 7, Choices: script, Events: events})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Error("the log does not end in a newline")
	}
	parsed, err := battle.ParseLog(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Seed != 7 {
		t.Errorf("the seed came back as %d", parsed.Seed)
	}
	if len(parsed.Choices) != len(script) {
		t.Fatalf("%d choices came back, want %d", len(parsed.Choices), len(script))
	}
	for i := range script {
		if parsed.Choices[i] != script[i] {
			t.Errorf("choice %d came back as %+v, want %+v", i, parsed.Choices[i], script[i])
		}
	}
	if len(parsed.Events) != len(events) {
		t.Fatalf("%d events came back, want %d", len(parsed.Events), len(events))
	}
	for i := range events {
		if parsed.Events[i] != events[i] {
			t.Errorf("event %d came back as %+v, want %+v", i, parsed.Events[i], events[i])
		}
	}
}

// TestKindsAndSidesAreWrittenByName keeps a saved log from depending on the order
// the constants happen to be declared in. A number would silently reinterpret
// every existing log the moment a kind was inserted.
func TestKindsAndSidesAreWrittenByName(t *testing.T) {
	raw, err := json.Marshal(battle.Event{
		Kind: battle.Damaged, Actor: "a", Side: hex.SideEnemy,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(raw)
	for _, want := range []string{`"kind":"damaged"`, `"side":"enemy"`} {
		if !strings.Contains(text, want) {
			t.Errorf("the encoded event %s is missing %s", text, want)
		}
	}
	// The fields a kind does not use stay out of the file.
	for _, unwanted := range []string{`"stacks"`, `"power"`, `"note"`} {
		if strings.Contains(text, unwanted) {
			t.Errorf("the encoded event %s carries an unused %s", text, unwanted)
		}
	}
	var back battle.Event
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Kind != battle.Damaged || back.Side != hex.SideEnemy {
		t.Errorf("the event came back as %+v", back)
	}
	for _, bad := range []string{`{"kind":"exploded"}`, `{"kind":7}`, `{"side":"nobody"}`} {
		var reject battle.Event
		if err := json.Unmarshal([]byte(bad), &reject); err == nil {
			t.Errorf("%s was accepted", bad)
		}
	}
}

func TestParseLogRejects(t *testing.T) {
	cases := []struct {
		name, raw, wantErr string
	}{
		{"malformed json", "{", "decode battle log"},
		{"no events", `{"seed":1,"events":[]}`, "records no events"},
		{"a choice with no unit", `{"seed":1,"events":[{"kind":"ended"}],
		  "choices":[{"turn":1,"skill":"strike"}]}`, "names no unit"},
		{"a choice that both passes and acts", `{"seed":1,"events":[{"kind":"ended"}],
		  "choices":[{"unit":"a","turn":1,"skill":"strike","passed":true}]}`, "both passes and uses"},
		{"a choice that does neither", `{"seed":1,"events":[{"kind":"ended"}],
		  "choices":[{"unit":"a","turn":1}]}`, "neither passes nor names"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := battle.ParseLog([]byte(testCase.raw))
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("error %q does not mention %q", err, testCase.wantErr)
			}
		})
	}
}

// TestReplayingAScriptReproducesTheBattle is what an undo and a verified log both
// rest on: the seed and the decisions are the battle, so replaying them is not an
// approximation of the state, it is the state.
func TestReplayingAScriptReproducesTheBattle(t *testing.T) {
	_, script, events := scripted(t)
	rerun := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	rerun.Begin()
	replayed, _, err := rerun.Replay(script, 500, nil)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replayed) != len(script) {
		t.Fatalf("%d decisions were replayed, want %d", len(replayed), len(script))
	}
	got := rerun.Drain()
	if len(got) != len(events) {
		t.Fatalf("the replay produced %d events, want %d", len(got), len(events))
	}
	for i := range events {
		if got[i] != events[i] {
			t.Fatalf("event %d differs:\n%+v\n%+v", i, events[i], got[i])
		}
	}
}

// TestReplayHandsBackTheTurnItStoppedOn is the property an undo needs. A replay
// that advanced a turn it had no decision for and then said nothing about it would
// leave a battle waiting for an action nobody was going to supply.
func TestReplayHandsBackTheTurnItStoppedOn(t *testing.T) {
	_, script, _ := scripted(t)
	if len(script) < 4 {
		t.Fatalf("the script is only %d decisions, too short to cut", len(script))
	}
	partial := script[:len(script)-2]
	fight := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	fight.Begin()
	replayed, pending, err := fight.Replay(partial, 500, nil)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(replayed) != len(partial) {
		t.Fatalf("%d decisions were replayed, want %d", len(replayed), len(partial))
	}
	// A replay that stops for want of a decision hands back the turn it stopped
	// on, so the caller does not have to advance a turn that has already begun.
	if pending == nil {
		t.Fatal("the replay stopped without handing back the open turn")
	}
	if pending.Skipped {
		t.Error("the replay handed back a skipped turn, which needs no decision")
	}
	if len(pending.Options) == 0 {
		t.Error("the turn handed back offers nothing to do")
	}
	// Carrying on from there reaches the end the full script did.
	resumed, stillPending, err := fight.Replay(script[len(partial):], 500, fight.Suggest)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if len(resumed) == 0 {
		t.Error("resuming took no decisions")
	}
	if stillPending != nil {
		t.Errorf("the resumed replay stopped on %+v rather than finishing", stillPending)
	}
}

func TestReplayRejectsAScriptThatDoesNotFit(t *testing.T) {
	fight := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	fight.Begin()
	_, _, err := fight.Replay(battle.Script{
		{Unit: "nobody", Turn: 1, Skill: "strike", Aim: hex.Offset{Col: 3, Row: 1}},
	}, 500, nil)
	if err == nil {
		t.Fatal("a script naming a unit that is not acting was accepted")
	}
	if !strings.Contains(err.Error(), "expects") {
		t.Errorf("error %q does not explain the mismatch", err)
	}

	fresh := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	fresh.Begin()
	if _, _, err := fresh.Replay(battle.Script{
		{Unit: "a", Turn: 1, Skill: "nonesuch", Aim: hex.Offset{Col: 3, Row: 1}},
	}, 500, nil); err == nil {
		t.Fatal("a script naming a skill the unit does not know was accepted")
	}
}

func TestPassReasonIsPartOfTheDecision(t *testing.T) {
	if got := (battle.Decision{}).PassReason(); got != "passed" {
		t.Errorf("a decision with no reason gives %q, want %q", got, "passed")
	}
	if got := (battle.Decision{Reason: "held back"}).PassReason(); got != "held back" {
		t.Errorf("the reason came back as %q", got)
	}
	// A passed turn records the words the decision carried, so a replay cannot
	// diverge from its log over a turn of phrase.
	fight := duel(t, []string{"strike", "jab"}, []string{"jab"}, 200, 80)
	fight.Begin()
	fight.Drain()
	if _, err := fight.Advance(); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := fight.Pass("held back"); err != nil {
		t.Fatalf("pass: %v", err)
	}
	skipped := find(fight.Drain(), battle.TurnSkipped)
	if len(skipped) != 1 || skipped[0].Note != "held back" {
		t.Errorf("the passed turn recorded %+v", skipped)
	}
}

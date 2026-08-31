package tui

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/scale"
)

// TestAmplifiedPowerReadsAsAMultiplier is the whole point of the change: the
// amplified line used to print the rules' own parts-per-thousand figure, which
// a reader had to divide by a thousand before it meant anything and which read
// as a damage number sitting next to real damage numbers.
func TestAmplifiedPowerReadsAsAMultiplier(t *testing.T) {
	line := Line(battle.Event{
		Kind: battle.Amplified, Actor: "a", Skill: "inferno",
		Status: "burn", Stacks: 1, Power: 3500,
	}, map[string]string{"a": "A2"}, nil)
	if !strings.Contains(line, "power x3.5") {
		t.Errorf("the amplified line reads %q, want a power of x3.5", line)
	}
	if strings.Contains(line, "3500") {
		t.Errorf("the amplified line reads %q, want no raw permille figure", line)
	}
}

func TestMultipleSpellsPermille(t *testing.T) {
	cases := []struct {
		permille int
		want     string
	}{
		// The figures the shipped data actually produces.
		{scale.Base, "1"},
		{3500, "3.5"},
		{2500, "2.5"},
		{500, "0.5"},
		{50, "0.05"},
		// A thousandth is the smallest a power can be, and it must survive
		// rather than round away: a log a reader cannot reproduce against the
		// rules is a log that lies.
		{1, "0.001"},
		{1125, "1.125"},
		{0, "0"},
		// Nothing on an event is negative today, and the branch is here so a
		// future one renders rather than printing a stray minus mid-number.
		{-1500, "-1.5"},
		{-1000, "-1"},
	}
	for _, testCase := range cases {
		if got := multiple(testCase.permille); got != testCase.want {
			t.Errorf("a power of %d reads as %q, want %q", testCase.permille, got, testCase.want)
		}
	}
}

package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// TestTheBudgetFlagIsWhatTheRoomIsOpenedWith goes in through the **flag set**,
// which is what catches a flag registered against another variable or never
// registered at all: setting the struct field directly proves nothing about it.
func TestTheBudgetFlagIsWhatTheRoomIsOpenedWith(t *testing.T) {
	for _, one := range []struct {
		name      string
		arguments []string
		want      int
	}{
		{name: "typed", arguments: []string{"-budget", "600"}, want: 600},
		{name: "left off", arguments: nil, want: 0},
	} {
		t.Run(one.name, func(t *testing.T) {
			var chosen settings
			set := flags(&chosen)
			set.SetOutput(io.Discard)
			if err := set.Parse(one.arguments); err != nil {
				t.Fatalf("parse %v: %v", one.arguments, err)
			}
			if chosen.budget != one.want {
				t.Fatalf("%v left the flag at %d, want %d", one.arguments, chosen.budget, one.want)
			}
			chosen.port = 0
			var out, errs bytes.Buffer
			held := hosting(t, chosen, documented, &out, &errs)
			if held.config.Budget != one.want {
				t.Errorf("%v parsed to budget=%d and opened a room on %d: the flag is not "+
					"what the room runs under", one.arguments, one.want, held.config.Budget)
			}
		})
	}
}

// TestTheBannerSaysBothClocksApply.
//
// ⚠️ **The line is drawn only when there is a budget**, for the draft and watch
// lines' reason: a line every ordinary host reads past is how the one that
// matters stops being read. And what it has to say is that the two clocks BOTH
// apply — a host who set a budget expecting it to replace the per-turn allowance
// would otherwise find turns still cut off at ninety seconds and no line anywhere
// explaining it.
func TestTheBannerSaysBothClocksApply(t *testing.T) {
	clocked := aRoom()
	clocked.budget = 600
	var out, errs bytes.Buffer
	held := hosting(t, clocked, documented, &out, &errs)

	var banners bytes.Buffer
	banner(held, "was told by -advertise", &banners)
	drawn := banners.String()
	if !strings.Contains(drawn, "budget") {
		t.Errorf("a room with a chess clock does not say so:\n%s", drawn)
	}
	if !strings.Contains(drawn, "on top of the allowance") {
		t.Errorf("the banner does not say both clocks apply, so a host who expected the "+
			"budget to replace the allowance is not told:\n%s", drawn)
	}

	// And a room without one says nothing extra.
	var quiet bytes.Buffer
	plain := hosting(t, aRoom(), documented, &quiet, &quiet)
	var plainBanner bytes.Buffer
	banner(plain, "was told by -advertise", &plainBanner)
	if strings.Contains(plainBanner.String(), "budget") {
		t.Errorf("a room with no chess clock explains one:\n%s", plainBanner.String())
	}
}

// TestTheBudgetIsInTheUsageAndInTheSettingsLine. A flag missing from the usage is
// a flag nobody finds, and one missing from settings.String is a flag missing
// from every failure this binary reports its own configuration in.
func TestTheBudgetIsInTheUsageAndInTheSettingsLine(t *testing.T) {
	var chosen settings
	set := flags(&chosen)
	var usage bytes.Buffer
	set.SetOutput(&usage)
	set.Usage()
	if !strings.Contains(usage.String(), "-budget") {
		t.Errorf("the usage never mentions -budget:\n%s", usage.String())
	}
	chosen.budget = 600
	if !strings.Contains(chosen.String(), "budget 600") {
		t.Errorf("the settings line does not carry the flag: %s", chosen.String())
	}
}

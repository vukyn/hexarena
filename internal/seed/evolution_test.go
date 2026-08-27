package seed_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/seed"
)

// evolving is a shipped character with more than one form, and the level at
// which every one of them is a legal choice. A cast of single-stage characters
// has nothing to say here and says so rather than passing.
func evolving(t *testing.T) (cast.Character, int) {
	t.Helper()
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	for _, character := range characters.All() {
		if len(character.Stages) > 1 {
			return character, progression.LevelCap
		}
	}
	t.Skip("no shipped character has more than one form, so there is no evolution to choose")
	return cast.Character{}, 0
}

func place(t *testing.T, character cast.Character, level int, stage string, skills []string) ([]byte, error) {
	t.Helper()
	entry := fmt.Sprintf(
		`{"id":"a","side":"ally","slot":[2,1],"character":%q,"level":%s`,
		character.ID, strconv.Itoa(level))
	if stage != progression.Furthest {
		entry += fmt.Sprintf(`,"stage":%q`, stage)
	}
	entry += `,"skills":["` + strings.Join(skills, `","`) + `"]}`
	raw := []byte(`{"units":[` + entry + `]}`)
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	roster, err := seed.ParseRoster(raw, characters)
	if err != nil {
		return raw, err
	}
	if len(roster) != 1 {
		t.Fatalf("the roster resolved to %d units", len(roster))
	}
	return raw, nil
}

// TestAPlacementMayFieldAnEarlierForm is chosen evolution where it is actually
// used: on a placement, which is the only thing in this repository that fields a
// unit rather than describing one.
func TestAPlacementMayFieldAnEarlierForm(t *testing.T) {
	character, level := evolving(t)
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	forms, err := character.StagesAt(level)
	if err != nil {
		t.Fatalf("stages at %d: %v", level, err)
	}
	if len(forms) < 2 {
		t.Skipf("%s has one form at level %d", character.ID, level)
	}
	young, grown := forms[0], forms[len(forms)-1]

	resolve := func(stage string) []byte {
		known := character.SkillsAt(level, stage)
		if len(known) > cast.SkillSlots {
			known = known[:cast.SkillSlots]
		}
		raw, err := place(t, character, level, stage, known)
		if err != nil {
			t.Fatalf("fielding %s as %q: %v", character.ID, stage, err)
		}
		return raw
	}
	youngRaw := resolve(young.Name)
	grownRaw := resolve(grown.Name)

	youngRoster, err := seed.ParseRoster(youngRaw, characters)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	grownRoster, err := seed.ParseRoster(grownRaw, characters)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if youngRoster[0].Stats == grownRoster[0].Stats {
		t.Errorf("both forms resolve to the same stat line, so the choice changes nothing")
	}
	if youngRoster[0].Stats[progression.HP] >= grownRoster[0].Stats[progression.HP] {
		t.Errorf("the earlier form is not the weaker one: %d against %d",
			youngRoster[0].Stats[progression.HP], grownRoster[0].Stats[progression.HP])
	}

	// And naming nothing is naming the furthest, which is what every placement
	// meant before it could choose — so a roster written earlier still says what
	// it always said.
	known := character.SkillsAt(level, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}
	defaultRaw, err := place(t, character, level, progression.Furthest, known)
	if err != nil {
		t.Fatalf("fielding %s without a choice: %v", character.ID, err)
	}
	defaultRoster, err := seed.ParseRoster(defaultRaw, characters)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if defaultRoster[0].Stats != grownRoster[0].Stats {
		t.Error("a placement that named no form did not field the furthest one")
	}
}

// TestGivingUpAnEvolutionKeepsWhatTheGrownFormNeverGets is the reason chosen
// evolution is a decision rather than a formality.
//
// Stage curves only rise, so fielding an earlier form is fielding a weaker unit.
// It buys something only if the earlier form can hold a skill the grown one
// cannot — which is what a stage allowlist on a learnset entry says, and what a
// threshold could never have said.
func TestGivingUpAnEvolutionKeepsWhatTheGrownFormNeverGets(t *testing.T) {
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	// The exclusive entry, found in the data rather than written down here: a
	// cast that stops keeping anything for an earlier form should fail this
	// loudly rather than quietly pass.
	var owner cast.Character
	var kept cast.Unlock
	for _, character := range characters.All() {
		for _, entry := range character.Skills {
			if len(entry.Stages) > 0 && len(entry.Stages) < len(character.Stages) {
				owner, kept = character, entry
			}
		}
	}
	if owner.ID == "" {
		t.Fatal("no shipped character keeps a skill for some of its forms, so giving up an evolution buys nothing and the choice is decorative")
	}

	grown := owner.Stages[len(owner.Stages)-1]
	if slices.Contains(kept.Stages, grown.Name) {
		t.Fatalf("%s keeps %q for its own grown form, so nothing is given up", owner.ID, kept.ID)
	}
	level := progression.LevelCap

	// The form that keeps it may bring it.
	keeper := kept.Stages[len(kept.Stages)-1]
	if _, err := place(t, owner, level, keeper, []string{kept.ID}); err != nil {
		t.Errorf("%s fielded as %s could not bring %q, which is kept for it: %v",
			owner.ID, keeper, kept.ID, err)
	}
	// The grown form may not, and is told why.
	_, err = place(t, owner, level, grown.Name, []string{kept.ID})
	if err == nil {
		t.Fatalf("%s fielded as %s brought %q, which is kept for %v",
			owner.ID, grown.Name, kept.ID, kept.Stages)
	}
	if !strings.Contains(err.Error(), "not learned") {
		t.Errorf("the refusal reads %q, want it to say the form does not have it", err)
	}
	// Which is the trade: the keeper is the weaker unit.
	youngStats, _, err := owner.Resolve(level, keeper)
	if err != nil {
		t.Fatalf("resolve as %s: %v", keeper, err)
	}
	grownStats, _, err := owner.Resolve(level, grown.Name)
	if err != nil {
		t.Fatalf("resolve as %s: %v", grown.Name, err)
	}
	if youngStats[progression.HP] >= grownStats[progression.HP] {
		t.Errorf("the form that keeps %q is not the weaker one, so there is no trade", kept.ID)
	}
}

// TestAFormThatIsWrongIsRefused covers the two ways a placement can name a form
// it may not field, and the flat entry that has no forms at all.
func TestAFormThatIsWrongIsRefused(t *testing.T) {
	character, _ := evolving(t)
	known := character.SkillsAt(1, progression.Furthest)
	if len(known) > cast.SkillSlots {
		known = known[:cast.SkillSlots]
	}

	last := character.Stages[len(character.Stages)-1]
	_, err := place(t, character, 1, last.Name, known)
	if err == nil {
		t.Errorf("a level 1 placement was fielded as %q, which begins at %d",
			last.Name, last.MinLevel)
	} else if !strings.Contains(err.Error(), "begins at level") {
		t.Errorf("the refusal reads %q, want it to say when the form begins", err)
	}

	_, err = place(t, character, progression.LevelCap, "Nonesuch", known)
	if err == nil {
		t.Error("a placement was fielded as a form the line does not have")
	} else if !strings.Contains(err.Error(), "no stage of this line") {
		t.Errorf("the refusal reads %q, want it to say the line has no such form", err)
	}

	// A flat entry writes its numbers out and has no evolution line at all, so a
	// stage there is a mistake rather than a choice.
	flat := `{"units":[{"id":"a","side":"ally","slot":[2,1],"stage":"Whatever",
	  "name":"Flat","element":["grass"],
	  "stats":{"hp":1000,"attack":100,"defense":100,"speed":50,"accuracy":50,"dodge":10},
	  "skills":["vine_whip"]}]}`
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the cast: %v", err)
	}
	if _, err := seed.ParseRoster([]byte(flat), characters); err == nil {
		t.Error("a flat entry named a stage and was accepted")
	} else if !strings.Contains(err.Error(), "no evolution line") {
		t.Errorf("the refusal reads %q, want it to say a flat entry has no forms", err)
	}
}

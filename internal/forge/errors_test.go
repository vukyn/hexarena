package forge

import (
	"errors"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
)

// A refusal is a value now, so that cmd/hexforge-tui can say it in Vietnamese.
// The value's own English is what cmd/hexforge prints, and it is pinned here
// word for word: those lines are what a script greps and a person has in their
// terminal history, so they are a contract rather than a wording.

// TestARefusalKeepsTheWordingTheCommandLinePrints is that contract.
func TestARefusalKeepsTheWordingTheCommandLinePrints(t *testing.T) {
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("the fire affinity: %v", err)
	}
	cases := []struct {
		err  error
		want string
	}{
		{&IDTakenError{ID: "a.b"}, `character "a.b" is already in the cast`},
		{&MissingNameError{}, "a character needs a display name"},
		{&MissingNameError{ID: "a.b"}, `character "a.b" needs a display name`},
		{&UnknownOriginError{ID: "nowhere"},
			`unknown origin "nowhere", add it with "hexforge origins add nowhere"`},
		{&UnknownArchetypeError{ID: "nobody", Known: []string{"duelist", "sentinel"}},
			`unknown archetype "nobody", want one of duelist, sentinel`},
		{&OriginTakenError{ID: "example-film"}, `origin "example-film" is already in the catalog`},
		{&EmptyKitError{}, "a character with no skills would have nothing to do on its turn"},
		{&DuplicateSkillError{ID: "strike"}, `"strike" is named twice`},
		{&MissingElementError{}, "no element given"},
		{&AffinityCountError{Raw: "fire/water/ice", Count: 3},
			`"fire/water/ice" lists 3 elements, want one or two separated by a slash`},
		{&CarryError{Affinity: fire, Skill: "sever", Element: element.Metal},
			`fire cannot carry "sever", which is metal`},
		{&CurveShapeError{Raw: "120"}, `"120" is not a curve, want base:max`},
		{&StatFieldError{Kind: progression.HP, Err: &CurveShapeError{Raw: "120"}},
			`hp: "120" is not a curve, want base:max`},
		{&YearError{Raw: "nineteen"},
			`the year "nineteen" is not a number; leave it empty if it is unknown`},
		{&SkillRenameError{From: "sever", To: "sunder"},
			`a skill's id cannot be edited, so "sever" cannot become "sunder"; ` +
				"renaming one has to change every kit and every restriction that names it, " +
				"which is a separate operation"},
		// The preset is carried in a field and left out of the sentence: whoever
		// shows this asked about that preset, and naming it here as well would
		// say it twice in one line.
		{&PresetOwnedSkillError{
			Archetype: "bulwark", Skill: "sever", Allowed: []string{"a.one", "a.two"},
		}, `"sever" belongs to a.one or a.two, and a preset is shared by every character built from it`},
		{&SkillEditBreaksError{
			Carrier: BrokenCharacter, ID: "a.one", Skill: "riptide",
			Err: errors.New("because"),
		}, `editing "riptide" would leave a.one unable to carry it: because`},
		{&SkillEditBreaksError{
			Carrier: BrokenPreset, ID: "bulwark", Skill: "sever", Err: errors.New("because"),
		}, `editing "sever" would leave the bulwark preset unable to carry it: because`},
		// No carrier named, which is the shape a refusal about a kit as a whole
		// takes: the parser's words and nobody blamed.
		{&SkillEditBreaksError{Skill: "bolt", Err: errors.New("because")},
			`editing "bolt" would stop the books loading: because`},
	}
	for _, test := range cases {
		if got := test.err.Error(); got != test.want {
			t.Errorf("%T reads\n %q\nwant\n %q", test.err, got, test.want)
		}
	}
}

// TestAWrappedRefusalKeepsTheWordsOfWhoeverMadeIt covers the other half: a
// refusal that came from internal/core is carried, not restated. Its own
// message reaches the command line unchanged, and errors.Is still finds it.
func TestAWrappedRefusalKeepsTheWordsOfWhoeverMadeIt(t *testing.T) {
	inner := errors.New("unknown skill \"nonesuch\"")
	wrapped := &UnknownSkillError{ID: "nonesuch", Err: inner}
	if got := wrapped.Error(); got != inner.Error() {
		t.Errorf("the wrapper reworded the refusal as %q", got)
	}
	if !errors.Is(wrapped, inner) {
		t.Error("the wrapped refusal is no longer findable with errors.Is")
	}
}

// TestARefusalCarriesWhatItIsAbout is what makes a second language possible: a
// front-end reads the fields rather than the sentence.
func TestARefusalCarriesWhatItIsAbout(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	kit, err := lib.LookupKit([]string{"strike", "riptide"})
	if err != nil {
		t.Fatalf("look up a kit: %v", err)
	}
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("the fire affinity: %v", err)
	}
	var carry *CarryError
	if !errors.As(CheckCarry(fire, kit), &carry) {
		t.Fatal("fire carrying a water kit was not refused with a *CarryError")
	}
	if carry.Skill != "riptide" {
		t.Errorf("the refusal names %q, want the first skill that cannot be carried", carry.Skill)
	}
	if carry.Element != element.Water {
		t.Errorf("the refused skill is reported as %s, want water", carry.Element)
	}
	if carry.Affinity != fire {
		t.Errorf("the refusal carries the affinity %s, want fire", carry.Affinity)
	}

	// A curve refusal classifies itself, so a front-end picks a sentence rather
	// than matching on the engine's English.
	var refused *CurveRefusedError
	if !errors.As(ValidateCurve(progression.HP, "900:400"), &refused) {
		t.Fatal("a curve that shrinks with level was not refused with a *CurveRefusedError")
	}
	if refused.Reason != CurveReasonShrinks {
		t.Errorf("the reason is %d, want the shrinking one", refused.Reason)
	}
	if !errors.As(ValidateCurve(progression.HP, "0:400"), &refused) {
		t.Fatal("a curve starting at zero was not refused")
	}
	if refused.Reason != CurveReasonNotPositive {
		t.Errorf("the reason is %d, want the not-positive one", refused.Reason)
	}

	// The per-answer checks a form applies as it is typed return the same types
	// the write does, which is what lets a form say "this id is taken" in one
	// language and cmd/hexforge say it in another.
	var taken *IDTakenError
	if !errors.As(lib.ValidateNewID(lib.Characters().All()[0].ID), &taken) {
		t.Error("an id already in the cast was not refused with an *IDTakenError")
	}
	var malformed *FieldRefusedError
	if !errors.As(lib.ValidateNewID("Not A Slug"), &malformed) {
		t.Error("a malformed id was not refused with a *FieldRefusedError")
	} else if malformed.Field != FieldID {
		t.Errorf("the refusal is about field %d, want the id", malformed.Field)
	}
	if err := lib.ValidateNewID("example-film.free"); err != nil {
		t.Errorf("a free, well-shaped id was refused: %v", err)
	}
	var empty *EmptyKitError
	if !errors.As(lib.ValidateKit("  "), &empty) {
		t.Error("an empty kit was not refused with an *EmptyKitError")
	}
	var twice *DuplicateSkillError
	if !errors.As(lib.ValidateKit("strike,strike"), &twice) {
		t.Error("a skill named twice was not refused with a *DuplicateSkillError")
	}

	// So does an affinity the chart will not allow.
	var pairing *AffinityRefusedError
	err = lib.ValidateElement("fire/water", kit)
	if !errors.As(err, &pairing) {
		t.Fatalf("a countering pair was refused with %T, want an *AffinityRefusedError", err)
	}
	if pairing.Reason != AffinityReasonCounters {
		t.Errorf("the reason is %d, want the countering one", pairing.Reason)
	}
}

// TestTheSummariesAreFactsBeforeTheyAreSentences covers the three summaries a
// front-end draws, each of which now has a value behind the line.
func TestTheSummariesAreFactsBeforeTheyAreSentences(t *testing.T) {
	lib, err := Load(shippedDataDir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	preset, known := lib.Archetypes().Get("sentinel")
	if !known {
		t.Fatal("the sentinel preset is not shipped")
	}
	facts := PresetFacts(preset)
	if len(facts.Skills) != len(preset.Skills) {
		t.Errorf("the preset's kit is %v, want %v", facts.Skills, preset.Skills)
	}
	summary := PresetSummary(preset)
	for _, skill := range facts.Skills {
		if !strings.Contains(summary, skill) {
			t.Errorf("the English summary %q leaves out %q", summary, skill)
		}
	}
	for _, demand := range facts.Demands {
		if !strings.Contains(summary, demand) {
			t.Errorf("the English summary %q leaves out the demand %q", summary, demand)
		}
	}

	character := lib.Characters().All()[0]
	stages := StageFacts(character)
	if len(stages) != len(character.Stages) {
		t.Fatalf("%d stages reported, want %d", len(stages), len(character.Stages))
	}
	for i, stage := range stages {
		if stage.Name != character.Stages[i].Name || stage.MinLevel != character.Stages[i].MinLevel {
			t.Errorf("stage %d is %+v, want %+v", i, stage, character.Stages[i])
		}
	}

	// And a write's confirmation, whose English is still the command line's.
	notes := lib.SaveNoteFacts(character)
	lines := lib.SaveNotes(character)
	if len(notes) != len(lines) {
		t.Fatalf("%d notes against %d lines", len(notes), len(lines))
	}
	if notes[0].Kind != NoteWrote || notes[0].ID != character.ID {
		t.Errorf("the first note is %+v, want the write", notes[0])
	}
	if want := "wrote " + character.ID + " to " + lib.CastPath(); lines[0] != want {
		t.Errorf("the first line is %q, want %q", lines[0], want)
	}
	if notes[len(notes)-1].Kind != NoteRebuild {
		t.Errorf("the last note is %+v, want the rebuild warning", notes[len(notes)-1])
	}
}

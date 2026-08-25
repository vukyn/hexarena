package i18n

// english is the client in English.
//
// It is not the original that the Vietnamese was made from — both arrays were
// written together — and it was rewritten while this package was built, so a
// line here may differ from what cmd/hexforge prints for the same fact. That is
// on purpose: the command line's wording is frozen because scripts read it, and
// the screen's is free to read better. The fact behind both is the same value
// from internal/forge, so they cannot disagree about anything but phrasing.
//
// Two rules were applied throughout: keep a line short enough for the layout,
// and do not abbreviate where the whole word fits.
var english = [keyCount]string{
	MeasuringTerminal: "measuring the terminal…",
	TerminalTooSmall: `terminal too small

needs at least %dx%d
this window is %dx%d

Make it bigger, or use
hexforge instead: same
cast, same checks, and
it fits any terminal.

q or ctrl+c to quit`,
	Truncated:   "… cut off; a taller window shows the rest",
	NoArguments: "takes no arguments, got %v",
	NotATerminal: "stdout is not a terminal, and a full-screen program would write control codes into it.\n" +
		"Use hexforge instead: it authors the same cast through the same checks, takes flags\n" +
		"and reads a pipe, and `hexforge check` prints what this program's check screen shows",
	DataFlagUsage:     "the data directory to read and write",
	LanguageFlagUsage: "the language to read the screens in: vi or en",

	MenuHeading:            "what would you like to do?",
	MenuCast:               "cast",
	MenuCastDetail:         "browse the authored characters, at any level",
	MenuNewCharacter:       "new character",
	MenuNewCharacterDetail: "author one, with the budget and the carry check live",
	MenuOrigins:            "origins",
	MenuOriginsDetail:      "the works the cast is borrowed from, and add one",
	MenuCheck:              "check",
	MenuCheckDetail:        "see that the art is there and the budget is kept",
	MenuNote: "Everything written here goes through the same checks as hexforge, and the\n" +
		"game boots from the embedded copy — rebuild before an edit reaches a battle.",
	MenuFooter: "↑/↓ move · enter open · ctrl+l tiếng Việt · q quit",

	ConfirmFooter:  "%s [y/N] · ctrl+c quit",
	ArtPresent:     "present",
	ArtMissing:     "MISSING",
	ChoicePosition: "%d of %d",

	FormHeading:         "new character",
	FormSubtitle:        "every check here is the write's own, at level %d",
	FormFooter:          "↑/↓ field · ←/→ pick · ctrl+s save · esc back · ctrl+l tiếng Việt · ctrl+c quit",
	FormDiscard:         "discard the character being authored?",
	FieldID:             "id",
	FieldName:           "name",
	FieldOrigin:         "origin",
	FieldArchetype:      "archetype",
	FieldArt:            "art",
	FieldKit:            "kit",
	FieldElement:        "element",
	FieldBiography:      "biography",
	NoneCatalogued:      "none catalogued",
	CurveAgainstCeiling: "%d → %d, ceiling %d",
	OverTheCeiling:      "OVER THE CEILING",
	LabelBudget:         "budget",
	LabelCarries:        "carries",
	BudgetWithin:        "%s %d of %d, %d to spare",
	BudgetOver:          "%s %d of %d, OVER THE BUDGET by %d",
	CarryNoElementYet:   "no element yet — %s",
	CarryRefused:        "NO — %s",
	CarryAccepted:       "YES — %s carries every skill in the kit",
	WriteRefused:        "cannot write: %s",

	KitTakesAnyElement:    "this kit is all neutral, so any element carries it",
	KitNeeds:              "this kit needs %s",
	PresetTakesAnyElement: "%s (any element)",
	PresetNeeds:           "%s (needs %s)",
	ElementJoiner:         " and ",

	BrowseHeading:          "cast",
	BrowseShowing:          "showing %s (%d of %d characters)",
	BrowseAllOrigins:       "all origins",
	BrowseFooter:           "↑/↓ character · ←/→ level · f filter · ctrl+l tiếng Việt · esc back · q quit",
	BrowseNothingHere:      "nothing to show here.",
	BrowseNothingAuthored:  "No characters have been authored yet. Pick \"new character\" from the menu.",
	BrowseNoneFromThisWork: "No character is borrowed from this work. Press f for the next filter.",
	LabelFrom:              "from",
	LabelTunedFrom:         "tuned from",
	LabelElement:           "element",
	LabelKit:               "kit",
	LabelArt:               "art",
	LabelStages:            "stages",
	LabelBiography:         "biography",
	LabelAtLevel:           "level %d",
	LabelAbsorbs:           "absorbs",
	StageInWords:           "stage %s",

	CheckHeading:        "check",
	CheckFooter:         "↑/↓ move · r re-read the files · ctrl+l tiếng Việt · esc back · q quit",
	CheckPassed:         "PASSED — no problems found",
	CheckFailed:         "FAILED — %d problem(s)",
	CheckCounts:         "%s: %d origins, %d archetypes, %d characters",
	CheckNothingToCheck: "no characters to check.",
	ColumnCharacter:     "character",
	ColumnArt:           "art",
	ColumnAbsorbs:       "absorbs of the budget, at the level cap",
	CheckDoesNotResolve: "does not resolve: %s",
	CheckOverBudget:     "OVER",
	CheckProblem:        "problem: %s",
	CheckNote: "this reads the files from disk; the game boots from the embedded copy, so an\n" +
		"edit needs a rebuild before it reaches a battle",

	OriginsHeading:     "origins",
	OriginsSubtitle:    "the works the cast is borrowed from",
	OriginsFooter:      "↑/↓ move · a add a work · ctrl+l tiếng Việt · esc back · q quit",
	OriginsEmpty:       "no works in the catalog yet. Press a to add one.",
	OriginsCastCount:   "%2d cast",
	OriginsTally:       "%d works · media: %s",
	OriginAdded:        "added %s (%s) to %s",
	LabelNote:          "note",
	OriginFormHeading:  "add a work",
	OriginFormSubtitle: "a character can only name a work the catalog holds",
	OriginFormFooter:   "↑/↓ move · ←/→ medium · ctrl+s add · esc back · ctrl+l tiếng Việt · ctrl+c quit",
	OriginFormHint:     "the year may be left empty when it is unknown; the note is free text",
	OriginFormDiscard:  "discard the work being added?",
	OriginFieldID:      "id",
	OriginFieldTitle:   "title",
	OriginFieldMedium:  "medium",
	OriginFieldYear:    "year",
	OriginFieldNote:    "note",
	AddRefused:         "cannot add: %s",

	ErrorIDTaken:            "the character %q is already in the cast",
	ErrorMissingName:        "a character needs a display name",
	ErrorUnknownOrigin:      "no origin %q in the catalog; add it with %q",
	ErrorUnknownArchetype:   "no archetype %q; the ones there are: %s",
	ErrorOriginTaken:        "the origin %q is already in the catalog",
	ErrorEmptyKit:           "a character with no skills would have nothing to do on its turn",
	ErrorDuplicateSkill:     "the skill %q is named twice",
	ErrorUnknownSkill:       "there is no skill named %q",
	ErrorUnknownElement:     "there is no element named %q",
	ErrorMissingElement:     "no element given yet",
	ErrorAffinityCount:      "%q lists %d elements; take one, or two separated by a slash",
	ErrorAffinityCounters:   "%s pairs two elements that already counter each other",
	ErrorAffinityUndeclared: "%s holds an element the chart does not declare",
	ErrorAffinityRefused:    "the element chart refuses %s: %v",
	ErrorCarry:              "%s cannot carry the skill %q, which is %s",
	ErrorCurveShape:         "%q is not a curve; write it as base:max",
	ErrorCurveNumber:        "%q has a %s that is not a number",
	ErrorCurveNotPositive:   "%s starts at %d; it has to be a positive number",
	ErrorCurveShrinks:       "%s ends at %d but starts at %d; a stat may not shrink as the level rises",
	ErrorCurveRefused:       "the %s curve is refused: %v",
	ErrorStatField:          "%s: %s",
	ErrorFieldID:            "that id will not do: %v",
	ErrorFieldImage:         "that art path will not do: %v",
	ErrorYear:               "the year %q is not a number; leave it empty if it is unknown",
	ErrorAsGiven:            "%v",

	ProblemMissingArt:     "character %s names the art %s, which is not at %s",
	ProblemDoesNotResolve: "character %s does not resolve: %v",
	NoteWrote:             "wrote %s to %s",
	NoteArtMissing:        "note: %s is not there yet; a check will keep saying so until it is",
	NoteRebuild:           "note: the game boots from the embedded copy — rebuild to see this in a battle",
}

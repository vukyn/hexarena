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
	MenuSkills:             "skills",
	MenuSkillsDetail:       "the declared skills, who may carry each, add or edit one",
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
	NoArtToChoose:       "no art found in %s — type a path here instead",
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

	SkillsHeading:               "skills",
	SkillsSubtitle:              "what each one does, and who may carry it",
	SkillsFooter:                "↑/↓ move · a add · e edit · ctrl+l tiếng Việt · esc back · q quit",
	SkillsTally:                 "%d skills · %d anyone may carry",
	ColumnWhoMayCarry:           "who may carry it",
	ColumnGloss:                 "translated name",
	SkillFormHeading:            "new skill",
	SkillFormSubtitle:           "the damage is the engine's own, worked out as you type",
	SkillFormFooter:             "↑/↓ field · ←/→ pick · space list · ctrl+s save · esc back · ctrl+l tiếng Việt",
	SkillFormDiscard:            "discard the skill being authored?",
	SkillFieldID:                "id",
	SkillFieldElement:           "element",
	SkillFieldTarget:            "aims at",
	SkillFieldRange:             "range",
	SkillFieldShape:             "shape",
	SkillFieldPower:             "damage multiplier",
	SkillFieldStrikes:           "strikes",
	SkillFieldAccuracy:          "accuracy",
	SkillFieldCooldown:          "cooldown (cd)",
	SkillFieldInflicts:          "inflicts",
	SkillFieldKeptForElements:   "kept for",
	SkillFieldKeptForRoles:      "kept for role",
	SkillFieldKeptForCharacters: "belongs to",
	LabelDamage:                 "damage",
	// The reference pair is named with the two stat labels this client leaves
	// untranslated everywhere else — see forge.ShortStat — rather than spelled
	// out. That is seven cells, and this row needs them: it is the widest fixed
	// row on the skill form, it grows with the number of digits in its own
	// figures, and it sat at exactly the 79 cells the window has before the
	// power field was renamed to something longer than "power". The command
	// line spells them out and is frozen that way because scripts read it; this
	// array is free to read better, which is what the package comment says.
	DamageLine:           "%d per strike, %d in all, against %d atk and %d def",
	DamageLineShort:      "%d per strike, %d in all",
	DamageAmplified:      "%d with its condition holding",
	DamageMoved:          "%d → %d per strike, %d → %d in all",
	SkillAdded:           "added %s to %s",
	SkillEdited:          "edited %s in %s",
	SkillFormEditHeading: "edit skill",
	SkillFormEditDiscard: "discard the changes?",

	// One line per field, describing the field the cursor is on rather than
	// footnoting two of the fourteen. See keys.go for why this replaced a
	// footnote: they say what the field does, not what it is called.
	SkillHelpID:                "the name the data files use: lower case and underscores — e.g. ember_lance",
	SkillHelpName:              "its Vietnamese name; empty uses the built-in one, then the bare id",
	SkillHelpElement:           "its element; only a unit sharing it may carry it, neutral suits everyone",
	SkillHelpTarget:            "which side it reaches: the far one, the caster's own, the caster, or both",
	SkillHelpRange:             "how many cells away it reaches, from where the caster stands — e.g. 2",
	SkillHelpShape:             "which cells besides the aim it catches; space draws it on the board",
	SkillHelpPower:             "parts per thousand of the caster's attack: 1000 is one times, 800 is 0.8x",
	SkillHelpStrikes:           "how many times one turn lands, each at the multiplier above — e.g. 3",
	SkillHelpAccuracy:          "parts per thousand chance to connect, before accuracy and dodge — e.g. 900",
	SkillHelpCooldown:          "turns of the caster's own to wait before using it again; 0 is every turn",
	SkillHelpInflicts:          "status:chance in parts per thousand, comma separated — e.g. poison:300",
	SkillHelpKeptForElements:   "only these elements may carry it; empty means any element — e.g. fire",
	SkillHelpKeptForRoles:      "only these role presets may carry it; empty means any role",
	SkillHelpKeptForCharacters: "only these characters may carry it; empty means anyone at all",

	// The shape diagram, opened from the shape chooser.
	SkillShapeHeading:  "which cells the skill catches",
	SkillShapeCoverage: "catches %d cells",
	SkillShapeShort:    "catches %d of %d cells — a second step off the board loses one",
	SkillShapeDrawnAt:  "drawn at %s, the middle of the enemy formation",
	SkillShapeLegend:   "%s is the aim, at full power · %s is a splash cell, at %s",
	SkillShapeFooter:   "←/→ shape · enter or esc done · ctrl+l tiếng Việt",

	WhoAnyone:          "anyone",
	WhoElementUnits:    "%s units",
	WhoKeptForElements: "kept for %s",
	WhoKeptForRoles:    "kept for the %s role",
	WhoBelongsTo:       "belongs to %s",

	PickerKitTitle:        "kit",
	PickerElementsTitle:   "elements allowed to carry it",
	PickerRolesTitle:      "roles allowed to carry it",
	PickerCharactersTitle: "characters allowed to carry it",
	PickerHint:            "space chooses · the number is the order · ! is one this character cannot take",
	PickerAllowlistHint:   "space chooses · leaving it empty lets anyone carry the skill",
	PickerFooter:          "space choose · ↑/↓ move · enter done · esc cancel · ctrl+l tiếng Việt",
	PickerNothingToPick:   "there is nothing to choose from.",
	PickerNothingChosen:   "nothing chosen yet",
	KitChooseHint:         "space to choose",

	PickerShowing:        "showing %s (%d of %d)",
	PickerNothingInGroup: "no characters from this work yet.",
	PickerFilterFooter:   "space choose · ↑/↓ move · f filter · enter done · esc back · ctrl+l tiếng Việt",

	PickerStatusesTitle: "the statuses this skill inflicts",
	PickerStatusHint:    "space chooses a status · type the chance in parts per thousand",
	PickerStatusFooter:  "space pick · ↑/↓ move · 0-9 chance · enter done · esc back · ctrl+l tiếng Việt",
	PickerChance:        "chance",
	StatusDetail:        "%s · %d turns · up to %d stacks",
	StatusTicks:         "damages every turn",

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
	LabelPlaystyle:         "playstyle",
	LabelElement:           "element",
	LabelKit:               "kit",
	LabelArt:               "art",
	LabelStages:            "stages",
	LabelBiography:         "biography",
	LabelAtLevel:           "level %d",
	LabelEffectiveHP:       "effective hp",
	StageInWords:           "stage %s",

	CheckHeading:        "check",
	CheckFooter:         "↑/↓ move · r re-read the files · ctrl+l tiếng Việt · esc back · q quit",
	CheckPassed:         "PASSED — no problems found",
	CheckFailed:         "FAILED — %d problem(s)",
	CheckCounts:         "%s: %d origins, %d archetypes, %d characters",
	CheckNothingToCheck: "no characters to check.",
	ColumnCharacter:     "character",
	ColumnArt:           "art",
	ColumnEffectiveHP:   "effective hp of the budget, at the cap",
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
	ErrorCarryRestricted:    "%s cannot carry the skill %q; it is kept for %s",
	// "kept for" rather than "restricted to": the row it lands on is already
	// narrow, and an author reading it wants who may carry the skill, not the
	// name of the mechanism keeping them from it.
	ErrorArchetypeRestricted: "%q cannot carry the skill %q; it is kept for the %s role",
	ErrorCharacterRestricted: "%q cannot carry the skill %q; it belongs to %s",
	ErrorSkillTaken:          "the skill %q is already in the book",
	ErrorMissingSkillID:      "a skill needs an id",
	// Shorter than the sentence forge.SkillRenameError prints, which spells out
	// what a rename would have to touch: the command line has room for a
	// paragraph and a form row does not.
	ErrorSkillRename:              "a skill's id cannot be edited here (%s → %s); renaming is a separate job",
	ErrorPresetOwnedSkill:         "%s belongs to %s, and every character built from a preset shares its kit",
	ErrorSkillEditBreaksCharacter: "editing %s would leave %s unable to carry it: %s",
	ErrorSkillEditBreaksPreset:    "editing %s would leave the %s preset unable to carry it: %s",
	ErrorSkillEditBreaks:          "editing %s would stop the books loading: %s",
	ErrorUnknownPattern:           "there is no shape named %q",
	ErrorUnknownTarget:            "there is nothing to aim at named %q; take enemy, ally or self",
	ErrorUnknownStatus:            "there is no status named %q",
	ErrorUnknownCharacter:         "there is no character %q in the cast",
	ErrorDuplicateEntry:           "%q is named twice",
	ErrorNotANumber:               "%q is not a number",
	ErrorApplicationShape:         "%q is not a status and a chance; write it as status:chance",
	ErrorCurveShape:               "%q is not a curve; write it as base:max",
	ErrorCurveNumber:              "%q has a %s that is not a number",
	ErrorCurveNotPositive:         "%s starts at %d; it has to be a positive number",
	ErrorCurveShrinks:             "%s ends at %d but starts at %d; a stat may not shrink as the level rises",
	ErrorCurveRefused:             "the %s curve is refused: %v",
	ErrorStatField:                "%s: %s",
	ErrorFieldID:                  "that id will not do: %v",
	ErrorFieldImage:               "that art path will not do: %v",
	ErrorYear:                     "the year %q is not a number; leave it empty if it is unknown",
	ErrorAsGiven:                  "%v",

	ProblemMissingArt:     "character %s names the art %s, which is not at %s",
	ProblemDoesNotResolve: "character %s does not resolve: %v",
	NoteWrote:             "wrote %s to %s",
	NoteEdited:            "edited %s in %s",
	NoteArtMissing:        "note: %s is not there yet; a check will keep saying so until it is",
	NoteRebuild:           "note: the game boots from the embedded copy — rebuild to see this in a battle",
	NoteGoldensMove:       "note: this is balance, so the golden files have moved — run make golden and read the diff",
}

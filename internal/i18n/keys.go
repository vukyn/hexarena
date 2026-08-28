package i18n

import "fmt"

// Key names one thing the client can say.
//
// It is an integer rather than a free string so that a message is chosen by a
// constant the compiler knows: a mistyped key is a build failure instead of a
// blank line on screen. The catalogs are arrays of the same length, so a
// language that forgets an entry leaves an empty string, and
// TestEveryKeyIsWordedInEveryLanguage fails on it — never an empty line for the
// author, and never a silent fall back to English, which is a worse outcome
// than a loud one because nobody who cannot read it will report it.
type Key int

const (
	// The program itself, and the two states it can be in before a screen.
	MeasuringTerminal Key = iota
	TerminalTooSmall
	Truncated
	NoArguments
	NotATerminal
	DataFlagUsage
	LanguageFlagUsage

	// The menu.
	MenuHeading
	MenuCast
	MenuCastDetail
	MenuNewCharacter
	MenuNewCharacterDetail
	MenuOrigins
	MenuOriginsDetail
	MenuSkills
	MenuSkillsDetail
	MenuCheck
	MenuCheckDetail
	MenuNote
	MenuFooter

	// Shared between screens.
	ConfirmFooter
	ArtPresent
	ArtMissing
	ArtSomeMissing
	ChoicePosition

	// The new-character form.
	FormHeading
	FormSubtitle
	FormFooter
	FormDiscard
	FieldID
	FieldName
	FieldOrigin
	FieldArchetype
	FieldArt
	FieldKit
	FieldSpecies
	SpeciesNothingInParticular
	FieldElement
	FieldBiography
	NoneCatalogued
	NoArtToChoose
	CurveAgainstCeiling
	OverTheCeiling
	LabelBudget
	LabelCarries
	BudgetWithin
	BudgetOver
	BudgetPierced
	CarryNoElementYet
	CarryRefused
	CarryAccepted
	WriteRefused

	// The skill book, and the form that adds to it.
	SkillsHeading
	SkillsSubtitle
	SkillsFooter
	SkillsTally
	ColumnWhoMayCarry
	ColumnGloss
	SkillFormHeading
	SkillFormSubtitle
	SkillFormFooter
	SkillFormDiscard
	SkillFieldID
	SkillFieldElement
	SkillFieldTarget
	SkillFieldRange
	SkillFieldShape
	SkillFieldPower
	SkillFieldStrikes
	SkillFieldAccuracy
	SkillFieldCooldown
	SkillFieldInflicts
	SkillFieldOnItself
	SkillFieldPierce
	SkillFieldRestores
	SkillFieldDrains
	SkillHelpOnItself
	SkillHelpPierce
	SkillHelpRestores
	SkillHelpDrains
	SkillFieldKeptForElements
	SkillFieldKeptForRoles
	SkillFieldKeptForCharacters
	SkillFieldKeptForSpecies
	SkillFieldKeptForOrigins
	LabelDamage
	DamageLine
	DamageLineShort
	DamageAmplified
	DamageMoved
	SkillAdded
	SkillEdited
	SkillFormEditHeading
	SkillFormEditDiscard

	// One line per field of the skill form, describing the field the cursor is
	// on: what the number or the name does, and an answer that would be valid.
	//
	// There is one of these for every field and they are worded here rather
	// than on the screen for the same reason every other line is — but the
	// reason they exist at all is worth writing down, because it replaced
	// something. The form used to carry a single footnote about parts per
	// thousand, which is a sentence about two of the fourteen fields, printed
	// whether or not the cursor was on either. A footnote covering a seventh of
	// a screen is a footnote nobody reads, and the fields nobody could guess —
	// what a shape covers, what syntax the statuses take, what an empty
	// allowlist means — were the ones it said nothing about.
	//
	// They say what the field *does*, not what it is called. A help line that
	// only expands the label ("power: the skill's power") is the footnote again.
	SkillHelpID
	SkillHelpName
	SkillHelpElement
	SkillHelpTarget
	SkillHelpRange
	SkillHelpShape
	SkillHelpPower
	SkillHelpStrikes
	SkillHelpAccuracy
	SkillHelpCooldown
	SkillHelpInflicts
	SkillHelpKeptForElements
	SkillHelpKeptForRoles
	SkillHelpKeptForCharacters
	SkillHelpKeptForSpecies
	SkillHelpKeptForOrigins

	// The shape diagram, opened from the shape chooser.
	//
	// A sub-screen rather than a pane on the form, and the reason is measured:
	// the form spends nineteen of the twenty body lines an 80x24 window has, and
	// the board is eight lines on its own. See skillFieldShape's help line,
	// which is where an author is told the key.
	SkillShapeHeading
	SkillShapeCoverage
	SkillShapeShort
	SkillShapeDrawnAt
	SkillShapeLegend
	SkillShapeFooter

	// Who may carry a skill, which is the same vocabulary in a listing, in a
	// picker's row and in a refusal.
	WhoAnyone
	WhoElementUnits
	WhoKeptForElements
	WhoKeptForRoles
	WhoBelongsTo
	WhoKeptForSpecies
	WhoKeptForOrigins

	// The multi-select sub-screen, and the row that opens it.
	PickerKitTitle
	PickerElementsTitle
	PickerRolesTitle
	PickerCharactersTitle
	PickerSpeciesTitle
	PickerOriginsTitle
	PickerHint
	PickerAllowlistHint
	PickerFooter
	PickerNothingToPick
	PickerNothingChosen
	KitChooseHint

	// Narrowing a picker's list, which one of the five has: the characters,
	// because that is the list that grows as a cast is authored. The key and
	// the cycle are the cast browser's own, so there is one interaction for
	// filtering a list of characters rather than two.
	PickerShowing
	PickerNothingInGroup
	PickerFilterFooter

	// The status picker, which is the multi-select with one answer more: a
	// status is an id *and* a chance, so it collects a number beside the list
	// rather than sending an author back to a text field to remember a syntax.
	PickerStatusesTitle
	PickerStatusHint
	PickerStatusFooter
	PickerChance
	StatusDetail
	StatusTicks
	CategoryStatDebuff
	CategoryControl
	CategoryBuff
	CategoryShield
	CategoryRegen
	CategoryTaunt

	// What a kit and a preset demand of an affinity.
	KitTakesAnyElement
	KitNeeds
	PresetTakesAnyElement
	PresetNeeds
	ElementJoiner

	// The cast browser.
	BrowseHeading
	BrowseShowing
	BrowseAllOrigins
	BrowseFooter
	PreviewFooter
	PreviewTitle
	PreviewArtUnreadable
	BrowseNothingHere
	BrowseNothingAuthored
	BrowseNoneFromThisWork
	LabelFrom
	LabelPlaystyle
	LabelElement
	LabelKit
	LabelTraits
	LabelArt
	LabelStages
	LabelBiography
	LabelAtLevel
	LabelEffectiveHP
	StageInWords

	// The check screen.
	CheckHeading
	CheckFooter
	CheckPassed
	CheckFailed
	CheckCounts
	CheckNothingToCheck
	ColumnCharacter
	ColumnArt
	ColumnEffectiveHP
	CheckDoesNotResolve
	CheckOverBudget
	CheckProblem
	CheckWarning
	CheckNote

	// The spar screen.
	SparHeading
	SparFooter
	SparSubject
	SparConditions
	SparOverall
	SparAloneInTheCast
	SparControl
	ColumnOpponent
	ColumnRate
	ColumnRecord
	ColumnTurns
	ColumnFirstMove
	SparRecord
	SparEndless
	SparNote

	// The origins catalog and the form that adds to it.
	OriginsHeading
	OriginsSubtitle
	OriginsFooter
	OriginsEmpty
	OriginsCastCount
	OriginsTally
	OriginAdded
	LabelNote
	OriginFormHeading
	OriginFormSubtitle
	OriginFormFooter
	OriginFormHint
	OriginFormDiscard
	OriginFieldID
	OriginFieldTitle
	OriginFieldMedium
	OriginFieldYear
	OriginFieldNote
	AddRefused

	// Refusals, worded from the values internal/forge hands over.
	ErrorIDTaken
	ErrorMissingName
	ErrorUnknownOrigin
	ErrorUnknownArchetype
	ErrorOriginTaken
	ErrorEmptyKit
	ErrorDuplicateSkill
	ErrorUnknownSkill
	ErrorUnknownElement
	ErrorMissingElement
	ErrorAffinityCount
	ErrorAffinityCounters
	ErrorAffinityUndeclared
	ErrorAffinityRefused
	ErrorCarry
	ErrorCarryRestricted
	ErrorArchetypeRestricted
	ErrorCharacterRestricted
	ErrorUnknownSpecies
	ErrorSpeciesRestricted
	ErrorOriginRestricted
	ErrorSkillTaken
	ErrorMissingSkillID
	ErrorSkillRename
	ErrorPresetOwnedSkill
	ErrorPresetLineageSkill
	ErrorSkillEditBreaksCharacter
	ErrorSkillEditBreaksPreset
	ErrorSkillEditBreaks
	ErrorUnknownPattern
	ErrorUnknownTarget
	ErrorUnknownStatus
	ErrorUnknownCharacter
	ErrorDuplicateEntry
	ErrorNotANumber
	ErrorApplicationShape
	ErrorCurveShape
	ErrorCurveNumber
	ErrorCurveNotPositive
	ErrorCurveShrinks
	ErrorCurveRefused
	ErrorStatField
	ErrorFieldID
	ErrorFieldImage
	ErrorYear
	ErrorAsGiven

	// What a check found, and what a write is worth saying afterwards.
	ProblemMissingArt
	ProblemMissingStageArt
	ProblemDoesNotResolve
	WarningShortReach
	WarningHeldBudget
	NoteWrote
	NoteEdited
	NoteArtMissing
	NoteRebuild
	NoteGoldensMove

	// What a skill does, in sentences, for somebody deciding whether to use it.
	// Every one is derived from the skill's own fields — see Lang.Describe — so a
	// wording here is a sentence with numbers dropped into it and never a fact of
	// its own.
	BlurbHits
	BlurbFlavoured
	LabelFlavour
	SkillHelpFlavour
	BlurbAims
	BlurbStrikes
	BlurbOnce
	BlurbPierces
	BlurbRestores
	// A summon, in two shapes: one that stays and one that does not. Two whole
	// wordings rather than a clause tacked on, for the reason a reply has three
	// — a sentence assembled from fragments is a sentence neither language gets
	// to choose the shape of.
	BlurbSummons
	BlurbSummonsBriefly
	// BlurbSummonedOne and BlurbSummonedMany are what is being called up, which
	// the two above take as their first blank. Separated because one copy reads
	// as a name and several read as a count, and a language may put the number
	// on either side of it.
	BlurbSummonedOne
	BlurbSummonedMany
	BlurbSummonedCopy
	BlurbSummonedCopies
	// BlurbSummonedCreature and BlurbSummonedCreatures are the same fallback for
	// a summon that is its own animal rather than a copy of its caster, told
	// apart by the stat spelling: a fixed line is what an author writes for
	// something that does not scale off whoever called it. Calling a toad a copy
	// was the fallback saying the one thing about it that is untrue.
	BlurbSummonedCreature
	BlurbSummonedCreatures
	// BlurbSummonedShare and BlurbSummonedShareEach put a copy's share of its
	// caster inside the subject rather than beside it, because the share is part
	// of what arrives: "two copies" and "two copies at a tenth of you" are
	// different things on the board, and a sentence that takes one subject
	// should be handed the whole of it.
	//
	// Two of them for the reason BlurbSummonedOne and BlurbSummonedMany are two:
	// several copies each carry the share and one copy simply has it, and a
	// language decides for itself whether that distinction needs a word.
	BlurbSummonedShare
	BlurbSummonedShareEach
	// BlurbSummonedOfCurrent and BlurbSummonedOfBase name *which* stats the
	// share is of, which is a difference a player can act on: a share of the
	// caster as it stands rewards buffing before the copy is made, and a share
	// of its base ignores every buff on the board. One wording for both would be
	// a sentence that is wrong for whichever of the two it was not written for.
	BlurbSummonedOfCurrent
	BlurbSummonedOfBase
	BlurbDrains
	BlurbInflicts
	BlurbGives
	BlurbSelfApplies
	BlurbStrips
	BlurbStripsOne
	BlurbWhenCarrying
	BlurbWhenHurt
	BlurbAmplified
	BlurbSelfAmplified
	BlurbConsumes
	BlurbCostRange
	BlurbCostCells
	BlurbCostSelf
	BlurbCostAccuracy
	BlurbCostCooldown
	BlurbCostCooldownOne
	BlurbCostEveryTurn
	BlurbAnd
	BlurbSideEnemy
	BlurbSideAlly
	BlurbSideSelf
	BlurbSideAll
	BlurbStatAttack
	BlurbStatDefense
	BlurbStatSpeed
	BlurbStatAccuracy
	BlurbStatDodge
	BlurbTraitGrants
	BlurbTraitGrantsGated
	BlurbTraitApplies
	BlurbTraitImmune
	BlurbTraitResists
	BlurbTraitReplyDamage
	BlurbTraitReplyStatus
	BlurbTraitReplyBoth
	BlurbTraitAmplifiesEffect
	BlurbTraitAmplifiesChance
	BlurbTraitWhile
	BlurbTraitDrains
	BlurbTraitNone
	BlurbFooter
	BlurbTraitsFooter
	BlurbMore

	// What a timed effect does, in sentences, for somebody who has just read its
	// name in a log and has nowhere to look it up. Derived from the status book
	// the same way the two above are derived from theirs — see
	// Lang.DescribeStatus — so a duration moving in statuses.json moves the
	// sentence with it.
	BlurbStatusTicks
	BlurbStatusHeals
	BlurbStatusLife
	BlurbStatusLifeCapped
	BlurbStatusControls
	BlurbStatusTaunts
	BlurbStatusShields
	BlurbStatusRaises
	BlurbStatusLowers
	BlurbStatusRaisesOnce
	BlurbStatusLowersOnce
	BlurbStatusStacked
	BlurbStatusNothing
	BlurbStatusLasts
	BlurbStatusLastsOne
	BlurbStatusAlways
	BlurbStatusStacks
	BlurbStatusOneStack
	BlurbStatusCaveat
	StatusesHeading
	StatusesSubtitle
	StatusesEmpty
	StatusesFooter
	MenuStatuses
	MenuStatusesDetail
	MenuPassives
	MenuPassivesDetail
	PassivesHeading
	PassivesSubtitle
	PassivesEmpty
	PassivesFooter
	PassivesNobodyCarries
	ColumnCarriedBy
	BlurbElementStrong
	BlurbElementWeak
	BlurbElementInert
	BlurbElementCaveat
	ElementsHeading
	ElementsSubtitle
	ElementsFooter
	ChartHeading
	ChartSubtitle
	ChartFooter
	ChartEmpty
	ChartMutual
	ChartInert
	ChartRates
	MenuElements
	MenuElementsDetail
	MenuSpecies
	MenuSpeciesDetail
	SpeciesHeading
	SpeciesSubtitle
	SpeciesEmpty
	SpeciesFooter
	SpeciesNobodyIs
	SpeciesNoNote

	keyCount
)

// catalog is one array of wordings per language, indexed by Key.
//
// An array rather than a map, because the index is a declared constant: the
// length is checked against keyCount at build time, and a language cannot grow
// an entry for a key that does not exist.
var catalog = [langCount]*[keyCount]string{
	Vi: &vietnamese,
	En: &english,
}

// Keys returns every declared key, which is what a completeness test walks.
func Keys() []Key {
	out := make([]Key, 0, keyCount)
	for i := range keyCount {
		out = append(out, Key(i))
	}
	return out
}

// Text is one wording, with nothing filled in.
func (l Lang) Text(key Key) string {
	if !l.Valid() || key < 0 || key >= keyCount {
		return ""
	}
	return catalog[l][key]
}

// Say is one wording with its blanks filled.
//
// A key whose wording takes no arguments is returned as it stands, so that a
// stray argument cannot decorate a line with %!(EXTRA ...).
func (l Lang) Say(key Key, args ...any) string {
	text := l.Text(key)
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

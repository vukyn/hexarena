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
	// VersionFlagUsage describes the flag that prints what the binary is and
	// exits: the build string it announces, the protocol number it speaks and
	// the digest of the data it embeds — the three a room's gate turns on.
	//
	// ⚠️ **Only the description is worded here; the output is not, and that is
	// deliberate rather than an omission.** What -version prints is
	// wire.Version.Report, whose two labels are `protocol` and `data` — the same
	// labels cmd/hexarena-host prints on its banner and the same ones both
	// version refusals tell a player to read, which is why JoinVersion leaves
	// them in English in both languages too. Translating them on one end would
	// break the one instruction those refusals give, and it would also give the
	// two binaries two spellings of one output.
	VersionFlagUsage

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
	// ChoiceSlots is ChoicePosition's answer for a list that fills slots
	// rather than a list being walked: the second figure is what binds, not
	// how many rows there are to choose between.
	ChoiceSlots

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
	// The filter on the listing: the footer while the field has the keyboard,
	// what it says with nothing typed yet, what it says with a query in it, and
	// what a query that found nothing says in place of the rows.
	//
	// Four keys rather than one wording with a blank left empty, because the
	// four are answers to different questions: an empty field needs to say what
	// to type into it, a full one needs to say how much of the book is left, and
	// "nothing matched" has to be said where the rows would have been or the
	// screen is an empty box.
	SkillsFilterFooter
	SkillsFilterPrompt
	SkillsFiltering
	SkillsFilterNothing
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
	SkillFieldCrit
	SkillFieldRestores
	SkillFieldDrains
	SkillHelpOnItself
	SkillHelpPierce
	SkillHelpCrit
	SkillHelpRestores
	SkillHelpDrains
	SkillFieldKeptForElements
	SkillFieldKeptForRoles
	SkillFieldKeptForCharacters
	SkillFieldKeptForSpecies
	SkillFieldKeptForOrigins
	LabelDamage
	DamageLine
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
	// the form spends nineteen of the twenty body lines a 120x24 window has, and
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

	// Reading the row under the cursor rather than choosing it, which is the
	// picker's other state. Only two of the kinds have a describer behind them,
	// so the key is announced on a footer of its own rather than on the one
	// every picker shares.
	PickerDescribeFooter
	PickerReadingFooter

	// The status picker, which is the multi-select with one answer more: a
	// status is an id *and* a chance, so it collects a number beside the list
	// rather than sending an author back to a text field to remember a syntax.
	PickerStatusesTitle
	PickerStatusHint
	PickerStatusFooter
	PickerChance
	StatusTicks
	CategoryStatDebuff
	CategoryControl
	CategoryBuff
	CategoryShield
	CategoryRegen
	CategoryTaunt
	CategoryHealCut
	CategoryCharge
	CategoryAbsorb
	CategoryReserve

	// The same nine categories as **noun phrases**, for a sentence that names
	// one rather than a column that explains one. The family above answers "what
	// does this kind of status do" and is worded as a predicate — "lowers a
	// stat" — which cannot be dropped into "strips 1 stack of ___"; these answer
	// "what is the kind called". Two families rather than one with a flag,
	// because they are two questions and a flag would be one function with two
	// jobs. See Lang.StatusCategoryNoun.
	//
	// The English shape is an **uncountable noun phrase**: no article, no
	// plural, lower case, so the same wording reads under "1 stack of" and "2
	// stacks of" alike. Every one of them is held to it. The alternative considered was
	// Vietnamese's own "hiệu ứng X" shape spelled out in English ("a
	// stat-lowering effect"), which needs an article the frame has nowhere to
	// put.
	//
	// These carry the Vietnamese wordings that used to live in gloss.go's
	// categoryGloss, which is gone: a category is a Go enum rather than a data
	// id, so Gloss's rule that English shows an id exactly as the data writes it
	// was what printed "stat_debuff" on an English line.
	CategoryNounDot
	CategoryNounStatDebuff
	CategoryNounControl
	CategoryNounBuff
	CategoryNounShield
	CategoryNounRegen
	CategoryNounTaunt
	CategoryNounHealCut
	CategoryNounCharge
	CategoryNounAbsorb
	CategoryNounReserve

	// What a kit and a preset demand of an affinity.
	KitTakesAnyElement
	KitNeeds
	PresetTakesAnyElement
	PresetNeeds
	ElementJoiner
	// ListComma separates every item of a list but the last two, which take the
	// conjunction instead. One key pair rather than one per joiner: the comma is
	// punctuation and identical in both languages, while the conjunction it hands
	// over to belongs to whoever is doing the joining.
	ListComma

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
	NoteWrote
	NoteBattleVerify
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
	// The chance only. What a critical strike multiplies by is one game-wide
	// constant on combat.Rules, and this package may not import that — nor
	// should a description restate a number every skill in the game shares.
	BlurbCritical
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
	// A CAP reads the same field the clause above does and means the opposite of
	// it, so it gets a clause of its own rather than a count inside that one.
	// "is carrying sundered" is what the shipped split said before this existed,
	// and it is exactly backwards: the gate opens while the caster holds FEWER
	// than the bound, so the sentence described the one state in which the skill
	// cannot be cast.
	//
	// It names no status on purpose. A cap's counter is bookkeeping — nobody chose
	// to hold it and it cannot be spent on anything else — so naming it spends the
	// line on a word that answers no question a reader has.
	BlurbWhenUnspent
	BlurbWhenHurt
	BlurbAmplified
	BlurbSelfAmplified
	BlurbSelfGradient
	BlurbConsumes
	BlurbConsumesStacks
	BlurbAmplifiedShape
	BlurbSelfAmplifiedShape
	// The openings a **gating** condition takes instead, in the sentence and in
	// the compact line.
	//
	// A gate is not an amplifier and the two cannot share an opening. Every other
	// condition here is read while the skill resolves, so "while this unit is
	// carrying five stacks" is the truth about it: the skill is cast either way
	// and the clause decides what it pays. A gate is read *before* the skill is
	// offered — skill.Condition.Gates says so, and battle.options acts on it — so
	// the same words are a plain lie: without the fuel there is no cast to be
	// carrying anything during. The compact line said something worse still,
	// because a shape-paid condition with no figure fell to the wording ending in
	// "spreads", which is the one thing a caster's own condition is forbidden to
	// do.
	//
	// Self-facing only, and that is a refusal rather than an omission:
	// resolveCondition rejects a gate on the target's condition outright, since
	// that reading would have to be taken per aim.
	//
	// Two of them for the reason the amplified pair is two — a gate may still be
	// paid a flat bonus on top, and the sentence quotes a figure exactly when
	// there is one to quote.
	BlurbSelfGated
	BlurbSelfGatedShape
	SummarySelfGated
	SummarySelfGatedShape
	SummaryAmplifiedShape
	SummaryAmplifiedArc
	SummarySelfAmplifiedShape
	SummarySelfScaled
	SummarySelfRestored
	BlurbArcs
	BlurbChains
	BlurbConsumesEachStrike
	// The singulars. English needs them and Vietnamese does not, which is why
	// they are keys rather than a count formatted into one wording — the same
	// shape BlurbStripsOne and BlurbStatusLastsOne already have.
	// The rider a condition pays only where it holds. It is its own key rather
	// than BlurbInflicts reused, because the sentence it joins has already said
	// what the condition is, so repeating the condition would say it twice.
	BlurbConditionInflicts
	BlurbConsumesStacksOne
	BlurbConsumesEachStrikeOne
	BlurbConsumesPile
	BlurbScalesPerStack
	BlurbRestoresPerStack
	BlurbConsumesUpTo
	BlurbAtLeast
	BlurbRepeats
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
	// The renewal, worded apart from the grant because the two differ in the one
	// thing a reader needs: a grant is simply carried, and this arrives again
	// every turn and can be taken off in between.
	BlurbTraitRenews
	BlurbTraitApplies
	BlurbTraitImmune
	BlurbTraitResists
	BlurbTraitVulnerable
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

	// The same skill in ONE line, for a list where four of them are compared at
	// once rather than read one at a time — see Lang.SummariseSkill, which says
	// why that is a fourth describer and not Describe with the prose dropped.
	// Every one of these is a clause rather than a sentence: no capital, no full
	// stop, joined with the same middle dot the cost line is joined with, and
	// carrying only figures the skill itself declares.
	SummaryDamage
	SummaryRestores
	SummaryDrains
	SummaryStatus
	SummarySelfApplies
	// A strip is a **count**, never the list Describe prints. Enumerating the
	// categories cannot fit and never could: purify names three, and in
	// Vietnamese "gỡ 3 hiệu ứng gây hại theo lượt và hiệu ứng giảm chỉ số và hiệu
	// ứng khống chế" is 79 cells before the aim and the cooldown are appended —
	// longer on its own than the whole line has. The full description keeps the
	// enumeration, which is its job and its line has room for it.
	//
	// Two wordings, chosen per skill from what that skill actually names, because
	// the category universe is **not** all harmful: dot, stat_debuff, control and
	// taunt are, and buff, shield and regen are not — status.Category.Harmful is
	// the function that separates a cleanse from a dispel. So a skill stripping
	// only harmful things may say so, and anything else gets the count and no
	// claim about it. A benign strip has no word of its own because nothing is
	// authored as one and the countng wording is *correct* for it; a third key
	// belongs to whoever writes the first dispel.
	SummaryStrips
	SummaryStripsHarmful
	// SummaryAmplified and SummarySelfAmplified are the two amplifiers, and they
	// are two wordings for the reason BlurbAmplified and BlurbSelfAmplified are:
	// the clause inside them says nothing about *whose* health or whose stacks
	// are being counted, so only the opening can tell a condition read against
	// the target from one read against the caster. Reading only the first is a
	// mistake this repository has already paid for once — forge.PreviewDamage
	// showed outrage and comeback, the two skills whose whole design is a
	// caster-side term, at their plain power.
	SummaryAmplified
	SummarySelfAmplified
	SummaryGradient
	SummaryHurt
	SummaryCooldown

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
	BlurbStatusStores
	BlurbStatusStocks
	BlurbStatusSeeps
	BlurbStatusAbsorbs
	BlurbStatusAbsorbsPool
	BlurbStatusSpills
	BlurbSkillUnstoppable
	BlurbTraitConverts
	BlurbSkillCosts
	SummaryCosts
	BlurbStatusCutsHealing
	BlurbStatusCutsHealingOnce
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

	// The build catalogue.
	BuildsHeading
	BuildsSubtitle
	BuildsFooter
	BuildsEmpty
	BuildsNoneForThisOne
	BuildsNoTrait
	LabelIntent
	MenuBuilds
	MenuBuildsDetail

	// The squad builder: the sides an author builds to fight each other.
	SquadsHeading
	SquadsSubtitle
	SquadsEmpty
	SquadsFooter
	SquadColumnID
	SquadColumnMembers
	SquadMemberCount
	SquadHeading
	SquadEditFooter
	SquadFieldID
	SquadFieldName
	SquadAddMember
	SquadFull
	// SquadFurthest is what an empty stage on a placement means: the furthest
	// form the level reaches. ⚠️ **It is only true on a line that does not
	// fork** — a level that has reached two grown forms has no furthest one, so
	// on a fork the two keys below are drawn instead.
	SquadFurthest
	// SquadForkUnnamed is the same empty stage read on a line that forks, and it
	// is the word SquadFurthest cannot be there: "furthest" names nothing where
	// there are two ends. It goes in the form chooser and in the member's row in
	// the squad, which are one fact drawn twice.
	SquadForkUnnamed
	// SquadForkArms is the line under the member's fields that names the arms and
	// what to do about them, in SquadHeldBack's place and for its reason: a value
	// the screen refuses to guess has to say what the reader can do instead. It
	// carries the consequences as well, because both are invisible — the two
	// loadout lists silently offer only what every arm learns, and the save
	// refuses the member outright.
	SquadForkArms
	SquadLoadoutCount
	SquadUnitHeading
	SquadUnitFooter
	SquadFieldCharacter
	// The line under the member's fields when the character it already holds has
	// been held back in cast.json. The chooser goes on offering that one — a
	// squad on the file may not be edited into naming somebody else behind its
	// author's back — so the screen has to say why a character nothing else
	// offers is on the list, or the row reads as the flag not working.
	SquadHeldBack
	SquadFieldLevel
	SquadFieldStage
	SquadFieldSlot
	// One wording per rank, read beside the cell on the slot row. A coordinate
	// says where a unit is in the file and says nothing about what standing
	// there is worth, and the thing that is worth something is the rank: reach
	// is counted in ranks from the far side, so the front one is what an attack
	// meets and everything behind it is screened.
	SquadRankFront
	SquadRankMiddle
	SquadRankBack
	SquadFieldSkills
	SquadFieldPassives
	SquadNothingChosen
	SquadPickSkills
	SquadPickPassives
	// A hint each rather than the picker's default, because the builder's two
	// lists are already the character's own learnset: nothing on either can be
	// refused, so the mark the default names never appears. They are two rather
	// than one because the lists differ in what is left to say — four slots in
	// an order against a single slot that may be left empty.
	SquadKitHint
	SquadTraitHint
	SquadFormation
	// The caption under the marker that sits beneath the front column. It is a
	// line of its own rather than a clause on SquadFormation, because the fact
	// is about one column and the marker is what ties it to that column — a
	// caption saying "on the left" leaves the reader to count.
	SquadFormationFront
	SquadFormationLegend
	SquadDiscard
	SquadDiscardSaved
	MenuSquads
	MenuSquadsDetail

	// The squad fight: two sides measured against each other.
	MenuFight
	MenuFightDetail
	FightHeading
	// What the screen says when the catalogue is empty. It is its own wording
	// rather than the catalogue's, because that one names a key this screen does
	// not have: reaching the fight with nothing built is only possible from the
	// menu, and the answer there is to go and build a side.
	FightNoSquads
	FightAgainst
	FightControl
	FightConditions
	FightRate
	FightRecord
	FightRecordLine
	FightBySide
	FightBySideLine
	FightLength
	FightLengthLine
	FightEndless
	FightEndlessLine
	FightCaution
	FightFooter

	// Playing a battle by hand against the opponent the engine plays.
	PlayHeading
	PlaySeed
	// Where the log's frame sits in the whole history, drawn on the heading row
	// rather than on a row of its own: this screen has no spare row — the budget
	// below is a whole feature about that — and the heading is about seventeen
	// cells of the seventy-nine there are.
	//
	// Shown whenever rows are hidden rather than only while the reader has
	// scrolled back, because the other half of the defect it answers is that
	// nothing on the screen said a history existed at all: a reader who cannot
	// see that there are three hundred rows will not go looking for the key that
	// reaches them.
	PlayLogRange
	PlayYourTurn
	PlayAimAt
	// The splash cells the aim under the cursor also reaches, named once above
	// the list rather than on every row: the mark is the same one the shape
	// diagram uses for the same thing, so a reader who has seen one screen has
	// seen the other, and repeating the sentence per row would spend three lines
	// saying what one says.
	PlayAimSplash
	// Why an option on the turn in front cannot be taken, one wording per
	// battle.Block that can reach a screen.
	//
	// The engine builds an English sentence of its own — Option.Reason — and it is
	// still what a client with no Lang in hand prints. These are the same four
	// facts said in the reader's language, off the enum and the three counts the
	// option carries, because internal/core may not import this package: a
	// sentence assembled there could only ever be assembled in one language.
	//
	// ⚠️ **The fuel one is the reason this family exists at all.** A cooldown
	// explains itself — the row says three and the reader waits three — so a
	// greyed row was survivable while every refusal was one. A skill whose
	// cooldown is nought and which is greyed out anyway explains itself with
	// nothing, so this has to carry all three facts: how much fuel, of what, and
	// how much the caster is actually holding. The status is named through its
	// gloss for the reason every data id on a Vietnamese screen is, and the count
	// held is read off the caster rather than restating what is needed — "needs 5,
	// holding 5" is the one pair of numbers this row can never draw.
	PlayBlockedUnknown
	PlayBlockedCooldown
	// English needs the singular and Vietnamese does not, which is two keys
	// rather than a plural rule — the same division BlurbCostCooldownOne,
	// BlurbStripsOne and PlayHiddenUnitsOne are two keys for. The row is drawn on
	// every turn a skill is cooling, so "1 turns of cooldown left" is the wording
	// a reader meets most often out of the four.
	PlayBlockedCooldownOne
	PlayBlockedFuel
	// A gate that has been spent out rather than one short of fuel, and it is
	// its own wording because the two say different things to do next: short of
	// fuel you wait and fill the tank, spent out you are finished with the skill
	// for the rest of the battle. Reusing the fuel wording would tell a reader
	// to wait for something that is never coming.
	PlayBlockedSpent
	PlayBlockedNoReach
	PlayWon
	PlayLost
	PlayDrawn
	PlayEmptied
	PlayFooter
	PlayAimFooter
	PlayOverFooter
	// What the battle screen gave up to fit the window, and why. The screen
	// budgets its own body — the heading and the option list are reserved and
	// everything else takes what is left — so a window shorter than the whole
	// board, roster, order line and log is the ordinary case rather than the odd
	// one, and a screen silently missing its board reads as a broken screen.
	//
	// One line rather than a marker per section: the rows are exactly what is
	// scarce when this is drawn at all.
	PlayHidden
	PlayHiddenBoard
	PlayHiddenOrder
	PlayHiddenLog
	PlayHiddenNote
	// English needs the singular and Vietnamese does not, which is two keys
	// rather than a plural rule — the same division BlurbCostCooldownOne and
	// BlurbStripsOne are two keys for.
	PlayHiddenUnits
	PlayHiddenUnitsOne

	// The read-only spellings of the three footers whose screens author.
	//
	// ⚠️ **A read-only footer is a second wording rather than the authoring one
	// with a clause taken out of it.** Deleting `a thêm` from a rendered line
	// would leave the separators either side of it, and nothing would measure
	// what was left — while a wording of its own is measured by every sweep the
	// catalog already has: both languages, one cell per letter, inside the floor.
	// Which of the two a screen draws is Context.Footer's one decision, taken
	// beside the decision that ignores the keystroke, so the two cannot disagree.
	SkillsReadFooter
	OriginsReadFooter
	SquadsReadFooter

	// The game client, cmd/hexarena-tui: a menu of the catalogues a player
	// reads, and a battle.
	//
	// Most of the authoring tool's menu wordings serve it unchanged — a listing
	// of the cast is a listing of the cast — so what is here is the entries whose
	// wording names an authoring key, plus the two lines that point somebody at
	// the other front-end and have to point at a different one.
	GameNotATerminal
	GameTerminalTooSmall
	GameMenuNote
	GameMenuSkillsDetail
	GameMenuWorksDetail
	GameMenuSquadsDetail
	GameMenuBattle
	GameMenuBattleDetail

	// The two protocol enums, worded. wire.Code is why a peer was turned away
	// at the gate and wire.Closure is why a match stopped for a reason the
	// board cannot show; both travel as an **id** precisely so the sentence
	// lives at this end, in the reader's own language, which is why
	// internal/wire may not import this package and these lines cannot live
	// there. See Lang.Refusal and Lang.Closure.
	//
	// Held **complete**, the way the status categories are and for the same
	// reason: a code is a Go enum rather than a data id, so a value with no
	// wording is a gap in the catalog rather than data nobody has reached yet.
	// TestEveryRefusalIsWorded and TestEveryClosureIsWorded walk
	// wire.CodeCount and wire.ClosureCount to say so.
	//
	// ⚠️ **A bare translation of the id would be worthless**, which is the one
	// rule every line here is written under: "dữ liệu không khớp" tells a
	// player nothing they can do, and the player reading one of these is stuck
	// at a lobby with no other information in front of them. Each says what
	// happened *and* what to do about it. The two that are a client bug or a
	// stale build rather than anything the player did — RefusalIllegalAction and
	// RefusalUnknownMessage — say so outright, because a player told only
	// "illegal action" will go looking for their own mistake.
	//
	// ⚠️ **RefusalNotYourTurn was the third of those and is not any more.** It
	// used to read "that is the program's mistake and not yours, and matching
	// builds is the fix", which was true while the only way to provoke it was a
	// peer that was wrong about whose turn it was. The game client's chooser now
	// gives up on a prompt at the allowance plus a grace and **passes** — the
	// third arm that stops a peer's death stranding it — so by then the room has
	// usually passed for that seat already and answers this code on the
	// **ordinary** timeout path. Measured on a real match over a socket: a
	// player who let the clock run out was told their program was broken. The
	// line now says what happened and that the board is right, which is the
	// whole of what there is to do about it.
	//
	// One wording per value and no second family, unlike the status categories:
	// they have two because two sentences genuinely needed two shapes, and here
	// there is one consumer — the lobby below — and one shape.
	//
	// ⚠️ **The ban-and-pick added one of each and every line above is unchanged.**
	// RefusalNotYourTurn and RefusalIllegalAction are *reused* by the draft rather
	// than twinned, because a decision out of turn and a decision the prompt did
	// not offer are the same facts about the same thing and take the same action
	// from a player; the two wire.Code comments carry that argument. What could
	// not be reused is the squad refusal, whose whole sentence is advice that
	// would be wrong here. → RefusalSquadUnwanted.
	RefusalNone
	RefusalProtocolMismatch
	RefusalDataMismatch
	RefusalBadPassword
	RefusalRoomUnknown
	RefusalRoomFull
	RefusalSquadRefused
	RefusalNotYourTurn
	RefusalIllegalAction
	RefusalUnknownMessage
	// RefusalSquadUnwanted is the ban-and-pick's one refusal at the **gate**, and
	// the reason it is worth a wording of its own rather than reusing the squad
	// one is that the advice is the opposite: nothing is wrong with the squad, so
	// a line telling a player to fix it would send them to check levels and forms
	// that are all fine. It says the room drafts and to come back with no squad.
	RefusalSquadUnwanted
	ClosedNone
	ClosedLeft
	ClosedStopped
	// ClosedDraftExpired is a draft that ran out of time.
	//
	// ⚠️ **It is spelled Closed* and not Closure*, like its three neighbours, and
	// that is load-bearing rather than tidy.** TestNoKeyIsOrphaned walks bare
	// identifier text module-wide, so a key called ClosureDraftExpired would be
	// counted as used by internal/wire's own constant of that name and could then
	// lose its last real reader without anything going red. Do not "fix" the
	// spelling. → the note on that test.
	ClosedDraftExpired

	// FormChoice is the row the three read-only views draw when the character in
	// front has more than one grown form at the level being read — a line that
	// forks, where progression.Line.StageAt refuses on purpose rather than pick an
	// arm for somebody.
	//
	// It carries the key that changes it rather than leaving that to a footer,
	// which is BlurbMore's own shape and is here for a sharper reason: a fork is a
	// fact about the character under the cursor, so a footer that named the key
	// would have to name it on every character the key does nothing on. Drawn only
	// while there is a choice, so a line that does not fork is exactly what it was.
	FormChoice
	// The two seats a room hands out, worded.
	//
	// wire.Seat travels as an id like every other protocol value, and this is
	// the far end of it — the same shape Lang.Refusal and Lang.Closure take,
	// keyed by the name wire.Seat already writes. It is worded rather than
	// printed raw because a seat goes on the waiting screen beside sentences: a
	// bare "host" on a Vietnamese screen is an English word in a column, which
	// is exactly the leak the sweeps hunt for elsewhere.
	SeatHost
	SeatGuest

	// # The lobby: joining a room, waiting for the second player, the result
	//
	// The three screens cmd/hexarena-tui grew for a match over a LAN, and they
	// live in that client rather than in internal/screen because a lobby is
	// drawn by one client — the authoring tool has no room to join. The
	// wordings are here all the same, because that is where every wording in
	// this repository lives and the client may hold none of its own.
	//
	// ⚠️ **This is what makes the ten refusals and the three closures readable
	// by a person.** They were worded and unread until these screens arrived,
	// which TODO.md records as one step narrower than "shipped dead": the join
	// screen is where a refusal at the gate is drawn, the live battle is where
	// a refusal mid-match is, and the result screen is where a closure is.
	GameMenuJoin
	GameMenuJoinDetail

	JoinHeading
	JoinCodeLabel
	JoinCodePlaceholder
	JoinPasswordLabel
	JoinPasswordPlaceholder
	JoinSquadLabel
	JoinHint
	JoinNoSquad
	JoinCodeLength
	// JoinDataEdited is the one honest consequence of the mirror being built
	// from the embedded books: with a --data directory whose files differ from
	// the ones this binary embeds, the catalogues are drawn from the library
	// and the battle is fought on the built-in data, so the two can disagree.
	//
	// ⚠️ It is drawn only when the two digests really **differ**, measured
	// rather than assumed: a --data pointing at an unmodified copy is the
	// common case, and warning about that would be noise on every join.
	JoinDataEdited
	JoinDialling
	JoinRefused
	// JoinVersion is what the binary drawing this screen **is**: the digest of
	// the data it embeds and the build string it announces — the two numbers
	// cmd/hexarena-host already prints on its banner.
	//
	// ⚠️ **It exists because the client was the side that could not read them,
	// and two refusals were already telling a player to.**
	// RefusalDataMismatch says *read the data line on each*, in both languages,
	// and RefusalProtocolMismatch says the same about the build line. Before
	// this key there was no data line and no build line on this end at all, so
	// both sentences asked for something one of the two screens did not have —
	// which is how a real refusal ended as a question to the author rather than
	// as two people comparing two short strings.
	//
	// ⚠️ **Mine, never theirs.** wire.Refused carries a Code and nothing else,
	// Welcome carries no version, and a refused client never receives a Welcome
	// anyway — so this end cannot learn the host's digest and this wording must
	// not read as though it had. Hence "this machine": what is drawn is one
	// side of a comparison a person makes.
	//
	// ⚠️ **`data` and `build` stay in English in both languages.** They are the
	// labels hexarena-host prints and the labels the two refusals name, so
	// translating them on this end would break the one instruction those
	// refusals give. Everything around them is the reader's language.
	JoinVersion
	JoinFooter

	WaitingHeading
	WaitingForPeer
	WaitingRoom
	WaitingSeat
	WaitingFormat
	WaitingFooter

	ResultHeading
	ResultStanding
	ResultBattleLine
	ResultWon
	ResultLost
	ResultDrawn
	ResultFooter

	// The battle screen in live mode. Three footers and one drawn line.
	//
	// ⚠️ **The footers are second wordings rather than the local ones with
	// clauses deleted**, which is the rule Context.Footer states for the
	// authoring pair: a program deleting a clause out of a rendered line leaves
	// the separators either side of it, and nothing measures what is left. A
	// live battle names neither `u`, `n` nor the save key.
	PlayLiveWaiting
	PlayLiveFooter
	PlayLiveAimFooter
	PlayLiveOverFooter

	// The two countdowns on a live battle's heading row: what is left of the
	// open turn's allowance for the player reading the screen, and for the
	// player on the other machine.
	//
	// ⚠️ **Two wordings rather than one with a marker moved into it**, which is
	// the rule the three live footers are written under. What differs between
	// them is which of the two clocks is the one running, and that is a word in
	// the label — `lượt bạn` / `your turn` against `lượt bên kia` / `their
	// turn` — because Palette's own rule is that colour is decoration and never
	// information, and an arrow would be read as one of the arrow **keys** the
	// footers beside them name.
	//
	// Both take the two clocks in the same order, always the reader's own
	// first: a number that changes position when the turn changes is a number
	// nobody can watch.
	PlayClockYours
	PlayClockTheirs

	// The composition-bonus reference: what a squad is paid for what it shares.
	//
	// The listing carries the scope in a column of its own rather than only in
	// the description, because the two kinds are the one thing a reader cannot
	// work out from the grants: a squad-wide rung and a sharers-only rung put
	// the same status on a unit, and only the row says which of them a unit is
	// carrying it for.
	BonusesHeading
	BonusesSubtitle
	BonusesEmpty
	BonusesFooter
	MenuBonuses
	MenuBonusesDetail
	// BlurbBonusCounts and BlurbBonusGoesTo are the two questions a bonus
	// answers before its rungs mean anything: what is being counted, and who
	// receives what the count buys.
	BlurbBonusCounts
	BlurbBonusGoesTo
	// BlurbBonusRung is one rung: how many have to share, and what each unit
	// that receives it is held. One line per rung rather than a table, so a
	// bonus that grows from two rungs to four grows the block by two lines
	// instead of needing a column nobody has measured.
	BlurbBonusRung
	BlurbBonusCaveat
	BonusAxisElement
	BonusScopeSquad
	BonusScopeSharers

	// # The ban and pick, as the game client draws it
	//
	// The draft screen's wording, and the one thing to know before adding to it:
	// ⚠️ **the glossary was coined at step 3 and these follow it rather than
	// re-coining** — `cấm` is a ban, `chọn tướng` is a pick, `lượt chọn tướng` is
	// a pick's turn. Three shipped wordings already use those words
	// (RefusalNotYourTurn, RefusalSquadUnwanted, ClosedDraftExpired), so a fourth
	// spelling here would be the same idea said two ways on two screens a player
	// sees minutes apart.
	//
	// ⚠️ **Six of the loadout's wordings are the squad builder's own and are
	// reused rather than restated** — SquadFieldStage, SquadFieldSkills,
	// SquadFieldPassives, SquadPickSkills, SquadPickPassives, SquadKitHint,
	// SquadTraitHint, SquadNothingChosen, SquadLoadoutCount, SquadFurthest and
	// SquadForkUnnamed. A drafted loadout and a saved member's loadout are the
	// same question asked of the same books through the same cast.ChooseLoadout,
	// and none of those lines says "squad" in either language. A second set would
	// be two wordings for one row, which is the mistake the two `Footer` halves
	// of Context already exist to avoid.
	//
	// ⚠️ **A cancelled draft reuses ClosedDraftExpired**, which is the closure the
	// room sends for the same event. The draft screen is where a player is
	// standing when the allowance runs out and the result screen is where they
	// land afterwards; wording it twice would let the two disagree about what
	// happened.

	// DraftHeading names the screen and DraftPoolLeft says how much of the pool
	// is still in it, which is the one number that makes the marks on the rows
	// mean something at a glance.
	DraftHeading
	DraftPoolLeft
	// The five states a row of the pool can be in. They are a **column of
	// wording** rather than a colour, for Palette's own reason: whether the
	// character a player is looking at is still available is the single fact this
	// screen exists to show, and an answer carried by colour is an answer a
	// monochrome terminal does not give.
	//
	// Banned and picked are told apart, and so is the side, because they are four
	// different things to a player: a character they banned is one they chose to
	// remove, one the opponent banned is one the opponent feared, one they picked
	// is theirs and one the opponent picked is what they will be fighting.
	DraftStateOpen
	DraftStateYouBanned
	DraftStateTheyBanned
	DraftStateYouPicked
	DraftStateTheyPicked
	// Whose decision is due and which kind, in six whole sentences rather than
	// two skeletons and a step name.
	//
	// ⚠️ **Six wordings and not two, deliberately.** A line assembled from "your"
	// plus a translated step name is a line neither language can inflect: "your
	// ban" and "your loadout for %s" have different shapes in English and
	// different ones again in Vietnamese, and the seat's own name has to sit
	// where each language puts a subject. The three `Their` wordings take the
	// **worded** seat (→ Lang.Seat) rather than the protocol's spelling.
	DraftYourBan
	DraftYourPick
	DraftYourLoadout
	DraftTheirBan
	DraftTheirPick
	DraftTheirLoadout
	// DraftNotBegun is the state this screen exists as much for as for the pool.
	//
	// ⚠️ **Nothing on the wire tells a client the room is full**, so a draft with
	// no decision recorded is either a room still waiting for its second player
	// or a room that filled a moment ago and has been asked nothing — and the two
	// are indistinguishable from here. What is *not* indistinguishable is the
	// consequence: a decision taken before the second seat is taken is refused
	// and thrown away, and a player who cannot see that has no way to know their
	// ban was not taken. So this says what is true of both readings and warns
	// about the one that costs something.
	DraftNotBegun
	// DraftRefused is the lead-in over a refusal the room sent, which is drawn
	// through Lang.Refusal exactly as a battle's is. → PlayScreen.LiveRefusal on
	// why a refusal crosses as a name.
	DraftRefused
	// DraftOnlyOne is a decision with one candidate, which is not a decision.
	//
	// ⚠️ **Whether it can happen is arithmetic and it has moved three times in two
	// days** — draft.Slack is the expression, the final pick sees `slack + 1`
	// candidates, and that was one character when the pool was sixteen. So the
	// line is drawn off the candidate list actually in hand rather than off any
	// figure written down, and a screen presenting a list of one as a choice is
	// the failure it is here to prevent.
	DraftOnlyOne
	// DraftArrangingNow is the ban and pick played out and both sides placing
	// their units, which is the phase the arrange screen draws. This is the line
	// that says the picking is over.
	DraftArrangingNow
	// The two sides' own blocks: how many each has picked, what each pick will
	// field, and what each side took out of the pool.
	//
	// DraftBannedNobody is a side with no ban to its name, and it is a wording
	// rather than an empty column for the reason every absence in this program is
	// declared: a blank where a list goes reads as a drawing that failed.
	DraftYourSide
	DraftTheirSide
	DraftNoPicks
	DraftLoadoutOpen
	DraftBanned
	DraftBannedNobody
	// The loadout editor's fourth row and what it says when the kit is legal.
	//
	// ⚠️ **A row of its own rather than a key chord, and that is the whole reason
	// this screen has four rows for three fields.** `enter` means "do what is
	// under the cursor" on every other screen in this program, and on a loadout
	// there are two things it could mean — open the list on this row, or send the
	// decision — so one of them had to become a row. What that buys is one footer
	// naming one key rather than a footer explaining that a key means two things.
	DraftFieldSend
	DraftSendReady
	// DraftForkArms is the line a forking evolution needs, and it is NOT the squad
	// builder's SquadForkArms even though the two say almost the same thing: that
	// one's last clause is *"the save refuses"*, and a draft has no save. What is
	// refused here is the **decision**, which is a different consequence to a
	// player — one loses a file they were editing, the other loses a turn of a
	// match — so the clause is worded rather than the sentence being shared with a
	// blank in it.
	DraftForkArms
	// JoinBringNone is the join screen's last squad position, and it is here rather
	// than beside the other Join keys for the reason it exists at all: a room that
	// **drafts** refuses a squad, so bringing none is the answer that room wants —
	// and a client cannot know a room drafts until it has been welcomed, because
	// the hello carrying the squad goes first. → joinScreen.Squad, and
	// wire.CodeSquadUnwanted.
	JoinBringNone
	// The four footers, one per thing this screen can be doing. The reading one
	// names no decision key at all, because the decision is not this player's to
	// take — a footer naming a key the screen ignores is the failure
	// Context.Footer exists to prevent, and here the difference is a whole match.
	DraftBanFooter
	DraftPickFooter
	DraftReadingFooter
	DraftLoadoutFooter

	// The arrange screen: the last decision of a draft, and the only one that is
	// about the board rather than about the pool.
	//
	// ArrangeHeading names it and ArrangePlaced counts what has been put down, so
	// the one number that says how far through the decision a player is sits
	// beside the name of it.
	ArrangeHeading
	ArrangePlaced
	// ArrangeDepth is why the whole board is drawn rather than one side's nine
	// cells, and it is the measured reason rather than a symmetry: placement in
	// this game is **purely defensive** and its whole value is rank depth — moving
	// a roster's aces to the back column and changing nothing else read 27.6% →
	// 47.3% over 4000 seeds. A player choosing cells without being able to see
	// which rank each one is cannot take the decision that is worth twenty points.
	ArrangeDepth
	// ⚠️ **ArrangeAllowance is the corrected reading of the clock and the screen
	// may not imply any other.** The phase's allowance is **not** one for the
	// phase: Server.settled re-arms off the reading after every batch, so the side
	// that arranges first hands its opponent a fresh full allowance and the worst
	// case is about twice one. Two readings of this were written into step 4's
	// brief and both were wrong, which is why the fact is a wording rather than a
	// countdown — a single number on a screen during a phase with two decisions
	// open is the wrong reading drawn in figures.
	ArrangeAllowance
	// ArrangeInHand is the pick waiting for a cell and ArrangeUnplaced is a row
	// that has not been given one, which is an absence declared rather than a
	// blank column — the rule every absence in this program is drawn under.
	ArrangeInHand
	ArrangeUnplaced
	// ArrangeSendReady is every pick placed, which is the one state enter sends
	// in.
	ArrangeSendReady
	// ArrangeSent is this side having arranged with the phase still open, and it
	// has to say **why** it is waiting: the phase is simultaneous and secret by
	// design, so a screen that only said "waiting" would read as a match that had
	// stalled.
	ArrangeSent
	// ArrangeLegend names the marks the board carries, for the reason the shape
	// diagram's legend does: two characters in a hex is a symbol, and a symbol
	// nothing names is a drawing a reader has to guess at.
	ArrangeLegend
	// ArrangeCursorAt is the cell under the cursor and the rank it stands in,
	// said together — the coordinate is what the data writes and the rank is the
	// half that means something, which is the squad builder's own pairing.
	ArrangeCursorAt
	// The three footers, one per thing this screen can be doing. The waiting one
	// names no decision key at all, because there is no decision left to take.
	ArrangeFooter
	ArrangeSendFooter
	ArrangeWaitingFooter

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

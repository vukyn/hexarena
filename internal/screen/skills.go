package screen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the new-skill form, in the order they are walked.
//
// Exported because both a raise site and a client's own fixtures name one: the
// picker a list field opens carries the field it fills, and cmd/hexforge-tui
// builds the form's states by hand for its width sweep. A fixture cannot follow
// a screen across a package boundary.
//
// Nine core fields, the display name, the statuses it inflicts (both sides), and
// the three allowlists. What is deliberately absent is requires, self_requires,
// self_gradient, strips, scaling and summons: each is a composite worth several
// questions of its own, and a form that asked a dozen more would be worse than an
// edit to skills.json. They survive a save untouched — see skill.Skill.MarshalJSON.
//
// ⚠️ This list was written down in three places (here, forge.resolveOnto, and
// TestTheShippedSkillBookSurvivesBeingWritten) and every copy was wrong the same
// way: each named self_applies, which this form *does* ask about, and none named
// self_requires or summons, which it does not. Corrected when self_gradient
// joined them. A list restated three times drifts three times; if it drifts
// again, derive it.
//
// The name sits second because that is where it is authored: a skill and the name
// it is called by are one thought, which is also why the name is a field on
// skill.Skill and not a translations file beside the book.
const (
	SkillFieldID = iota
	SkillFieldName
	SkillFieldFlavour
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
	// Beside pierce, so the form reads in the order the file is written in.
	SkillFieldCrit
	SkillFieldRestores
	SkillFieldDrains
	SkillFieldKeptForElements
	SkillFieldKeptForRoles
	SkillFieldKeptForCharacters
	SkillFieldKeptForSpecies
	SkillFieldKeptForOrigins
	SkillFieldCount
)

// SkillsScreen is the skill book, and the form that adds to it and changes it.
//
// It is the other half of the new-character form in the same way the origins
// screen is: a kit can only name a skill the book holds, so an author who finds
// the skill they want missing can add it and go straight back.
//
// Nothing here decides whether an answer is acceptable, and nothing here works
// out what a skill is worth. The damage comes from forge.Library.PreviewDraft,
// which is combat.Rules.Damage against the reference pair skills.golden's own
// table is measured from, and the write goes through forge.SkillDraft.Resolve,
// which appends to the book and therefore applies exactly the checks a load
// applies.
type SkillsScreen struct {
	Skills []skill.Skill
	Cursor int

	// Query is the typed filter, and Filtering is whether the field under the
	// heading has the keyboard.
	//
	// ⚠️ **The two are carried side by side rather than one derived from the
	// other**, and that is the decision the whole mode rests on: enter closes
	// the field and *keeps* the query, so a narrowed listing with the keyboard
	// back on the rows is an ordinary state and the one the feature exists for.
	// Reading focus off "the query is not empty" would make that state
	// unreachable and would also make a filter impossible to open before
	// anything has been typed into it — the same shape as PlayScreen carrying
	// LogFollow beside LogOffset instead of encoding "follow" as a sentinel
	// offset, and as a queue reading having to declare absence rather than
	// detect it.
	//
	// Query is a plain string and not a bubbles textinput, which every other
	// typed field on this screen is. The reason is the keys: the arrows belong
	// to the *listing* while this field has focus — letters are text, so k and j
	// cannot walk the rows and the arrows are all that is left — so a text field
	// here would draw a caret that none of its own keys could move. What is
	// left of a text field once its arrows are gone is a string with a
	// backspace, which is what this is.
	Query     string
	Filtering bool

	// Adding is whether the form is in front of the listing to author a new
	// skill, and Editing is the id it is in front to change. They are two fields
	// rather than one because one form serves both jobs and the difference
	// decides three things: the heading, what Escape is asking to throw away, and
	// whether the write appends or replaces. FormInFront is the two together, and
	// they are never both set.
	Adding  bool
	Editing string

	// Inputs is every typed field of the form, indexed by the SkillField
	// constants above.
	//
	// ⚠️ **A slice, so two copies of this screen SHARE it, and that is the one
	// place "a screen is a value" is not true here.** Every other field is
	// copied when the screen is; these text fields are not — writing through
	// `a.Inputs[i]` is visible through `b.Inputs[i]` for any copy `b` taken
	// before the write. It was measured rather than reasoned about, in #216:
	// dropping the client's write-back of the screen after a status pick left
	// the inflicts entry filled in anyway, because the landing wrote through
	// storage the copies share, while every flag beside it was lost.
	//
	// It is moved here **as it is** and deliberately unchanged. The behaviour is
	// relied on today and changing it is a measurement of its own, not a side
	// effect of a move — so TestTheSkillFormsFieldsAreSharedBetweenCopies pins
	// it in both directions, and whoever changes it later has to mean it.
	Inputs []textinput.Model
	Field  int

	// The three choosers and the five allowlists, held as their answers rather
	// than as text fields: an element, a side and a shape are all ids out of a
	// book, so typing one is only a way to get it wrong.
	ElementIndex int
	TargetIndex  int
	ShapeIndex   int
	KeptElements []string
	KeptRoles    []string
	KeptWho      []string
	KeptKinds    []string
	KeptWorlds   []string
	Touched      bool

	// ShapeDrawn is whether the shape diagram is over the form.
	//
	// A flag rather than a state of its own, because the diagram has no answer
	// to hold: it draws the shape the chooser is already on and the arrow keys
	// on it are the chooser's own, so there is one ShapeIndex and the drawing
	// cannot disagree with the field behind it. That is the difference from the
	// picker, which collects an answer and hands it back on enter.
	ShapeDrawn bool

	Err error
	// Added is the last skill written, kept as what it was rather than as the
	// line announcing it, so a language switch redraws the announcement.
	Added *skill.Skill
	// Edited is the last change written, kept whole rather than as its id: the
	// damage before and after is what an author needs from an edit, and it is
	// carried so a language switch redraws that too.
	Edited *forge.SkillChange
}

// FormInFront reports whether the form is over the listing, either to author a
// skill or to change one.
func (s SkillsScreen) FormInFront() bool { return s.Adding || s.Editing != "" }

// NewSkillsScreen is the listing filled from a library, with an empty form
// behind it.
//
// ⚠️ **It takes a Context rather than a library**, which is the one signature
// this move had to change, and the reason is that the form dresses text fields:
// NewInput wants to know whether colour would be noise, this package may not
// read an environment, and Palette already carries the answer the binary was
// handed. Same arrangement the preview's cache key has.
func NewSkillsScreen(c Context) SkillsScreen {
	return SkillsScreen{}.Refresh(c).ResetForm(c)
}

// Refresh re-reads the book, keeping the query and clamping the cursor —
// entering the screen after a write should show the changed row rather than a
// stale list.
func (s SkillsScreen) Refresh(c Context) SkillsScreen {
	s.Skills = c.Lib.Skills().Skills()
	// Against the rows the filter leaves and not against the book, for the reason
	// every other read of this cursor goes through Rows(): a write can shorten
	// what a standing query finds, and a cursor clamped against the book would
	// point past the end of the listing that is actually drawn.
	//
	// The query itself survives a refresh. A save re-reads the book through here,
	// and an author who narrowed the listing to find a skill, edited it and came
	// back is looking at the same question — clearing the filter under them would
	// make every write also a navigation.
	s.Cursor = Clamp(s.Cursor, 0, len(s.Rows())-1)
	if s.Inputs == nil {
		s = s.ResetForm(c)
	}
	return s
}

// FilterLimit is how many letters the typed filter takes.
//
// A bound rather than a free-running field, and it is what keeps the filter row
// inside the window by construction: the wording around the query is thirty-one
// cells at its longest, so a query this long cannot reach the seventy-nine the
// floor leaves. Nothing an author would type comes near it — the longest shipped
// skill name is well under half of it — so the limit is a guard rather than a
// constraint, and the row clips as well, because a guard nothing measures is a
// guard that has already stopped working.
const FilterLimit = 32

// Rows is the skills the filter in force leaves on screen, and it is the ONE
// place the listing's cursor may be indexed against.
//
// ⚠️ **Every read goes through here or through Selected below.** The cursor
// counts the *visible* rows, so a read that reached into s.Skills with it would
// silently act on a different skill than the one under the marker — and the two
// agree exactly while nothing is typed, which is every existing test. That is
// the same funnel PickState.Visible is for, and the reason it is one function
// rather than a filter written out at each site.
//
// Matching is i18n.MatchesSkill, which reads the skill's **id and its authored
// Vietnamese name** and asks nothing about the language in front. See that
// function for why: a query that found different rows after ctrl+l would be the
// screen mutating behind the author.
func (s SkillsScreen) Rows() []skill.Skill {
	if strings.TrimSpace(s.Query) == "" {
		return s.Skills
	}
	out := make([]skill.Skill, 0, len(s.Skills))
	for _, current := range s.Skills {
		if i18n.MatchesSkill(s.Query, current) {
			out = append(out, current)
		}
	}
	return out
}

// Selected is the skill under the cursor, and false when the filter has left
// nothing to point at.
//
// The second return is what the three keys that read a row ask, rather than each
// of them measuring the list itself: a query matching nothing is an ordinary
// state that arrives one keystroke at a time, so e and ? have to decline rather
// than index an empty slice.
func (s SkillsScreen) Selected() (skill.Skill, bool) {
	rows := s.Rows()
	if len(rows) == 0 {
		return skill.Skill{}, false
	}
	return rows[Clamp(s.Cursor, 0, len(rows)-1)], true
}

// Subject is what the description screen raised from here is about: the skill
// under the cursor, and where it sits in the rows the filter has left.
//
// Off Rows() rather than off the book, for the reason Selected is: the cursor
// indexes the filtered view, so the position beside the name — "3 / 5" — has to
// count the same list the marker was moving through.
func (s SkillsScreen) Subject() Subject {
	rows := s.Rows()
	if len(rows) == 0 {
		return Subject{Kind: SkillSubject}
	}
	at := Clamp(s.Cursor, 0, len(rows)-1)
	return Subject{Kind: SkillSubject, ID: rows[at].ID, At: at + 1, Of: len(rows)}
}

// ResetForm empties the form and puts every chooser back on its default, which
// is what both opening a new one and discarding a half-written one come to.
func (s SkillsScreen) ResetForm(c Context) SkillsScreen {
	s.Inputs = make([]textinput.Model, SkillFieldCount)
	for i := range s.Inputs {
		input := NewInput(c.Style.Plain)
		input.Prompt = ""
		input.CharLimit = 200
		input.SetWidth(24)
		s.Inputs[i] = input
	}
	s.Inputs[SkillFieldID].SetWidth(32)
	s.Inputs[SkillFieldName].SetWidth(32)
	s.Inputs[SkillFieldFlavour].SetWidth(48)
	s.Inputs[SkillFieldInflicts].SetWidth(40)
	s.Inputs[SkillFieldOnItself].SetWidth(40)
	// The defaults are the shape of an ordinary single-target attack, and the
	// element among them is the one worth spelling out: neutral is the common
	// pool, so a skill authored without an opinion about its element is one
	// every character can take. Power and accuracy have none, because both are
	// balance and a default would write a number nobody chose.
	s.Inputs[SkillFieldRange].SetValue(defaultSkillRange)
	s.Inputs[SkillFieldStrikes].SetValue(defaultSkillStrikes)
	s.Inputs[SkillFieldCooldown].SetValue(defaultSkillCooldown)
	s.ElementIndex = IndexOf(forge.ElementNames(), defaultSkillElement)
	s.TargetIndex = IndexOf(forge.TargetNames(), defaultSkillTarget)
	s.ShapeIndex = IndexOf(c.Lib.PatternNames(), defaultSkillPattern)
	s.KeptElements, s.KeptRoles, s.KeptWho = nil, nil, nil
	s.KeptKinds, s.KeptWorlds = nil, nil
	s.Field = SkillFieldID
	s.Touched = false
	s.ShapeDrawn = false
	s.Err = nil
	s.Editing = ""
	s.Inputs[s.Field].Focus()
	return s
}

// Prefill is ResetForm over a skill that already exists, which is what makes one
// form serve both jobs.
//
// Every value comes from forge.SkillAnswers rather than being formatted here, so
// that accepting the form as it stands reproduces the skill exactly. A screen
// that wrote its own "1200" or turned an absent restriction into an empty list
// would turn opening the form into a change.
func (s SkillsScreen) Prefill(c Context, current skill.Skill) SkillsScreen {
	s = s.ResetForm(c)
	answers := forge.SkillAnswers(current)
	for _, filled := range []struct {
		field int
		value string
	}{
		{SkillFieldID, answers.ID},
		{SkillFieldName, answers.Name},
		{SkillFieldFlavour, answers.Flavour},
		{SkillFieldRange, answers.Range},
		{SkillFieldPower, answers.Power},
		{SkillFieldStrikes, answers.Strikes},
		{SkillFieldAccuracy, answers.Accuracy},
		{SkillFieldCooldown, answers.Cooldown},
		{SkillFieldInflicts, answers.Applies},
		{SkillFieldOnItself, answers.SelfApplies},
		{SkillFieldPierce, answers.Pierce},
		{SkillFieldCrit, answers.Crit},
		{SkillFieldRestores, answers.Restores},
		{SkillFieldDrains, answers.Drains},
	} {
		s.Inputs[filled.field].SetValue(filled.value)
	}
	s.ElementIndex = IndexOf(forge.ElementNames(), answers.Element)
	s.TargetIndex = IndexOf(forge.TargetNames(), answers.Target)
	s.ShapeIndex = IndexOf(c.Lib.PatternNames(), answers.Pattern)
	s.KeptElements = forge.SplitList(answers.RestrictElements)
	s.KeptRoles = forge.SplitList(answers.RestrictArchetypes)
	s.KeptWho = forge.SplitList(answers.RestrictCharacters)
	s.KeptKinds = forge.SplitList(answers.RestrictSpecies)
	s.KeptWorlds = forge.SplitList(answers.RestrictOrigins)
	s.Editing = current.ID
	return s
}

// The defaults a skill takes when nobody says otherwise. They are the same
// strings cmd/hexforge's prompts default to; both front-ends offering different
// defaults would be two answers to one question.
const (
	defaultSkillElement  = "neutral"
	defaultSkillTarget   = "enemy"
	defaultSkillPattern  = "single"
	defaultSkillRange    = "1"
	defaultSkillStrikes  = "1"
	defaultSkillCooldown = "0"
)

// IndexOf is where a value sits in a chooser's list, and nought when it is not
// in one at all.
//
// Exported because a client's own fixtures set a chooser by name — a shape, an
// element — and an index written out by hand would be a second reading of the
// book's order.
func IndexOf(values []string, want string) int {
	for i, value := range values {
		if value == want {
			return i
		}
	}
	return 0
}

// Draft is the answers as internal/forge wants them, which is the only thing
// this screen hands outwards.
//
// Exported because a client test compares what the form resolves to against what
// the command line's flags resolve to — the two front-ends must be incapable of
// disagreeing, and that comparison needs the draft.
func (s SkillsScreen) Draft(c Context) forge.SkillDraft {
	return forge.SkillDraft{
		ID:                 strings.TrimSpace(s.Inputs[SkillFieldID].Value()),
		Name:               s.Inputs[SkillFieldName].Value(),
		Flavour:            s.Inputs[SkillFieldFlavour].Value(),
		Element:            at(forge.ElementNames(), s.ElementIndex),
		Target:             at(forge.TargetNames(), s.TargetIndex),
		Range:              s.Inputs[SkillFieldRange].Value(),
		Pattern:            at(c.Lib.PatternNames(), s.ShapeIndex),
		Power:              s.Inputs[SkillFieldPower].Value(),
		Strikes:            s.Inputs[SkillFieldStrikes].Value(),
		Accuracy:           s.Inputs[SkillFieldAccuracy].Value(),
		Cooldown:           s.Inputs[SkillFieldCooldown].Value(),
		Applies:            s.Inputs[SkillFieldInflicts].Value(),
		SelfApplies:        s.Inputs[SkillFieldOnItself].Value(),
		Pierce:             s.Inputs[SkillFieldPierce].Value(),
		Crit:               s.Inputs[SkillFieldCrit].Value(),
		Restores:           s.Inputs[SkillFieldRestores].Value(),
		Drains:             s.Inputs[SkillFieldDrains].Value(),
		RestrictElements:   strings.Join(s.KeptElements, ","),
		RestrictArchetypes: strings.Join(s.KeptRoles, ","),
		RestrictCharacters: strings.Join(s.KeptWho, ","),
		RestrictSpecies:    strings.Join(s.KeptKinds, ","),
		RestrictOrigins:    strings.Join(s.KeptWorlds, ","),
	}
}

func at(values []string, index int) string {
	if len(values) == 0 {
		return ""
	}
	return values[Clamp(index, 0, len(values)-1)]
}

// chooserField reports whether a field is stepped through rather than typed.
func skillChooserField(field int) bool {
	switch field {
	case SkillFieldElement, SkillFieldTarget, SkillFieldShape:
		return true
	default:
		return false
	}
}

// listField reports whether a field is a list chosen on the sub-screen.
func skillListField(field int) bool {
	switch field {
	case SkillFieldKeptForElements, SkillFieldKeptForRoles,
		SkillFieldKeptForCharacters, SkillFieldKeptForSpecies,
		SkillFieldKeptForOrigins:
		return true
	default:
		return false
	}
}

// Update routes a keystroke on the listing, the form over it, or the diagram
// over that, and says what the client owes it.
//
// ⚠️ **Three returns rather than two, and the third is a tea.Cmd.** Every other
// screen here answers (itself, Action); this one has text fields, and a bubbles
// textinput answers an Update with a command — the cursor's blink — which has to
// come out or the field loses its cursor. That is the same fact PickResult.Cmd
// carries. It is not a field on Action, deliberately: an Action is a comparable
// value a screen returns and a test writes out as a literal, and a func field
// would take that away from every screen to serve one. A result type of exactly
// {Action, Cmd} would be a tuple with a name; PickResult earned its type by
// carrying four things.
//
// e is the edit key. It does not collide with anything: this screen is a list
// rather than a form, so no field has the keyboard, and its only other letters
// are q, k, j and a. It sits beside a for the reason those two belong together —
// adding a skill and changing one are the same form reached two ways.
//
// The filter is a **mode** and not another letter, and that follows from the
// line above rather than from taste: every letter this screen has is already a
// command, so a field sharing the keyboard with them could take no query at all.
// / is the key because it is the one neither client had spent anywhere, and
// because it is what a reader of any other full-screen program will already try.
func (s SkillsScreen) Update(c Context, message tea.KeyPressMsg) (SkillsScreen, Action, tea.Cmd) {
	if s.FormInFront() {
		return s.updateForm(c, message)
	}
	// Answered ahead of the listing's own keys, and while it is open the listing
	// has only the two arrows: q, a, e, k, j and ? are letters and belong in the
	// query.
	if s.Filtering {
		return s.updateFilter(c, message), Action{}, nil
	}
	switch message.String() {
	case "q":
		return s, Action{Kind: Quit}, nil
	case "esc":
		return s, Action{Kind: Back}, nil
	case "/":
		// Whatever was typed before is still there: enter closes the field
		// without throwing the query away, so reopening it has to pick that
		// query back up or the two keys would disagree about what enter did.
		s.Filtering = true
	case "up", "k":
		s.Cursor = Clamp(s.Cursor-1, 0, len(s.Rows())-1)
	case "down", "j":
		s.Cursor = Clamp(s.Cursor+1, 0, len(s.Rows())-1)
	case "a":
		// Deliberately not guarded on there being a row: adding is the one key
		// here that indexes nothing, and a filter that has found nothing is
		// exactly the moment an author wants to write the skill they were
		// looking for.
		//
		// It is guarded on the client being able to write, which is the other
		// question entirely — see Context.Authoring. A game client draws this
		// listing so a player can read what a skill does, and the form over it
		// writes skills.json.
		if c.Authoring {
			s = s.ResetForm(c)
			s.Adding = true
			s.Added, s.Edited = nil, nil
		}
	case "e":
		if selected, held := s.Selected(); held && c.Authoring {
			s = s.Prefill(c, selected)
			s.Added, s.Edited = nil, nil
		}
	case "?":
		// The description screen keeps no cursor of its own, so this one hands
		// over the skill it is pointing at — the same arrangement the art
		// preview has with the browser. The subject is built off the *visible*
		// rows through the same accessor this key does, so the paragraph and the
		// marker cannot come from two different skills.
		if _, held := s.Selected(); held {
			return s, Action{Kind: Raise, Target: Blurb, Subject: s.Subject()}, nil
		}
	}
	return s, Action{}, nil
}

// updateFilter routes a keystroke while the filter field has the keyboard.
//
// Five keys and everything else is text. The arrows are the arrows themselves
// rather than k and j, which is the whole reason this is a mode: a letter typed
// here has to reach the query, so the vim pair cannot be borrowed and the arrows
// are what is left to walk the narrowed rows with.
func (s SkillsScreen) updateFilter(_ Context, message tea.KeyPressMsg) SkillsScreen {
	switch message.String() {
	case "esc":
		// One key undoes the whole thing. Clearing and closing together is what
		// makes escape safe to press: there is no state where the listing is
		// still narrowed by a query the author has just dismissed, and no second
		// keystroke to work out.
		s.Query, s.Filtering = "", false
	case "enter":
		// The query stays and the keyboard goes back to the rows, which is what
		// makes j, k, a, e and ? work on what was found.
		s.Filtering = false
	case "backspace":
		if letters := []rune(s.Query); len(letters) > 0 {
			s.Query = string(letters[:len(letters)-1])
		}
	case "up":
		s.Cursor = Clamp(s.Cursor-1, 0, len(s.Rows())-1)
	case "down":
		s.Cursor = Clamp(s.Cursor+1, 0, len(s.Rows())-1)
	default:
		// Text rather than the key's name, which is what lets a space through:
		// a bare space stringifies as "space" and carries " " as its text, and
		// a Vietnamese name is two words often as not, so a filter that could
		// not take a space would stop at the first one. A chord carries no text
		// at all, so ctrl+anything falls through here harmlessly — and ctrl+c
		// and ctrl+l never reach this function: a client answers both before any
		// screen is asked.
		if letters := []rune(s.Query + message.Text); message.Text != "" &&
			len(letters) <= FilterLimit {
			s.Query = string(letters)
		}
	}
	// Clamped wherever the query changed rather than only where the cursor moved:
	// a keystroke that narrows the listing can leave the cursor past its end, and
	// a query matching nothing has to leave it at nought rather than at minus one.
	s.Cursor = Clamp(s.Cursor, 0, len(s.Rows())-1)
	return s
}

// Confirmed throws the half-written skill away and closes the form.
//
// ⚠️ **Two questions reach it and there is one answer**, which is why it does not
// read the question: SkillFormDiscard and SkillFormEditDiscard differ only in
// what they say is being lost — a skill nobody has written yet, or a set of
// changes to one already in the book — and ResetForm is what both of them meant.
// A wording is not an identity, so a screen branching on which one was asked
// would be a screen with two answers where the book has one.
//
// ⚠️ **What the question was about arrives as an `any` and is ignored**, which
// is honest rather than lazy: a guard's subject is a client's own vocabulary —
// the squad builder's two questions are told apart by a squad id — and this
// screen's one question is about the screen itself. The parameter is here so the
// shape matches the other four Confirmed methods, which a client dispatch is
// held total over.
func (s SkillsScreen) Confirmed(c Context, _ any) (SkillsScreen, Action) {
	s = s.ResetForm(c)
	s.Adding = false
	return s, Action{}
}

func (s SkillsScreen) updateForm(c Context, message tea.KeyPressMsg) (SkillsScreen, Action, tea.Cmd) {
	// The diagram is answered first, before Escape can be read as leaving the
	// form and before Enter can be read as moving to the next field: while it is
	// over the form, both of those close it and nothing else on the form is
	// reachable.
	if s.ShapeDrawn {
		switch message.String() {
		case "esc", "enter", "space":
			s.ShapeDrawn = false
		case "left":
			s = s.cycle(c, -1)
		case "right":
			s = s.cycle(c, 1)
		}
		return s, Action{}, nil
	}
	// After the diagram and before the switch: saving answers to more than one
	// keystroke, and IsSaveKey is the single declaration of which. It comes
	// second because the diagram takes every key while it is open.
	if IsSaveKey(message) {
		return s.save(c), Action{}, nil
	}
	switch message.String() {
	case "esc":
		if !s.Touched {
			s.Adding, s.Editing = false, ""
			return s, Action{}, nil
		}
		// The question names what is being thrown away, which is a different
		// thing in each case: a skill nobody has written yet, or a set of changes
		// to one that is already in the book.
		question := i18n.SkillFormDiscard
		if s.Editing != "" {
			question = i18n.SkillFormEditDiscard
		}
		return s, Action{Kind: Ask, Question: question}, nil
	case "up", "shift+tab":
		return s.moveTo(s.Field - 1), Action{}, nil
	case "down", "tab", "enter":
		return s.moveTo(s.Field + 1), Action{}, nil
	}
	if skillChooserField(s.Field) {
		switch message.String() {
		case "left":
			s = s.cycle(c, -1)
		case "right":
			s = s.cycle(c, 1)
		case "space":
			// Space is the same key the three allowlists open their picker with,
			// and it is free on a chooser: a chooser is stepped with the arrows.
			// Only the shape has anything to open.
			if s.Field == SkillFieldShape {
				s.ShapeDrawn = true
			}
		}
		return s, Action{}, nil
	}
	// The inflicts field is the one text field with a list behind it. Space
	// opens it rather than typing a space, and that costs nothing: the syntax
	// ParseApplications reads has no spaces in it, and every other way of
	// filling this field still works, because the field is the record.
	if (s.Field == SkillFieldInflicts || s.Field == SkillFieldOnItself) && message.String() == "space" {
		return s, Action{Kind: Pick, Picker: s.OpenStatuses(c)}, nil
	}
	if skillListField(s.Field) {
		if message.String() == "space" || message.String() == "right" {
			return s, Action{Kind: Pick, Picker: s.OpenAllowlist(c, s.Field)}, nil
		}
		return s, Action{}, nil
	}
	updated, command := s.Inputs[s.Field].Update(message)
	if updated.Value() != s.Inputs[s.Field].Value() {
		s.Touched = true
		s.Err = nil
		s.Added, s.Edited = nil, nil
	}
	s.Inputs[s.Field] = updated
	return s, Action{}, command
}

// Paste puts a pasted string where a typed one would go: into the form's focused
// field, or into the filter's query, or nowhere.
//
// ⚠️ **This screen has two text targets and they belong to different clients.**
// The form's fields open behind Context.Authoring; the typed filter opens on `/`
// and is guarded by nothing, so it is the one text target a **game** client has
// on a screen out of this package. The order below is Update's own — form, then
// filter — so a paste cannot reach a form the keystrokes could not have opened.
//
// ⚠️ **`FormInFront` is load-bearing twice over.** ResetForm focuses the id field
// and runs at construction, so a client on the bare **listing** holds a focused
// text field nobody is looking at — a paste there would fill a form that has not
// been opened. And it is what keeps a read-only client out of the authoring half:
// the form can only be in front because `a` or `e` put it there, and both are
// guarded on Context.Authoring, so a paste on cmd/hexarena-tui falls to the
// filter or to nothing.
//
// There is no predicate beside this. The two arms have to find their target
// anyway, and a `Pasting` restating them would be one rule written twice.
func (s SkillsScreen) Paste(_ Context, text string) (SkillsScreen, tea.Cmd) {
	if s.FormInFront() {
		field := focusedField(s.Inputs, s.Field)
		if field == nil {
			return s, nil
		}
		before := field.Value()
		command := PasteInto(field, text)
		if field.Value() != before {
			// updateForm's bookkeeping, on updateForm's condition.
			s.Touched = true
			s.Err = nil
			s.Added, s.Edited = nil, nil
		}
		return s, command
	}
	if s.Filtering {
		s = s.pasteFilter(text)
	}
	return s, nil
}

// pasteFilter appends a pasted string to the query, under the same two rules
// updateFilter's own text arm is under: the field takes what it can hold and the
// cursor is re-clamped because a narrowed listing can leave it past the end.
//
// ⚠️ **A paste over the limit is truncated rather than refused**, which is the
// opposite of what a number field does and is the right answer for a different
// reason: the query is a search rather than a value, so thirty-two letters of a
// long paste narrow the listing exactly as thirty-two typed ones would, and a
// refusal would leave a reader with no way to search for anything long. Nothing
// is being written to a file here.
func (s SkillsScreen) pasteFilter(text string) SkillsScreen {
	letters := []rune(s.Query + PasteText(text))
	if len(letters) > FilterLimit {
		letters = letters[:FilterLimit]
	}
	s.Query = string(letters)
	s.Cursor = Clamp(s.Cursor, 0, len(s.Rows())-1)
	return s
}

// OpenAllowlist builds the picker for one of the five lists.
//
// The list is offered rather than typed for the reason the origin and the
// archetype are on the other form: every entry is an id out of a book, so a
// picker cannot produce a name that does not exist — and an allowlist naming
// somebody who does not exist is the same mistake as an empty one, satisfied by
// nobody.
//
// It **builds** the state and does not put it in front, which is the whole of
// what moving this screen changed here: a picker's lifetime is the client's,
// because the client is the only thing with a field to keep one in. The state
// travels back as Action.Pick's payload and the client raises it.
//
// Exported because this client's own width fixture registers three of these
// pickers as screens of their own and has to reach them the way a keystroke
// does; a fixture cannot follow a screen across a package boundary.
func (s SkillsScreen) OpenAllowlist(c Context, field int) *PickState {
	switch field {
	case SkillFieldKeptForElements:
		return &PickState{
			Title: i18n.PickerElementsTitle, Kind: PickElements,
			Hint:    i18n.PickerAllowlistHint,
			Options: IDOptions(forge.ElementNames()), Chosen: s.KeptElements,
			Into: SkillsPickElements,
		}
	case SkillFieldKeptForRoles:
		return &PickState{
			Title: i18n.PickerRolesTitle, Kind: PickArchetypes,
			Hint:    i18n.PickerAllowlistHint,
			Options: IDOptions(c.Lib.Archetypes().IDs()), Chosen: s.KeptRoles,
			Into: SkillsPickRoles,
		}
	case SkillFieldKeptForOrigins:
		return &PickState{
			Title: i18n.PickerOriginsTitle, Kind: PickOrigins,
			Hint:    i18n.PickerAllowlistHint,
			Options: IDOptions(c.Lib.Origins().IDs()), Chosen: s.KeptWorlds,
			Into: SkillsPickWorlds,
		}
	case SkillFieldKeptForSpecies:
		return &PickState{
			Title: i18n.PickerSpeciesTitle, Kind: PickSpecies,
			Hint:    i18n.PickerAllowlistHint,
			Options: IDOptions(c.Lib.Species().IDs()), Chosen: s.KeptKinds,
			Into: SkillsPickKinds,
		}
	default:
		// The one list with a filter, because it is the one that grows: the
		// elements are eleven and fixed and the role presets are five, while the
		// cast is whatever has been authored. It narrows by origin and takes the
		// cast browser's own key, so filtering a list of characters is one
		// interaction wherever it happens.
		return &PickState{
			Title: i18n.PickerCharactersTitle, Kind: PickCharacters,
			Hint:    i18n.PickerAllowlistHint,
			Footer:  i18n.PickerFilterFooter,
			Options: CharacterOptions(c.Lib), Groups: c.Lib.OriginIDs(),
			Chosen: s.KeptWho,
			Into:   SkillsPickWho,
		}
	}
}

// OpenStatuses builds the picker over the status book, whose answer is written
// into the inflicts field.
//
// The shortest path from "I want a poison" to a valid entry: pick the status out
// of the book, type the chance in the field under the list, enter. The syntax is
// forge.AddApplications' — the same spelling ParseApplications reads back — so
// the screen never spells an entry itself, and nothing about the field changes:
// it is still a text field, an author who knows the syntax still types it, and a
// script writes the same thing.
//
// Nothing is preselected. The field may already hold entries and the picker
// appends to them, so starting with those rows ticked would mean the author had
// to untick them to avoid writing each one twice.
//
// ⚠️ **The destination is the inflicts field whichever of the two status fields
// raised it**, which is what the field write below already did and is preserved
// unchanged: space on "on itself" opens this list and its answer lands in
// "inflicts". Moving the screen is not the change that gets to decide whether
// that is right.
func (s SkillsScreen) OpenStatuses(c Context) *PickState {
	return &PickState{
		Title: i18n.PickerStatusesTitle, Kind: PickStatuses,
		Hint: i18n.PickerStatusHint, Footer: i18n.PickerStatusFooter,
		Options: StatusOptions(c.Lib),
		Typed:   NumberField(c.Style.Plain, forge.DefaultApplicationChance),
		Label:   i18n.PickerChance,
		Into:    SkillsPickInflicts,
	}
}

// SkillsPick is where one of this form's six lists lands: one named field of
// this screen.
//
// ⚠️ **It is declared here and not in a client, and that is the move rather than
// a widening.** PickState.Into is an `any` because a destination used to name a
// field of a *client's* screen — so cmd/hexforge-tui declared all ten of them.
// Six of the ten named fields of this form, and this form is no longer a
// client's, so its destinations came with it. The four that still name a
// client's screens — the character form's two and the squad builder's two —
// stayed where they are, and the client is the one thing that knows both
// vocabularies, which is what a client is for.
//
// ⚠️ **A destination is *where*, never *how*.** Five of the six assign the
// answer to a list field; SkillsPickInflicts composes it into a text field
// through forge.AddApplications. Both are destinations, and what the receiving
// field does when the answer arrives is that field's business.
type SkillsPick uint8

const (
	// SkillsPickNothing is the zero value: a destination that names no field.
	// No raise here uses it, and Picked declines it, which is what a picker
	// carrying no destination at all already gets.
	SkillsPickNothing SkillsPick = iota
	// The five allowlists and the inflicts field, each named after the field it
	// fills, so a destination pointed at the wrong one reads wrong at the raise
	// site as well as at the landing.
	SkillsPickElements
	SkillsPickRoles
	SkillsPickWorlds
	SkillsPickKinds
	SkillsPickWho
	SkillsPickInflicts
	// SkillsPickCount is the count a dispatch is held total against, in the
	// shape SubjectKindCount, TargetCount and KindCount already have: a
	// destination added above it enters the walk without anybody remembering to
	// list it, which a hand-written list of six would not give.
	SkillsPickCount
)

// Picked is what the six lists this form raises write back.
//
// ⚠️ **Five of the six are one line and the sixth is not, and that is the shape
// rather than an exception.** A destination says *where* an answer lands; what
// the field does with it belongs to the field. The five allowlists take the
// chosen ids as the list they are, and the inflicts field is text that already
// holds entries, so its answer is composed into what is there through
// forge.AddApplications — the same call the typed syntax goes through, which is
// why it is a branch here rather than a second kind of picker.
//
// The destination arrives as an `any` because that is what PickState carries it
// as; a value that is not one of this screen's own is a picker somebody else
// raised, and it lands nowhere for the same reason SkillsPickNothing does.
func (s SkillsScreen) Picked(c Context, into any, answer PickAnswer) (SkillsScreen, Action) {
	switch into {
	case SkillsPickElements:
		s.KeptElements = answer.Chosen
	case SkillsPickRoles:
		s.KeptRoles = answer.Chosen
	case SkillsPickWorlds:
		s.KeptWorlds = answer.Chosen
	case SkillsPickKinds:
		s.KeptKinds = answer.Chosen
	case SkillsPickWho:
		s.KeptWho = answer.Chosen
	case SkillsPickInflicts:
		return s.inflict(c, answer), Action{}
	default:
		// A destination this form does not own — the picker carries an `any`, so a
		// value out of another screen's vocabulary can arrive here.
		return s, Action{}
	}
	s.Touched = true
	// No list leaves the form, so no list carries an action.
	return s, Action{}
}

// inflict writes what the status picker chose into the inflicts field.
//
// It is a method of its own rather than a sixth arm inline, because it is the
// one landing that reads a book, can refuse, and clears more than the dirty
// flag — an arm of that size in a switch of one-liners reads as though the
// others had been abbreviated.
func (s SkillsScreen) inflict(c Context, answer PickAnswer) SkillsScreen {
	if len(answer.Chosen) == 0 {
		return s
	}
	field := &s.Inputs[SkillFieldInflicts]
	written, err := c.Lib.AddApplications(field.Value(), answer.Chosen, answer.Typed)
	if err != nil {
		// A refusal from here can only be a chance that is not a number, which
		// the field refuses a keystroke at a time — so this is unreachable and
		// reported rather than swallowed, on the form's own error line.
		s.Err = err
		return s
	}
	field.SetValue(written)
	// The cursor goes to the end, because what was just written is at the end
	// and the author's next move is usually to adjust it.
	field.CursorEnd()
	s.Touched = true
	s.Err = nil
	s.Added, s.Edited = nil, nil
	return s
}

func (s SkillsScreen) moveTo(target int) SkillsScreen {
	s.Inputs[s.Field].Blur()
	s.Field = (target + SkillFieldCount) % SkillFieldCount
	if !skillChooserField(s.Field) && !skillListField(s.Field) {
		s.Inputs[s.Field].Focus()
	}
	return s
}

func (s SkillsScreen) cycle(c Context, by int) SkillsScreen {
	step := func(index int, total int) int {
		if total == 0 {
			return 0
		}
		return (index + by + total) % total
	}
	switch s.Field {
	case SkillFieldElement:
		s.ElementIndex = step(s.ElementIndex, len(forge.ElementNames()))
	case SkillFieldTarget:
		s.TargetIndex = step(s.TargetIndex, len(forge.TargetNames()))
	case SkillFieldShape:
		s.ShapeIndex = step(s.ShapeIndex, len(c.Lib.PatternNames()))
	}
	s.Touched = true
	s.Err = nil
	s.Added = nil
	return s
}

// save resolves the draft and writes it, as an addition or as a change to the
// skill the form was opened on.
//
// Every half belongs to internal/forge: Resolve and ResolveEdit each refuse a
// skill a load would refuse, SaveSkill and EditSkill each write through the
// temp-file-then-rename that keeps a crash from truncating skills.json, and the
// second of those refuses an edit no character or preset could survive. Nothing
// on this screen decides which of those is true.
func (s SkillsScreen) save(c Context) SkillsScreen {
	if s.Editing != "" {
		return s.saveEdit(c)
	}
	built, err := s.Draft(c).Resolve(c.Lib)
	if err != nil {
		s.Err = err
		return s
	}
	if err := c.Lib.SaveSkill(built); err != nil {
		s.Err = err
		return s
	}
	s = s.Refresh(c).ResetForm(c)
	s.Adding = false
	s.Added = &built
	return s
}

func (s SkillsScreen) saveEdit(c Context) SkillsScreen {
	built, err := s.Draft(c).ResolveEdit(c.Lib, s.Editing)
	if err != nil {
		s.Err = err
		return s
	}
	change, err := c.Lib.EditSkill(built)
	if err != nil {
		s.Err = err
		return s
	}
	// ResetForm clears Editing, which is what closes the form: the listing behind
	// it is refreshed from the library the write went through, so the changed row
	// is the changed row.
	s = s.Refresh(c).ResetForm(c)
	s.Adding = false
	s.Edited = &change
	return s
}

// skillsRoom is how many rows the listing has, measured from the window in hand:
// the body has c.Height-4 lines and this screen spends ten of them on a
// heading, the filter row, a blank, a column header, a blank, the two damage
// rows, the two lines a write leaves behind, and the tally.
//
// The ten are enumerated because three of them are the busiest state rather than
// every state, and the reserve is for the busiest: the second damage row is only
// drawn for a skill with a condition, the second write line only after an edit
// that moved the damage, and the filter row only once there is a filter. A
// reserve that counted what is about to be drawn would be a reserve that changes
// under the author, which is worse than one row spent on a screen that does not
// need it.
//
// ⚠️ **That is why opening the filter costs no listing row**, which is the point
// of paying for it unconditionally: pressing / narrows the list without also
// shifting every row under it up by one, so the row the cursor was on stays
// where it was on screen.
//
// It stayed at nine when editing added the second write line, and the reason is
// worth recording rather than leaving as a coincidence: the number was one too
// high before. The count it came with listed "the empty string the body's
// trailing newline leaves", copied from the picker's own count, and this body has no trailing
// newline — the tally is written without one. So the real spend was eight against
// a reserve of nine, and the second write line is what that spare line has now
// gone on. There is no spare left, which is what
// TestTheSkillListingFitsTheSmallestWindowAfterAnEdit measures at the 120x24
// floor — and the filter row is why this is ten rather than nine, measured at
// the same floor by TestTheSkillListingFitsTheSmallestWindowWhileFiltering. An
// eleventh line on this screen needs an eleventh here.
func skillsRoom(c Context) int {
	room := c.Height - 4 - 10
	if room < 3 {
		return 3
	}
	return room
}

// skillRow lays out one row of the listing, and the header above it, from one
// place so the two cannot drift apart.
//
// glossColumn of zero drops the translated-name column entirely rather than
// drawing it empty. That one rule covers both cases that need it: English,
// where nothing is glossed, and a book whose ids all happen to be unglossed —
// a column of blanks would read as missing data rather than as a column that
// does not apply.
//
// powerColumn is a parameter for the same reason glossColumn is, and it stopped
// being a constant when the power column's header stopped being the word
// "power": the header is the label the form authored the number with, so a
// column of 8 held "1000x1" but cut "damage multiplier" — or, since Pad only
// widens, let the header run 9 cells past the column and push the last column's
// header right of the rows it names. One header out of line with its own rows is
// the one failure this function exists to prevent.
func skillRow(idColumn, glossColumn, powerColumn int, id, gloss, member, power, who string) string {
	name := Pad(id, idColumn)
	if glossColumn > 0 {
		name += " " + Pad(gloss, glossColumn)
	}
	return fmt.Sprintf("%s %s %s%s", name, Pad(member, 9), Pad(power, powerColumn), who)
}

// skillPowerColumn is the width the power column takes: enough for the figures
// the rows hold, and enough for the header naming them.
//
// The figures are short by construction — a power and a strike count, "1000x1"
// — so the header is what decides this, and it differs per language for the
// same reason every other measured column here does.
//
// The measured label gets a cell added, and the 8 has one already: it is the
// last column before free text, so without a gap a header exactly as wide as its
// column runs straight into the next one. "hệ số sát thương" is 16 cells, which
// is what made that visible.
func skillPowerColumn(c Context) int {
	const figures = 8
	if width := lipgloss.Width(c.Text(i18n.SkillFieldPower)) + 1; width > figures {
		return width
	}
	return figures
}

// filterRow is what has been typed into the filter, and what it has left.
//
// Drawn only when there is a filter to describe, while skillsRoom pays for it
// whichever state the screen is in — see that comment for why the reserve is
// unconditional and this is not.
//
// Two wordings rather than one with the query left blank. An empty field has to
// say what to type into it, because a bare label reads as a broken row; a field
// with a query in it has to say how much of the book is left, which is the
// browser's own reading (i18n.BrowseShowing) asked the same way.
//
// ⚠️ **Nothing here says whether the field has the keyboard, and that is the
// footer's job.** Marking focus on the row itself would have to be colour — the
// palette's rule is that colour is decoration and never information — or a caret
// character, and there is no caret to draw: the arrows belong to the listing, so
// a caret would be one this field's own keys could not move. The footer is where
// the keys are, and it is the line that changes.
func (s SkillsScreen) filterRow(c Context, showing int) string {
	typed := strings.TrimSpace(s.Query)
	if !s.Filtering && typed == "" {
		return ""
	}
	if typed == "" {
		return "  " + c.Style.Dim.Render(c.Text(i18n.SkillsFilterPrompt)) + "\n"
	}
	// The query is the one part of this row with no length of its own, so it is
	// what gets clipped rather than the counts after it: how much of the book is
	// left is the answer, and the author can already see what they typed. Against
	// the window, because both halves are data. FilterLimit is what keeps this
	// from being reachable at the floor; the clip is what says so if it stops.
	spent := lipgloss.Width(c.Text(i18n.SkillsFiltering, "", showing, len(s.Skills)))
	room := max(c.UsableWidth()-3-spent, 1)
	return "  " + c.Style.Dim.Render(c.Text(i18n.SkillsFiltering,
		Clip(s.Query, room), showing, len(s.Skills))) + "\n"
}

// View draws the listing, the form over it, or the diagram over that — a body
// and the footer that names the keys it answers to.
func (s SkillsScreen) View(c Context) (string, string) {
	if s.FormInFront() {
		return s.viewForm(c)
	}
	footer := c.Footer(i18n.SkillsFooter, i18n.SkillsReadFooter)
	if s.Filtering {
		// One footer for both clients, because the filter takes the keyboard: a
		// and e are letters in the query on either, so there is nothing for the
		// two spellings to differ about.
		footer = c.Text(i18n.SkillsFilterFooter)
	}
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SkillsHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.SkillsSubtitle)) + "\n")

	rows := s.Rows()
	out.WriteString(s.filterRow(c, len(rows)))
	out.WriteString("\n")

	anyone := 0
	for _, current := range s.Skills {
		if forge.AnyoneMayCarry(current) {
			anyone++
		}
	}
	// The tally is the book's own count and stays that whatever is filtered: how
	// many skills there are and how many anybody may carry are facts about the
	// data, and the filter row above already says how much of it is on screen.
	tally := c.Style.Dim.Render(c.Text(i18n.SkillsTally, len(s.Skills), anyone))
	if len(rows) == 0 {
		// Said rather than drawn as an empty box, and the two reasons a listing
		// can be empty are told apart: a book with nothing in it is a data
		// directory to go and look at, and a query that found nothing is a
		// keystroke to take back. The column header is not drawn either — a
		// header over no rows names columns that are not there.
		empty := i18n.SkillsFilterNothing
		if len(s.Skills) == 0 {
			empty = i18n.NoneCatalogued
		}
		out.WriteString("  " + c.Text(empty) + "\n\n")
		out.WriteString(tally)
		return out.String(), footer
	}

	// Measured over the whole book rather than over the visible rows, which is
	// PickState.IDColumn's own scope and here it earns it twice: the filter
	// narrows on a keystroke, so columns measured from what is left would slide
	// sideways under the author as they type. The last column is free text
	// against the window, so what a wide id column costs is a few cells of a
	// restriction summary that clips anyway.
	column, glossColumn := 0, 0
	for _, current := range s.Skills {
		if width := lipgloss.Width(current.ID); width > column {
			column = width
		}
		if width := lipgloss.Width(c.Lang.SkillName(current)); width > glossColumn {
			glossColumn = width
		}
	}
	if glossColumn > 0 {
		// The header has to fit the column it names, and a newly authored skill
		// has no gloss, so this width is the widest of the two rather than of
		// the glosses alone.
		if width := lipgloss.Width(c.Text(i18n.ColumnGloss)); width > glossColumn {
			glossColumn = width
		}
	}
	from, to := Window(len(rows), s.Cursor, skillsRoom(c))
	// The header names the one column nobody could guess. The other three are
	// an id, an element and a power, each labelled with the word the form that
	// authored it uses — which is the point of naming them from the same keys:
	// an author who has just typed a damage multiplier on the form should find
	// that column called the same thing here, rather than the shorter word the
	// form stopped using.
	powerColumn := skillPowerColumn(c)
	out.WriteString("  " + c.Style.Dim.Render(skillRow(column+1, glossColumn, powerColumn,
		c.Text(i18n.SkillFieldID), c.Text(i18n.ColumnGloss), c.Text(i18n.LabelElement),
		c.Text(i18n.SkillFieldPower), c.Text(i18n.ColumnWhoMayCarry))) + "\n")
	for index := from; index < to; index++ {
		current := rows[index]
		marker := "  "
		// The power and the strike count are the balance, so they are the two
		// numbers on the row; everything else about a skill is a keypress away
		// on the form that authored it.
		row := skillRow(column+1, glossColumn, powerColumn,
			current.ID, c.Lang.SkillName(current),
			current.Element.String(),
			strconv.Itoa(current.Power)+"x"+strconv.Itoa(current.StrikeCount()), "")
		// Measured against the window rather than against the floor. MinWidth is
		// the width this program promises to draw in, not a ceiling on what it
		// may spend, and this last column is data: a restriction cut to "để dành
		// cho loài dr…" is a row that stopped saying which species it is for,
		// on a terminal with a hundred spare columns beside it. Prose still
		// wraps at the floor — a paragraph run across a wide terminal is a line
		// a reader loses their place in — but a table cell is read by scanning
		// down it, so width is the one thing it can always use.
		row += Clip(c.Lang.WhoMaySummary(current), c.UsableWidth()-3-lipgloss.Width(row))
		if index == s.Cursor {
			marker = "> "
			row = c.Style.Selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}
	out.WriteString("\n")
	// What the skill under the cursor is worth, against the same reference the
	// form previews an unwritten one against — so a power being authored can be
	// compared with the powers already in the book without leaving the program.
	// Through the same accessor e and ? read, so the figure is the marked row's.
	if selected, held := s.Selected(); held {
		preview := c.Lib.PreviewDamage(selected)
		out.WriteString(c.Label(c.Text(i18n.LabelDamage), "%s",
			c.Lang.Damage(preview)))
		if preview.Amplified > 0 {
			out.WriteString(c.Continued("%s",
				c.Text(i18n.DamageAmplified, preview.Amplified)))
		}
	}
	if s.Added != nil {
		out.WriteString(c.Style.Good.Render(c.Text(i18n.SkillAdded,
			s.Added.ID, c.Lib.SkillsPath())) + "\n")
	}
	if s.Edited != nil {
		out.WriteString(c.Style.Good.Render(c.Text(i18n.SkillEdited,
			s.Edited.After.ID, c.Lib.SkillsPath())) + "\n")
		// The before and after, and only when there is something to compare: an
		// edit to a restriction or a targeting side moves no damage, and a line
		// saying a number did not change has to be read to learn nothing.
		if s.Edited.MovesDamage() {
			out.WriteString(c.Style.Dim.Render(c.Lang.DamageMoved(*s.Edited)) + "\n")
		}
	}
	out.WriteString(tally)
	return out.String(), footer
}

// SkillFieldLabel is what each row of the form is called.
//
// Exported with SkillFieldHelp and SkillLabelWidth for the reason the field
// constants are: a client's width sweep measures every row of this form in both
// languages, and a fixture cannot follow a screen across a package boundary.
func SkillFieldLabel(c Context, field int) string {
	keys := [SkillFieldCount]i18n.Key{
		SkillFieldID: i18n.SkillFieldID,
		// The same key the listing's own column uses, because they are the same
		// thing: an author who has just typed a name here should find the column
		// that shows it called what they typed it into.
		SkillFieldName:              i18n.ColumnGloss,
		SkillFieldFlavour:           i18n.LabelFlavour,
		SkillFieldElement:           i18n.SkillFieldElement,
		SkillFieldTarget:            i18n.SkillFieldTarget,
		SkillFieldRange:             i18n.SkillFieldRange,
		SkillFieldShape:             i18n.SkillFieldShape,
		SkillFieldPower:             i18n.SkillFieldPower,
		SkillFieldStrikes:           i18n.SkillFieldStrikes,
		SkillFieldAccuracy:          i18n.SkillFieldAccuracy,
		SkillFieldCooldown:          i18n.SkillFieldCooldown,
		SkillFieldInflicts:          i18n.SkillFieldInflicts,
		SkillFieldOnItself:          i18n.SkillFieldOnItself,
		SkillFieldPierce:            i18n.SkillFieldPierce,
		SkillFieldCrit:              i18n.SkillFieldCrit,
		SkillFieldRestores:          i18n.SkillFieldRestores,
		SkillFieldDrains:            i18n.SkillFieldDrains,
		SkillFieldKeptForElements:   i18n.SkillFieldKeptForElements,
		SkillFieldKeptForRoles:      i18n.SkillFieldKeptForRoles,
		SkillFieldKeptForCharacters: i18n.SkillFieldKeptForCharacters,
		SkillFieldKeptForSpecies:    i18n.SkillFieldKeptForSpecies,
		SkillFieldKeptForOrigins:    i18n.SkillFieldKeptForOrigins,
	}
	return c.Text(keys[field])
}

// SkillFieldHelp is the line describing the field the cursor is on: what it
// means, and an answer that would be valid.
//
// One entry per field, and the array is indexed by the field constant for the
// same reason SkillFieldLabel's is — a field added without a help line is a
// blank line rather than a build failure, so TestEveryFieldOfTheSkillFormHasHelp
// walks every field and reads the screen.
//
// It replaced a static footnote about parts per thousand. That footnote was true
// of two fields out of fourteen and drawn whichever field had the cursor, which
// is why it explained nothing about the fields nobody could guess: what a shape
// covers, what syntax the statuses take, what an empty allowlist means.
func SkillFieldHelp(c Context, field int) string {
	keys := [SkillFieldCount]i18n.Key{
		SkillFieldID:                i18n.SkillHelpID,
		SkillFieldName:              i18n.SkillHelpName,
		SkillFieldFlavour:           i18n.SkillHelpFlavour,
		SkillFieldElement:           i18n.SkillHelpElement,
		SkillFieldTarget:            i18n.SkillHelpTarget,
		SkillFieldRange:             i18n.SkillHelpRange,
		SkillFieldShape:             i18n.SkillHelpShape,
		SkillFieldPower:             i18n.SkillHelpPower,
		SkillFieldStrikes:           i18n.SkillHelpStrikes,
		SkillFieldAccuracy:          i18n.SkillHelpAccuracy,
		SkillFieldCooldown:          i18n.SkillHelpCooldown,
		SkillFieldInflicts:          i18n.SkillHelpInflicts,
		SkillFieldOnItself:          i18n.SkillHelpOnItself,
		SkillFieldPierce:            i18n.SkillHelpPierce,
		SkillFieldCrit:              i18n.SkillHelpCrit,
		SkillFieldRestores:          i18n.SkillHelpRestores,
		SkillFieldDrains:            i18n.SkillHelpDrains,
		SkillFieldKeptForElements:   i18n.SkillHelpKeptForElements,
		SkillFieldKeptForRoles:      i18n.SkillHelpKeptForRoles,
		SkillFieldKeptForCharacters: i18n.SkillHelpKeptForCharacters,
		SkillFieldKeptForSpecies:    i18n.SkillHelpKeptForSpecies,
		SkillFieldKeptForOrigins:    i18n.SkillHelpKeptForOrigins,
	}
	return c.Text(keys[Clamp(field, 0, SkillFieldCount-1)])
}

// SkillLabelWidth is the column the field names sit in, measured from the labels
// themselves rather than declared: the longest is "cooldown" in one language and
// "để dành cho mẫu" in the other.
func SkillLabelWidth(c Context) int {
	widest := 0
	for field := range SkillFieldCount {
		if width := lipgloss.Width(SkillFieldLabel(c, field)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

// The two marks the diagram puts in a cell. hex.Render gives each cell two
// characters, so both are two.
//
// Characters and not colour, for the reason the picker's own marks are: the
// meaning has to survive NO_COLOR, a monochrome terminal and a recording that
// lost its escape codes. The dense mark is the cell the skill is aimed at and
// the sparse one is a cell that only catches the splash share, which is the
// weight each carries as well as the ink.
const (
	shapeAimMark    = "##"
	shapeSplashMark = ".."
)

// viewShape is the shape diagram: the board with the cells this shape catches
// marked, drawn from forge.ShapeCoverage.
//
// It is a sub-screen rather than a pane beside the form, and that was measured
// rather than judged: the form spends nineteen of the twenty body lines a 120x24
// window has — twenty with a refusal under it — and hex.Render is eight lines
// before a heading, a legend or the blanks around it. There was no room, and
// hiding half a board is worse than opening one.
//
// The arrows here are the chooser's own, so the drawing follows the field rather
// than holding a copy of it, and nothing needs applying when it closes.
func (s SkillsScreen) viewShape(c Context) (string, string) {
	footer := c.Text(i18n.SkillShapeFooter)
	shapes := c.Lib.PatternNames()
	name := at(shapes, s.ShapeIndex)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SkillShapeHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.ChoicePosition,
			Clamp(s.ShapeIndex, 0, len(shapes)-1)+1, len(shapes))) + "\n")

	// The draft rather than the shape alone: what a shape covers depends on the
	// side the skill aims at, and that is the field above this one.
	draft := s.Draft(c)
	coverage, err := c.Lib.ShapeCoverage(draft.Pattern, draft.Target)
	if err != nil {
		// Unreachable through the chooser, which offers the book's own names,
		// and drawn rather than swallowed for the same reason the picker draws a
		// lost id: a shape the book cannot resolve is worth seeing.
		out.WriteString(c.Style.Bad.Render("  "+c.Lang.Error(err)) + "\n")
		return out.String(), footer
	}
	// How many cells it really catches, and — when the aim loses one — that it
	// did, so a shape drawing short reads as this aim rather than as this shape.
	caught := c.Text(i18n.SkillShapeCoverage, coverage.Covered())
	if !coverage.Whole() {
		caught = c.Text(i18n.SkillShapeShort, coverage.Covered(), coverage.Max)
	}
	out.WriteString("  " + c.Style.Selected.Render(name) + "  " +
		c.Style.Dim.Render(caught) + "\n")
	out.WriteString("  " + c.Style.Dim.Render(
		c.Text(i18n.SkillShapeDrawnAt, coverage.Primary)) + "\n\n")

	for _, line := range strings.Split(shapeBoard(coverage), "\n") {
		out.WriteString("  " + line + "\n")
	}
	out.WriteString("\n  " + c.Style.Dim.Render(c.Text(i18n.SkillShapeLegend,
		shapeAimMark, shapeSplashMark, c.Lib.SplashShare())))
	return out.String(), footer
}

// shapeBoard draws the battlefield with a coverage marked on it.
//
// The board is hex.Render, the same drawing the terminal client shows a
// formation with, and the cells are the ones pattern.Targets returned — so this
// is a rendering of two existing functions and holds no geometry of its own.
func shapeBoard(coverage forge.ShapeCoverage) string {
	return hex.Render(func(cell hex.Offset) string {
		if cell == coverage.Primary {
			return shapeAimMark
		}
		if slices.Contains(coverage.Splash, cell) {
			return shapeSplashMark
		}
		return ""
	})
}

// formRoom is how many field rows the window has.
//
// The body gets c.Height-4. This form spends the rest on a heading, a blank, a
// blank, the damage row and the help line — five — plus a row for each Ellipsis
// the window draws and one for a refusal when there is one. Counting them here
// rather than guessing is what the listing had to learn twice: a reserve one out
// truncates the screen's own summary and looks like a layout bug rather than an
// arithmetic one.
func (s SkillsScreen) formRoom(c Context) int {
	spent := 5
	if s.Err != nil {
		spent++
	}
	// Room for both ellipses whenever the window cannot hold everything, so the
	// count does not change as the cursor moves and shift every row under it.
	room := c.Height - 4 - spent
	if room < SkillFieldCount {
		room -= 2
	}
	if room < 1 {
		room = 1
	}
	return room
}

func (s SkillsScreen) viewForm(c Context) (string, string) {
	if s.ShapeDrawn {
		return s.viewShape(c)
	}
	footer := c.Text(i18n.SkillFormFooter, SaveKeyLabel())
	// The heading is the whole of what tells an author which of the two jobs this
	// form is doing, so it is not shared: every field is prefilled on an edit, and
	// a prefilled form under "new skill" reads as a form that has remembered the
	// last thing typed into it.
	heading := i18n.SkillFormHeading
	if s.Editing != "" {
		heading = i18n.SkillFormEditHeading
	}
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(heading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.SkillFormSubtitle)) + "\n\n")

	width := SkillLabelWidth(c)
	// The form scrolls now. It spent every row a 120x24 window has at fourteen
	// fields, and healing brought three more, so the choice was between a
	// sub-screen for the rest and a window over all of them — and a form split
	// in two makes an author hunt for a field rather than scroll to it.
	//
	// The window follows the cursor rather than the top, so tabbing to the last
	// field brings it into view instead of leaving the cursor off screen.
	from, to := Window(SkillFieldCount, s.Field, s.formRoom(c))
	if from > 0 {
		out.WriteString("  " + c.Style.Dim.Render(Ellipsis) + "\n")
	}
	for field := from; field < to; field++ {
		marker := "  "
		if field == s.Field {
			marker = "> "
		}
		name := Pad(SkillFieldLabel(c, field), width)
		if field == s.Field {
			name = c.Style.Selected.Render(name)
		} else {
			name = c.Style.Label.Render(name)
		}
		out.WriteString(marker + name + " " + s.FieldValue(c, field, width) + "\n")
	}

	if to < SkillFieldCount {
		out.WriteString("  " + c.Style.Dim.Render(Ellipsis) + "\n")
	}

	out.WriteString("\n")
	out.WriteString(s.DamageRow(c, width))
	// The help for the field the cursor is on, at the body's own indent rather
	// than in the value column. The label column is measured per language and
	// takes a third of the row in Vietnamese; a sentence that has to say what a
	// field means *and* show a valid answer cannot spend that, and it is not a
	// value belonging to a row anyway.
	//
	// The last line carries no newline of its own, and that is a line rather than
	// a tidy: frame splits the body on newlines, so a trailing one leaves an
	// empty string that costs a row of the twenty a 120x24 window has. This form
	// spent all twenty before the name field arrived; dropping the newline is
	// what paid for it, with nothing to see on screen either way. It is the same
	// accounting skillsRoom records for the listing, which has never had one.
	tail := []string{"  " + c.Style.Dim.Render(SkillFieldHelp(c, s.Field))}
	if s.Err != nil {
		tail = append(tail,
			c.Style.Bad.Render(c.Text(i18n.WriteRefused, c.Lang.Error(s.Err))))
	}
	out.WriteString(strings.Join(tail, "\n"))
	return out.String(), footer
}

// FieldValue is what one row shows: a chooser, a chosen list, or what was typed.
//
// Exported for the reason DamageRow is — the client's width sweep reads two of
// these rows, the allowlist and the chance reading, as the two callers of
// FieldValueRoom that live on this screen.
func (s SkillsScreen) FieldValue(c Context, field, labelWidth int) string {
	choice := func(values []string, index int) string {
		if len(values) == 0 {
			return c.Style.Bad.Render(c.Text(i18n.NoneCatalogued))
		}
		return fmt.Sprintf(ChoiceFormat, c.Lang.Glossed(at(values, index)),
			c.Style.Dim.Render(c.Text(i18n.ChoicePosition,
				Clamp(index, 0, len(values)-1)+1, len(values))))
	}
	switch field {
	case SkillFieldElement:
		return choice(forge.ElementNames(), s.ElementIndex)
	case SkillFieldTarget:
		return choice(forge.TargetNames(), s.TargetIndex)
	case SkillFieldShape:
		return choice(c.Lib.PatternNames(), s.ShapeIndex)
	case SkillFieldKeptForElements:
		return s.listValue(c, s.KeptElements, labelWidth)
	case SkillFieldKeptForRoles:
		return s.listValue(c, s.KeptRoles, labelWidth)
	case SkillFieldKeptForCharacters:
		return s.listValue(c, s.KeptWho, labelWidth)
	case SkillFieldKeptForSpecies:
		return s.listValue(c, s.KeptKinds, labelWidth)
	case SkillFieldKeptForOrigins:
		return s.listValue(c, s.KeptWorlds, labelWidth)
	case SkillFieldAccuracy, SkillFieldPower, SkillFieldPierce, SkillFieldCrit,
		SkillFieldRestores, SkillFieldDrains:
		// Every one of these is authored in parts per thousand because that is
		// what the engine multiplies and divides by, but nobody reads 850 as a
		// chance or 2200 as "twice over". The percentage sits beside the field rather than
		// replacing it: the number written to the file is still the number on
		// screen.
		return s.Inputs[field].View() + s.percentHint(c, field)
	case SkillFieldInflicts:
		// The chances in this field are parts per thousand too, but the field
		// holds a whole list in the syntax ParseApplications reads, so the
		// reading goes beside it rather than into it.
		return s.Inputs[field].View() + s.chanceHint(c, labelWidth)
	default:
		return s.Inputs[field].View()
	}
}

// chanceHint reads out the chances in the inflicts field. A list being typed is
// unparseable most of the time, and that is not an error to announce, so it says
// nothing until the whole list parses.
func (s SkillsScreen) chanceHint(c Context, labelWidth int) string {
	typed := strings.TrimSpace(s.Inputs[SkillFieldInflicts].Value())
	if typed == "" {
		return ""
	}
	applications, err := c.Lib.ParseApplications(typed)
	if err != nil || len(applications) == 0 {
		return ""
	}
	// The field itself is a fixed width, so the row's only unbounded part is
	// this reading: a skill may apply any number of statuses, and five of them
	// would push the row past the floor. Clipping the reading is right where
	// clipping the value would not be — a chance you cannot see is still
	// written in the field beside it.
	//
	// What is left is measured from the field as it is *drawn* rather than from
	// the Width it was given, and the difference is a real cell: a bubbles text
	// field renders its own trailing cursor, so its View is a cell wider than
	// its Width. A room computed from the declaration left this row 80 cells
	// wide in an 80-column window — inside frame's clip, so nothing was cut, and
	// over the edge on the terminals that wrap a line filling the final cell.
	// It only became visible when the label column grew.
	//
	// Against the window and not the floor: a chance is data, and a list of them
	// cut short on a wide terminal hides one of the numbers being authored.
	room := FieldValueRoom(c.UsableWidth(), labelWidth,
		lipgloss.Width(s.Inputs[SkillFieldInflicts].View()))
	return "  " + c.Style.Dim.Render(Clip(forge.ApplicationChances(applications), room))
}

// percentHint is the dim reading of a parts-per-thousand field, or nothing at
// all while the field does not hold one. A half-typed number is the normal
// state of a text field, so it is not an error to say nothing about.
func (s SkillsScreen) percentHint(c Context, field int) string {
	permille, err := strconv.Atoi(strings.TrimSpace(s.Inputs[field].Value()))
	if err != nil || permille == 0 {
		// A support skill declares no power, and "0%" says nothing the zero did
		// not.
		return ""
	}
	return "  " + c.Style.Dim.Render(forge.Percent(permille))
}

// listValue draws one of the three allowlists: what is in it, or that anybody
// may carry the skill, which is what an empty list means.
//
// The list is ids, so it takes the window: an allowlist clipped at the floor
// stops naming the last character it is kept for, and which characters those
// are is the whole content of the field.
func (s SkillsScreen) listValue(c Context, chosen []string, labelWidth int) string {
	room := FieldValueRoom(c.UsableWidth(), labelWidth,
		lipgloss.Width(c.Text(i18n.KitChooseHint)))
	if len(chosen) == 0 {
		return c.Style.Dim.Render(c.Text(i18n.WhoAnyone) + "  " + c.Text(i18n.KitChooseHint))
	}
	return Clip(strings.Join(chosen, " "), room) + "  " +
		c.Style.Dim.Render(c.Text(i18n.KitChooseHint))
}

// DamageRow is the point of authoring a skill on a screen rather than in a file:
// what the power being typed is actually worth, before it is written.
//
// Exported for the reason SkillLabelWidth and FieldValue are: the client's width
// sweep measures this row at two windows in both languages, and a fixture cannot
// follow a screen across a package boundary.
func (s SkillsScreen) DamageRow(c Context, labelWidth int) string {
	preview, err := c.Lib.PreviewDraft(s.Draft(c))
	if err != nil {
		return c.LabelAt(c.Text(i18n.LabelDamage), labelWidth, "%s",
			c.Style.Bad.Render(c.Lang.Error(err)))
	}
	return c.LabelAt(c.Text(i18n.LabelDamage), labelWidth, "%s",
		c.Lang.Damage(preview))
}

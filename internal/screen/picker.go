package screen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The multi-select: a list of ids, several of which may be chosen, drawn over
// whichever screen raised it.
//
// It is the eighth screen in this package and the first with no listing of its
// own — it is **handed** its rows, which is what lets five different questions
// share one set of keys — so a client keeps it beside its screens rather than on
// one, and hands it every keystroke while it is up. Where the answer goes is the
// client's word and never read here; see PickState.Into.
//
// # Why a sub-screen and not a pane on the form
//
// Measured, not judged. The new-character form draws fifteen rows plus a
// heading, a blank and two live-check rows: nineteen lines of a body that has
// c.Height-4 to work in, so twenty in the minimum window. There is no room
// beside it for a list of nineteen skills, and the alternative — hiding the
// skills that do not fit — is the one thing a picker must not do, because a
// hidden skill reads as a skill that does not exist.
//
// So the list takes the whole screen while it is open, and scrolls: the body
// gets c.Height-4, this screen spends five of those on its heading, hint and
// summary, and the rest is rows. That is thirteen rows in a 120x24 terminal
// against nineteen skills, so it scrolls there and shows everything in anything
// taller. A scrolling list rather than paging, because the position counter and
// the cursor together already say where you are, and a page boundary is one more
// thing to learn.
//
// # Why one picker for five different lists
//
// The kit is chosen from the skill book; a restriction's three allowlists are
// chosen from the element, archetype and cast books; and what a skill inflicts is
// chosen from the status book. All five are "pick several from a list, keep the
// order", and one implementation is one set of keys to learn. What differs is
// only how a row's detail reads, which is what PickKind selects, and the detail
// is worded at draw time rather than stored — a stored sentence would still be in
// the old language after ctrl+l.
//
// Two of the five carry something more, and each earns it. The statuses collect a
// chance beside the list, because a status is an id *and* a number. The
// characters carry a filter, and the reason is not only that the list grows —
// the skill book grows too — but that a character has an obvious axis to narrow
// by and the cast browser already narrows by it. A filter needs somewhere to come
// from; the skill book has no equally natural grouping, so the kit picker scrolls
// and does not filter.
//
// # Why one of the five collects a number as well
//
// A status is not an id. The field it goes into takes "status:chance" in parts
// per thousand, so a picker that handed back ids alone would leave the author
// back in a text field remembering a syntax — which is the complaint the picker
// was raised to answer. So the status picker carries one text field under its
// list, and enter applies both halves.
//
// One chance for everything picked, which is a limit rather than an oversight: a
// chance per row would be a form inside a list. Two statuses at two chances is
// two trips, or one trip and an edit — the field is still a text field and
// forge.ParseApplications is still the only parser.

// PickKind is what a picker is choosing.
type PickKind int

const (
	// PickSkills is the kit, whose rows say who may carry each skill.
	PickSkills PickKind = iota
	// PickElements, PickArchetypes, PickCharacters, PickSpecies and PickOrigins
	// are a restriction's five allowlists, whose rows are ids and their
	// Vietnamese names.
	PickElements
	PickArchetypes
	PickCharacters
	// PickSpecies is one of the two whose name comes out of a data file rather
	// than a compiled gloss, because a species is authored with its name beside
	// it -- the same arrangement a passive has.
	PickSpecies
	// PickOrigins is the second of those, and for the same reason: an origin is
	// authored with its title beside it in origins.json.
	PickOrigins
	// PickStatuses is what a skill inflicts, whose rows carry each status's own
	// facts — how long it lasts, how far it stacks, whether it ticks — because
	// choosing between "mire" and "expose" on the ids alone is not choosing.
	PickStatuses
	// PickPassives is the trait a placement keeps, and it exists because the
	// squad builder was raising its trait picker as PickSkills: the rows were
	// trait ids and the detail looked each one up in the skill book, so every
	// row on that screen drew "unknown skill" in red where its detail belonged.
	// A kind of its own rather than a lookup that tries both books, for the
	// reason every other kind here is one — what a row *is* decides how it
	// reads, and a detail guessing at that is a detail that can guess wrong.
	PickPassives
)

// PickOption is one row: an id, and why the carrier may not take it.
//
// The refusal is an error value rather than a sentence, and it is the same value
// forge.CheckSkill hands the write. A picker that decided availability for
// itself would be a second copy of the rule, and the copy would be the one an
// author trusted.
type PickOption struct {
	ID      string
	Refusal error
	// Group is what a filter narrows the list by, empty for a picker with none.
	//
	// One string per row rather than a predicate per filter, because the only
	// thing a filter has to do here is answer "is this row in the group in
	// front" — and a row carrying its own group means the filtering is the same
	// code whatever the group turns out to be.
	Group string
}

// PickAnswer is what a picker hands back: what was chosen, and the one thing
// that was typed beside the list.
//
// A struct rather than two parameters so that the pickers with nothing to
// type are not written as though they had ignored something — and so that a
// sixth picker needing a second answer does not change four call sites.
type PickAnswer struct {
	// Chosen is in the order it was picked, because for a kit that order *is*
	// the kit.
	Chosen []string
	// Typed is the extra field's answer, empty for a picker that has none.
	Typed string
}

// PickState is a picker while it is open.
type PickState struct {
	Title   i18n.Key
	Hint    i18n.Key
	Footer  i18n.Key
	Kind    PickKind
	Options []PickOption
	// Chosen is the answer, in the order it was chosen, because for a kit that
	// order *is* the kit.
	Chosen []string
	// Slots is how many the answer may hold, and **nought is uncapped**.
	//
	// Uncapped is what every picker but two is, so nought is the honest default
	// rather than a flag beside a number: the five restriction allowlists take
	// as many ids as an author cares to name, the status picker likewise, and
	// the character form's kit picker is bounded by the write rather than by a
	// count. The two the squad builder raises are the exception, because a
	// placement spends exactly cast.SkillSlots and cast.TraitSlots.
	//
	// It is enforced in Toggle rather than only where the answer lands, so an
	// answer past the limit cannot be built at all — the refusal used to arrive
	// after enter, by which time the author had picked six and had to reopen the
	// list to give two back.
	Slots  int
	Cursor int
	// Typed is the one answer collected beside the list, and Label names it.
	// Both are absent for the pickers that choose ids and nothing else.
	Typed *textinput.Model
	Label i18n.Key
	// Groups are the values f narrows the list by, and Filter indexes them from
	// one — zero is the filter that hides nothing, exactly as the cast browser's
	// own filter is numbered, and for the same reason: its name is a translated
	// word while every entry in the list is an id that reads the same in either
	// language.
	//
	// Empty for a picker with nothing to narrow. The list a filter is worth
	// having on is the one that grows with the cast, which is the characters.
	Groups []string
	Filter int
	// Reading is whether the description of the row under the cursor is in
	// front of the list, and Scroll is how far down that description has been
	// walked.
	//
	// A state of the picker rather than a screen, and that is forced rather than
	// preferred: the picker is drawn over whichever screen raised it and takes
	// keys before any screen does, so a description reached by switching screens
	// would be a screen the picker went on swallowing the keys of. It is the
	// same reading BlurbScreen gives, from the one place a kit is actually
	// chosen — an author picking four of nine wants to know what the fifth does
	// *there*, not on the listing two screens away.
	//
	// Scroll is not a second cursor, for the reason BlurbScreen's is not: it
	// selects nothing, it is which lines of one answer are visible, and every
	// key that changes *which* row is described resets it so it cannot survive
	// into a shorter answer and leave a reader looking at nothing.
	//
	// ⚠️ **Measured: no shipped description reaches it.** BlurbScreen scrolls
	// because it draws all five of a character's traits at once, which is some
	// thirty lines against the seventeen an 120-by-24 window leaves;
	// one row is at most three, in either language, across every shipped skill
	// and trait. So this is the guard and not a live path — kept because a trait
	// is allowed six sentences and the room falls to three in a small window, and
	// because dropping it would leave the one screen built for reading a
	// description unable to finish reading one the day somebody writes a long
	// one. What that costs is two cases; what the answer being clamped where it
	// is *read* buys is that an offset past a short answer still draws it.
	Reading bool
	Scroll  int
	// Into is where the answer lands when the list is closed with enter, and
	// **this package never looks at it**: it is stored, carried through the
	// keystroke that closes the list, and handed straight back on the result.
	//
	// ⚠️ **It is deliberately opaque, and that is what let the picker move.** A
	// destination names one field of one *client's* screen — the kit on that
	// client's character form, the fifth allowlist on its skill form — so it is
	// that client's vocabulary and cannot be declared here. Carrying it as `any`
	// is the same arrangement the unsaved-changes guard already has, where the
	// question records the client's own screen enum and nothing in the question
	// reads it. The client is the only thing that knows what a value here means,
	// and the only thing entitled to interpret one.
	//
	// ⚠️ **It used to be an `apply func(model, PickAnswer) model`**, and that
	// closure is what kept this screen in one binary: a callback naming `model`
	// names one client, so a screen holding one is a screen written for the
	// client it was written in. Measured before it was replaced — there were
	// **ten** of them, in three files, and every one wrote the answer into one
	// named field of its own screen and then set a flag or two beside it. That
	// is a destination, and a destination is data.
	//
	// A nil Into is a picker whose answer goes nowhere, which is what a state
	// built by hand carries: enter closes the list and writes nothing.
	Into any
}

// Describes reports whether the row under the cursor has a description behind
// it, which is what decides whether ? is offered at all.
//
// Two of the eight kinds, and the division is which books hold a describer a
// player would read: a skill has Describe and a trait has DescribePassive. An
// element, a role, a character, a species and an origin have none — a row there
// is an id and its name, which is already the whole of what could be said.
//
// A status has one and is deliberately left out. Its rows already carry the
// facts a description would derive, its picker spends the keys under the list on
// a chance field, and Lang.DescribeStatus has a reference screen of its own that
// reaches every status rather than only the ones a skill happens to inflict.
func (p *PickState) Describes() bool {
	return p.Kind == PickSkills || p.Kind == PickPassives
}

// Visible is the rows the filter in force leaves on screen.
//
// Everything chosen stays chosen, whether or not the filter shows it: the
// summary under the list is the whole answer and it is drawn from chosen rather
// than from this, so narrowing the list is a way to find a row and never a way
// to lose one.
func (p *PickState) Visible() []PickOption {
	wanted := p.Group()
	if wanted == "" {
		return p.Options
	}
	out := make([]PickOption, 0, len(p.Options))
	for _, option := range p.Options {
		if option.Group == wanted {
			out = append(out, option)
		}
	}
	return out
}

// Group is the value in force, or empty when every row is shown.
func (p *PickState) Group() string {
	if p.Filter <= 0 || p.Filter > len(p.Groups) {
		return ""
	}
	return p.Groups[p.Filter-1]
}

// groupName is what the filter is called on screen: the group's own id, or the
// translated word for everything.
func (p *PickState) groupName(c Context) string {
	if group := p.Group(); group != "" {
		return group
	}
	return c.Text(i18n.BrowseAllOrigins)
}

// NextFilter steps the filter on, wrapping through "everything".
//
// The key and the cycle are the cast browser's, deliberately: an author who has
// filtered the cast once should not have a second interaction to learn for the
// same job on a list of the same things.
func (p *PickState) NextFilter() {
	p.Filter = (p.Filter + 1) % (len(p.Groups) + 1)
	p.Cursor = Clamp(p.Cursor, 0, len(p.Visible())-1)
}

// Raise settles the defaults a raiser did not fill in and hands the picker back
// ready to be put in front. Nothing is applied until it is closed with enter.
//
// It returns the same pointer it was given rather than a value, because the
// pointer **is** the presence flag: a client stores it, branches on it not being
// nil to decide whether a picker is up, and closes the list by writing nil. Ten
// raise sites build a literal and hand it here, so the three answers below are
// settled once rather than at each of them.
func (p *PickState) Raise() *PickState {
	p.Cursor = Clamp(p.Cursor, 0, len(p.Options)-1)
	if p.Hint == 0 {
		p.Hint = i18n.PickerHint
	}
	if p.Footer == 0 {
		// The footer that names ? is the default only where ? does something.
		// A key announced on a screen that ignores it is worse than one nobody
		// was told about, and the two footers are one line each rather than one
		// line assembled from parts, because the line has to be measured whole:
		// the English one is 79 cells of the 79 the minimum window leaves.
		p.Footer = i18n.PickerFooter
		if p.Describes() {
			p.Footer = i18n.PickerDescribeFooter
		}
	}
	return p
}

// NumberField is the chance field the status picker carries: a text field that
// takes digits and nothing else.
//
// It takes the terminal's answer as a parameter for the reason NewInput does:
// this package reads no environment, so the binary asks and hands the answer
// down.
//
// Narrow on purpose. A picker's other keys are space, the arrows, k, j, enter
// and escape, so letters have to stay out of the field or moving the cursor
// would type into it — and the field holds a figure in parts per thousand, which
// is four digits at most.
//
// The default is the *placeholder* rather than the value, and that was measured
// rather than chosen: a four-digit default in a field with a four-character limit
// is a field that refuses the next keystroke, so the first thing an author has to
// do is delete an answer they did not give. Empty with the default shown means
// typing works immediately and pressing enter without typing still writes it —
// forge.AddApplications reads a blank chance as the default, so the two front-ends
// cannot disagree about what nothing means.
func NumberField(plain bool, placeholder string) *textinput.Model {
	input := NewInput(plain)
	input.Prompt = ""
	input.CharLimit = 4
	input.SetWidth(6)
	input.Placeholder = placeholder
	input.Focus()
	return &input
}

// NumberKey reports whether a keystroke belongs in a number field: a digit, or
// one of the keys that edit or move within one.
//
// The filter is here rather than on the field's own Validate, and that is not a
// preference. bubbles' Validate does not refuse a keystroke — it takes the value
// and records an error beside it — so a letter typed at a "numeric" field sits in
// the field looking accepted, and only a caller reading Err would ever know. It
// was written the other way first and a test found the letters in the field.
// Refusing the key is what makes the field numeric.
// A keystroke is read through Code and Text rather than through a key type,
// because bubbletea v2 has none: an editing key is a named Code, and a
// printable one carries the characters it produced in Text. Text is empty
// whenever a modifier is held, which is what keeps ctrl+1 out of the field
// without a second check for it.
func NumberKey(message tea.KeyPressMsg) bool {
	switch message.Code {
	case tea.KeyBackspace, tea.KeyDelete, tea.KeyLeft, tea.KeyRight,
		tea.KeyHome, tea.KeyEnd:
		return true
	}
	if message.Text == "" {
		return false
	}
	for _, letter := range message.Text {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return true
}

// KitOptions is the skill book as rows, each carrying whether this character may
// take it and why not.
//
// Every skill is listed, including the ones this character cannot take. That is
// the point: a kit is chosen against a book an author is still learning, and a
// row saying "kept for the bulwark role" teaches something a missing row does
// not.
func KitOptions(lib *forge.Library, who forge.Carrier) []PickOption {
	skills := lib.Skills().Skills()
	out := make([]PickOption, 0, len(skills))
	for _, carried := range skills {
		out = append(out, PickOption{ID: carried.ID, Refusal: forge.CheckSkill(who, carried)})
	}
	return out
}

// IDOptions is a plain list of ids, for the three allowlists.
func IDOptions(ids []string) []PickOption {
	out := make([]PickOption, 0, len(ids))
	for _, id := range ids {
		out = append(out, PickOption{ID: id})
	}
	return out
}

// CharacterOptions is the authored cast as rows, each carrying the work it was
// borrowed from so the list can be narrowed by it.
//
// The order is the cast book's own, which is what the browser lists too, and the
// group is the character's origin — the same axis the browser filters on, so an
// author filtering here is filtering by the thing they already filter by there.
//
// ⚠️ **A held-back character is offered here, and that is not an oversight —
// do not "finish the job" by filtering cast.Character.Hidden out of this list.**
// The squad builder honours the flag because it is choosing *who fights*, and a
// character held back is one an author has taken out of that choice. This list
// answers a different question: which characters is this skill kept for, written
// into `restrict.characters`. Hiding a row here would make an existing
// restriction naming that character **unauthorable** — a skill whose allowlist
// already names it could not be re-saved with the name it has, and there is no
// second way in, because the field is a picker and nothing else writes it. The
// two lists share a shape and share nothing else.
func CharacterOptions(lib *forge.Library) []PickOption {
	characters := lib.Characters().All()
	out := make([]PickOption, 0, len(characters))
	for _, character := range characters {
		out = append(out, PickOption{ID: character.ID, Group: character.Origin})
	}
	return out
}

// StatusOptions is the status book as rows.
//
// Every status is offered and none can be refused: a skill may inflict anything
// the book declares, so there is no carrier to check against and no row that
// carries a mark. What each row carries instead is the status's own facts, which
// is the thing an id does not say.
func StatusOptions(lib *forge.Library) []PickOption {
	book := lib.StatusBook()
	out := make([]PickOption, 0, len(book))
	for _, kind := range book {
		out = append(out, PickOption{ID: kind.ID})
	}
	return out
}

// PickResult is what a keystroke did to a picker, beside the picker itself.
//
// ⚠️ **The pair is not (itself, Action), which every other screen here answers
// with, and the reason is that an Action cannot carry an answer.** A closed
// picker hands back a set of ids and one typed string; Action holds a Kind, a
// Target and a Subject, and a Subject is one id — so a kit of four would have to
// be encoded into a field built to name one thing, which is the shape this
// repository has twice paid for. The two are not even the same question: an
// Action says *what a screen wants a client to do next*, and none of the ten
// destinations wants anything done — a pick fills in a field and leaves the
// reader in front of the form they were filling. So there is no Action here at
// all, and there is a result type instead.
//
// The **picker** is the other half of the pair and is returned rather than
// carried in here, because the pointer is the presence flag: nil is the list
// coming down, which is what esc and enter both do.
type PickResult struct {
	// Answered reports the list came down on enter rather than on esc, which is
	// the difference between a finished pick and an abandoned one.
	//
	// ⚠️ It is carried **beside** Into rather than read off it. A picker with no
	// destination is legal — a state built by hand carries none — so a nil Into
	// means "this answer lands nowhere", which is a different sentence from
	// "there was no answer", and the two would be one field otherwise. Absence
	// declared, never detected: the rule the log's follow flag and the queue's
	// pending reading are both written under.
	Answered bool
	// Into is the destination the picker was raised with, handed straight back
	// and never looked at. See PickState.Into.
	Into any
	// Answer is what was chosen and what was typed, and is only filled in when
	// Answered.
	Answer PickAnswer
	// Cmd is whatever the field under the list asked for — a cursor blink, on
	// the one picker in five that has a field at all.
	//
	// ⚠️ **A tea.Cmd here is the one thing no other screen in this package
	// returns**, and it is unavoidable rather than a widening of the vocabulary:
	// bubbles' textinput answers an Update with a command, and dropping it would
	// leave the chance field with no cursor. Everything else a keystroke wants is
	// still said in data.
	Cmd tea.Cmd
}

// Update reads one keystroke and hands back the picker and what the client owes
// it.
//
// Three outcomes, and the pair says which without a fourth field: the picker
// comes back with the list still up, or nil with nothing answered, which is esc,
// or nil with an answer and the destination it belongs to, which is enter.
func (p *PickState) Update(_ Context, message tea.KeyPressMsg) (*PickState, PickResult) {
	if p.Reading {
		return p.read(message)
	}
	switch message.String() {
	case "?":
		// Only where a row has something behind it. Elsewhere ? goes nowhere,
		// which is what it did before and what f does on a picker with no
		// groups — and the one thing that could still have wanted it, the
		// chance field, refuses it already, because ? is not a digit.
		if p.Describes() {
			p.Reading, p.Scroll = true, 0
		}
	case "esc":
		return nil, PickResult{}
	case "enter":
		answer := PickAnswer{Chosen: append([]string(nil), p.Chosen...)}
		if p.Typed != nil {
			answer.Typed = p.Typed.Value()
		}
		// The list comes down and what the landing needs travels back with it,
		// which is the guard's ordering for the same reason: a landing that read
		// the client's own picker field would have to run while the picker it is
		// closing is still in front.
		//
		// ⚠️ This used to read `apply := p.apply` before clearing the client's
		// field and call it after. Nothing was lost with that ordering — the
		// receiver is a local pointer and clearing a field on the model never
		// reached it — and it stops being something a reader has to hold, because
		// a destination cannot be un-set by anything the client does next.
		return nil, PickResult{Answered: true, Into: p.Into, Answer: answer}
	case "up", "k":
		p.Cursor = Clamp(p.Cursor-1, 0, len(p.Visible())-1)
	case "down", "j":
		p.Cursor = Clamp(p.Cursor+1, 0, len(p.Visible())-1)
	case "space":
		p.Toggle()
	case "f":
		// Only where there is something to narrow. On a picker with no groups f
		// is a letter nothing is listening for, which is what it was before.
		if len(p.Groups) > 0 {
			p.NextFilter()
			return p, PickResult{}
		}
		fallthrough
	default:
		// Anything left goes to the field, when there is one and the key belongs
		// in it. It is checked last so that a picker's own keys cannot be
		// swallowed by a text field, and only digits get through, so k and j stay
		// movement rather than becoming text.
		if p.Typed != nil && NumberKey(message) {
			updated, command := p.Typed.Update(message)
			*p.Typed = updated
			return p, PickResult{Cmd: command}
		}
	}
	return p, PickResult{}
}

// read is the picker with a description in front of the list.
//
// The keys are BlurbScreen's, deliberately: an author who has read a skill from
// the listing should not have a second interaction to learn for the same job
// here. ? or esc closes it, and closes only it — the list comes back with
// everything that was chosen still chosen, because a description is a question
// asked *while* choosing and answering it must not cost the answer.
//
// Walking the cursor from here is why one description can be read after the
// next without going back and forth, and it walks the picker's own cursor
// rather than a cursor of this state's — the same borrowing BlurbScreen does,
// and for the same reason: two cursors are two things that can point at
// different rows.
//
// It takes no Context because none of these keys draws anything, and it cannot
// close the list: only the outer switch does that, so this arm always comes back
// with the picker still up.
func (p *PickState) read(message tea.KeyPressMsg) (*PickState, PickResult) {
	switch message.String() {
	case "?", "esc":
		p.Reading = false
	// [ and ] alias the page keys, here and at the other two sites that scroll,
	// for a keyboard that has no PgUp and no PgDn. Reading is entered before the
	// typed field is ever reached — PickState.Update returns into this function
	// at its first line — and the field only ever accepts a digit anyway
	// (NumberKey), so neither bracket can be swallowed as text.
	case "pgdown", "]":
		p.Scroll++
	case "pgup", "[":
		p.Scroll = max(p.Scroll-1, 0)
	case "up", "k":
		p.Cursor = Clamp(p.Cursor-1, 0, len(p.Visible())-1)
		p.Scroll = 0
	case "down", "j":
		p.Cursor = Clamp(p.Cursor+1, 0, len(p.Visible())-1)
		p.Scroll = 0
	}
	return p, PickResult{}
}

// Toggle takes the row under the cursor in or out.
//
// Taking one out always works, including one that has become unavailable since
// it was chosen and including one taken out of a list that is already over its
// slots — that is the state an author most needs to be able to fix, which is why
// it is the first branch and has to stay the first branch. A loadout hand-edited
// past the limit in squads.json opens here, and the whole reason to open it is
// to give a row back.
//
// Taking one in is refused when the carrier may not have it, and refused again
// when the slots are full. Nothing is said about either refusal here because it
// is already on screen: the row carries its mark, the line under the list
// carries the whole sentence, and the counter beside the title says how much of
// the loadout is spoken for — all before space is ever pressed. A message raised
// by the keypress would be the same words twice.
//
// A full list **blocks** rather than swapping the oldest row out. With one trait
// slot that swap is tempting and it would make space two verbs at once, worded
// differently on the two pickers of the same screen depending on how many slots
// each has. One rule instead: the list is full, space does nothing, take one out
// first — which is what the trait hint already says.
func (p *PickState) Toggle() {
	rows := p.Visible()
	if len(rows) == 0 {
		return
	}
	option := rows[Clamp(p.Cursor, 0, len(rows)-1)]
	if at := slices.Index(p.Chosen, option.ID); at >= 0 {
		p.Chosen = append(p.Chosen[:at:at], p.Chosen[at+1:]...)
		return
	}
	if option.Refusal != nil {
		return
	}
	if p.Slots > 0 && len(p.Chosen) >= p.Slots {
		return
	}
	p.Chosen = append(p.Chosen, option.ID)
}

// Room is how many rows the list has, measured from the window in hand.
//
// The count was wrong the first time it was written and the screen said so: the
// body has c.Height-4 lines to draw in, and this screen spends **seven** of
// them — its heading, its hint, a blank, the rows, a blank, the summary, the
// refusal for the row under the cursor, and the empty string that the trailing
// newline leaves behind when frame splits the body. Miss one and frame truncates
// the bottom of the list, which is exactly the failure a scrolling list is for.
//
// Three rows is the floor: below that the list is not a list, and a window that
// small is already refused by the too-small screen.
//
// A picker carrying a field of its own spends two more — the field's row and the
// blank above it — and it is counted here rather than subtracted at the call
// site, so the one place that knows what this screen draws is the one place that
// counts it.
func (p *PickState) Room(c Context) int {
	spent := 7
	if p.Typed != nil {
		spent += 2
	}
	if len(p.Groups) > 0 {
		spent++
	}
	room := c.Height - 4 - spent
	if room < 3 {
		return 3
	}
	return room
}

// Window and Clip were this screen's private helpers once, and both are the
// package's own rules now — Window in listing.go beside Clamp, because every
// listing here scrolls by it, and Clip in screen.go beside Pad, because a
// client's frame cuts every line of every screen through it. Their comments
// there carry the arguments.

// IDColumn is the column the ids sit in, measured from the ids being drawn.
func (p *PickState) IDColumn() int {
	widest := 0
	for _, option := range p.Options {
		if width := lipgloss.Width(option.ID); width > widest {
			widest = width
		}
	}
	return widest + 1
}

// detailColumn is the column the ids are padded to, and **nought when the list
// has no detail column at all** — the spelling speciesRow and passiveRow already
// use for a column that is dropped rather than drawn empty.
//
// Worked out once per draw over the rows being drawn, never per row. A collapse
// decided row by row would take the padding off the rows whose cell happens to
// be empty and leave it on the rest, which is a ragged table rather than no
// table; the whole point of a column is that every row shares it.
//
// ⚠️ The condition is **"nothing in this list has a detail"**, which is the true
// statement, and deliberately not "the language is English" or "the kind is
// PickPassives". Those two are proxies: they hold on the shipped books today and
// stop holding the day a Vietnamese list has an empty column — a trait book with
// an unnamed trait — or an English one gains a detail. A proxy that stops
// holding draws the wrong table and fails nothing.
//
// The scope is the **visible** rows rather than the window, because the filter
// chooses which list is on screen (and the "showing" line says so) while the
// scroll position is only where in that list a reader has got to: a
// window-scoped reading would make the column appear and disappear as the
// cursor walked. The width itself still comes from IDColumn, which measures
// every option, so filtering cannot shift the ids sideways either.
//
// Emptiness is measured with lipgloss.Width and not against "", because a detail
// is styled before it is returned and an empty cell rendered through a style is
// escape codes around nothing.
func (p *PickState) detailColumn(c Context, rows []PickOption) int {
	for _, option := range rows {
		if lipgloss.Width(p.Detail(c, option.ID)) > 0 {
			return p.IDColumn()
		}
	}
	return 0
}

// Detail is what a row says about its option, beyond the id.
//
// For a skill that is who may carry it, in the same words the listing and a
// refusal use. For the three allowlists it is the id's Vietnamese name, which is
// nothing at all in English — there, an id is shown as the data writes it, and
// detailColumn drops the whole column rather than drawing a row of blanks.
//
// ⚠️ It is exported for the client's width sweep, which enumerates this column
// over every row of whichever picker a screen is holding open. That sweep calls
// this rather than rebuilding one, deliberately: a second reading would have to
// know the eight kinds apart, and a data column enumerated by a copy of the
// thing that draws it goes quietly wrong the day a ninth is added.
func (p *PickState) Detail(c Context, id string) string {
	if p.Kind == PickStatuses {
		// A status's facts, glossed name first: "trúng độc · dot · 3 lượt ·
		// tối đa 3 lớp". The category and the numbers are what tell a poison
		// from a burn, and the ticking note is the one fact that changes what a
		// skill applying it is worth.
		for _, kind := range c.Lib.StatusBook() {
			if kind.ID != id {
				continue
			}
			// The wording stands in for the category rather than joining it:
			// "dot" is the category's own id and "damages every turn" is what it
			// means, so printing both prints it twice — and the row is long
			// enough that the second copy is what gets clipped.
			sort := c.Lang.StatusCategory(kind.Category)
			facts := c.Text(i18n.StatusDetail, sort, kind.Duration, kind.MaxStacks)
			if name := c.Lang.Gloss(id); name != "" {
				facts = name + " · " + facts
			}
			return c.Style.Dim.Render(facts)
		}
		return ""
	}
	if p.Kind == PickSpecies {
		// The authored name through Lang.SpeciesName, which is the same reading
		// the species listing gives the same book — and not c.Lang.Gloss, since
		// the word lives on the declaration rather than in the compiled tables,
		// so a lookup there would find nothing in either language.
		//
		// The accessor is what makes English right rather than what makes it
		// bare: a data name is Vietnamese whoever asks, so SpeciesName returns
		// nothing in English and the row is the id, which is the data's own name
		// for itself. Reading kind.Name raw drew "dragon  rồng" on an English
		// screen — a leak rather than a translation. Every shipped kind carries
		// a name, so in English no row has a detail at all and detailColumn
		// drops the column, exactly as the trait picker's does.
		if kind, known := c.Lib.Species().Get(id); known {
			return c.Style.Dim.Render(c.Lang.SpeciesName(kind))
		}
		return ""
	}
	if p.Kind == PickOrigins {
		// The authored title, raw, and that is not the species branch's mistake
		// repeated: a work's title is a proper noun of the work — "Pokémon",
		// "Naruto" — so it is the same word in either language, and there is no
		// Lang accessor being gone round. An origin picker therefore keeps its
		// column in English, where the species one drops its.
		if work, known := c.Lib.Origins().Get(id); known {
			return c.Style.Dim.Render(work.Title)
		}
		return ""
	}
	if p.Kind == PickPassives {
		held, err := c.Lib.Passives().Lookup(id)
		if err != nil {
			// Unreachable for the reason the skill branch below is: a learnset
			// names traits the passive book has already been parsed against.
			// Written down rather than ignored, since the alternative is the
			// blank cell this kind was added to stop being a red one.
			return c.Style.Bad.Render(c.Lang.Error(err))
		}
		// The trait's own name, which is what the listing puts beside an id and
		// what GlossedPassives gives a kit — authored in the passive book, in
		// the one language the data is written in. So an English row is the
		// bare id, exactly as the listing drops its gloss column there, and ?
		// is what an English reader gets the sentences from.
		return c.Style.Dim.Render(c.Lang.PassiveName(held))
	}
	if p.Kind != PickSkills {
		return c.Style.Dim.Render(c.Lang.Gloss(id))
	}
	carried, err := c.Lib.Skills().Lookup(id)
	if err != nil {
		// Unreachable through the library the options were built from, and
		// written down rather than ignored: an id the book has lost is a bug
		// worth seeing on the row it belongs to.
		return c.Style.Bad.Render(c.Lang.Error(err))
	}
	return c.Lang.WhoMaySummary(carried)
}

// refusalUnderCursor is the refusal on the row the cursor is on, or nothing when
// the filter has left no rows at all.
func refusalUnderCursor(rows []PickOption, cursor int) error {
	if len(rows) == 0 {
		return nil
	}
	return rows[Clamp(cursor, 0, len(rows)-1)].Refusal
}

// counted is the figure beside the title, and which figure it is depends on
// whether the answer has slots to fill.
//
// A picker with none says where in the list the answer has got to, which is what
// this screen has always drawn. A picker with slots says how much of the loadout
// is spoken for instead, because the length of the list answers nothing an author
// choosing four of nine wants to know — four skills bind and nineteen rows do
// not.
//
// The over-full reading is drawn in the refusing style, and it is not dead code
// even though toggle can no longer build one: a loadout already past its slots in
// squads.json opens here for editing, and that is exactly the state the colour is
// for.
func (p *PickState) counted(c Context) string {
	if p.Slots <= 0 {
		return c.Style.Dim.Render(c.Text(i18n.ChoicePosition, len(p.Chosen), len(p.Options)))
	}
	style := c.Style.Dim
	if len(p.Chosen) > p.Slots {
		style = c.Style.Bad
	}
	return style.Render(c.Text(i18n.ChoiceSlots, len(p.Chosen), p.Slots))
}

func (p *PickState) View(c Context) (string, string) {
	if p.Reading {
		return p.viewReading(c)
	}
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(p.Title)) + "  " + p.counted(c) + "\n")
	out.WriteString(c.Style.Dim.Render(c.Text(p.Hint)) + "\n")
	// The filter in force, on its own line and only where there is one. It names
	// both counts for the reason the browser's does: "showing fixture-anime" says
	// nothing about how much of the list is hidden.
	rows := p.Visible()
	if len(p.Groups) > 0 {
		out.WriteString(c.Style.Dim.Render(c.Text(i18n.PickerShowing,
			p.groupName(c), len(rows), len(p.Options))) + "\n")
	}
	out.WriteString("\n")
	if len(p.Options) == 0 {
		out.WriteString("  " + c.Text(i18n.PickerNothingToPick) + "\n")
		return out.String(), c.Text(p.Footer)
	}
	if len(rows) == 0 {
		// A filter that hides everything is a state an author has to be able to
		// see and step out of, not an empty screen.
		out.WriteString("  " + c.Text(i18n.PickerNothingInGroup) + "\n")
	}

	column := p.detailColumn(c, rows)
	from, to := Window(len(rows), p.Cursor, p.Room(c))
	for index := from; index < to; index++ {
		option := rows[index]
		marker := "  "
		if index == p.Cursor {
			marker = "> "
		}
		order := ""
		if at := slices.Index(p.Chosen, option.ID); at >= 0 {
			order = strconv.Itoa(at + 1)
		}
		// The state is characters and not colour: a chosen row carries its
		// position in the kit and an unavailable one carries a mark, so the
		// screen reads on a monochrome terminal and through a recording that
		// lost its escape codes.
		flag := " "
		if option.Refusal != nil {
			flag = "!"
		}
		state := fmt.Sprintf("%2s %s ", order, flag)
		// A column of nought is a list with no detail column at all, so the id
		// is not widened to one — the shape speciesRow gives a listing whose
		// name column is dropped. The selection bar then covers the bare id,
		// which is what the species and trait listings already draw when they
		// drop theirs: padding exists to give the following cell a column to
		// start at, and with no cell after it there is nothing left for the
		// width to be for.
		name := option.ID
		if column > 0 {
			name = Pad(option.ID, column)
		}
		if index == p.Cursor {
			name = c.Style.Selected.Render(name)
		} else if option.Refusal != nil {
			name = c.Style.Dim.Render(name)
		}
		row := marker + state + name
		if column > 0 {
			// Every row says who its option is for, in the same words the
			// listing and a refusal use, whether or not this character
			// qualifies. The mark is what says "not you"; the words are what say
			// why, and they are worth having on a row that is available too.
			//
			// The room is measured from the column, so it is computed here
			// rather than above: a list with no column has no width to subtract.
			//
			// And it is measured against the window rather than against the
			// floor, by the rule the skill listing's last column states:
			// MinWidth is the width this program promises to draw in, not a
			// ceiling on what it may spend, and this cell is data — a gloss, a
			// species name, an allowlist note. A restriction cut to "để dành
			// cho loài dr…" is a row that stopped saying which species it is
			// for, on a terminal with a hundred spare columns beside it. Prose
			// still wraps at the floor; a table cell is read by scanning down
			// it, so width is the one thing it can always use.
			room := c.UsableWidth() - 1 - lipgloss.Width(marker+state) - column - 1
			detail := Clip(p.Detail(c, option.ID), room)
			if option.Refusal != nil {
				detail = c.Style.Dim.Render(detail)
			}
			row += " " + detail
		}
		out.WriteString(row + "\n")
	}

	out.WriteString("\n")
	chosen := c.Style.Dim.Render(c.Text(i18n.PickerNothingChosen))
	if len(p.Chosen) > 0 {
		chosen = strings.Join(p.Chosen, " ")
	}
	// The window, not the floor: what this clips is a list of ids the author has
	// chosen, which is data, and the list has no length the program can promise.
	// The kit is the bounded case and it fits — cast.SkillSlots is 4 and the
	// widest shipped ids are 13 cells, so a full kit is 53 and the indent makes
	// 55 of the 77 there are. An **allowlist** is the unbounded one: nothing caps
	// it, so it is as long as the book behind it. The whole shipped cast is 4 ids
	// and 69 cells, which is three characters short of the floor and one
	// character away from being over it, and the status book is already 21 kinds.
	// A cell that fits until the next id is authored is not a cell that fits.
	//
	// The line is one branch narrower than it looks: with nothing chosen it holds
	// PickerNothingChosen, which is wording. That branch is short enough that the
	// wider cut never reaches it, so it stays where it is rather than being
	// split into a room of its own.
	out.WriteString("  " + Clip(chosen, c.UsableWidth()-3) + "\n")
	// The whole refusal for the row under the cursor, which is where there is
	// room for a sentence. It is drawn before anything is pressed, so a space
	// that cannot take the row in has already explained itself.
	//
	// ⚠️ **This is prose, and it takes the window anyway. The data rule is not
	// the argument — do not fold it into one.** A sentence still wraps at the
	// floor everywhere it wraps; what is different here is that this one does not
	// wrap at all, and the reason it is clipped rather than widened was never
	// stated, so it read as a floor measurement somebody would either copy or
	// "fix" for the wrong reason. The argument is:
	//
	//   - **a client's frame already clips every line it draws, at the window.**
	//     So a line
	//     handed over whole is not a line that overflows a terminal; it is a line
	//     the frame cuts at the window. Clipping at the floor buys nothing that
	//     frame was not already going to do — it only does it seventy-nine cells
	//     early.
	//   - **Twenty-four sites render c.Lang.Error, and this was the only one that
	//     clipped at all** — the only one measuring the floor. Its twenty-three
	//     siblings hand the sentence to frame. So on a two-hundred-column terminal
	//     this refusal alone was cut at seventy-nine while every other refusal in
	//     the program ran to the edge, which is a difference no reader could
	//     explain and nothing here intended.
	//
	// **Wrapping is not the fix and must not be attempted.** Clip shortens rather
	// than wraps for the reason a frame does, and here it is load-bearing rather
	// than a preference: (*PickState).Room counts this refusal as **one** of the
	// seven rows this screen spends, so a second line would push the bottom of the
	// list under frame's cut — the exact failure a scrolling list exists to
	// prevent.
	//
	// **And the cut stays rather than going away**, which is the half worth being
	// deliberate about. Deleting it and letting the frame do the cutting would
	// make this row consistent with the other twenty-three, but a frame's cut is
	// **silent** while Clip appends an ellipsis. Of the twenty-four this one was
	// the most wrong about width and the most honest about truncation; the fix
	// keeps the honesty and drops the wrongness.
	if refusal := refusalUnderCursor(rows, p.Cursor); refusal != nil {
		out.WriteString(c.Style.Bad.Render("  "+Clip(c.Lang.Error(refusal), c.UsableWidth()-3)) + "\n")
	}
	// The one answer typed beside the list, under it and named, with its reading
	// beside it — the same reading the form gives the field this picker writes
	// into, because 300 is not a chance anybody reads as one.
	if p.Typed != nil {
		// The reading is of what the field will contribute, which is the default
		// while nothing has been typed — the same value enter would write, so the
		// percentage cannot describe a chance the write does not use.
		answer := strings.TrimSpace(p.Typed.Value())
		if answer == "" {
			answer = p.Typed.Placeholder
		}
		reading := ""
		if permille, err := strconv.Atoi(answer); err == nil {
			reading = "  " + c.Style.Dim.Render(forge.Percent(permille))
		}
		out.WriteString("\n  " + c.Style.Label.Render(c.Text(p.Label)) + "  " +
			p.Typed.View() + reading)
	}
	return out.String(), c.Text(p.Footer)
}

// viewReading is the description of the row under the cursor, over the list
// rather than beside it.
//
// Over it for the reason the list itself covers the form that raised it: the
// sentences run to more lines than an 120-by-24 window has beside
// thirteen rows, and a trait's already wrap past that on their own. It scrolls
// for the same reason BlurbScreen does, and shares that screen's room: the two
// spend their lines identically — a heading, a blank, the sentences, and the
// line saying how much is left.
func (p *PickState) viewReading(c Context) (string, string) {
	footer := c.Text(i18n.PickerReadingFooter)
	rows := p.Visible()
	if len(rows) == 0 {
		return "  " + c.Text(i18n.PickerNothingToPick) + "\n", footer
	}
	cursor := Clamp(p.Cursor, 0, len(rows)-1)
	name, body := p.described(c, rows[cursor].ID)

	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(name) + "  " +
		c.Style.Dim.Render(c.Text(i18n.ChoicePosition, cursor+1, len(rows))) + "\n\n")
	room := TraitRoom(c)
	// Clamped here rather than where it is incremented, because the key that
	// moves it does not know how long the answer is: the answer belongs to the
	// row under the cursor, and that can have moved since.
	scroll := Clamp(p.Scroll, 0, max(len(body)-room, 0))
	for _, line := range body[scroll:min(scroll+room, len(body))] {
		out.WriteString(line + "\n")
	}
	if len(body) > room {
		out.WriteString(c.Style.Dim.Render("  " + c.Text(i18n.BlurbMore,
			min(scroll+room, len(body)), len(body))))
	}
	// No trailing newline, for the reason the trait blurb has none: frame splits
	// this on newlines and cuts from the bottom, so a trailing one costs the
	// line saying there is more to read.
	return strings.TrimRight(out.String(), "\n"), footer
}

// described is one row's name and the lines under it, from the describer its
// kind belongs to.
//
// One method for both halves rather than a title function beside a body
// function, because the two are one lookup: asking the book twice is asking it
// to answer differently once.
func (p *PickState) described(c Context, id string) (string, []string) {
	if p.Kind == PickPassives {
		held, err := c.Lib.Passives().Lookup(id)
		if err != nil {
			return id, []string{"  " + c.Style.Bad.Render(c.Lang.Error(err))}
		}
		return c.Lang.GlossedPassive(held), TraitSentences(c, held)
	}
	declared, err := c.Lib.Skills().Lookup(id)
	if err != nil {
		return id, []string{"  " + c.Style.Bad.Render(c.Lang.Error(err))}
	}
	return c.Lang.GlossedSkill(declared), SkillLines(c, declared)
}

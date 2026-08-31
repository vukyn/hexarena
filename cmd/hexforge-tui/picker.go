package main

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

// The multi-select, held by the model rather than by a screen, and drawn over
// whichever screen raised it — the same shape the unsaved-changes guard has.
//
// # Why a sub-screen and not a pane on the form
//
// Measured, not judged. The new-character form draws fifteen rows plus a
// heading, a blank and two live-check rows: nineteen lines of a body that has
// m.height-4 to work in, so twenty in the minimum window. There is no room
// beside it for a list of nineteen skills, and the alternative — hiding the
// skills that do not fit — is the one thing a picker must not do, because a
// hidden skill reads as a skill that does not exist.
//
// So the list takes the whole screen while it is open, and scrolls: the body
// gets m.height-4, this screen spends five of those on its heading, hint and
// summary, and the rest is rows. That is thirteen rows in an 80x24 terminal
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
// only how a row's detail reads, which is what pickKind selects, and the detail
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

// pickKind is what a picker is choosing.
type pickKind int

const (
	// pickSkills is the kit, whose rows say who may carry each skill.
	pickSkills pickKind = iota
	// pickElements, pickArchetypes, pickCharacters, pickSpecies and pickOrigins
	// are a restriction's five allowlists, whose rows are ids and their
	// Vietnamese names.
	pickElements
	pickArchetypes
	pickCharacters
	// pickSpecies is one of the two whose name comes out of a data file rather
	// than a compiled gloss, because a species is authored with its name beside
	// it -- the same arrangement a passive has.
	pickSpecies
	// pickOrigins is the second of those, and for the same reason: an origin is
	// authored with its title beside it in origins.json.
	pickOrigins
	// pickStatuses is what a skill inflicts, whose rows carry each status's own
	// facts — how long it lasts, how far it stacks, whether it ticks — because
	// choosing between "mire" and "expose" on the ids alone is not choosing.
	pickStatuses
	// pickPassives is the trait a placement keeps, and it exists because the
	// squad builder was raising its trait picker as pickSkills: the rows were
	// trait ids and the detail looked each one up in the skill book, so every
	// row on that screen drew "unknown skill" in red where its detail belonged.
	// A kind of its own rather than a lookup that tries both books, for the
	// reason every other kind here is one — what a row *is* decides how it
	// reads, and a detail guessing at that is a detail that can guess wrong.
	pickPassives
)

// pickOption is one row: an id, and why the carrier may not take it.
//
// The refusal is an error value rather than a sentence, and it is the same value
// forge.CheckSkill hands the write. A picker that decided availability for
// itself would be a second copy of the rule, and the copy would be the one an
// author trusted.
type pickOption struct {
	id      string
	refusal error
	// group is what a filter narrows the list by, empty for a picker with none.
	//
	// One string per row rather than a predicate per filter, because the only
	// thing a filter has to do here is answer "is this row in the group in
	// front" — and a row carrying its own group means the filtering is the same
	// code whatever the group turns out to be.
	group string
}

// pickAnswer is what a picker hands back: what was chosen, and the one thing
// that was typed beside the list.
//
// A struct rather than two parameters so that the pickers with nothing to
// type are not written as though they had ignored something — and so that a
// sixth picker needing a second answer does not change four call sites.
type pickAnswer struct {
	// Chosen is in the order it was picked, because for a kit that order *is*
	// the kit.
	Chosen []string
	// Typed is the extra field's answer, empty for a picker that has none.
	Typed string
}

// pickState is a picker while it is open.
type pickState struct {
	title   i18n.Key
	hint    i18n.Key
	footer  i18n.Key
	kind    pickKind
	options []pickOption
	// chosen is the answer, in the order it was chosen, because for a kit that
	// order *is* the kit.
	chosen []string
	// slots is how many the answer may hold, and **nought is uncapped**.
	//
	// Uncapped is what every picker but two is, so nought is the honest default
	// rather than a flag beside a number: the five restriction allowlists take
	// as many ids as an author cares to name, the status picker likewise, and
	// the character form's kit picker is bounded by the write rather than by a
	// count. The two the squad builder raises are the exception, because a
	// placement spends exactly cast.SkillSlots and cast.TraitSlots.
	//
	// It is enforced in toggle rather than only in the apply callback, so an
	// answer past the limit cannot be built at all — the refusal used to arrive
	// after enter, by which time the author had picked six and had to reopen the
	// list to give two back.
	slots  int
	cursor int
	// typed is the one answer collected beside the list, and label names it.
	// Both are absent for the pickers that choose ids and nothing else.
	typed *textinput.Model
	label i18n.Key
	// groups are the values f narrows the list by, and filter indexes them from
	// one — zero is the filter that hides nothing, exactly as the cast browser's
	// own filter is numbered, and for the same reason: its name is a translated
	// word while every entry in the list is an id that reads the same in either
	// language.
	//
	// Empty for a picker with nothing to narrow. The list a filter is worth
	// having on is the one that grows with the cast, which is the characters.
	groups []string
	filter int
	// reading is whether the description of the row under the cursor is in
	// front of the list, and scroll is how far down that description has been
	// walked.
	//
	// A state of the picker rather than a screen, and that is forced rather than
	// preferred: the picker is drawn over whichever screen raised it and takes
	// keys before any screen does, so a description reached by switching screens
	// would be a screen the picker went on swallowing the keys of. It is the
	// same reading blurbScreen gives, from the one place a kit is actually
	// chosen — an author picking four of nine wants to know what the fifth does
	// *there*, not on the listing two screens away.
	//
	// scroll is not a second cursor, for the reason blurbScreen's is not: it
	// selects nothing, it is which lines of one answer are visible, and every
	// key that changes *which* row is described resets it so it cannot survive
	// into a shorter answer and leave a reader looking at nothing.
	//
	// ⚠️ **Measured: no shipped description reaches it.** blurbScreen scrolls
	// because it draws all five of a character's traits at once, which is some
	// thirty lines against the seventeen an eighty-by-twenty-four window leaves;
	// one row is at most three, in either language, across every shipped skill
	// and trait. So this is the guard and not a live path — kept because a trait
	// is allowed six sentences and the room falls to three in a small window, and
	// because dropping it would leave the one screen built for reading a
	// description unable to finish reading one the day somebody writes a long
	// one. What that costs is two cases; what the answer being clamped where it
	// is *read* buys is that an offset past a short answer still draws it.
	reading bool
	scroll  int
	// apply hands the answer back to whoever raised the picker.
	apply func(model, pickAnswer) model
}

// describes reports whether the row under the cursor has a description behind
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
func (p *pickState) describes() bool {
	return p.kind == pickSkills || p.kind == pickPassives
}

// visible is the rows the filter in force leaves on screen.
//
// Everything chosen stays chosen, whether or not the filter shows it: the
// summary under the list is the whole answer and it is drawn from chosen rather
// than from this, so narrowing the list is a way to find a row and never a way
// to lose one.
func (p *pickState) visible() []pickOption {
	wanted := p.group()
	if wanted == "" {
		return p.options
	}
	out := make([]pickOption, 0, len(p.options))
	for _, option := range p.options {
		if option.group == wanted {
			out = append(out, option)
		}
	}
	return out
}

// group is the value in force, or empty when every row is shown.
func (p *pickState) group() string {
	if p.filter <= 0 || p.filter > len(p.groups) {
		return ""
	}
	return p.groups[p.filter-1]
}

// groupName is what the filter is called on screen: the group's own id, or the
// translated word for everything.
func (p *pickState) groupName(m model) string {
	if group := p.group(); group != "" {
		return group
	}
	return m.text(i18n.BrowseAllOrigins)
}

// nextFilter steps the filter on, wrapping through "everything".
//
// The key and the cycle are the cast browser's, deliberately: an author who has
// filtered the cast once should not have a second interaction to learn for the
// same job on a list of the same things.
func (p *pickState) nextFilter() {
	p.filter = (p.filter + 1) % (len(p.groups) + 1)
	p.cursor = clamp(p.cursor, 0, len(p.visible())-1)
}

// pick raises a picker. Nothing is applied until it is closed with enter.
func (m model) pick(state *pickState) model {
	state.cursor = clamp(state.cursor, 0, len(state.options)-1)
	if state.hint == 0 {
		state.hint = i18n.PickerHint
	}
	if state.footer == 0 {
		// The footer that names ? is the default only where ? does something.
		// A key announced on a screen that ignores it is worse than one nobody
		// was told about, and the two footers are one line each rather than one
		// line assembled from parts, because the line has to be measured whole:
		// the English one is 79 cells of the 79 the minimum window leaves.
		state.footer = i18n.PickerFooter
		if state.describes() {
			state.footer = i18n.PickerDescribeFooter
		}
	}
	m.picker = state
	return m
}

// numberField is the chance field the status picker carries: a text field that
// takes digits and nothing else.
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
func numberField(placeholder string) *textinput.Model {
	input := newInput()
	input.Prompt = ""
	input.CharLimit = 4
	input.SetWidth(6)
	input.Placeholder = placeholder
	input.Focus()
	return &input
}

// numberKey reports whether a keystroke belongs in a number field: a digit, or
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
func numberKey(message tea.KeyPressMsg) bool {
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

// kitOptions is the skill book as rows, each carrying whether this character may
// take it and why not.
//
// Every skill is listed, including the ones this character cannot take. That is
// the point: a kit is chosen against a book an author is still learning, and a
// row saying "kept for the bulwark role" teaches something a missing row does
// not.
func kitOptions(lib *forge.Library, who forge.Carrier) []pickOption {
	skills := lib.Skills().Skills()
	out := make([]pickOption, 0, len(skills))
	for _, carried := range skills {
		out = append(out, pickOption{id: carried.ID, refusal: forge.CheckSkill(who, carried)})
	}
	return out
}

// idOptions is a plain list of ids, for the three allowlists.
func idOptions(ids []string) []pickOption {
	out := make([]pickOption, 0, len(ids))
	for _, id := range ids {
		out = append(out, pickOption{id: id})
	}
	return out
}

// characterOptions is the authored cast as rows, each carrying the work it was
// borrowed from so the list can be narrowed by it.
//
// The order is the cast book's own, which is what the browser lists too, and the
// group is the character's origin — the same axis the browser filters on, so an
// author filtering here is filtering by the thing they already filter by there.
func characterOptions(lib *forge.Library) []pickOption {
	characters := lib.Characters().All()
	out := make([]pickOption, 0, len(characters))
	for _, character := range characters {
		out = append(out, pickOption{id: character.ID, group: character.Origin})
	}
	return out
}

// statusOptions is the status book as rows.
//
// Every status is offered and none can be refused: a skill may inflict anything
// the book declares, so there is no carrier to check against and no row that
// carries a mark. What each row carries instead is the status's own facts, which
// is the thing an id does not say.
func statusOptions(lib *forge.Library) []pickOption {
	book := lib.StatusBook()
	out := make([]pickOption, 0, len(book))
	for _, kind := range book {
		out = append(out, pickOption{id: kind.ID})
	}
	return out
}

func (p *pickState) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if p.reading {
		return p.read(m, message)
	}
	switch message.String() {
	case "?":
		// Only where a row has something behind it. Elsewhere ? goes nowhere,
		// which is what it did before and what f does on a picker with no
		// groups — and the one thing that could still have wanted it, the
		// chance field, refuses it already, because ? is not a digit.
		if p.describes() {
			p.reading, p.scroll = true, 0
		}
	case "esc":
		m.picker = nil
		return m, nil
	case "enter":
		answer := pickAnswer{Chosen: append([]string(nil), p.chosen...)}
		if p.typed != nil {
			answer.Typed = p.typed.Value()
		}
		apply := p.apply
		m.picker = nil
		if apply != nil {
			m = apply(m, answer)
		}
		return m, nil
	case "up", "k":
		p.cursor = clamp(p.cursor-1, 0, len(p.visible())-1)
	case "down", "j":
		p.cursor = clamp(p.cursor+1, 0, len(p.visible())-1)
	case "space":
		p.toggle()
	case "f":
		// Only where there is something to narrow. On a picker with no groups f
		// is a letter nothing is listening for, which is what it was before.
		if len(p.groups) > 0 {
			p.nextFilter()
			m.picker = p
			return m, nil
		}
		fallthrough
	default:
		// Anything left goes to the field, when there is one and the key belongs
		// in it. It is checked last so that a picker's own keys cannot be
		// swallowed by a text field, and only digits get through, so k and j stay
		// movement rather than becoming text.
		if p.typed != nil && numberKey(message) {
			updated, command := p.typed.Update(message)
			*p.typed = updated
			m.picker = p
			return m, command
		}
	}
	m.picker = p
	return m, nil
}

// read is the picker with a description in front of the list.
//
// The keys are blurbScreen's, deliberately: an author who has read a skill from
// the listing should not have a second interaction to learn for the same job
// here. ? or esc closes it, and closes only it — the list comes back with
// everything that was chosen still chosen, because a description is a question
// asked *while* choosing and answering it must not cost the answer.
//
// Walking the cursor from here is why one description can be read after the
// next without going back and forth, and it walks the picker's own cursor
// rather than a cursor of this state's — the same borrowing blurbScreen does,
// and for the same reason: two cursors are two things that can point at
// different rows.
func (p *pickState) read(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "?", "esc":
		p.reading = false
	// [ and ] alias the page keys, here and at the other two sites that scroll,
	// for a keyboard that has no PgUp and no PgDn. Reading is entered before the
	// typed field is ever reached — pickState.update returns into this function
	// at its first line — and the field only ever accepts a digit anyway
	// (numberKey), so neither bracket can be swallowed as text.
	case "pgdown", "]":
		p.scroll++
	case "pgup", "[":
		p.scroll = max(p.scroll-1, 0)
	case "up", "k":
		p.cursor = clamp(p.cursor-1, 0, len(p.visible())-1)
		p.scroll = 0
	case "down", "j":
		p.cursor = clamp(p.cursor+1, 0, len(p.visible())-1)
		p.scroll = 0
	}
	m.picker = p
	return m, nil
}

// toggle takes the row under the cursor in or out.
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
func (p *pickState) toggle() {
	rows := p.visible()
	if len(rows) == 0 {
		return
	}
	option := rows[clamp(p.cursor, 0, len(rows)-1)]
	if at := slices.Index(p.chosen, option.id); at >= 0 {
		p.chosen = append(p.chosen[:at:at], p.chosen[at+1:]...)
		return
	}
	if option.refusal != nil {
		return
	}
	if p.slots > 0 && len(p.chosen) >= p.slots {
		return
	}
	p.chosen = append(p.chosen, option.id)
}

// room is how many rows the list has, measured from the window in hand.
//
// The count was wrong the first time it was written and the screen said so: the
// body has m.height-4 lines to draw in, and this screen spends **seven** of
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
func (p *pickState) room(m model) int {
	spent := 7
	if p.typed != nil {
		spent += 2
	}
	if len(p.groups) > 0 {
		spent++
	}
	room := m.height - 4 - spent
	if room < 3 {
		return 3
	}
	return room
}

// window is the slice of a list to draw, keeping the cursor inside it.
//
// It scrolls by the least it can: the window only moves when the cursor would
// leave it, so stepping back and forth across one boundary does not make the
// whole list jump about.
func window(total, cursor, room int) (from, to int) {
	if room >= total {
		return 0, total
	}
	from = cursor - room/2
	if from < 0 {
		from = 0
	}
	if from+room > total {
		from = total - room
	}
	return from, from + room
}

// clip shortens a line to a number of cells, keeping the front, which is where
// the id and the element are.
//
// Shortened rather than wrapped, for the reason frame clips: a wrapped row
// pushes every row under it down by one, which is how the footer leaves the
// bottom of the screen.
func clip(text string, room int) string {
	if room < 1 {
		return ""
	}
	if lipgloss.Width(text) <= room {
		return text
	}
	letters := []rune(text)
	for len(letters) > 0 && lipgloss.Width(string(letters)+ellipsis) > room {
		letters = letters[:len(letters)-1]
	}
	return string(letters) + ellipsis
}

// idColumn is the column the ids sit in, measured from the ids being drawn.
func (p *pickState) idColumn() int {
	widest := 0
	for _, option := range p.options {
		if width := lipgloss.Width(option.id); width > widest {
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
// pickPassives". Those two are proxies: they hold on the shipped books today and
// stop holding the day a Vietnamese list has an empty column — a trait book with
// an unnamed trait — or an English one gains a detail. A proxy that stops
// holding draws the wrong table and fails nothing.
//
// The scope is the **visible** rows rather than the window, because the filter
// chooses which list is on screen (and the "showing" line says so) while the
// scroll position is only where in that list a reader has got to: a
// window-scoped reading would make the column appear and disappear as the
// cursor walked. The width itself still comes from idColumn, which measures
// every option, so filtering cannot shift the ids sideways either.
//
// Emptiness is measured with lipgloss.Width and not against "", because a detail
// is styled before it is returned and an empty cell rendered through a style is
// escape codes around nothing.
func (p *pickState) detailColumn(m model, rows []pickOption) int {
	for _, option := range rows {
		if lipgloss.Width(p.detail(m, option.id)) > 0 {
			return p.idColumn()
		}
	}
	return 0
}

// detail is what a row says about its option, beyond the id.
//
// For a skill that is who may carry it, in the same words the listing and a
// refusal use. For the three allowlists it is the id's Vietnamese name, which is
// nothing at all in English — there, an id is shown as the data writes it, and
// detailColumn drops the whole column rather than drawing a row of blanks.
func (p *pickState) detail(m model, id string) string {
	if p.kind == pickStatuses {
		// A status's facts, glossed name first: "trúng độc · dot · 3 lượt ·
		// tối đa 3 lớp". The category and the numbers are what tell a poison
		// from a burn, and the ticking note is the one fact that changes what a
		// skill applying it is worth.
		for _, kind := range m.lib.StatusBook() {
			if kind.ID != id {
				continue
			}
			// The wording stands in for the category rather than joining it:
			// "dot" is the category's own id and "damages every turn" is what it
			// means, so printing both prints it twice — and the row is long
			// enough that the second copy is what gets clipped.
			sort := m.lang.StatusCategory(kind.Category)
			facts := m.text(i18n.StatusDetail, sort, kind.Duration, kind.MaxStacks)
			if name := m.lang.Gloss(id); name != "" {
				facts = name + " · " + facts
			}
			return m.style.dim.Render(facts)
		}
		return ""
	}
	if p.kind == pickSpecies {
		// The authored name through Lang.SpeciesName, which is the same reading
		// the species listing gives the same book — and not m.lang.Gloss, since
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
		if kind, known := m.lib.Species().Get(id); known {
			return m.style.dim.Render(m.lang.SpeciesName(kind))
		}
		return ""
	}
	if p.kind == pickOrigins {
		// The authored title, raw, and that is not the species branch's mistake
		// repeated: a work's title is a proper noun of the work — "Pokémon",
		// "Naruto" — so it is the same word in either language, and there is no
		// Lang accessor being gone round. An origin picker therefore keeps its
		// column in English, where the species one drops its.
		if work, known := m.lib.Origins().Get(id); known {
			return m.style.dim.Render(work.Title)
		}
		return ""
	}
	if p.kind == pickPassives {
		held, err := m.lib.Passives().Lookup(id)
		if err != nil {
			// Unreachable for the reason the skill branch below is: a learnset
			// names traits the passive book has already been parsed against.
			// Written down rather than ignored, since the alternative is the
			// blank cell this kind was added to stop being a red one.
			return m.style.bad.Render(m.lang.Error(err))
		}
		// The trait's own name, which is what the listing puts beside an id and
		// what GlossedPassives gives a kit — authored in the passive book, in
		// the one language the data is written in. So an English row is the
		// bare id, exactly as the listing drops its gloss column there, and ?
		// is what an English reader gets the sentences from.
		return m.style.dim.Render(m.lang.PassiveName(held))
	}
	if p.kind != pickSkills {
		return m.style.dim.Render(m.lang.Gloss(id))
	}
	carried, err := m.lib.Skills().Lookup(id)
	if err != nil {
		// Unreachable through the library the options were built from, and
		// written down rather than ignored: an id the book has lost is a bug
		// worth seeing on the row it belongs to.
		return m.style.bad.Render(m.lang.Error(err))
	}
	return m.lang.WhoMaySummary(carried)
}

// refusalUnderCursor is the refusal on the row the cursor is on, or nothing when
// the filter has left no rows at all.
func refusalUnderCursor(rows []pickOption, cursor int) error {
	if len(rows) == 0 {
		return nil
	}
	return rows[clamp(cursor, 0, len(rows)-1)].refusal
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
func (p *pickState) counted(m model) string {
	if p.slots <= 0 {
		return m.style.dim.Render(m.text(i18n.ChoicePosition, len(p.chosen), len(p.options)))
	}
	style := m.style.dim
	if len(p.chosen) > p.slots {
		style = m.style.bad
	}
	return style.Render(m.text(i18n.ChoiceSlots, len(p.chosen), p.slots))
}

func (p *pickState) view(m model) (string, string) {
	if p.reading {
		return p.viewReading(m)
	}
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(p.title)) + "  " + p.counted(m) + "\n")
	out.WriteString(m.style.dim.Render(m.text(p.hint)) + "\n")
	// The filter in force, on its own line and only where there is one. It names
	// both counts for the reason the browser's does: "showing fixture-anime" says
	// nothing about how much of the list is hidden.
	rows := p.visible()
	if len(p.groups) > 0 {
		out.WriteString(m.style.dim.Render(m.text(i18n.PickerShowing,
			p.groupName(m), len(rows), len(p.options))) + "\n")
	}
	out.WriteString("\n")
	if len(p.options) == 0 {
		out.WriteString("  " + m.text(i18n.PickerNothingToPick) + "\n")
		return out.String(), m.text(p.footer)
	}
	if len(rows) == 0 {
		// A filter that hides everything is a state an author has to be able to
		// see and step out of, not an empty screen.
		out.WriteString("  " + m.text(i18n.PickerNothingInGroup) + "\n")
	}

	column := p.detailColumn(m, rows)
	from, to := window(len(rows), p.cursor, p.room(m))
	for index := from; index < to; index++ {
		option := rows[index]
		marker := "  "
		if index == p.cursor {
			marker = "> "
		}
		order := ""
		if at := slices.Index(p.chosen, option.id); at >= 0 {
			order = strconv.Itoa(at + 1)
		}
		// The state is characters and not colour: a chosen row carries its
		// position in the kit and an unavailable one carries a mark, so the
		// screen reads on a monochrome terminal and through a recording that
		// lost its escape codes.
		flag := " "
		if option.refusal != nil {
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
		name := option.id
		if column > 0 {
			name = pad(option.id, column)
		}
		if index == p.cursor {
			name = m.style.selected.Render(name)
		} else if option.refusal != nil {
			name = m.style.dim.Render(name)
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
			// minWidth is the width this program promises to draw in, not a
			// ceiling on what it may spend, and this cell is data — a gloss, a
			// species name, an allowlist note. A restriction cut to "để dành
			// cho loài dr…" is a row that stopped saying which species it is
			// for, on a terminal with a hundred spare columns beside it. Prose
			// still wraps at the floor; a table cell is read by scanning down
			// it, so width is the one thing it can always use.
			room := m.usableWidth() - 1 - lipgloss.Width(marker+state) - column - 1
			detail := clip(p.detail(m, option.id), room)
			if option.refusal != nil {
				detail = m.style.dim.Render(detail)
			}
			row += " " + detail
		}
		out.WriteString(row + "\n")
	}

	out.WriteString("\n")
	chosen := m.style.dim.Render(m.text(i18n.PickerNothingChosen))
	if len(p.chosen) > 0 {
		chosen = strings.Join(p.chosen, " ")
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
	// wider clip never reaches it, so it stays where it is rather than being
	// split into a room of its own.
	out.WriteString("  " + clip(chosen, m.usableWidth()-3) + "\n")
	// The whole refusal for the row under the cursor, which is where there is
	// room for a sentence. It is drawn before anything is pressed, so a space
	// that cannot take the row in has already explained itself.
	if refusal := refusalUnderCursor(rows, p.cursor); refusal != nil {
		out.WriteString(m.style.bad.Render("  "+clip(m.lang.Error(refusal), minWidth-3)) + "\n")
	}
	// The one answer typed beside the list, under it and named, with its reading
	// beside it — the same reading the form gives the field this picker writes
	// into, because 300 is not a chance anybody reads as one.
	if p.typed != nil {
		// The reading is of what the field will contribute, which is the default
		// while nothing has been typed — the same value enter would write, so the
		// percentage cannot describe a chance the write does not use.
		answer := strings.TrimSpace(p.typed.Value())
		if answer == "" {
			answer = p.typed.Placeholder
		}
		reading := ""
		if permille, err := strconv.Atoi(answer); err == nil {
			reading = "  " + m.style.dim.Render(forge.Percent(permille))
		}
		out.WriteString("\n  " + m.style.label.Render(m.text(p.label)) + "  " +
			p.typed.View() + reading)
	}
	return out.String(), m.text(p.footer)
}

// viewReading is the description of the row under the cursor, over the list
// rather than beside it.
//
// Over it for the reason the list itself covers the form that raised it: the
// sentences run to more lines than an eighty-by-twenty-four window has beside
// thirteen rows, and a trait's already wrap past that on their own. It scrolls
// for the same reason blurbScreen does, and shares that screen's room: the two
// spend their lines identically — a heading, a blank, the sentences, and the
// line saying how much is left.
func (p *pickState) viewReading(m model) (string, string) {
	footer := m.text(i18n.PickerReadingFooter)
	rows := p.visible()
	if len(rows) == 0 {
		return "  " + m.text(i18n.PickerNothingToPick) + "\n", footer
	}
	cursor := clamp(p.cursor, 0, len(rows)-1)
	name, body := p.described(m, rows[cursor].id)

	var out strings.Builder
	out.WriteString(m.style.heading.Render(name) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition, cursor+1, len(rows))) + "\n\n")
	room := traitRoom(m)
	// Clamped here rather than where it is incremented, because the key that
	// moves it does not know how long the answer is: the answer belongs to the
	// row under the cursor, and that can have moved since.
	scroll := clamp(p.scroll, 0, max(len(body)-room, 0))
	for _, line := range body[scroll:min(scroll+room, len(body))] {
		out.WriteString(line + "\n")
	}
	if len(body) > room {
		out.WriteString(m.style.dim.Render("  " + m.text(i18n.BlurbMore,
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
func (p *pickState) described(m model, id string) (string, []string) {
	if p.kind == pickPassives {
		held, err := m.lib.Passives().Lookup(id)
		if err != nil {
			return id, []string{"  " + m.style.bad.Render(m.lang.Error(err))}
		}
		return m.lang.GlossedPassive(held), traitSentences(m, held)
	}
	declared, err := m.lib.Skills().Lookup(id)
	if err != nil {
		return id, []string{"  " + m.style.bad.Render(m.lang.Error(err))}
	}
	return m.lang.GlossedSkill(declared), skillLines(m, declared)
}

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
	// pickElements, pickArchetypes and pickCharacters are a restriction's three
	// allowlists, whose rows are ids and their Vietnamese names.
	pickElements
	pickArchetypes
	pickCharacters
	// pickStatuses is what a skill inflicts, whose rows carry each status's own
	// facts — how long it lasts, how far it stacks, whether it ticks — because
	// choosing between "mire" and "expose" on the ids alone is not choosing.
	pickStatuses
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
	// apply hands the answer back to whoever raised the picker.
	apply func(model, pickAnswer) model
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
		state.footer = i18n.PickerFooter
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
	switch message.String() {
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

// toggle takes the row under the cursor in or out.
//
// Taking one out always works, including one that has become unavailable since
// it was chosen — that is the state an author most needs to be able to fix.
// Taking one in is refused when the carrier may not have it, and nothing is said
// about the refusal here because it is already on screen: the row carries its
// mark and the line under the list carries the whole sentence, before space is
// ever pressed. A message raised by the keypress would be the same words twice.
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

// detail is what a row says about its option, beyond the id.
//
// For a skill that is who may carry it, in the same words the listing and a
// refusal use. For the three allowlists it is the id's Vietnamese name, which is
// nothing at all in English — there, an id is shown as the data writes it.
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

func (p *pickState) view(m model) (string, string) {
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(p.title)) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition, len(p.chosen), len(p.options))) + "\n")
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

	column := p.idColumn()
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
		name := pad(option.id, column)
		if index == p.cursor {
			name = m.style.selected.Render(name)
		} else if option.refusal != nil {
			name = m.style.dim.Render(name)
		}
		// Every row says who its option is for, in the same words the listing
		// and a refusal use, whether or not this character qualifies. The mark
		// is what says "not you"; the words are what say why, and they are worth
		// having on a row that is available too.
		room := minWidth - 1 - lipgloss.Width(marker+state) - column - 1
		detail := clip(p.detail(m, option.id), room)
		if option.refusal != nil {
			detail = m.style.dim.Render(detail)
		}
		out.WriteString(marker + state + name + " " + detail + "\n")
	}

	out.WriteString("\n")
	chosen := m.style.dim.Render(m.text(i18n.PickerNothingChosen))
	if len(p.chosen) > 0 {
		chosen = strings.Join(p.chosen, " ")
	}
	out.WriteString("  " + clip(chosen, minWidth-3) + "\n")
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

package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
// # Why one picker for four different lists
//
// The kit is chosen from the skill book; a restriction's three allowlists are
// chosen from the element, archetype and cast books. All four are "pick several
// from a list, keep the order", and one implementation is one set of keys to
// learn. What differs is only how a row's detail reads, which is what pickKind
// selects, and the detail is worded at draw time rather than stored — a stored
// sentence would still be in the old language after ctrl+l.

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
}

// pickState is a picker while it is open.
type pickState struct {
	title   i18n.Key
	kind    pickKind
	options []pickOption
	// chosen is the answer, in the order it was chosen, because for a kit that
	// order *is* the kit.
	chosen []string
	cursor int
	// apply hands the answer back to whoever raised the picker.
	apply func(model, []string) model
}

// pick raises a picker. Nothing is applied until it is closed with enter.
func (m model) pick(state *pickState) model {
	state.cursor = clamp(state.cursor, 0, len(state.options)-1)
	m.picker = state
	return m
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

func (p *pickState) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		m.picker = nil
		return m, nil
	case "enter":
		apply, chosen := p.apply, append([]string(nil), p.chosen...)
		m.picker = nil
		if apply != nil {
			m = apply(m, chosen)
		}
		return m, nil
	case "up", "k":
		p.cursor = clamp(p.cursor-1, 0, len(p.options)-1)
	case "down", "j":
		p.cursor = clamp(p.cursor+1, 0, len(p.options)-1)
	case " ":
		p.toggle()
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
	if len(p.options) == 0 {
		return
	}
	option := p.options[clamp(p.cursor, 0, len(p.options)-1)]
	if at := slices.Index(p.chosen, option.id); at >= 0 {
		p.chosen = append(p.chosen[:at:at], p.chosen[at+1:]...)
		return
	}
	if option.refusal != nil {
		return
	}
	p.chosen = append(p.chosen, option.id)
}

// pickerRoom is how many rows the list has, measured from the window in hand.
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
func pickerRoom(m model) int {
	const spent = 7
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

func (p *pickState) view(m model) (string, string) {
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(p.title)) + "  " +
		m.style.dim.Render(m.text(i18n.ChoicePosition, len(p.chosen), len(p.options))) + "\n")
	out.WriteString(m.style.dim.Render(m.text(i18n.PickerHint)) + "\n\n")
	if len(p.options) == 0 {
		out.WriteString("  " + m.text(i18n.PickerNothingToPick) + "\n")
		return out.String(), m.text(i18n.PickerFooter)
	}

	column := p.idColumn()
	from, to := window(len(p.options), p.cursor, pickerRoom(m))
	for index := from; index < to; index++ {
		option := p.options[index]
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
	if refusal := p.options[clamp(p.cursor, 0, len(p.options)-1)].refusal; refusal != nil {
		out.WriteString(m.style.bad.Render("  "+clip(m.lang.Error(refusal), minWidth-3)) + "\n")
	}
	return out.String(), m.text(i18n.PickerFooter)
}

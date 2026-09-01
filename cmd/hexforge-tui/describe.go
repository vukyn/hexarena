package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/progression"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// The two describers — the blurb and the art preview — are not screens in the
// sense the others are: they describe whatever raised them. This file is the
// half of that which belongs to the **client**, and the split is the whole point
// of the step it arrived in.
//
// ⚠️ **They used to reach backwards.** blurb.go read m.browse.level eight times,
// m.skills.cursor five, m.browse.cursor five and m.play four ways; preview.go
// read m.browse three ways. A screen that pulls from three other screens cannot
// move into internal/screen and cannot be drawn by a client whose screens are
// different ones — so the direction is inverted: the raiser builds a
// draw.Subject and pushes it, and the describer draws what it was handed.
//
// What lives here is therefore everything a describer may not know: which of
// this client's screens a subject belongs on, and which raiser an arrow key
// walks while a describer is in front. The describers themselves are in
// blurb.go and preview.go and read nothing but a draw.Context.

// subjects is what a draw.Subject means to this client: which of its screens is
// handed the subject, and what landing on it costs.
//
// A map keyed by kind and read by key, never ranged over into anything that
// reaches a screen — the same discipline internal/core holds about map order, and
// the same shape raiseTargets already has one field over.
//
// ⚠️ It has to be **total**: a kind with no entry here makes a raise hand the
// describer nothing, which draws an empty screen off a keystroke that did
// everything right. That is the failure `Action.Focus` shipped with — it answered
// only the statuses reference, and a Focus aimed anywhere else declined the whole
// trip in silence, one class weaker than this because nothing counted the cases.
// TestEverySubjectKindIsAppliedByThisClient walks screen.SubjectKindCount rather
// than this map, so a kind added over there fails here instead of going quiet.
var subjects = map[draw.SubjectKind]func(model, draw.Subject) (model, bool){
	draw.StatusSubject:    landStatus,
	draw.SkillSubject:     landSkill,
	draw.CharacterSubject: landCharacter,
}

// landStatus puts the statuses reference's cursor on the status a raise named.
//
// It reports rather than settling for the nearest, and raise declines the whole
// trip when it fails: a trait naming a status the book has lost is a trait
// already printing a bare id, and a cursor moved to whatever sorted next would
// answer a question nobody asked. This is the only applier that can say no.
func landStatus(m model, subject draw.Subject) (model, bool) {
	statuses, found := m.statuses.Focus(subject.ID)
	if !found {
		return m, false
	}
	m.statuses = statuses
	return m, true
}

// landSkill hands the description screen a skill to describe.
func landSkill(m model, subject draw.Subject) (model, bool) {
	m.blurb.Subject = subject
	return m, true
}

// landCharacter hands **both** describers of a character the same subject.
//
// ⚠️ Two rather than one, and deliberately: the traits blurb and the art preview
// are two readings of one thing — a character at a level — so the subject names
// the character and the Target names which reading was asked for. Writing only
// the screen the raise is aimed at would need this applier to be told the target,
// which is the map above answering a question the map beside it already answers.
// Writing the other one costs a field on a value that is copied every keystroke
// anyway, and it is what keeps the two from disagreeing about who is in front.
func landCharacter(m model, subject draw.Subject) (model, bool) {
	m.blurb.Subject = subject
	m.preview.Subject = subject
	return m, true
}

// applySubject lands whatever a raise was about, and says whether it could.
//
// NoSubject is handled here rather than in the map for the reason NoTarget is
// kept out of raiseTargets: it is what every raise that is about nothing carries,
// so an entry for it would be an entry every Quit and every Back could reach.
func (m model) applySubject(subject draw.Subject) (model, bool) {
	if subject.Kind == draw.NoSubject {
		return m, true
	}
	land, known := subjects[subject.Kind]
	if !known {
		return m, false
	}
	return land(m, subject)
}

// hand is the route into that applier for a raiser that has not been converted
// to return a draw.Action yet.
//
// The three screens that raise the blurb and the one that raises the preview
// still write m.screen themselves — converting them is the next step — so they
// push their subject through here and set the screen as they always did. What
// matters is the direction: they push, and the describer no longer pulls.
//
// The bool is dropped because none of the three subjects they build can be
// declined: only a status has somewhere to fail to be found, and none of them
// names one.
func (m model) hand(subject draw.Subject) model {
	handed, landed := m.applySubject(subject)
	if !landed {
		return m
	}
	return handed
}

// updateBlurb routes a keystroke while the description screen is in front.
//
// ⚠️ **The raiser is walked from here rather than from the describer**, which is
// the client half of the inversion this file exists for. An author reading one
// description can read the next without going back and forth, and that has to
// move the listing behind — so the cursor stays in one place, the screen it
// belongs to, and the new subject is pushed the way the raising key pushed the
// first one.
//
// The two key sets are disjoint on purpose: the describer owns leaving and
// scrolling, the raiser owns everything that changes *what* is described.
func (m model) updateBlurb(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	b := m.blurb
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "?":
		m.screen = b.from
		// ⚠️ And the raise is forgotten as it is used, exactly as navigate does
		// with its own Back. This describer answers esc itself rather than
		// through navigate, so nothing else clears the slot — and the browser's
		// esc is a draw.Back now, so a record left behind would send the browser
		// back to the browser.
		m.raisedFrom = screenMenu
		return m, nil
	}
	if b.from == screenPlay {
		// The option behind moves from here too, the way the listing's cursor
		// does, so four options can be read one after another without going back
		// and forth. It goes through playScreen.move, which is the one place that
		// knows an unavailable option is stepped over.
		//
		// While aiming they do nothing: the skill is settled and the cell is
		// what is being chosen, so walking the options would change what is
		// described out from under a decision already half taken.
		if m.play.pending != nil && !m.play.aiming {
			switch message.String() {
			case "up", "k":
				m.play = m.play.move(-1)
			case "down", "j":
				m.play = m.play.move(1)
			}
		}
		m.blurb.Subject = m.play.subject()
		return m, nil
	}
	if b.from == screenBrowse {
		rows := len(m.browse.Rows())
		switch message.String() {
		// [ and ] alias the page keys, here and at the other two sites that
		// scroll, because a compact keyboard has neither PgUp nor PgDn and the
		// trait sentences are exactly what a window this size cannot finish. It
		// is safe to take a printable key here: this branch feeds no text field,
		// which is what makes the alias free rather than a letter stolen from
		// something being typed.
		case "pgdown", "]":
			b.Scroll++
			m.blurb = b
			return m, nil
		case "pgup", "[":
			b.Scroll = max(b.Scroll-1, 0)
			m.blurb = b
			return m, nil
		case "up", "k":
			m.browse.Cursor = clamp(m.browse.Cursor-1, 0, rows-1)
		case "down", "j":
			m.browse.Cursor = clamp(m.browse.Cursor+1, 0, rows-1)
		case "left", "h":
			m.browse.Level = clamp(m.browse.Level-1, 1, progression.LevelCap)
		case "right", "l":
			m.browse.Level = clamp(m.browse.Level+1, 1, progression.LevelCap)
		case "home":
			m.browse.Level = 1
		case "end":
			m.browse.Level = progression.LevelCap
		}
		// Anything that changed which character or which level is in front
		// changed the answer, so the offset into the old one means nothing.
		b.Scroll = 0
		m.blurb = b
		return m.hand(m.browse.Subject()), nil
	}
	// Through the listing's own visible rows, not its book: the cursor counts what
	// the filter left, so stepping it against the full book would walk past the
	// end of the narrowed listing and describe a skill the screen behind is not
	// pointing at. skillsScreen.rows is the one funnel — see it for why.
	rows := len(m.skills.rows())
	switch message.String() {
	case "up", "k":
		m.skills.cursor = clamp(m.skills.cursor-1, 0, rows-1)
	case "down", "j":
		m.skills.cursor = clamp(m.skills.cursor+1, 0, rows-1)
	}
	m.blurb.Subject = m.skills.subject()
	return m, nil
}

// updatePreview routes a keystroke while the art preview is in front, and walks
// the browser behind it for the reason updateBlurb does.
func (m model) updatePreview(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "p":
		m.screen = screenBrowse
		// Forgotten as it is used, for the reason updateBlurb's esc forgets it.
		m.raisedFrom = screenMenu
		return m, nil
	case "left", "h":
		m.browse.Level = clamp(m.browse.Level-1, 1, progression.LevelCap)
	case "right", "l":
		m.browse.Level = clamp(m.browse.Level+1, 1, progression.LevelCap)
	case "home":
		m.browse.Level = 1
	case "end":
		m.browse.Level = progression.LevelCap
	}
	return m.hand(m.browse.Subject()), nil
}

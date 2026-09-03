package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/progression"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// The two describers — the blurb and the art preview — are not screens in the
// sense the others are: they describe whatever raised them. This file is the
// half of that which belongs to the **client**, and it is the half no screen in
// internal/screen may hold: which of *this client's* screens a subject lands
// on, and which raiser an arrow key walks while a describer is in front.
//
// It is cmd/hexforge-tui/describe.go's own shape, and that is the point rather
// than a copy: the describers themselves are one body in internal/screen, drawn
// by both clients, and what each client owes them is a subject and a way back.
// The two appliers differ in exactly one entry — a squad, which one client turns
// into a measurement and the other into a battle.

// subjects is what a draw.Subject means to this client: which of its screens is
// handed the subject, and what landing on it costs.
//
// A map keyed by kind and read by key, never ranged over into anything that
// reaches a screen — the same discipline internal/core holds about map order,
// one layer up.
//
// ⚠️ It has to be **total**: a kind with no entry makes a raise hand the
// describer nothing, which draws an empty screen off a keystroke that did
// everything right. TestEverySubjectKindIsAppliedByThisClient walks
// screen.SubjectKindCount rather than this map, so a kind added over there fails
// here instead of going quiet.
var subjects = map[draw.SubjectKind]func(model, draw.Subject) (model, bool){
	draw.StatusSubject:    landStatus,
	draw.SkillSubject:     landSkill,
	draw.CharacterSubject: landCharacter,
	draw.SquadSubject:     landSquad,
}

// landStatus puts the statuses reference's cursor on the status a raise named.
//
// It reports rather than settling for the nearest, and raise declines the whole
// trip when it fails: a trait naming a status the book has lost is a trait
// already printing a bare id, and a cursor moved to whatever sorted next would
// answer a question nobody asked.
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
// Two rather than one, deliberately: the traits blurb and the art preview are
// two readings of one thing — a character at a level — so the subject names the
// character and the Target names which reading was asked for. Writing only the
// screen the raise is aimed at would need this applier to be told the target,
// which is the map above answering a question raiseTargets already answers.
func landCharacter(m model, subject draw.Subject) (model, bool) {
	m.blurb.Subject = subject
	m.preview.Subject = subject
	return m, true
}

// landSquad records which squad the reader is taking into a battle.
//
// ⚠️ **This is the entry the two clients answer differently, and it is the seam
// the network work replaces.** The squad catalogue lives in internal/screen and
// its `f` raises draw.Fight about a squad by id; in the authoring tool that id
// becomes the home side of a **measurement** — two squads, both ways round, a
// win rate — and here it becomes the side the player fights *with*. Neither
// meaning could be written into the catalogue, which is exactly what a Target and
// a Subject are for: a raise names what it wants and asks nobody what the client
// in front will make of it.
//
// It reports, like landStatus, and a raise that cannot land declines the whole
// trip. Nothing a keystroke produces reaches that — the catalogue writes itself
// back onto the model before navigate is called, so the id it named is on the
// list this reads — so a squad the catalogue does not hold is a fault rather than
// a state, and opening a battle on whichever row `taking` happened to hold would
// answer a question nobody asked.
func landSquad(m model, subject draw.Subject) (model, bool) {
	for index, squad := range m.squads.Saved {
		if squad.ID == subject.ID {
			m.taking = index
			return m, true
		}
	}
	return m, false
}

// applySubject lands whatever a raise was about, and says whether it could.
//
// NoSubject is handled here rather than in the map for the reason NoTarget is
// kept out of raiseTargets: it is what every raise about nothing carries, so an
// entry for it would be an entry every Quit and every Back could reach.
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

// hand is the route into that applier for a raiser that is already in front:
// the arrow keys below change *what* is described without navigating anywhere.
//
// The bool is dropped because neither describer's raiser can build a subject
// that declines — only a status has somewhere to fail to be found, and neither
// the browser nor the battle names one.
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
// the client half of the inversion this file exists for. A reader reading one
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
		// ⚠️ Through goBack, exactly as navigate's own Back is, so nothing else
		// clears the slot — and it is what restores the way back of a screen that
		// was raised and then raised this. See model.raisedOver: the battle is that
		// screen.
		return m.goBack(), nil
	}
	if m.raisedFrom == screenBattle {
		// The option behind moves from here too, the way the listing's cursor
		// does, so four options can be read one after another. It goes through
		// draw.PlayScreen.Move, which is the one place that knows an unavailable
		// option is stepped over.
		//
		// While aiming they do nothing: the skill is settled and the cell is what
		// is being chosen, so walking the options would change what is described
		// out from under a decision already half taken.
		if m.battle.Pending != nil && !m.battle.Aiming {
			switch message.String() {
			case "up", "k":
				m.battle = m.battle.Move(-1)
			case "down", "j":
				m.battle = m.battle.Move(1)
			}
		}
		m.blurb.Subject = m.battle.Subject()
		return m, nil
	}
	if m.raisedFrom == screenCast {
		rows := len(m.cast.Rows())
		switch message.String() {
		// [ and ] alias the page keys, here and at the other two sites that
		// scroll, because a compact keyboard has neither PgUp nor PgDn and five
		// traits at the cap are exactly what a window this size cannot finish. It
		// is safe to take a printable key here: this branch feeds no text field.
		case "pgdown", "]":
			b.Scroll++
			m.blurb = b
			return m, nil
		case "pgup", "[":
			b.Scroll = max(b.Scroll-1, 0)
			m.blurb = b
			return m, nil
		// The arm of a fork, walked here for the reason the level is: this screen
		// keeps no cursor and no level of its own, so what a key over it moves is
		// the browser behind it. Through CycleForm rather than by writing the field,
		// because which arms there are is the browser's own question and a second
		// answer to it is how two screens come to disagree about what a fork is.
		case "s":
			m.cast = m.cast.CycleForm()
		case "up", "k":
			m.cast.Cursor = draw.Clamp(m.cast.Cursor-1, 0, rows-1)
		case "down", "j":
			m.cast.Cursor = draw.Clamp(m.cast.Cursor+1, 0, rows-1)
		case "left", "h":
			m.cast.Level = draw.Clamp(m.cast.Level-1, 1, progression.LevelCap)
		case "right", "l":
			m.cast.Level = draw.Clamp(m.cast.Level+1, 1, progression.LevelCap)
		case "home":
			m.cast.Level = 1
		case "end":
			m.cast.Level = progression.LevelCap
		}
		// Anything that changed which character or which level is in front changed
		// the answer, so the offset into the old one means nothing.
		b.Scroll = 0
		m.blurb = b
		return m.hand(m.cast.Subject()), nil
	}
	// Through the listing's own visible rows, not its book: the cursor counts what
	// the filter left, so stepping it against the full book would walk past the
	// end of the narrowed listing and describe a skill the screen behind is not
	// pointing at. draw.SkillsScreen.Rows is the one funnel.
	rows := len(m.skills.Rows())
	switch message.String() {
	case "up", "k":
		m.skills.Cursor = draw.Clamp(m.skills.Cursor-1, 0, rows-1)
	case "down", "j":
		m.skills.Cursor = draw.Clamp(m.skills.Cursor+1, 0, rows-1)
	}
	m.blurb.Subject = m.skills.Subject()
	return m, nil
}

// updatePreview routes a keystroke while the art preview is in front, and walks
// the browser behind it for the reason updateBlurb does.
func (m model) updatePreview(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "p":
		// Through goBack, for the reason updateBlurb's esc goes through it. The
		// browser is the only thing that raises this, so what it follows is the
		// browser — read rather than written down, which is the whole point of the
		// client keeping the record.
		return m.goBack(), nil
	case "left", "h":
		m.cast.Level = draw.Clamp(m.cast.Level-1, 1, progression.LevelCap)
	case "right", "l":
		m.cast.Level = draw.Clamp(m.cast.Level+1, 1, progression.LevelCap)
	case "home":
		m.cast.Level = 1
	case "end":
		m.cast.Level = progression.LevelCap
	// The arm of a fork, walked here for the reason the level is: this screen
	// keeps no cursor and no level of its own, so what a key over it moves is
	// the browser behind it. Through CycleForm rather than by writing the field,
	// because which arms there are is the browser's own question and a second
	// answer to it is how two screens come to disagree about what a fork is.
	case "s":
		m.cast = m.cast.CycleForm()
	}
	return m.hand(m.cast.Subject()), nil
}

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
// m.skills.Cursor five, m.browse.cursor five and m.play four ways; preview.go
// read m.browse three ways. A screen that pulls from three other screens cannot
// move into internal/screen and cannot be drawn by a client whose screens are
// different ones — so the direction is inverted: the raiser builds a
// draw.Subject and pushes it, and the describer draws what it was handed.
//
// ⚠️ **And the describer's own way back is m.raisedFrom now.** The blurb carried
// a `from screen` of its own for three releases, because one of its raisers still
// wrote m.screen itself rather than returning a draw.Raise. All three return one
// now, so navigate records the raiser in the slot it already keeps for a Back,
// and the field is gone. `Subject.Kind` could not have replaced it: #207 measured
// a listed skill and a battle option to be **one** subject and collapsed them
// into SkillSubject, so the subject cannot tell the listing from the battle. The
// raiser can, because it is this client's own enum and this is this client's own
// file.
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
	draw.SquadSubject:     landSquad,
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

// landSquad puts the fight's home side on the squad a raise named.
//
// ⚠️ **The id is turned into a row here, and that is the whole reason the raise
// names an id.** fightScreen.home is an index into the catalogue — the list its
// choosers walk — and an index is a fact about *this client's* two screens
// standing next to each other, which is not something a screen in
// internal/screen may know or should have to.
//
// It reports, like landStatus, and a raise that cannot land declines the whole
// trip. Nothing a keystroke produces reaches that: the catalogue writes itself
// back onto the model before navigate is called, so the id it named is on the
// list this reads. A squad the catalogue does not hold is a fault rather than a
// state, and opening the fight on whichever row happened to be under `home`
// would answer a question nobody asked.
func landSquad(m model, subject draw.Subject) (model, bool) {
	for index, squad := range m.squad.Saved {
		if squad.ID == subject.ID {
			m.fight.home = index
			return m, true
		}
	}
	return m, false
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
		// ⚠️ Through goBack, exactly as navigate's own Back is. This describer
		// answers esc itself rather than through navigate, so nothing else
		// clears the slot — and the browser's esc is a draw.Back now, so a record
		// left behind would send the browser back to the browser. It is also
		// what restores the way back of a screen that was raised and then raised
		// this: see model.raisedOver.
		return m.goBack(), nil
	}
	if m.raisedFrom == screenPlay {
		// The option behind moves from here too, the way the listing's cursor
		// does, so four options can be read one after another without going back
		// and forth. It goes through draw.PlayScreen.Move, which is the one place that
		// knows an unavailable option is stepped over.
		//
		// While aiming they do nothing: the skill is settled and the cell is
		// what is being chosen, so walking the options would change what is
		// described out from under a decision already half taken.
		if m.play.Pending != nil && !m.play.Aiming {
			switch message.String() {
			case "up", "k":
				m.play = m.play.Move(-1)
			case "down", "j":
				m.play = m.play.Move(1)
			}
		}
		m.blurb.Subject = m.play.Subject()
		return m, nil
	}
	if m.raisedFrom == screenBrowse {
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
		// The arm of a fork, walked here for the reason the level is: this screen
		// keeps no cursor and no level of its own, so what a key over it moves is
		// the browser behind it. Through CycleForm rather than by writing the field,
		// because which arms there are is the browser's own question and a second
		// answer to it is how two screens come to disagree about what a fork is.
		case "s":
			m.browse = m.browse.CycleForm()
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
	// pointing at. draw.SkillsScreen.Rows is the one funnel — see it for why.
	rows := len(m.skills.Rows())
	switch message.String() {
	case "up", "k":
		m.skills.Cursor = clamp(m.skills.Cursor-1, 0, rows-1)
	case "down", "j":
		m.skills.Cursor = clamp(m.skills.Cursor+1, 0, rows-1)
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
		// browser — read rather than written down, which is the whole point of
		// the client keeping the record.
		return m.goBack(), nil
	case "left", "h":
		m.browse.Level = clamp(m.browse.Level-1, 1, progression.LevelCap)
	case "right", "l":
		m.browse.Level = clamp(m.browse.Level+1, 1, progression.LevelCap)
	case "home":
		m.browse.Level = 1
	case "end":
		m.browse.Level = progression.LevelCap
	// The arm of a fork, walked here for the reason the level is: this screen
	// keeps no cursor and no level of its own, so what a key over it moves is
	// the browser behind it. Through CycleForm rather than by writing the field,
	// because which arms there are is the browser's own question and a second
	// answer to it is how two screens come to disagree about what a fork is.
	case "s":
		m.browse = m.browse.CycleForm()
	}
	return m.hand(m.browse.Subject()), nil
}

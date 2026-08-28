package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/tui"
)

// playScreen is a battle fought by hand against the opponent the engine plays.
//
// It is raised from the fight for the reason the fight is raised from the
// catalogue: that is where a pairing is already chosen, and a battle wants two
// squads before it wants anything else. What the simulation answers over two
// hundred battles, this answers once — and the two are worth having side by
// side, because a rate says which squad is better and playing one says why.
//
// ⚠️ **This is the one screen holding something the model does not copy.** Every
// other screen is a value the model carries, so a field written while drawing is
// thrown away with the copy; a battle is a pointer, and a mutation reaches every
// copy of the model there is. So the battle is stepped in update and **never**
// touched in view — view reads it and draws, and that division is what keeps a
// redraw from playing a turn.
type playScreen struct {
	// seed is which battle this is. Walking it is how a player asks for another
	// arrangement of the same pairing, and it is the same number the log carries,
	// so a battle played here can be replayed by the game client.
	seed uint64
	// side is the half the player is fighting on: the home squad's, always, so
	// that the squad under the catalogue's cursor is the one being played rather
	// than the one being played against.
	side hex.Side

	fight  *battle.Battle
	tags   map[string]string
	names  map[string]string
	events []battle.Event
	// script is every decision taken, the player's and the engine's alike. It is
	// what undo shortens and what the whole battle is rebuilt from, which is why
	// a rebuild needs nothing else: a battle is a pure function of its seed and
	// the decisions taken.
	script battle.Script
	// pending is the turn waiting on the player. Nil means the battle is between
	// turns, which is where the engine's own units act.
	pending *battle.Prompt

	// option and aim are the two questions a turn asks, and aiming says which of
	// them is in front.
	option int
	aim    int
	aiming bool

	err error
}

// playTurnLimit is where a battle is abandoned, and it is cmd/hexarena's number
// deliberately: a battle this screen calls endless and a battle the game calls
// endless have to be the same battle.
const playTurnLimit = 4000

func newPlayScreen() playScreen {
	return playScreen{seed: 1, side: hex.SideAlly}
}

// begin builds the battle from the pairing in front and runs it up to the
// player's first decision.
func (p playScreen) begin(m model) playScreen {
	p.fight, p.tags, p.names = nil, nil, nil
	p.events, p.script, p.pending = nil, nil, nil
	p.option, p.aim, p.aiming = 0, 0, false
	p.err = nil

	home, away, ok := m.fight.sides(m)
	if !ok {
		return p
	}
	roster, err := home.Take(hex.SideAlly, m.lib.Characters())
	if err != nil {
		p.err = err
		return p
	}
	facing, err := away.Take(hex.SideEnemy, m.lib.Characters())
	if err != nil {
		p.err = err
		return p
	}
	fight, err := battle.New(m.lib.Books(), p.seed, append(roster, facing...))
	if err != nil {
		p.err = err
		return p
	}
	fight.Begin()
	p.fight = fight
	p.tags = tui.Tags(fight.Units())
	p.names = tui.Names(fight.Units())
	p.collect()
	return p.run()
}

// collect drains whatever the battle has recorded since it was last asked.
//
// The event log is the only contract a reader of a battle has, and this screen
// is a reader like any other: a screen that summed its own strikes would be a
// second place where what a battle did is decided.
func (p *playScreen) collect() {
	if p.fight == nil {
		return
	}
	p.events = append(p.events, p.fight.Drain()...)
}

// run takes every turn that is not the player's, and stops on the one that is.
//
// A skipped turn is stepped past rather than shown: a unit that lost its action
// to control has no decision in it, and a screen that stopped there would ask a
// question with no answers.
func (p playScreen) run() playScreen {
	if p.fight == nil {
		return p
	}
	for steps := 0; steps < playTurnLimit; steps++ {
		if p.fight.Finished() {
			p.pending = nil
			return p
		}
		prompt := p.pending
		p.pending = nil
		if prompt == nil {
			opened, err := p.fight.Advance()
			if err != nil {
				p.err = err
				return p
			}
			prompt = opened
		}
		p.collect()
		if prompt.Skipped {
			continue
		}
		unit, known := p.fight.Unit(prompt.Unit)
		if !known {
			// A turn offered to somebody who is not fighting is stepped past
			// rather than reported. It cannot happen — the queue holds units the
			// battle enlisted — and a screen is the wrong place to discover that
			// it did; the step limit above is what stops this being a loop.
			continue
		}
		if unit.Side == p.side {
			p.pending = prompt
			p.option = p.firstAvailable(prompt)
			p.aim, p.aiming = 0, false
			return p
		}
		if err := p.engineOrder(prompt); err != nil {
			p.err = err
			return p
		}
	}
	return p
}

// firstAvailable is the option the cursor starts on: the first the unit may
// actually take, rather than the first declared. A cursor that opened on a skill
// on cooldown would be one press from a refusal on most turns.
func (p playScreen) firstAvailable(prompt *battle.Prompt) int {
	for index, option := range prompt.Options {
		if option.Available() {
			return index
		}
	}
	return 0
}

// engineOrder takes the turn the engine would take, which is how the opponent
// plays and how the "let it pick" key answers.
//
// It lands in the script as a decision like the player's own, because the script
// is what the whole battle is rebuilt from: a half of it that was not written
// down would replay as a different battle.
func (p *playScreen) engineOrder(prompt *battle.Prompt) error {
	choice, acted := p.fight.Suggest(prompt)
	if !acted {
		return p.skip(prompt, battle.NoActionReason)
	}
	return p.take(prompt, choice.Skill, choice.Aim)
}

// take and skip are the two things a turn can be spent on, and they are two
// methods rather than one taking a decision so that a decision with a skill and
// no aim — which the engine would refuse and nothing here should be able to
// build — cannot be written at all.
func (p *playScreen) take(prompt *battle.Prompt, skill string, aim hex.Offset) error {
	if err := p.fight.Act(skill, aim); err != nil {
		return err
	}
	return p.record(battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Skill: skill, Aim: hex.At(aim),
	})
}

func (p *playScreen) skip(prompt *battle.Prompt, reason string) error {
	decision := battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn, Passed: true, Reason: reason,
	}
	if err := p.fight.Pass(decision.PassReason()); err != nil {
		return err
	}
	return p.record(decision)
}

func (p *playScreen) record(decision battle.Decision) error {
	p.script = append(p.script, decision)
	p.collect()
	return nil
}

// rewind rebuilds the battle from the seed and a shortened script.
//
// It is the whole of undo, and it works because a battle is a pure function of
// its seed and the decisions taken: there is no state to unwind, only a shorter
// list to replay. That is the same property the log's --verify rests on.
func (p playScreen) rewind(m model, script battle.Script) playScreen {
	seed, side := p.seed, p.side
	fresh := newPlayScreen()
	fresh.seed, fresh.side = seed, side
	fresh = fresh.begin(m)
	if fresh.fight == nil || fresh.err != nil {
		return fresh
	}
	replayed, pending, err := fresh.fight.Replay(script, playTurnLimit, nil)
	if err != nil {
		fresh.err = err
		return fresh
	}
	fresh.script = replayed
	fresh.pending = pending
	fresh.events = fresh.fight.Drain()
	return fresh.run()
}

// undo drops the player's last decision and everything that followed it.
func (p playScreen) undo(m model) playScreen {
	cut := -1
	for index := len(p.script) - 1; index >= 0; index-- {
		unit, known := p.fight.Unit(p.script[index].Unit)
		if known && unit.Side == p.side {
			cut = index
			break
		}
	}
	if cut < 0 {
		return p
	}
	shortened := make(battle.Script, cut)
	copy(shortened, p.script[:cut])
	return p.rewind(m, shortened)
}

func (p playScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		if p.aiming {
			p.aiming = false
			m.play = p
			return m, nil
		}
		m.screen = screenFight
		return m, nil
	case "n":
		p.seed++
		p = p.begin(m)
	case "u":
		if p.fight != nil {
			p = p.undo(m)
		}
	}
	if p.pending == nil {
		m.play = p
		return m, nil
	}
	switch message.String() {
	case "up", "k":
		p = p.move(-1)
	case "down", "j":
		p = p.move(1)
	case "enter", "space":
		p = p.choose(m)
	case "a":
		// The engine's own answer, taken as the player's: somebody who wants to
		// see what it would do here should not have to guess it, and it lands in
		// the script as a decision like any other.
		if err := p.engineOrder(p.pending); err != nil {
			p.err = err
		} else {
			p.pending = nil
			p = p.run()
		}
	case "p":
		if err := p.skip(p.pending, ""); err != nil {
			p.err = err
		} else {
			p.pending = nil
			p = p.run()
		}
	}
	m.play = p
	return m, nil
}

// move walks whichever of the two lists is in front.
func (p playScreen) move(by int) playScreen {
	if p.aiming {
		aims := p.pending.Options[p.option].Aims
		p.aim = (p.aim + by + len(aims)) % len(aims)
		return p
	}
	// Unavailable options are stepped over rather than hidden: a player deciding
	// what to do needs to know a skill exists and is two turns away, and a cursor
	// that could rest on one would be a cursor whose enter does nothing.
	for step := 1; step <= len(p.pending.Options); step++ {
		next := (p.option + by*step + len(p.pending.Options)*len(p.pending.Options)) % len(p.pending.Options)
		if p.pending.Options[next].Available() {
			p.option = next
			return p
		}
	}
	return p
}

// choose answers whichever question is in front: which skill, and then where.
//
// A skill with one legal cell does not ask the second question, because a
// question with one answer is not a decision.
func (p playScreen) choose(m model) playScreen {
	option := p.pending.Options[clamp(p.option, 0, len(p.pending.Options)-1)]
	if !option.Available() {
		return p
	}
	if !p.aiming && len(option.Aims) > 1 {
		p.aiming, p.aim = true, 0
		return p
	}
	aim := option.Aims[0]
	if p.aiming {
		aim = option.Aims[clamp(p.aim, 0, len(option.Aims)-1)]
	}
	if err := p.take(p.pending, option.Skill, aim); err != nil {
		p.err = err
		return p
	}
	p.pending, p.aiming = nil, false
	return p.run()
}

// playLogLines is how much of the log the screen keeps in front of the player.
//
// The last few rather than all of it: what a player has to read is what happened
// since they last chose, and a screen that grew a line per event would push the
// board off the top by the third turn.
const playLogLines = 8

func (p playScreen) view(m model) (string, string) {
	footer := m.text(i18n.PlayFooter)
	if p.aiming {
		footer = m.text(i18n.PlayAimFooter)
	}
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.PlayHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.PlaySeed, p.seed)) + "\n")
	if p.err != nil {
		out.WriteString("\n  " + m.style.bad.Render(m.lang.Error(p.err)) + "\n")
		return out.String(), footer
	}
	if p.fight == nil {
		out.WriteString("\n  " + m.text(i18n.SquadsEmpty) + "\n")
		return out.String(), footer
	}
	out.WriteString("\n" + tui.Board(p.fight, p.tags) + "\n")
	out.WriteString(tui.Roster(p.fight, p.tags) + "\n\n")
	out.WriteString(m.style.dim.Render(tui.Order(p.fight.Queue(), p.tags, 6)) + "\n\n")
	out.WriteString(p.recent(m))
	if p.fight.Finished() {
		out.WriteString("\n" + m.style.emphasis.Render(p.ending(m)) + "\n")
		return strings.TrimRight(out.String(), "\n"), m.text(i18n.PlayOverFooter)
	}
	if p.pending == nil {
		return strings.TrimRight(out.String(), "\n"), footer
	}
	out.WriteString("\n" + p.choices(m))
	return strings.TrimRight(out.String(), "\n"), footer
}

// recent is the tail of the log, which is what happened since the player last
// chose.
func (p playScreen) recent(m model) string {
	from := len(p.events) - playLogLines
	if from < 0 {
		from = 0
	}
	var out strings.Builder
	for _, event := range p.events[from:] {
		line := tui.Line(event, p.tags)
		if line == "" {
			continue
		}
		out.WriteString("  " + m.style.dim.Render(line) + "\n")
	}
	return out.String()
}

// choices is the turn in front: whose it is, what they may do, and where it may
// be pointed once a skill is picked.
func (p playScreen) choices(m model) string {
	unit, known := p.fight.Unit(p.pending.Unit)
	if !known {
		return ""
	}
	var out strings.Builder
	out.WriteString(m.style.label.Render(m.text(i18n.PlayYourTurn,
		p.tags[unit.ID], unit.Name, p.pending.Turn)) + "\n")
	for index, option := range p.pending.Options {
		marker := "  "
		line := option.Skill
		if !option.Available() {
			line += "  " + option.Reason
		}
		switch {
		case !option.Available():
			line = m.style.dim.Render(line)
		case index == p.option && !p.aiming:
			marker = "> "
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}
	if !p.aiming {
		return out.String()
	}
	option := p.pending.Options[clamp(p.option, 0, len(p.pending.Options)-1)]
	out.WriteString("\n" + m.style.label.Render(m.text(i18n.PlayAimAt, option.Skill)) + "\n")
	for index, cell := range option.Aims {
		marker := "  "
		line := cell.String()
		if held := p.occupant(cell); held != "" {
			line += "  " + held
		}
		if index == p.aim {
			marker = "> "
			line = m.style.selected.Render(line)
		}
		out.WriteString(marker + line + "\n")
	}
	return out.String()
}

// occupant is the tag and name standing on a cell, so an aim reads as somebody
// rather than as a coordinate.
func (p playScreen) occupant(cell hex.Offset) string {
	for _, unit := range p.fight.Units() {
		if unit.Dead || unit.Cell != cell {
			continue
		}
		return p.tags[unit.ID] + " " + unit.Name
	}
	return ""
}

// ending is how the battle finished, in the words the game client uses for it.
func (p playScreen) ending(m model) string {
	switch p.fight.Outcome() {
	case battle.Victory:
		winner, _ := p.fight.Winner()
		if winner == p.side {
			return m.text(i18n.PlayWon)
		}
		return m.text(i18n.PlayLost)
	case battle.Stalemate:
		return m.text(i18n.PlayDrawn)
	default:
		return m.text(i18n.PlayEmptied)
	}
}

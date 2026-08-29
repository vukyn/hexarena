package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"path/filepath"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/forge"
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

	fight *battle.Battle
	// roster is what the battle was built from, kept because a log records it:
	// a log carrying the resolved placement is what makes it re-runnable across
	// a data edit, and asking the battle for it afterwards would be asking for
	// the units as they are now rather than as they were placed.
	roster []battle.Roster
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
	// notes are what a write left behind, held as facts rather than as a
	// sentence so ctrl+l redraws them in the other language.
	notes []forge.Note
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
	p.roster = nil
	p.events, p.script, p.pending = nil, nil, nil
	p.option, p.aim, p.aiming = 0, 0, false
	p.err, p.notes = nil, nil

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
	placed := append(roster, facing...)
	fight, err := battle.New(m.lib.Books(), p.seed, placed)
	if err != nil {
		p.err = err
		return p
	}
	fight.Begin()
	p.fight = fight
	p.roster = placed
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
	// Saving is asked before the switch because it answers to more than one
	// keystroke; isSaveKey is the single declaration of which.
	if isSaveKey(message) {
		p = p.save(m)
		m.play = p
		return m, nil
	}
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
	case "?":
		// The full description of the option under the cursor, on the screen the
		// rest of this tool already raises with this key. It is free here and it
		// costs no turn: the blurb reads the option and nothing else, so raising
		// it and leaving it cannot step the battle.
		//
		// Reached with a prompt open, which is what the guard above this switch
		// buys, and it works while aiming too — the skill is chosen and the cell
		// is not, so "what does this do" is still the live question. An empty
		// option list is refused rather than indexed: a turn with nothing to take
		// is a turn with nothing to describe.
		if len(p.pending.Options) > 0 {
			m.play = p
			m.blurb.from = screenPlay
			m.blurb.scroll = 0
			m.screen = screenBlurb
			return m, nil
		}
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

// save writes the battle out where the game client can replay it.
//
// It may be pressed at any point rather than only at the end, and that is not a
// concession: a battle stopped halfway is a battle, its script is consistent,
// and re-running it reproduces exactly the half that was played. What a log
// records is what happened, not what finished.
func (p playScreen) save(m model) playScreen {
	if p.fight == nil {
		return p
	}
	home, away, ok := m.fight.sides(m)
	if !ok {
		return p
	}
	path, err := m.lib.SaveBattleLog(home.ID, away.ID, p.seed, battle.Log{
		Seed: p.seed, Roster: p.roster, Choices: p.script, Events: p.events,
	})
	if err != nil {
		p.err, p.notes = err, nil
		return p
	}
	p.err = nil
	// The second note is the whole reason the first one is worth having, and it
	// carries the rebuild warning itself: --verify re-runs against the copy
	// baked into the game binary, so a log written after an edit nobody rebuilt
	// will not verify, and the mismatch would read as corruption.
	//
	// It names the file relative to the data directory, because what goes after
	// --replay is a path somebody has to type and the absolute one is mostly the
	// part they are already standing in.
	relative := path
	if shortened, err := filepath.Rel(m.lib.Dir(), path); err == nil {
		relative = shortened
	}
	p.notes = []forge.Note{
		{Kind: forge.NoteWrote, ID: filepath.Base(path), Path: path},
		{Kind: forge.NoteBattleVerify, Path: relative},
	}
	return p
}

// playLogLines is how much of the log the screen keeps in front of the player.
//
// The last few rather than all of it: what a player has to read is what happened
// since they last chose, and a screen that grew a line per event would push the
// board off the top by the third turn.
const playLogLines = 8

func (p playScreen) view(m model) (string, string) {
	footer := m.text(i18n.PlayFooter, saveKeyLabel())
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
		out.WriteString(p.wrote(m))
		return strings.TrimRight(out.String(), "\n"), m.text(i18n.PlayOverFooter, saveKeyLabel())
	}
	if p.pending == nil {
		return strings.TrimRight(out.String(), "\n"), footer
	}
	out.WriteString("\n" + p.choices(m))
	out.WriteString(p.wrote(m))
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
	// The id column is measured over the options this turn offers rather than
	// fixed, for the reason menuLabelWidth and every detail pane measure theirs.
	// Over the options and not over the book: the widest id in the game is
	// thirteen cells and this unit may be bringing four short ones, so a
	// book-wide column would spend the summary's room on a skill nobody here can
	// cast.
	width := p.optionWidth()
	// Clipped to the floor rather than to the window in hand, which is what the
	// caution line and the trait sentences do and for the same reason: measuring
	// the real terminal gives one line two shapes and leaves the width sweep
	// nothing to hold. Below the floor no screen is drawn at all, so the floor is
	// also the narrowest room this row ever really has.
	clip := lipgloss.NewStyle().MaxWidth(max(minWidth-1-markerWidth-width-optionGap, 0))
	for index, option := range p.pending.Options {
		marker := "  "
		// ⚠️ An unavailable option keeps its reason and **drops its summary**.
		// The row has one slot and the two answer different questions: the reason
		// is why this cannot be cast, which is the live question the moment a
		// cursor steps over it, and what the skill does is a ? away. Do not
		// "fix" this by drawing both — the second one would be the half that got
		// clipped.
		tail := p.summarise(m, option.Skill)
		if !option.Available() {
			tail = option.Reason
		}
		line := option.Skill
		if tail != "" {
			line += strings.Repeat(" ",
				width-lipgloss.Width(option.Skill)+optionGap) + clip.Render(tail)
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

// The two fixed columns a row spends before its summary: the cursor marker, and
// the gap between the id column and whatever follows it.
const (
	markerWidth = 2
	optionGap   = 2
)

// optionWidth is the id column, measured over the turn's own options.
func (p playScreen) optionWidth() int {
	width := 0
	for _, option := range p.pending.Options {
		if drawn := lipgloss.Width(option.Skill); drawn > width {
			width = drawn
		}
	}
	return width
}

// summarise is what a skill does, in the one line that fits beside its id.
//
// The whole reason the list is worth reading: an id is a name, and nothing in
// "venoshock" says it is the skill that doubles into a poison. It is
// i18n.Lang.SummariseSkill rather than anything assembled here — this screen may
// hold no wording of its own, and the compact line is tied to the full
// description by a test in that package rather than by a screen's good
// intentions.
//
// A skill the book cannot find summarises as nothing rather than as an error.
// The options come out of the battle, which was built from the same library, so
// a miss is not reachable; and a row is the wrong place to report that it was.
func (p playScreen) summarise(m model, id string) string {
	declared, err := m.lib.Skills().Lookup(id)
	if err != nil {
		return ""
	}
	return m.lang.SummariseSkill(declared, m.lib.Patterns())
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

// wrote is the line a save leaves behind, in the shape every other write in this
// client reports itself.
func (p playScreen) wrote(m model) string {
	if len(p.notes) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	for index, note := range m.lang.Notes(p.notes) {
		style := m.style.dim
		if index == 0 {
			style = m.style.good
		}
		// Wrapped against minWidth rather than the window in hand, for the
		// reason the fight's caution is: measuring the real terminal would give
		// one sentence two shapes and leave the width sweep nothing to hold.
		for _, line := range wrapWords(note, minWidth-1) {
			out.WriteString(style.Render(line) + "\n")
		}
	}
	return out.String()
}

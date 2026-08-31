package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/scale"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// sparScreen is forge.Spar drawn.
//
// It is the one screen that answers a question about a character rather than
// about the file it was written in. Everything else here checks that a character
// is legal — the budget, the carry rule, the art — and none of that says whether
// it belongs beside the ones already written. This fights it against them.
//
// It reads who is in front off the check screen's cursor and owns the level
// itself. That is the opposite arrangement from the preview, and deliberately:
// the preview borrows the browser's level because it draws the picture *for* a
// level somebody was already walking, while the level a comparison is made at is
// the comparison's own question.
type sparScreen struct {
	// level is what both sides are fielded at. Both, always: a spar between two
	// levels would be measuring the curve rather than the characters, and the
	// curve is what the browser is for.
	level int
	// seeds is how many duels each half of each row is fought over. It is here
	// rather than on the report because it is a question the author asks, and
	// asking it again is what re-running costs.
	seeds  int
	cursor int
	// fought is the reports already run, keyed by who, at what level, over how
	// many seeds. A map rather than a field for the reason the preview's cache is
	// one: every method here has a value receiver, so a plain field written while
	// drawing would be thrown away with the copy.
	//
	// The preview caches because rasterising art is slow. This caches for the
	// opposite reason — a spar is fast, but it is fast per keystroke, and the
	// level moves under an arrow key that repeats.
	fought map[string]forge.SparReport
	// failed is the same cache for the reports that could not be run at all, so
	// a refusal is not re-derived on every redraw either.
	failed map[string]error
}

// The number of duels a row is fought over, and how far it may be moved.
//
// A hundred is enough to tell a lopsided pairing from an even one and cheap
// enough to run while an arrow key repeats; a thousand is where a percentage
// point starts to mean something and is worth waiting for. The step doubles
// rather than adding, because what an author is choosing between is orders of
// confidence rather than exact counts.
const (
	sparSeeds    = 100
	sparMinSeeds = 10
	sparMaxSeeds = 1000
)

func newSparScreen() sparScreen {
	return sparScreen{
		level:  progression.LevelCap,
		seeds:  sparSeeds,
		fought: map[string]forge.SparReport{},
		failed: map[string]error{},
	}
}

// refresh empties the cache, and is called on entering the screen.
//
// The preview's cache survives because a picture on disk does not change while
// the program is up. A report does: it is derived from the whole library, and
// the library is written to by two other screens. Entering the spar after
// authoring a character has to fight the character that was authored.
func (s sparScreen) refresh() sparScreen {
	s.fought = map[string]forge.SparReport{}
	s.failed = map[string]error{}
	if s.level == 0 {
		s.level = progression.LevelCap
	}
	return s
}

func (s sparScreen) update(m model, message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "q":
		return m, tea.Quit
	case "esc", "s":
		m.screen = screenCheck
		return m, nil
	case "up", "k":
		s.cursor = clamp(s.cursor-1, 0, s.rowCount(m)-1)
	case "down", "j":
		s.cursor = clamp(s.cursor+1, 0, s.rowCount(m)-1)
	case "left", "h":
		s.level = clamp(s.level-1, 1, progression.LevelCap)
	case "right", "l":
		s.level = clamp(s.level+1, 1, progression.LevelCap)
	case "home":
		s.level = 1
	case "end":
		s.level = progression.LevelCap
	case "+", "=":
		s.seeds = clamp(s.seeds*2, sparMinSeeds, sparMaxSeeds)
	case "-":
		s.seeds = clamp(s.seeds/2, sparMinSeeds, sparMaxSeeds)
	}
	m.spar = s
	return m, nil
}

// rowCount is how many opponents the cursor has to walk, which is the whole
// cast whenever there is a report at all.
func (s sparScreen) rowCount(m model) int {
	report, _, ok := s.report(m)
	if !ok {
		return 0
	}
	return len(report.Matchups)
}

// subject is the character the check screen has under its cursor.
//
// The id is taken from the report and the character looked up in the book,
// rather than the report carrying one: a CharacterReport is a finding about a
// character and not the character, and the day it carries one is the day two
// screens can disagree about what is in the file.
// ⚠️ The row's FORM comes back with it, and that is what makes a forking line
// sparrable from here at all. Inspect reports one row an arm, so the cursor is
// already sitting on a chosen form — asking Spar for "the furthest" instead
// would hand it two answers and get a refusal, on a screen with nowhere to
// choose.
func (s sparScreen) subject(m model) (cast.Character, string, bool) {
	rows := m.check.report.Rows
	if len(rows) == 0 {
		return cast.Character{}, "", false
	}
	row := rows[clamp(m.check.cursor, 0, len(rows)-1)]
	character, known := m.lib.Characters().Get(row.ID)
	return character, row.Stage, known
}

// report is the run for who is in front, fighting it if it has not been fought.
func (s sparScreen) report(m model) (forge.SparReport, error, bool) {
	character, stage, ok := s.subject(m)
	if !ok {
		return forge.SparReport{}, nil, false
	}
	key := fmt.Sprintf("%s|%s|%d|%d", character.ID, stage, s.level, s.seeds)
	if done, cached := s.fought[key]; cached {
		return done, nil, true
	}
	if failure, cached := s.failed[key]; cached {
		return forge.SparReport{}, failure, true
	}
	done, err := m.lib.Spar(character.ID, s.level, s.seeds, stage)
	if err != nil {
		s.failed[key] = err
		return forge.SparReport{}, err, true
	}
	s.fought[key] = done
	return done, nil, true
}

// The fixed columns of the listing, all of them numbers. The opponent column is
// measured instead: it holds an id and the control marker beside one of them,
// and a constant wide enough for both in both languages is a constant nobody can
// check by looking at it.
const (
	sparRateWidth   = 8
	sparRecordWidth = 16
	sparTurnsWidth  = 7
)

// sparNameWidth is the column the opponent ids sit in, measured from the widest
// row being drawn rather than declared. Measure, do not guess: the marker beside
// the control row is a translated word, so the same constant would be right in
// at most one language, and an id is as long as the author made it.
func sparNameWidth(m model, report forge.SparReport) int {
	widest := lipgloss.Width(m.text(i18n.ColumnOpponent))
	for _, matchup := range report.Matchups {
		if width := lipgloss.Width(sparRowName(m, matchup)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

// sparRowName is an opponent's id, with the control marked as one.
func sparRowName(m model, matchup forge.Matchup) string {
	if !matchup.Mirror {
		return matchup.Against.ID
	}
	return matchup.Against.ID + " (" + m.text(i18n.SparControl) + ")"
}

// signed is a difference drawn with its direction, which a bare figure cannot
// carry: an advantage to the first slot and an advantage to the second are the
// same number and opposite findings. forge.PercentInColumn writes the minus; the
// plus is here because a column of differences needs both signs or neither.
//
// The figure itself is not translated, and that is a decision rather than an
// oversight. Vietnamese writes a decimal comma, but every other percentage this
// program shows goes through forge.Percent and prints a point — so a comma here
// would not be one screen localised, it would be one screen disagreeing with the
// rest of the tool about what a number looks like.
func signed(permille int) string {
	if permille < 0 {
		return forge.PercentInColumn(permille)
	}
	return "+" + forge.PercentInColumn(permille)
}

func (s sparScreen) view(m model) (string, string) {
	footer := m.text(i18n.SparFooter)
	character, _, ok := s.subject(m)
	if !ok {
		return "  " + m.text(i18n.CheckNothingToCheck) + "\n", footer
	}
	report, failure, _ := s.report(m)
	var out strings.Builder
	out.WriteString(m.style.Heading.Render(m.text(i18n.SparHeading)) + "  " +
		m.style.Label.Render(character.ID+" — "+character.Name) + "\n")
	if failure != nil {
		out.WriteString("  " + m.style.Bad.Render(m.lang.Error(failure)) + "\n")
		return out.String(), footer
	}

	challenger := report.Challenger
	fmt.Fprintf(&out, "  %s\n", m.style.Dim.Render(m.text(i18n.SparSubject,
		challenger.Level, challenger.Stage)))
	fmt.Fprintf(&out, "  %s\n\n", m.style.Dim.Render(m.text(i18n.SparConditions,
		len(report.Matchups), report.Seeds, sparBattles(report))))

	// What was fielded, drawn before any figure it produced. A win rate for four
	// skills the reader cannot see is a number they have to take on trust, and
	// this screen is the one that has to be trusted least.
	labels := detailLabelWidth(m)
	fmt.Fprintf(&out, "  %s %s\n", m.style.Label.Render(pad(m.text(i18n.LabelKit), labels)),
		strings.Join(challenger.Skills, " "))
	fmt.Fprintf(&out, "  %s %s\n\n", m.style.Label.Render(pad(m.text(i18n.LabelTraits), labels)),
		strings.Join(challenger.Passives, " "))

	if report.Opponents() == 0 {
		out.WriteString("  " + m.style.Dim.Render(m.text(i18n.SparAloneInTheCast)) + "\n\n")
	} else {
		fmt.Fprintf(&out, "  %s %s %s\n\n",
			m.style.Label.Render(pad(m.text(i18n.SparOverall), labels)),
			draw.Bar(draw.StatBarWidth, int64(report.Rate()), scale.Base),
			forge.PercentInColumn(report.Rate()))
	}

	names := sparNameWidth(m, report)
	fmt.Fprintf(&out, "  %s%s%s%s%s\n",
		pad(m.text(i18n.ColumnOpponent), names),
		pad(m.text(i18n.ColumnRate), sparRateWidth),
		pad(m.text(i18n.ColumnRecord), sparRecordWidth),
		pad(m.text(i18n.ColumnTurns), sparTurnsWidth),
		m.text(i18n.ColumnFirstMove))
	endless := 0
	for i, matchup := range report.Matchups {
		marker := "  "
		if i == s.cursor {
			marker = "> "
		}
		name := pad(sparRowName(m, matchup), names)
		if i == s.cursor {
			name = m.style.Selected.Render(name)
		}
		total := matchup.Total()
		endless += total.Endless
		fmt.Fprintf(&out, "%s%s%s%s%s%s\n", marker, name,
			pad(forge.PercentInColumn(matchup.Rate()), sparRateWidth),
			pad(m.text(i18n.SparRecord, total.Wins, total.Losses, total.Draws), sparRecordWidth),
			pad(fmt.Sprint(matchup.Turns), sparTurnsWidth),
			signed(matchup.Edge()))
	}
	if endless > 0 {
		out.WriteString("\n  " + m.style.Bad.Render(m.text(i18n.SparEndless, endless)) + "\n")
	}
	out.WriteString("\n" + m.style.Dim.Render(
		m.text(i18n.SparNote, cast.SkillSlots, cast.TraitSlots)))
	return out.String(), footer
}

// sparBattles is how many duels the whole report cost: every row, both ways.
func sparBattles(report forge.SparReport) int {
	return 2 * report.Seeds * len(report.Matchups)
}

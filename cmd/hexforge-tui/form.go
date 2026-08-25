package main

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the new-character form, in the order they are walked.
//
// The order is hexforge's prompt order, and on this screen it is a reading order
// rather than a constraint. At a prompt the kit has to come before the element,
// because an answer once given is given; here both fields are on screen at once
// and either one re-checks against the other the moment it changes, so the
// author may fill them in whichever way round they think.
//
// That is why the kit is chosen from a list and the element is still typed. The
// list marks the skills this character cannot take and says why — with no
// element answered yet, nothing is marked, because an unanswered element
// restricts nothing (forge.Carrier) — and the carry line under the form refuses
// an element the chosen kit cannot take, naming the skill. Neither direction
// silently drops the other's answer: changing the element does not empty the
// kit, it turns the offending rows into marked rows and the carry line red,
// which is a state an author can see and fix rather than one that happened
// behind them.
const (
	fieldID = iota
	fieldName
	fieldOrigin
	fieldArchetype
	fieldImage
	fieldKit
	fieldElement
	fieldBio
	fieldStatBase
)

const fieldCount = fieldStatBase + progression.KindCount

// formScreen authors one character.
//
// Nothing here decides whether an answer is acceptable. The budget comes from
// forge.Library.Budget, which is progression.EffectiveHP; the carry check comes
// from forge.Library.ValidateElement, which is skill.CanCarry; and the write
// goes through forge.Draft.Resolve, which appends to the book and therefore
// applies exactly the checks a load applies. What this type owns is when to ask
// those questions — which is on every keystroke, because they are integer
// arithmetic and a map lookup and cost nothing.
type formScreen struct {
	inputs     []textinput.Model
	origins    []cast.Origin
	archetypes []cast.Archetype
	// art is the images internal/forge found under the data directory, which
	// is what the art field offers. Empty means there is nothing to offer and
	// that field is a text field instead — see choiceField.
	art []string

	// kit is the chosen skills, in the order they were chosen, because that
	// order is the kit. It replaced a comma separated text field: nineteen ids
	// typed by hand is nineteen chances to name one that does not exist, and the
	// list can say who each skill is kept for while it is being chosen.
	kit []string

	originIndex    int
	archetypeIndex int
	artIndex       int
	cursor         int

	// touched is whether anything has been typed or chosen. It is the whole of
	// the unsaved-changes guard: a form nobody has edited is one an Escape may
	// throw away in silence.
	touched bool

	// These three record whether a field is still following the value something
	// else supplies it — the art path follows the id, the kit and the curves
	// follow the archetype — so that changing the id or the preset updates them,
	// and editing one by hand stops it being overwritten under the author.
	imageFollowsID    bool
	kitFollowsPreset  bool
	statFollowsPreset [progression.KindCount]bool

	// err is the last refusal from a save, and notes is what the last successful
	// one reported. Both are kept as internal/forge produced them — an error
	// value and a slice of forge.Note — rather than as the lines they will be
	// drawn as, so that switching language redraws them instead of leaving the
	// previous language's sentence sitting under a translated form.
	err   error
	notes []forge.Note
}

func newFormScreen(lib *forge.Library) formScreen {
	// A directory that cannot be read is, from the author's side, a directory
	// with nothing in it: either way there is no art to offer, and the answer to
	// both is the text field and a line naming where it looked. Refusing to draw
	// the form over it would be the one outcome that helps nobody.
	art, _ := lib.ArtFiles()
	f := formScreen{
		inputs:           make([]textinput.Model, fieldCount),
		origins:          lib.Origins().All(),
		archetypes:       lib.Archetypes().All(),
		art:              art,
		imageFollowsID:   true,
		kitFollowsPreset: true,
	}
	for kind := range f.statFollowsPreset {
		f.statFollowsPreset[kind] = true
	}
	for i := range f.inputs {
		input := textinput.New()
		input.Prompt = ""
		input.CharLimit = 200
		input.Width = 40
		f.inputs[i] = input
	}
	f.inputs[fieldBio].CharLimit = 400
	// A curve is nine characters at its longest, so a narrow field leaves room
	// for the meter and the numbers beside it inside the minimum window.
	for _, kind := range progression.Kinds() {
		f.inputs[fieldStatBase+int(kind)].Width = 11
	}
	f = f.applyPreset()
	f.inputs[f.cursor].Focus()
	return f
}

// applyPreset refills every field that is still following something else.
func (f formScreen) applyPreset() formScreen {
	f = f.applyArt()
	if len(f.archetypes) == 0 {
		return f
	}
	preset := f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)]
	if f.kitFollowsPreset {
		f.kit = append([]string(nil), preset.Skills...)
	}
	for _, kind := range progression.Kinds() {
		if f.statFollowsPreset[kind] {
			f.inputs[fieldStatBase+int(kind)].SetValue(forge.FormatCurve(preset.Stats[kind]))
		}
	}
	return f
}

// applyArt points the art field at what the id suggests, while it is still
// following the id.
//
// Which field that is depends on what is on disk. With art to choose from, the
// suggestion is worth honouring only when it names one of the entries:
// SuggestedImage derives a path from the id, so it names a real file exactly
// when the art was filed where the id says it would be. When it does not, the
// selection stays where it is — the first entry to begin with, and whatever was
// chosen once somebody has chosen — rather than jumping back to the top of the
// list on every keystroke in the id.
func (f formScreen) applyArt() formScreen {
	if !f.imageFollowsID {
		return f
	}
	suggested := forge.SuggestedImage(strings.TrimSpace(f.inputs[fieldID].Value()))
	if len(f.art) == 0 {
		f.inputs[fieldImage].SetValue(suggested)
		// The cursor goes to the end of what was just put there, which is where
		// an author editing a prefilled path starts from. SetValue moves it on
		// its own only when the field was empty, so filling this one a letter at
		// a time as the id is typed would otherwise leave the cursor stranded
		// where the first suggestion happened to end.
		f.inputs[fieldImage].CursorEnd()
		return f
	}
	if at := slices.Index(f.art, suggested); at >= 0 {
		f.artIndex = at
	}
	return f
}

// imageAnswer is the art path the form is offering, from whichever of the two
// fields is in front.
func (f formScreen) imageAnswer() string {
	if len(f.art) == 0 {
		return strings.TrimSpace(f.inputs[fieldImage].Value())
	}
	return f.art[clamp(f.artIndex, 0, len(f.art)-1)]
}

// Draft is the answers as internal/forge wants them. It is the only thing this
// screen hands outwards, which is what makes "the form produces the character
// the command line produces" a statement a test can make.
func (f formScreen) draft() forge.Draft {
	draft := forge.Draft{
		ID:      strings.TrimSpace(f.inputs[fieldID].Value()),
		Name:    f.inputs[fieldName].Value(),
		Image:   f.imageAnswer(),
		Skills:  strings.Join(f.kit, ","),
		Element: strings.TrimSpace(f.inputs[fieldElement].Value()),
		Bio:     f.inputs[fieldBio].Value(),
	}
	if len(f.origins) > 0 {
		draft.Origin = f.origins[clamp(f.originIndex, 0, len(f.origins)-1)].ID
	}
	if len(f.archetypes) > 0 {
		draft.Archetype = f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)].ID
	}
	for _, kind := range progression.Kinds() {
		draft.Stats[kind] = f.inputs[fieldStatBase+int(kind)].Value()
	}
	return draft
}

// choiceField reports whether a field is picked from a list rather than typed.
//
// An origin and an archetype are ids in books, so typing one is a way to get it
// wrong; the list cannot produce an answer the book does not hold. Art is the
// same shape with one difference — its list is what is on disk rather than what
// a book declares, so it can be empty, and an empty one leaves that field a
// text field. A form that cannot be completed because a folder is empty is
// worse than one that lets an author name a file that is not there yet, which
// is what the command line does and what the write already warns about.
func (f formScreen) choiceField(field int) bool {
	switch field {
	case fieldOrigin, fieldArchetype, fieldKit:
		return true
	case fieldImage:
		return len(f.art) > 0
	default:
		return false
	}
}

func (f formScreen) update(m model, message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		return f.leave(m), nil
	case "ctrl+s":
		f = f.save(m)
		m.form = f
		return m, nil
	case "up", "shift+tab":
		f = f.moveTo(f.cursor - 1)
		m.form = f
		return m, nil
	case "down", "tab", "enter":
		f = f.moveTo(f.cursor + 1)
		m.form = f
		return m, nil
	}
	// The kit is a list rather than a cycle, so it opens a sub-screen instead of
	// stepping. Nineteen skills do not fit beside a form — see picker.go, which
	// measures that rather than asserting it.
	if f.cursor == fieldKit {
		m.form = f
		if message.String() == " " || message.String() == "right" {
			return m.openKit(), nil
		}
		return m, nil
	}
	if f.choiceField(f.cursor) {
		switch message.String() {
		case "left":
			f = f.cycle(-1)
		case "right":
			f = f.cycle(1)
		}
		m.form = f
		return m, nil
	}

	updated, command := f.inputs[f.cursor].Update(message)
	changed := updated.Value() != f.inputs[f.cursor].Value()
	f.inputs[f.cursor] = updated
	if changed {
		f.touched = true
		f.err = nil
		f.notes = nil
		switch {
		case f.cursor == fieldID:
			// The art field follows the id until somebody sets it themselves.
			f = f.applyArt()
		case f.cursor == fieldImage:
			f.imageFollowsID = false
		case f.cursor >= fieldStatBase:
			f.statFollowsPreset[f.cursor-fieldStatBase] = false
		}
	}
	m.form = f
	return m, command
}

// openKit raises the picker over the skill book.
//
// The options carry a refusal each, and it is forge.CheckSkill's: the same
// predicate the write applies, against whatever the form has settled so far. So
// the reason a row is unavailable cannot disagree with the reason the write
// refuses it, which is the whole point of the answer coming from internal/forge.
func (m model) openKit() model {
	return m.pick(&pickState{
		title: i18n.PickerKitTitle, kind: pickSkills,
		options: kitOptions(m.lib, m.form.draft().Carrier()),
		chosen:  m.form.kit,
		apply: func(m model, answer pickAnswer) model {
			m.form.kit = answer.Chosen
			// Choosing is setting it by hand, so it stops following the preset,
			// on the same terms as typing a kit used to.
			m.form.kitFollowsPreset = false
			m.form.touched = true
			m.form.err = nil
			m.form.notes = nil
			return m
		},
	})
}

// leave goes back to the menu, asking first if there is anything to lose.
func (f formScreen) leave(m model) model {
	if !f.touched {
		m.screen = screenMenu
		return m
	}
	return m.ask(i18n.FormDiscard, func(m model) model {
		m.form = newFormScreen(m.lib)
		m.screen = screenMenu
		return m
	})
}

// moveTo changes the focused field, wrapping at both ends so the form is a ring
// rather than a dead end.
func (f formScreen) moveTo(target int) formScreen {
	f.inputs[f.cursor].Blur()
	f.cursor = (target + fieldCount) % fieldCount
	if !f.choiceField(f.cursor) {
		f.inputs[f.cursor].Focus()
	}
	return f
}

// cycle steps a chooser, and refills whatever the new preset supplies.
func (f formScreen) cycle(by int) formScreen {
	switch f.cursor {
	case fieldOrigin:
		if len(f.origins) > 0 {
			f.originIndex = (f.originIndex + by + len(f.origins)) % len(f.origins)
			f.touched = true
		}
	case fieldArchetype:
		if len(f.archetypes) > 0 {
			f.archetypeIndex = (f.archetypeIndex + by + len(f.archetypes)) % len(f.archetypes)
			f.touched = true
			f = f.applyPreset()
		}
	case fieldImage:
		if len(f.art) > 0 {
			f.artIndex = (f.artIndex + by + len(f.art)) % len(f.art)
			// Choosing is setting it by hand, so it stops following the id, on
			// the same terms as typing a path used to.
			f.imageFollowsID = false
			f.touched = true
		}
	}
	f.err = nil
	f.notes = nil
	return f
}

// save resolves the draft and writes it.
//
// Both halves belong to internal/forge: Resolve refuses a character a load
// would refuse, SaveCharacter writes through the temp-file-then-rename that
// keeps a crash from truncating cast.json, and SaveNotes is the same
// confirmation hexforge prints — including the warning that the art is not
// there yet and the reminder that the game boots from the embedded copy.
func (f formScreen) save(m model) formScreen {
	character, err := f.draft().Resolve(m.lib)
	if err != nil {
		f.err = err
		f.notes = nil
		return f
	}
	if err := m.lib.SaveCharacter(character); err != nil {
		f.err = err
		f.notes = nil
		return f
	}
	notes := m.lib.SaveNoteFacts(character)
	fresh := newFormScreen(m.lib)
	fresh.notes = notes
	return fresh
}

// fieldLabel is what each row is called.
//
// Stat rows are named by their short label — hp, atk, def — in every language,
// and forge.ShortStat says why: those six are the flag names and the data
// files' own keys, so an author needs them as they are written to act on them.
func fieldLabel(m model, field int) string {
	labels := map[int]i18n.Key{
		fieldID:        i18n.FieldID,
		fieldName:      i18n.FieldName,
		fieldOrigin:    i18n.FieldOrigin,
		fieldArchetype: i18n.FieldArchetype,
		fieldImage:     i18n.FieldArt,
		fieldKit:       i18n.FieldKit,
		fieldElement:   i18n.FieldElement,
		fieldBio:       i18n.FieldBiography,
	}
	// A map read by key, never ranged over into anything drawn: the order on
	// screen is the field order, which is the constants above.
	if key, named := labels[field]; named {
		return m.text(key)
	}
	return forge.ShortStat(progression.Kind(field - fieldStatBase))
}

// formLabelWidth is the column the field names sit in. It is measured from the
// labels themselves rather than declared, because the longest of them is
// "archetype" in one language and "mẫu vai trò" in the other, and a constant
// would be wrong for one of them the next time either is reworded.
func formLabelWidth(m model) int {
	widest := 0
	for field := range fieldCount {
		if width := lipgloss.Width(fieldLabel(m, field)); width > widest {
			widest = width
		}
	}
	return widest + 1
}

func (f formScreen) view(m model) (string, string) {
	footer := m.text(i18n.FormFooter)
	var out strings.Builder
	out.WriteString(m.style.heading.Render(m.text(i18n.FormHeading)) + "  " +
		m.style.dim.Render(m.text(i18n.FormSubtitle, progression.LevelCap)) + "\n\n")

	width := formLabelWidth(m)
	for field := range fieldCount {
		out.WriteString(f.row(m, field, width))
	}
	out.WriteString("\n")
	out.WriteString(f.liveChecks(m, width))
	return out.String(), footer
}

// row draws one field: a marker, its name, and either its text or its choice.
func (f formScreen) row(m model, field, labelWidth int) string {
	marker := "  "
	if field == f.cursor {
		marker = "> "
	}
	name := pad(fieldLabel(m, field), labelWidth)
	if field == f.cursor {
		name = m.style.selected.Render(name)
	} else {
		name = m.style.label.Render(name)
	}

	var value string
	switch {
	case field == fieldOrigin:
		value = f.choice(m, f.originIndex, len(f.origins), f.originLabel())
	case field == fieldArchetype:
		value = f.choice(m, f.archetypeIndex, len(f.archetypes), f.archetypeLabel(m))
	case field == fieldImage && f.choiceField(fieldImage):
		value = f.choice(m, f.artIndex, len(f.art), f.artLabel(m))
	case field == fieldKit:
		value = f.kitValue(m, labelWidth)
	case field >= fieldStatBase:
		value = f.statRow(m, progression.Kind(field-fieldStatBase))
	default:
		value = f.inputs[field].View()
	}
	drawn := marker + name + " " + value + "\n"
	if field == fieldImage && !f.choiceField(fieldImage) {
		// An art field that is a text field is the exception now, so it says
		// why: "nothing to choose from" without naming the folder it looked in
		// is a line nobody can act on. It sits under the row in the same column
		// as the values, which is what m.labelAt with no name draws.
		drawn += m.labelAt("", labelWidth, "%s",
			m.style.dim.Render(m.text(i18n.NoArtToChoose, m.lib.AssetsPath())))
	}
	return drawn
}

// kitValue is the chosen kit, with the key that opens the list.
//
// The ids and not their names, for the reason no id is translated anywhere here:
// they are what cast.json holds and what --skills takes. Their Vietnamese names
// are one keypress away, on the rows of the list itself.
func (f formScreen) kitValue(m model, labelWidth int) string {
	hint := m.style.dim.Render(m.text(i18n.KitChooseHint))
	room := minWidth - 3 - labelWidth - lipgloss.Width(m.text(i18n.KitChooseHint)) - 2
	if len(f.kit) == 0 {
		return m.style.bad.Render(m.text(i18n.PickerNothingChosen)) + "  " + hint
	}
	return clip(strings.Join(f.kit, " "), room) + "  " + hint
}

// choiceFormat is how a chooser draws: the value between arrows, then where it
// sits in the list.
//
// It is a constant because the art chooser measures its own room from it. A
// second copy of the decoration would drift from the one being drawn, and the
// measurement would then be of a row nobody sees.
const choiceFormat = "< %s >  %s"

// choice renders a picked value with its position, so "there are more of these"
// is visible without pressing anything.
func (f formScreen) choice(m model, index, total int, label string) string {
	if total == 0 {
		return m.style.bad.Render(m.text(i18n.NoneCatalogued))
	}
	return fmt.Sprintf(choiceFormat, label,
		m.style.dim.Render(m.text(i18n.ChoicePosition, index+1, total)))
}

// artLabel is the chosen art path, shortened to what its row has room for.
func (f formScreen) artLabel(m model) string {
	return elidePath(f.imageAnswer(), artRoom(m, f.artIndex, len(f.art)))
}

// artRoom is how many cells the art chooser has for its path.
//
// A path is the one chooser value here with no bound. An origin and an
// archetype are ids from a book, but art is a file under a folder an author
// fills as they like: assets/fixture/sprout.svg is already 24 cells before
// anybody nests one deeper. So the row is measured rather than trusted — what
// is left once the marker, the label column, the chooser's own decoration and
// the position counter have taken theirs, all of them as they are really drawn.
//
// What it measures against is the floor and not the window in hand, and that is
// deliberate: minWidth is the width this program promises to draw in, while
// measuring the real terminal would give the same row two lengths and leave
// TestEveryWordingFitsTheMinimumWidth nothing to hold.
func artRoom(m model, index, total int) int {
	const marker = 2
	decoration := lipgloss.Width(fmt.Sprintf(choiceFormat, "",
		m.text(i18n.ChoicePosition, index+1, total)))
	// The window's last column is left empty, as it is everywhere here: a line
	// filling a terminal's final cell wraps on some of them.
	return minWidth - 1 - marker - formLabelWidth(m) - 1 - decoration
}

// ellipsis marks a path that did not fit. One rune rather than three dots, so
// that its cell count is its character count, which is the unit every column in
// this client is measured in.
const ellipsis = "…"

// elidePath shortens a path to a number of cells, keeping the end.
//
// The end is the informative half: the file name says which piece of art this
// is and the extension says what kind, while the folder above it is either
// obvious from the character being authored or one keypress away. So a path
// that does not fit loses its directories first — assets/deep/hero.svg becomes
// …/hero.svg — and only a file name that is too long on its own is cut, from
// the front, so that the extension survives.
//
// Shortened rather than wrapped, for the reason frame clips: a wrapped row
// pushes every row under it down by one, which is how the footer leaves the
// bottom of the screen.
func elidePath(image string, room int) string {
	if room < 1 {
		return ""
	}
	if lipgloss.Width(image) <= room {
		return image
	}
	base := path.Base(image)
	if shortened := ellipsis + "/" + base; lipgloss.Width(shortened) <= room {
		return shortened
	}
	// Measured a rune at a time rather than counted, so that a name holding a
	// character wider than one cell cannot slip past the budget.
	letters := []rune(base)
	for len(letters) > 0 && lipgloss.Width(ellipsis+string(letters)) > room {
		letters = letters[1:]
	}
	return ellipsis + string(letters)
}

func (f formScreen) originLabel() string {
	if len(f.origins) == 0 {
		return ""
	}
	origin := f.origins[clamp(f.originIndex, 0, len(f.origins)-1)]
	return fmt.Sprintf("%s — %s", origin.ID, origin.Title)
}

func (f formScreen) archetypeLabel(m model) string {
	if len(f.archetypes) == 0 {
		return ""
	}
	preset := f.archetypes[clamp(f.archetypeIndex, 0, len(f.archetypes)-1)]
	return fmt.Sprintf("%s — %s", preset.ID, m.lang.PresetSummary(preset))
}

// statRow is a curve as "base:max", a meter of where its maximum sits against
// the ceiling for that stat, and the ceiling itself.
func (f formScreen) statRow(m model, kind progression.Kind) string {
	input := f.inputs[fieldStatBase+int(kind)]
	ceiling := m.lib.Limits().Ceilings[kind]
	curve, err := forge.ParseCurve(input.Value())
	meter := m.style.dim.Render(strings.Repeat("-", statBarWidth+2))
	trailing := ""
	if err != nil {
		// The refusal is forge.ParseCurve's, worded by the catalog. This screen
		// does not decide what is wrong with a curve, only where to put it.
		return fmt.Sprintf("%s %s %s", input.View(), meter, m.style.bad.Render(m.lang.Error(err)))
	}
	meter = bar(statBarWidth, curve.Max, ceiling)
	trailing = m.text(i18n.CurveAgainstCeiling, curve.Base, curve.Max, ceiling)
	if curve.Max > ceiling {
		trailing = m.style.bad.Render(trailing + "  " + m.text(i18n.OverTheCeiling))
	} else {
		trailing = m.style.dim.Render(trailing)
	}
	return fmt.Sprintf("%s %s %s", input.View(), meter, trailing)
}

// liveChecks is the two answers that make this screen worth having: what the
// stat line spends of the joint budget, and whether the affinity can carry the
// kit. Both are recomputed on every keystroke because both are cheap, and both
// come from internal/forge so that neither can disagree with the write.
func (f formScreen) liveChecks(m model, labelWidth int) string {
	var out strings.Builder
	draft := f.draft()

	table, err := draft.Table(m.lib)
	if err != nil {
		out.WriteString(m.labelAt(m.text(i18n.LabelBudget), labelWidth, "%s", m.style.bad.Render(m.lang.Error(err))))
	} else {
		values := table.At(progression.LevelCap)
		out.WriteString(m.labelAt(m.text(i18n.LabelBudget), labelWidth, "%s", budgetLine(m, m.lib.Budget(values))))
	}

	out.WriteString(m.labelAt(m.text(i18n.LabelCarries), labelWidth, "%s", f.carryLine(m, draft)))

	switch {
	case f.err != nil:
		out.WriteString("\n" +
			m.style.bad.Render(m.text(i18n.WriteRefused, m.lang.Error(f.err))) + "\n")
	case len(f.notes) > 0:
		out.WriteString("\n")
		for i, note := range m.lang.Notes(f.notes) {
			if i == 0 {
				out.WriteString(m.style.good.Render(note) + "\n")
				continue
			}
			out.WriteString(m.style.dim.Render(note) + "\n")
		}
	}
	return out.String()
}

// carryLine says whether the affinity carries every skill in the kit, and names
// the first one it cannot.
//
// The judgement is not this screen's doing: forge.CheckCarry, reached through
// ValidateElement, decides it and hands back a *forge.CarryError holding the
// affinity, the skill and the skill's element, so the form, the prompt and the
// parser cannot disagree about a mismatch. Only the sentence is chosen here,
// and only by asking the catalog for it.
func (f formScreen) carryLine(m model, draft forge.Draft) string {
	names := draft.KitNames(m.lib)
	kit, err := m.lib.LookupKit(names)
	if err != nil {
		return m.style.bad.Render(m.lang.Error(err))
	}
	if strings.TrimSpace(draft.Element) == "" {
		return m.style.dim.Render(m.text(i18n.CarryNoElementYet, m.lang.KitSummary(kit)))
	}
	if err := m.lib.ValidateElement(draft.Element, kit); err != nil {
		return m.style.bad.Render(m.text(i18n.CarryRefused, m.lang.Error(err)))
	}
	return m.style.good.Render(m.text(i18n.CarryAccepted, draft.Element))
}

package screen

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The fields of the add-a-work form.
//
// Exported for the reason the skill form's own field constants are: a client's
// fixture reaches a field the way a keystroke does, and a test cannot follow a
// screen across a package boundary.
const (
	OriginFieldID = iota
	OriginFieldTitle
	OriginFieldMedium
	OriginFieldYear
	OriginFieldNote
	OriginFieldCount
)

// OriginsScreen is the catalog of works the cast is borrowed from, and the form
// that adds one.
//
// A work has to exist before a character can name it, so this screen is the
// other half of the new-character form rather than a listing for its own sake:
// an author who finds the origin they want missing can add it and go straight
// back without leaving the program.
//
// It is the **seventh and last** of the catalogues a game client will offer, and
// the cleanest of the moves: it kept no cursor of anybody else's, raised nothing
// and read no other screen's state, so the whole of what it named of its client
// was one way back and one question. The way back is a Back and the question is
// an Ask carrying no subject — this screen names nothing, so it passes nil where
// the squad builder passes a SquadsAsk.
type OriginsScreen struct {
	Origins []cast.Origin
	Counts  map[string]int
	Cursor  int

	// Adding is whether the form is in front of the catalog.
	Adding bool

	// Inputs is every typed field of the form, indexed by the OriginField
	// constants above.
	//
	// ⚠️ **A slice, so two copies of this screen SHARE it**, exactly as the skill
	// form's own Inputs do and with the same caveat: every other field is copied
	// when the screen is and these text fields are not. It is moved as it is,
	// deliberately — the behaviour is relied on today and changing it is a
	// measurement of its own rather than a side effect of a move.
	Inputs      []textinput.Model
	MediumIndex int
	Field       int
	Touched     bool

	Err error
	// Added is the last work written, kept as what it was rather than as the
	// line announcing it, so a language switch redraws the announcement.
	Added *cast.Origin
}

// NewOriginsScreen is the catalog filled from a library, with an empty form
// behind it.
//
// ⚠️ **It takes a Context rather than a library**, which is the one signature
// this move had to change, and the reason is the one NewSkillsScreen carries:
// the form dresses text fields, NewInput wants to know whether colour would be
// noise, this package may not read an environment, and Palette already holds the
// answer the binary was handed.
func NewOriginsScreen(c Context) OriginsScreen {
	return OriginsScreen{}.Refresh(c).ResetForm(c)
}

// Refresh re-reads the catalog and the count of characters borrowed from each
// work, clamping the cursor: entering the screen after a write should show the
// new row rather than a stale list.
func (o OriginsScreen) Refresh(c Context) OriginsScreen {
	o.Origins = c.Lib.Origins().All()
	o.Counts = make(map[string]int, len(o.Origins))
	for _, origin := range o.Origins {
		// A map keyed by id, read by key. Nothing here ranges over it into an
		// ordered output; the order on screen is the book's.
		o.Counts[origin.ID] = len(c.Lib.Characters().OfOrigin(origin.ID))
	}
	o.Cursor = Clamp(o.Cursor, 0, len(o.Origins)-1)
	if o.Inputs == nil {
		o = o.ResetForm(c)
	}
	return o
}

// ResetForm empties every field of the add-a-work form and puts the keyboard
// back on the first of them.
func (o OriginsScreen) ResetForm(c Context) OriginsScreen {
	o.Inputs = make([]textinput.Model, OriginFieldCount)
	for i := range o.Inputs {
		input := NewInput(c.Style.Plain)
		input.Prompt = ""
		input.CharLimit = 200
		input.SetWidth(44)
		o.Inputs[i] = input
	}
	o.Field = OriginFieldID
	o.MediumIndex = 0
	o.Touched = false
	o.Err = nil
	o.Inputs[o.Field].Focus()
	return o
}

// Update routes a keystroke on the catalog or on the form over it, and says what
// the client owes it.
//
// ⚠️ **Three returns rather than two, and the third is a tea.Cmd** — the same
// arrangement, for the same measured reason, as draw.SkillsScreen.Update. This
// screen has text fields, and a bubbles textinput answers an Update with a
// command (the cursor's blink) which has to come out or the field loses its
// cursor. It is not a field on Action, deliberately: an Action is a comparable
// value a screen returns and a test writes out as a literal, and a func field
// would take that away from every screen to serve two.
func (o OriginsScreen) Update(c Context, message tea.KeyPressMsg) (OriginsScreen, Action, tea.Cmd) {
	if o.Adding {
		return o.updateForm(c, message)
	}
	switch message.String() {
	case "q":
		return o, Action{Kind: Quit}, nil
	case "esc":
		return o, Action{Kind: Back}, nil
	case "up", "k":
		o.Cursor = Clamp(o.Cursor-1, 0, len(o.Origins)-1)
	case "down", "j":
		o.Cursor = Clamp(o.Cursor+1, 0, len(o.Origins)-1)
	case "a":
		// Guarded on the client being able to write — see Context.Authoring. A
		// game client draws this catalogue so a reader can see which fiction the
		// cast is borrowed from, and the form over it writes origins.json.
		if c.Authoring {
			o = o.ResetForm(c)
			o.Adding = true
			o.Added = nil
		}
	}
	return o, Action{}, nil
}

// Confirmed throws the half-typed work away and closes the form, which is the
// whole of what the question above offered.
//
// It stays on this screen — the listing behind the form is where the reader was
// — so the action is the zero one, Stay.
//
// ⚠️ **What the question was about arrives as an `any` and is ignored.** This
// screen asks one question and it is about the screen itself, so there is
// nothing to tell apart; the parameter is here so the shape matches every other
// Confirmed a client dispatch is held total over.
func (o OriginsScreen) Confirmed(c Context, _ any) (OriginsScreen, Action) {
	o = o.ResetForm(c)
	o.Adding = false
	return o, Action{}
}

func (o OriginsScreen) updateForm(c Context, message tea.KeyPressMsg) (OriginsScreen, Action, tea.Cmd) {
	// Before the switch, because saving answers to more than one keystroke and
	// IsSaveKey is the single declaration of which.
	if IsSaveKey(message) {
		return o.save(c), Action{}, nil
	}
	switch message.String() {
	case "esc":
		if !o.Touched {
			o.Adding = false
			return o, Action{}, nil
		}
		return o, Action{Kind: Ask, Question: i18n.OriginFormDiscard}, nil
	case "up", "shift+tab":
		return o.moveTo(o.Field - 1), Action{}, nil
	case "down", "tab", "enter":
		return o.moveTo(o.Field + 1), Action{}, nil
	}
	if o.Field == OriginFieldMedium {
		mediums := cast.MediumNames()
		switch message.String() {
		case "left":
			o.MediumIndex = (o.MediumIndex - 1 + len(mediums)) % len(mediums)
			o.Touched = true
		case "right":
			o.MediumIndex = (o.MediumIndex + 1) % len(mediums)
			o.Touched = true
		}
		return o, Action{}, nil
	}
	updated, command := o.Inputs[o.Field].Update(message)
	if updated.Value() != o.Inputs[o.Field].Value() {
		o.Touched = true
		o.Err = nil
	}
	o.Inputs[o.Field] = updated
	return o, Action{}, command
}

func (o OriginsScreen) moveTo(target int) OriginsScreen {
	o.Inputs[o.Field].Blur()
	o.Field = (target + OriginFieldCount) % OriginFieldCount
	if o.Field != OriginFieldMedium {
		o.Inputs[o.Field].Focus()
	}
	return o
}

// save writes the work through forge.Library.SaveOrigin, which validates it
// exactly as a load would and replaces the file atomically.
//
// ⚠️ This is the one method in this file that reaches a real directory, and it
// reaches it the way the skill form and the squad builder do: through
// internal/forge, which is the one part of the module allowed to touch a file.
// Nothing in this package opens one itself, which is why the end-to-end write is
// driven in cmd/hexforge-tui, where a scratch data directory already exists.
func (o OriginsScreen) save(c Context) OriginsScreen {
	// The year's refusal is forge.ParseYear's, like every other refusal on this
	// screen: a rule worded in a front-end is a rule declared twice.
	year, err := forge.ParseYear(o.Inputs[OriginFieldYear].Value())
	if err != nil {
		o.Err = err
		return o
	}
	medium, err := cast.ParseMedium(cast.MediumNames()[o.MediumIndex])
	if err != nil {
		o.Err = err
		return o
	}
	origin := cast.Origin{
		ID:     strings.TrimSpace(o.Inputs[OriginFieldID].Value()),
		Title:  strings.TrimSpace(o.Inputs[OriginFieldTitle].Value()),
		Medium: medium,
		Year:   year,
		Note:   strings.TrimSpace(o.Inputs[OriginFieldNote].Value()),
	}
	if err := c.Lib.SaveOrigin(origin); err != nil {
		o.Err = err
		return o
	}
	o = o.Refresh(c).ResetForm(c)
	o.Adding = false
	o.Added = &origin
	return o
}

// View draws the catalog, or the form when it is in front, as a body and a
// footer.
func (o OriginsScreen) View(c Context) (string, string) {
	if o.Adding {
		return o.viewForm(c)
	}
	footer := c.Footer(i18n.OriginsFooter, i18n.OriginsReadFooter)
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.OriginsHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.OriginsSubtitle)) + "\n\n")
	if len(o.Origins) == 0 {
		out.WriteString("  " + c.Text(i18n.OriginsEmpty) + "\n")
		return out.String(), footer
	}
	counted := originsCountWidth(c)
	for i, origin := range o.Origins {
		marker := "  "
		// Blank rather than a figure when nobody recorded one, which is what
		// makes a zero year readable: Pad below gives the cell its five cells
		// either way, so the guard is about the word and not the width.
		year := ""
		if origin.Year != 0 {
			year = strconv.Itoa(origin.Year)
		}
		row := fmt.Sprintf("%s %s %s %s  %s",
			Pad(origin.ID, 16), Pad(origin.Medium.String(), 7), Pad(year, 5),
			Pad(c.Text(i18n.OriginsCastCount, o.Counts[origin.ID]), counted), origin.Title)
		if i == o.Cursor {
			marker = "> "
			row = c.Style.Selected.Render(row)
		}
		out.WriteString(marker + row + "\n")
	}
	if selected := o.Origins[Clamp(o.Cursor, 0, len(o.Origins)-1)]; selected.Note != "" {
		out.WriteString("\n" + c.Label(c.Text(i18n.LabelNote), "%s", selected.Note))
	}
	if o.Added != nil {
		out.WriteString("\n" + c.Style.Good.Render(c.Text(i18n.OriginAdded,
			o.Added.ID, o.Added.Medium, c.Lib.OriginsPath())) + "\n")
	}
	out.WriteString("\n" + c.Style.Dim.Render(c.Text(i18n.OriginsTally,
		len(o.Origins), strings.Join(cast.MediumNames(), " "))))
	return out.String(), footer
}

// originsCountWidth is the column the "how many characters" cell sits in. It is
// measured rather than declared because the cell is a counted noun — two words
// in Vietnamese, one in English — and the widest count is the whole cast.
func originsCountWidth(c Context) int {
	widest := 0
	for _, origin := range c.Lib.Origins().All() {
		width := lipgloss.Width(c.Text(i18n.OriginsCastCount,
			len(c.Lib.Characters().OfOrigin(origin.ID))))
		if width > widest {
			widest = width
		}
	}
	return widest
}

// originFieldLabel is what each row of the add-a-work form is called.
func originFieldLabel(c Context, field int) string {
	keys := [OriginFieldCount]i18n.Key{
		OriginFieldID:     i18n.OriginFieldID,
		OriginFieldTitle:  i18n.OriginFieldTitle,
		OriginFieldMedium: i18n.OriginFieldMedium,
		OriginFieldYear:   i18n.OriginFieldYear,
		OriginFieldNote:   i18n.OriginFieldNote,
	}
	return c.Text(keys[field])
}

func (o OriginsScreen) viewForm(c Context) (string, string) {
	footer := c.Text(i18n.OriginFormFooter, SaveKeyLabel())
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.OriginFormHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.OriginFormSubtitle)) + "\n\n")
	width := 0
	for field := range OriginFieldCount {
		if measured := lipgloss.Width(originFieldLabel(c, field)); measured > width {
			width = measured
		}
	}
	width++
	for field := range OriginFieldCount {
		marker := "  "
		if field == o.Field {
			marker = "> "
		}
		name := Pad(originFieldLabel(c, field), width)
		if field == o.Field {
			name = c.Style.Selected.Render(name)
		} else {
			name = c.Style.Label.Render(name)
		}
		value := o.Inputs[field].View()
		if field == OriginFieldMedium {
			mediums := cast.MediumNames()
			value = fmt.Sprintf(ChoiceFormat, mediums[o.MediumIndex],
				c.Style.Dim.Render(c.Text(i18n.ChoicePosition, o.MediumIndex+1, len(mediums))))
		}
		out.WriteString(marker + name + " " + value + "\n")
	}
	out.WriteString("\n" + c.Style.Dim.Render(c.Text(i18n.OriginFormHint)) + "\n")
	if o.Err != nil {
		out.WriteString("\n" +
			c.Style.Bad.Render(c.Text(i18n.AddRefused, c.Lang.Error(o.Err))) + "\n")
	}
	return out.String(), footer
}

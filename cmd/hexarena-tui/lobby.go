package main

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
	"github.com/vukyn/hexarena/internal/socket"
	"github.com/vukyn/hexarena/internal/wire"
)

// # The lobby: three screens this client draws and the other one never will
//
// ⚠️ **They live here rather than in internal/screen, and the reason is the
// import graph.** A lobby screen holding a wire.RoomCode or a wire.Code would
// pull the protocol into the package two clients share — and
// i18n.Lang.Refusal(name string) takes a **name** precisely so that never has to
// happen. Beside that: internal/screen's package comment describes the screens
// *two clients draw*, and cmd/hexforge-tui has no room to join.
//
// **The stated cost.** internal/screen/testdata/screens.golden cannot see these,
// so a layout regression here is caught by one golden rather than two. Accepted:
// there is no data column and no drawing on any of the three — a heading, some
// prose, two fields and a code.
//
// What holds them instead is this client's own net:
// TestEveryScreenThisClientDrawsIsSwept goes red the moment screenCount grows,
// so each of the three is registered in everyScreen in the commit that added it,
// and from there it gets the width sweep, the two-language sweep, the gloss-leak
// sweep and an entry in this client's golden.

// joinScreen is the room code, the room's password, and which squad to bring.
type joinScreen struct {
	// Code and Password are the two fields, and Field says which has the
	// cursor.
	Code     textinput.Model
	Password textinput.Model
	Field    int

	// Squads is the catalogue as it stood on the way in, and Squad the row
	// under the chooser. A slice rather than a reference to the catalogue
	// screen: this one is re-read by Refresh on every entrance, for the reason
	// enter(screenBattle) re-reads it.
	Squads []placement.Squad
	Squad  int

	// Dialling is a room being called, and At the code that is being called.
	// A screen that said nothing while a network round trip was in flight would
	// look like a key that did nothing.
	Dialling bool
	At       wire.RoomCode

	// Refused is the **name** of the wire.Code a refusal carried, and empty
	// when the last failure was not one. A name and not the typed value: that
	// is what keeps internal/wire out of internal/i18n, and it costs this
	// screen one String call.
	Refused string
	// Err is a failure that was not a refusal — a code of the wrong shape, a
	// machine that is not there — drawn through Lang.Error, which is the house
	// rule for a diagnostic in the parser's own English behind a lead-in.
	Err error
	// BadLength says the last submission was a code of the wrong length, and
	// Mistyped how long it was.
	//
	// ⚠️ **A flag beside the number rather than a nought meaning "fine".** An
	// empty field is a code of length nought, which is the commonest wrong
	// length there is, so a nought standing for absence would swallow exactly
	// the case a player hits first. That is the sentinel-in-a-legal-value
	// mistake this repository has paid for twice.
	//
	// ⚠️ **And a number rather than an error carrying a sentence.** The first
	// draft was a typed error with an Error() method holding prose;
	// TestNoScreenHoldsItsOwnWording caught it on the first run. It is also the
	// rule internal/forge is written under — return values, not sentences — so
	// that a front-end cannot become a second declaration of a refusal.
	BadLength bool
	Mistyped  int

	// Edited says the data in the directory this client is reading differs from
	// the copy the binary embeds, which is a thing the reader has to be told
	// once and only when it is true. → i18n.JoinDataEdited.
	Edited bool
}

// The two fields, in the order tab walks them.
const (
	joinFieldCode = iota
	joinFieldPassword
	joinFieldCount
)

// The widths the two fields are drawn at.
//
// ⚠️ **They are sized by the PLACEHOLDER rather than by the value**, which is
// the wider of the two and was measured rather than guessed: a room code is
// twelve characters, but the line offering one runs to twenty-one in English and
// a field narrower than its own hint draws a hint with the end cut off. The
// password's is longer still. Both are checked by the sweep, which asserts the
// placeholder is on the drawn screen — a cut one is not.
const (
	joinCodeWidth     = 24
	joinPasswordWidth = 34
)

// newJoinScreen builds the two fields, dressed the way every field in these two
// clients is. → style.go on why the dress is a forwarder rather than a copy.
func newJoinScreen() joinScreen {
	code, password := newInput(), newInput()
	code.Prompt, password.Prompt = "", ""
	code.CharLimit, password.CharLimit = 64, 200
	code.SetWidth(joinCodeWidth)
	password.SetWidth(joinPasswordWidth)
	// ⚠️ The password is masked by the field rather than by anything this screen
	// does with the value. wire.Password redacts itself under every fmt verb,
	// which covers the value travelling; this covers it being over somebody's
	// shoulder.
	password.EchoMode = textinput.EchoPassword
	code.Focus()
	return joinScreen{Code: code, Password: password}
}

// Refresh is the screen as a reader finds it on the way in: the catalogue
// re-read, the last attempt's refusal cleared, and the data notice measured.
//
// ⚠️ **The notice is measured rather than assumed.** A --data directory whose
// files are byte-identical to the embedded copy is the common case, and telling
// a player their edits will not reach the battle when they have made none is
// noise on every join. An unreadable directory is not a difference either — it
// is a library that would not have loaded.
func (j joinScreen) Refresh(c draw.Context, saved []placement.Squad) joinScreen {
	j.Squads = saved
	j.Squad = draw.Clamp(j.Squad, 0, max(len(saved)-1, 0))
	j.Dialling, j.At = false, ""
	j.Refused, j.Err = "", nil
	j.BadLength, j.Mistyped = false, 0
	same, err := c.Lib.MatchesEmbeddedData()
	j.Edited = err == nil && !same
	return j
}

// Chosen is the squad this screen would bring, and whether there is one.
func (j joinScreen) Chosen() (placement.Squad, bool) {
	if len(j.Squads) == 0 {
		return placement.Squad{}, false
	}
	return j.Squads[draw.Clamp(j.Squad, 0, len(j.Squads)-1)], true
}

// Update routes one keystroke.
//
// Three returns rather than two, and this is the client's **first** screen that
// genuinely needs the third: a bubbles textinput answers an Update with the
// cursor's blink, and dropping it leaves the field with no cursor. navigateWith
// has carried that command since it was written and it has been nil every time
// until now.
func (j joinScreen) Update(c draw.Context, message tea.KeyPressMsg) (joinScreen, draw.Action, tea.Cmd) {
	if j.Dialling {
		// A dial in flight takes no keys but esc: the answer is a message, and a
		// second enter would open a second socket the session has nowhere to put.
		if message.String() == "esc" {
			return j, draw.Action{Kind: draw.Back}, nil
		}
		return j, draw.Action{}, nil
	}
	switch message.String() {
	case "esc":
		return j, draw.Action{Kind: draw.Back}, nil
	case "tab":
		j.Field = (j.Field + 1) % joinFieldCount
		return j.focused(), draw.Action{}, nil
	case "shift+tab":
		j.Field = (j.Field + joinFieldCount - 1) % joinFieldCount
		return j.focused(), draw.Action{}, nil
	case "left":
		if len(j.Squads) > 0 {
			j.Squad = (j.Squad + len(j.Squads) - 1) % len(j.Squads)
		}
		return j, draw.Action{}, nil
	case "right":
		if len(j.Squads) > 0 {
			j.Squad = (j.Squad + 1) % len(j.Squads)
		}
		return j, draw.Action{}, nil
	case "enter":
		return j.submit(), draw.Action{}, nil
	}
	updated, command := j.field().Update(message)
	j = j.replace(updated)
	return j, draw.Action{}, command
}

// Paste puts a pasted string into whichever of the two fields has the cursor, and
// nowhere when neither has it.
//
// ⚠️ **A dial in flight takes no paste, for the reason it takes no keys.** The
// answer to a dial is a message, and a code changing under a round trip that is
// already carrying one would leave the field disagreeing with the room being
// called — which is the state j.At exists to keep straight.
//
// ⚠️ **Nothing is trimmed here, and that is submit's job rather than an
// oversight.** A pasted room code arrives with a newline more often than not and
// bubbles turns each one into a space, so the field really does end up holding
// `"7QK4M2XZ9BTF "` — measured, and pinned by
// TestATextFieldTurnsANewlineInAPasteIntoASpace one layer down. submit already
// TrimSpaces before it measures the length, because that is where the value is
// read and where a refusal about it is worded; trimming a second time on the way
// in would be a second declaration of the same rule, and it would also make the
// field refuse to hold a space somebody meant to type.
//
// What it costs is one invisible cell of a twenty-four-cell field: the cursor
// sits one past the code rather than against it. Nothing else on the screen
// moves, the length refusal counts the trimmed value, and the dial re-encodes
// into canonical form regardless.
func (j joinScreen) Paste(text string) (joinScreen, tea.Cmd) {
	field := j.field()
	if j.Dialling || !field.Focused() {
		return j, nil
	}
	command := draw.PasteInto(&field, text)
	return j.replace(field), command
}

// submit reads the code out of the field and asks for it to be dialled, or says
// why it will not.
//
// ⚠️ **The client owes nothing about case and one thing about length.**
// RoomCode.Decode upper-cases before it decodes — the alphabet is upper-case
// only and the fold is total — and socket.Dial re-encodes into canonical form
// before it builds the path, exactly as the server's own roomOf does. So a code
// typed in lower case is a good code and nothing here touches it. What is owed
// is a TrimSpace, because a pasted code arrives with a newline more often than
// not, and a friendly refusal for a code of the wrong **length** — because
// Decode's own message is developer-facing English and reads badly as the first
// thing a player ever sees. Everything past length goes to Dial and comes back
// through Lang.Error.
func (j joinScreen) submit() joinScreen {
	j.Refused, j.Err = "", nil
	j.BadLength, j.Mistyped = false, 0
	if _, have := j.Chosen(); !have {
		return j
	}
	typed := strings.TrimSpace(j.Code.Value())
	if len(typed) != wire.RoomCodeLength {
		j.BadLength, j.Mistyped = true, len(typed)
		return j
	}
	j.Dialling, j.At = true, wire.RoomCode(typed)
	return j
}

// Refused is the join screen after a dial came back with something to say.
func (j joinScreen) Failed(err error) joinScreen {
	j.Dialling, j.At = false, ""
	j.Refused, j.Err = "", nil
	j.BadLength, j.Mistyped = false, 0
	var refusal *socket.Refusal
	if errors.As(err, &refusal) {
		// The **name**, not the value. → the field.
		j.Refused = refusal.Code.String()
		return j
	}
	j.Err = err
	return j
}

func (j joinScreen) field() textinput.Model {
	if j.Field == joinFieldPassword {
		return j.Password
	}
	return j.Code
}

func (j joinScreen) replace(field textinput.Model) joinScreen {
	if j.Field == joinFieldPassword {
		j.Password = field
		return j
	}
	j.Code = field
	return j
}

// focused puts the cursor in the field the tab landed on and takes it out of the
// other, which is what makes one of the two draw a cursor.
func (j joinScreen) focused() joinScreen {
	j.Code.Blur()
	j.Password.Blur()
	if j.Field == joinFieldPassword {
		j.Password.Focus()
		return j
	}
	j.Code.Focus()
	return j
}

// View draws the screen and its footer.
func (j joinScreen) View(c draw.Context) (string, string) {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.JoinHeading)) + "\n\n")
	for _, line := range draw.WrapWords(c.Text(i18n.JoinHint), draw.MinWidth-3) {
		out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
	}
	if j.Edited {
		out.WriteString("\n")
		for _, line := range draw.WrapWords(c.Text(i18n.JoinDataEdited), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
		}
	}
	out.WriteString("\n")

	width := joinLabelWidth(c)
	out.WriteString(j.row(c, width, joinFieldCode, i18n.JoinCodeLabel,
		placeholder(j.Code, c.Text(i18n.JoinCodePlaceholder))))
	out.WriteString(j.row(c, width, joinFieldPassword, i18n.JoinPasswordLabel,
		placeholder(j.Password, c.Text(i18n.JoinPasswordPlaceholder))))
	out.WriteString("  " + draw.Pad(c.Text(i18n.JoinSquadLabel), width) + " " + j.squadValue(c) + "\n")

	out.WriteString("\n")
	switch {
	case len(j.Squads) == 0:
		out.WriteString("  " + c.Style.Bad.Render(c.Text(i18n.JoinNoSquad)) + "\n")
	case j.Dialling:
		out.WriteString("  " + c.Style.Dim.Render(c.Text(i18n.JoinDialling, j.At)) + "\n")
	}
	if j.Refused != "" {
		out.WriteString("  " + c.Style.Bad.Render(c.Text(i18n.JoinRefused)) + "\n")
		for _, line := range draw.WrapWords(c.Lang.Refusal(j.Refused), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Bad.Render(line) + "\n")
		}
	}
	if j.BadLength {
		out.WriteString("  " + c.Style.Bad.Render(
			c.Text(i18n.JoinCodeLength, wire.RoomCodeLength, j.Mistyped)) + "\n")
	}
	if j.Err != nil {
		// Lang.Error is the house rule for a diagnostic that is not this
		// program's to word: the lead-in is the reader's language and the
		// parser's own English follows it.
		for _, line := range draw.WrapWords(c.Lang.Error(j.Err), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Bad.Render(line) + "\n")
		}
	}
	return out.String(), c.Text(i18n.JoinFooter)
}

// row is one field, drawn the way every labelled row in these clients is.
func (j joinScreen) row(c draw.Context, width, field int, label i18n.Key, drawn string) string {
	marker := "  "
	if j.Field == field {
		marker = "> "
	}
	return marker + draw.Pad(c.Text(label), width) + " " + drawn + "\n"
}

// placeholder puts the hint on a field before it has been typed in.
//
// It is set here rather than at construction because a placeholder is **wording**
// and ctrl+l swaps the language on any screen: a placeholder written once at
// startup would stay in the language the program opened in.
func placeholder(field textinput.Model, hint string) string {
	field.Placeholder = hint
	return field.View()
}

// squadValue is the chooser row: the side's name between arrows, and its id
// beside it. Both are free text an author wrote, so neither is measured.
func (j joinScreen) squadValue(c draw.Context) string {
	chosen, have := j.Chosen()
	if !have {
		return c.Style.Dim.Render(draw.Ellipsis)
	}
	return fmt.Sprintf(draw.ChoiceFormat, chosen.Name, c.Style.Dim.Render(chosen.ID))
}

// joinLabelWidth is the column the three labels sit in, measured over the
// language in front for the reason menuLabelWidth is measured: one number for
// two languages is only right for both by luck.
func joinLabelWidth(c draw.Context) int {
	widest := 0
	for _, key := range []i18n.Key{i18n.JoinCodeLabel, i18n.JoinPasswordLabel, i18n.JoinSquadLabel} {
		if width := lipgloss.Width(c.Text(key)); width > widest {
			widest = width
		}
	}
	return widest
}

// waitingScreen is the room joined and the second seat still empty.
//
// It holds values rather than reading the mirror while it draws, and that is the
// same rule PlayScreen is written under one layer down: the mirror belongs to
// the Play goroutine and is read under a lock, so what a screen keeps is what
// somebody handed it inside one.
type waitingScreen struct {
	Code    wire.RoomCode
	Seat    wire.Seat
	Welcome wire.Welcome
	Seated  bool
}

func (w waitingScreen) View(c draw.Context) (string, string) {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.WaitingHeading)) + "\n\n")
	for _, line := range draw.WrapWords(c.Text(i18n.WaitingForPeer), draw.MinWidth-3) {
		out.WriteString("  " + line + "\n")
	}
	out.WriteString("\n")
	out.WriteString("  " + c.Style.Dim.Render(c.Text(i18n.WaitingRoom, w.Code)) + "\n")
	out.WriteString("  " + c.Style.Dim.Render(
		c.Text(i18n.WaitingSeat, c.Lang.Seat(string(w.Seat)))) + "\n")
	if w.Seated {
		out.WriteString("  " + c.Style.Dim.Render(c.Text(i18n.WaitingFormat,
			w.Welcome.Format, w.Welcome.Battles, w.Welcome.Allowance)) + "\n")
	}
	return out.String(), c.Text(i18n.WaitingFooter)
}

// resultScreen is how a match ended: the standing this client's own engine
// settled, and the closure when the room sent one.
//
// ⚠️ **The standing is computed here from socket.Mirror.Fought and there is no
// standing message.** A client is a mirror: it learns each battle's outcome from
// its own Ended event and the series length from the welcome, and a standing on
// the wire would be a second declaration of something both peers compute — the
// one place two peers could disagree about who was winning while both of their
// battles agreed.
type resultScreen struct {
	Fought []socket.Fought
	// Closure is the name of the wire.Closure the room sent, empty when it sent
	// none. A **name**, for the reason joinScreen.Refused is one.
	Closure string
	// Err is what Play returned — a divergence, most of all, which is the one
	// failure this whole design exists to make loud.
	Err error
}

func (r resultScreen) View(c draw.Context) (string, string) {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.ResultHeading)) + "\n\n")
	mine, theirs := r.standing()
	out.WriteString("  " + c.Style.Emphasis.Render(c.Text(i18n.ResultStanding, mine, theirs)) + "\n\n")
	for _, one := range r.Fought {
		out.WriteString("  " + c.Text(i18n.ResultBattleLine, one.Battle, c.Text(outcomeKey(one))) + "\n")
	}
	if r.Closure != "" {
		out.WriteString("\n")
		for _, line := range draw.WrapWords(c.Lang.Closure(r.Closure), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
		}
	}
	if r.Err != nil {
		out.WriteString("\n")
		for _, line := range draw.WrapWords(c.Lang.Error(r.Err), draw.MinWidth-3) {
			out.WriteString("  " + c.Style.Bad.Render(line) + "\n")
		}
	}
	return out.String(), c.Text(i18n.ResultFooter)
}

// standing is how many of the series each side took. A battle nobody won counts
// for neither, which is what a draw is.
func (r resultScreen) standing() (mine, theirs int) {
	for _, one := range r.Fought {
		switch {
		case !one.Decided:
		case one.Mine():
			mine++
		default:
			theirs++
		}
	}
	return mine, theirs
}

// outcomeKey is the one word a battle's row carries.
func outcomeKey(one socket.Fought) i18n.Key {
	switch {
	case !one.Decided:
		return i18n.ResultDrawn
	case one.Mine():
		return i18n.ResultWon
	default:
		return i18n.ResultLost
	}
}

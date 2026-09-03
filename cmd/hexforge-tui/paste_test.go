package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/clipboard"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// # Pasting into the authoring client
//
// This client has by far the most to lose from a deaf field: a character's form
// alone is fifteen fields plus a curve each, and the skill form, the works form
// and the squad builder are three more forms on top. Every one of them ignored a
// paste, for the same reason the game client's join screen did — bracketed paste
// delivers one tea.PasteMsg and this model's Update named tea.KeyPressMsg and
// nothing else.
//
// What is measured here is the **route**: the guard, the picker and the four
// screens, in that order, and every other screen doing nothing. Where a paste
// lands once a screen has it is internal/screen's own suite; the character form
// is this package's own and is measured here.

// pasteOf is one bracketed paste as a terminal delivers it.
//
// ⚠️ **One message carrying the whole string.** A helper that sent a key per
// character would measure the typed path under another name, and every assertion
// below would pass with the route deleted.
func pasteOf(text string) tea.PasteMsg { return tea.PasteMsg{Content: text} }

// TestAPasteFillsTheCharacterFormsFocusedField is the biggest of the four forms,
// and the one whose fields are this package's own.
//
// Three fields rather than one, because the form's paste arm has three
// consequences to get right and a single field would exercise none of them: the
// id is followed by the art path, the art path stops following once it is set by
// hand, and a curve stops following the preset.
//
// *Sees:* the tea.PasteMsg arm deleted from Update, the form dropped from the
// route, formScreen.paste writing a field the cursor is not on, and any of the
// three follow-links left unbroken by a paste that a keystroke breaks.
// *Cannot see:* that a real terminal sends the message.
func TestAPasteFillsTheCharacterFormsFocusedField(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	if !m.form.inputs[m.form.cursor].Focused() {
		t.Fatal("the form opened with no field focused")
	}

	m = send(t, m, pasteOf("mot-nhan-vat"))
	if got := m.form.inputs[fieldID].Value(); got != "mot-nhan-vat" {
		t.Fatalf("the id field holds %q after a paste", got)
	}
	if !m.form.touched {
		t.Error("a paste left the form untouched, so escape would throw it away in silence")
	}
	if !m.form.imageFollowsID {
		t.Error("a paste into the id stopped the art path following it")
	}

	// The name, one row down, so the cursor is what decides and not the field
	// index.
	m = key(t, m, "down")
	if m.form.cursor != fieldName {
		t.Fatalf("down landed on field %d, want the name", m.form.cursor)
	}
	m = send(t, m, pasteOf("Tên Dán"))
	if got := m.form.inputs[fieldName].Value(); got != "Tên Dán" {
		t.Errorf("the name field holds %q after a paste", got)
	}
	if got := m.form.inputs[fieldID].Value(); got != "mot-nhan-vat" {
		t.Errorf("the second paste changed the id to %q, so both landed in one field", got)
	}

	// A curve, which is where a paste has to break a link a keystroke breaks.
	for m.form.cursor < fieldStatBase {
		m = key(t, m, "down")
	}
	kind := m.form.cursor - fieldStatBase
	if !m.form.statFollowsPreset[kind] {
		t.Fatalf("curve %d already stopped following the preset, so this measures nothing", kind)
	}
	m = send(t, m, pasteOf("thang"))
	if m.form.statFollowsPreset[kind] {
		t.Errorf("a paste into curve %d left it following the preset, so the next preset "+
			"would overwrite what was pasted", kind)
	}
}

// TestAPasteFillsTheSkillFormsFocusedFieldHere is the same claim through the
// other route arm, driven on the real model rather than on the screen alone.
//
// ⚠️ **It is not a duplicate of internal/screen's.** That one calls Paste on a
// screen it built; this one presses `a` on the model and drives a message through
// Update, so it measures the arm in this client's route and the Context it hands
// over. A screen test cannot see a missing route arm and a route test cannot see
// a screen writing the wrong field.
func TestAPasteFillsTheSkillFormsFocusedFieldHere(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	if !m.skills.FormInFront() {
		t.Fatal("`a` did not open the skill form")
	}
	m = send(t, m, pasteOf("mot-chieu-dan"))
	if got := m.skills.Inputs[draw.SkillFieldID].Value(); got != "mot-chieu-dan" {
		t.Errorf("the skill form's id holds %q after a paste", got)
	}
}

// TestAPasteFillsThePickersChanceField is the fourth route arm, and the one that
// is asked before the screen switch rather than inside it.
//
// ⚠️ **A picker is drawn OVER whichever screen raised it**, so a route that
// walked the screen switch first would put the paste into the skill form behind
// the list the author is looking at. That is what the ordering assertion below is
// about: the chance takes it and the form's own fields do not move.
//
// *Sees:* the picker arm moved after the switch, or dropped; PasteDigits swapped
// for PasteInto on a numeric field.
func TestAPasteFillsThePickersChanceField(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = typeText(t, m, "a")
	for m.skills.Field != draw.SkillFieldInflicts {
		m = key(t, m, "down")
	}
	before := m.skills.Inputs[draw.SkillFieldInflicts].Value()
	m = key(t, m, "space")
	if m.picker == nil || m.picker.Typed == nil {
		t.Fatal("space on the inflicts field raised no picker with a chance on it")
	}
	m.picker.Typed.SetValue("")

	m = send(t, m, pasteOf("250"))
	if got := m.picker.Typed.Value(); got != "250" {
		t.Errorf("the chance field holds %q after a paste", got)
	}
	if got := m.skills.Inputs[draw.SkillFieldInflicts].Value(); got != before {
		t.Errorf("a paste aimed at the picker reached the form behind it: %q", got)
	}
}

// TestAPasteWithAQuestionPendingChangesNothing is the guard, which is the one
// thing drawn over everything and has no field at all.
//
// *Sees:* the guard arm dropped from the route, which would let a paste fill the
// form a discard question is being asked about.
func TestAPasteWithAQuestionPendingChangesNothing(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	m = typeText(t, m, "x")
	m = key(t, m, "esc")
	if m.guard == nil {
		t.Fatal("escape from a touched form asked nothing, so this test measures nothing")
	}
	before := m.screenContent()
	m = send(t, m, pasteOf("dan-vao-cau-hoi"))
	if got := m.form.inputs[fieldID].Value(); got != "x" {
		t.Errorf("a paste behind a pending question filled the id with %q", got)
	}
	if m.screenContent() != before {
		t.Error("a paste behind a pending question changed what is drawn")
	}
}

// TestAPasteWithNoFieldInFrontChangesNothingHere is the vacuity guard.
//
// ⚠️ **Aimed at a chooser row inside an OPEN form, and at the works catalogue.**
// Both are screens that own text fields, which is what makes the assertion able
// to fail: a screen with no fields would satisfy it with the whole route deleted.
// The chooser row is the sharper of the two — the form really is in front, so
// only Focused tells the row apart from the one above it.
//
// *Sees:* formScreen.pasting dropping its Focused check; OriginsScreen.Pasting
// dropping its mode check; a route that pasted into whatever screen was in front.
func TestAPasteWithNoFieldInFrontChangesNothingHere(t *testing.T) {
	t.Run("a chooser row inside the character form", func(t *testing.T) {
		m, _, _ := start(t, i18n.Vi)
		m = m.enter(screenNew)
		for !m.form.choiceField(m.form.cursor) {
			m = key(t, m, "down")
		}
		if m.form.inputs[m.form.cursor].Focused() {
			t.Fatalf("field %d is a chooser and is focused, so this measures the wrong thing",
				m.form.cursor)
		}
		before := fieldValues(m)
		m = send(t, m, pasteOf("dan-vao-cho-chon"))
		for field, got := range fieldValues(m) {
			if got != before[field] {
				t.Errorf("a paste on a chooser row changed field %d from %q to %q",
					field, before[field], got)
			}
		}
		if m.form.touched {
			t.Error("a paste on a chooser row made the form dirty without changing anything")
		}
	})

	t.Run("the works catalogue", func(t *testing.T) {
		m, _, _ := start(t, i18n.Vi)
		m = m.enter(screenOrigins)
		if !m.origins.Inputs[draw.OriginFieldID].Focused() {
			t.Fatal("the hidden form's id field is not focused, so this test measures nothing")
		}
		before := m.screenContent()
		m = send(t, m, pasteOf("dan-vao-danh-muc"))
		if got := m.origins.Inputs[draw.OriginFieldID].Value(); got != "" {
			t.Errorf("a paste on the catalogue filled the hidden form's id with %q", got)
		}
		if m.screenContent() != before {
			t.Error("a paste on the works catalogue changed what it draws")
		}
	})

	t.Run("the menu", func(t *testing.T) {
		m, _, _ := start(t, i18n.Vi)
		before := m.screenContent()
		m = send(t, m, pasteOf("dan-vao-menu"))
		if m.screenContent() != before {
			t.Error("a paste on the menu changed what it draws")
		}
	})
}

// TestCtrlVPastesTheClipboardIntoAFocusedFieldHere is the second route: the key a
// terminal does **not** intercept.
//
// ⌘V and ctrl+shift+V are injected by the terminal as bracketed paste and are the
// tests above. Plain ctrl+v arrives as an ordinary keystroke, so this client reads
// the clipboard itself and says the answer as the same tea.PasteMsg.
//
// ⚠️ **The reader is injected and the real clipboard is never driven.** What that
// cannot see is that the real reader is still what is wired in;
// internal/clipboard's own test holds that by identity.
//
// *Sees:* the ctrl+v arm deleted; a command whose message this model cannot name
// — which is what bubbles' own textinput.Paste produces and is exactly why it is
// not what is wired here.
func TestCtrlVPastesTheClipboardIntoAFocusedFieldHere(t *testing.T) {
	handClipboard(t, "dan-tu-bang-nho")

	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	next, command := m.Update(keyPresses["ctrl+v"])
	m = next.(model)
	if command == nil {
		t.Fatal("ctrl+v asked for no command, so no clipboard was ever read")
	}
	message := command()
	if _, named := message.(tea.PasteMsg); !named {
		t.Fatalf("ctrl+v produced a %T, which this model cannot name and would drop", message)
	}
	m = send(t, m, message)
	if got := m.form.inputs[fieldID].Value(); got != "dan-tu-bang-nho" {
		t.Errorf("the id field holds %q after ctrl+v", got)
	}
}

// TestCtrlVWithNoFieldFocusedChangesNothingHere is the vacuity guard for the
// second route, aimed at a chooser row inside an open form for the reason the
// first one is.
func TestCtrlVWithNoFieldFocusedChangesNothingHere(t *testing.T) {
	handClipboard(t, "dan-tu-bang-nho")

	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenNew)
	for !m.form.choiceField(m.form.cursor) {
		m = key(t, m, "down")
	}
	before := fieldValues(m)
	drawn := m.screenContent()

	next, command := m.Update(keyPresses["ctrl+v"])
	m = next.(model)
	if command == nil {
		t.Fatal("ctrl+v asked for no command at all, so the route below is never reached")
	}
	m = send(t, m, command())
	for field, got := range fieldValues(m) {
		if got != before[field] {
			t.Errorf("ctrl+v on a chooser row changed field %d from %q to %q",
				field, before[field], got)
		}
	}
	if m.screenContent() != drawn {
		t.Error("ctrl+v on a chooser row changed what the form draws")
	}
}

// TestCtrlVCollidesWithNothingThisClientBinds derives the claim the ctrl+v arm's
// comment makes, rather than restating it.
//
// Every chord this suite can send is pressed on the character form with a field
// focused, and ctrl+v has to be the only one that asks for a clipboard read. It
// is derived because "I grepped and found no collision" is a claim about the day
// it was written.
//
// *Sees:* a second binding for ctrl+v added anywhere ahead of the arm; the arm
// moved below a screen that answers the chord first.
// *Cannot see:* a chord no entry of keyPresses names. The table is what this suite
// can send, which is why ctrl+v was added to it in the same commit as the arm.
func TestCtrlVCollidesWithNothingThisClientBinds(t *testing.T) {
	handClipboard(t, "dan-tu-bang-nho")
	reads := []string{}
	for name, press := range keyPresses {
		if press.Mod == 0 {
			continue
		}
		m, _, _ := start(t, i18n.Vi)
		m = m.enter(screenNew)
		_, command := m.Update(press)
		if command == nil {
			continue
		}
		if _, pastes := command().(tea.PasteMsg); pastes {
			reads = append(reads, name)
		}
	}
	if len(reads) != 1 || reads[0] != "ctrl+v" {
		t.Errorf("the chords that read a clipboard are %v, want exactly ctrl+v", reads)
	}
}

// handClipboard hands the program a clipboard holding one string, for the test's
// lifetime.
func handClipboard(t *testing.T, text string) {
	t.Helper()
	restore := clipboard.Read
	t.Cleanup(func() { clipboard.Read = restore })
	clipboard.Read = func() (string, error) { return text, nil }
}

// fieldValues is every text field of the character form, so a test can claim
// nothing moved without naming fifteen of them.
func fieldValues(m model) map[int]string {
	held := map[int]string{}
	for field := range m.form.inputs {
		held[field] = m.form.inputs[field].Value()
	}
	return held
}

package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/clipboard"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	draw "github.com/vukyn/hexarena/internal/screen"
)

// # Pasting into this client
//
// The bug this file exists for was reported here: ⌘V on the join screen with a
// room code on the clipboard, and nothing appeared. Bracketed paste is on by
// default under bubbletea v2, so a paste arrives as one tea.PasteMsg rather than
// as a run of key presses, and this model's Update switched on tea.KeyPressMsg
// and its own messages and nothing else — so the message died in the model and
// no field ever saw it. It compiled, it drew, and pasting did nothing.
//
// What is measured here is the **route**: which screen a paste reaches on this
// client and, much more of the work, which screens it must not. Where a paste
// lands once a screen has it is internal/screen's own suite.

// pasteOf is one bracketed paste as a terminal delivers it.
//
// ⚠️ **One message carrying the whole string.** A helper that sent a key per
// character would measure the typed path under another name, and every assertion
// below would pass with the route deleted — which is exactly the state this
// client shipped in.
func pasteOf(text string) tea.PasteMsg { return tea.PasteMsg{Content: text} }

// TestAPastedRoomCodeReachesTheFieldAndJoinsTheRoom is the reported bug, driven
// end to end against a real room on a loopback listener.
//
// *Sees:* the tea.PasteMsg arm deleted from Update, the join screen dropped from
// the route, joinScreen.Paste writing the wrong field, a paste that reached the
// field but left a value the dial refuses.
// *Cannot see:* that a real terminal sends the message. That is bubbletea's
// renderer enabling the mode, one dependency away.
func TestAPastedRoomCodeReachesTheFieldAndJoinsTheRoom(t *testing.T) {
	held, library := openARoom(t, 3)
	m, _ := joiningWithNothingTyped(t, held, library, i18n.Vi)

	m = send(t, m, pasteOf(string(held.code)))
	if got := m.join.Code.Value(); got != string(held.code) {
		t.Fatalf("the code field holds %q after a paste of %q", got, held.code)
	}

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter after a pasted code asked for no command, so no room was called")
	}
	if !m.join.Dialling {
		t.Fatal("the join screen is not calling anything after a pasted code")
	}
	if m.join.BadLength {
		t.Fatalf("the pasted code was refused for its length: %d characters", m.join.Mistyped)
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the dial from a pasted code was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	if m.screen != screenWaiting {
		t.Fatalf("a pasted code landed on screen %v, want the waiting room", m.screen)
	}
}

// TestAPastedRoomCodeWithATrailingNewlineStillJoins is the case a clipboard
// actually holds.
//
// ⚠️ **The field really does end up a character longer, and that is measured
// rather than worked around.** bubbles turns each newline into a space, so the
// value is thirteen characters; submit TrimSpaces before it measures the length,
// which is why the dial goes out anyway and why nothing trims on the way in. Both
// halves are asserted, because a paste route that quietly trimmed would pass the
// second half and make the comment on joinScreen.Paste a lie.
//
// *Sees:* a trim added to the paste route, submit's TrimSpace removed, bubbles
// changing what it does with a newline.
// *Cannot see:* what a particular terminal puts in the message — the newline is
// the clipboard's, not the terminal's.
func TestAPastedRoomCodeWithATrailingNewlineStillJoins(t *testing.T) {
	held, library := openARoom(t, 3)
	m, _ := joiningWithNothingTyped(t, held, library, i18n.Vi)

	m = send(t, m, pasteOf(string(held.code)+"\n"))
	if got, want := m.join.Code.Value(), string(held.code)+" "; got != want {
		t.Fatalf("a pasted code with a newline left the field holding %q, want %q", got, want)
	}

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if m.join.BadLength {
		t.Fatalf("a pasted code with a newline was refused for its length: %d characters",
			m.join.Mistyped)
	}
	if command == nil {
		t.Fatal("enter after a pasted code with a newline called no room")
	}
	if m.join.At != held.code {
		t.Errorf("the room being called is %q, want the code that was pasted %q",
			m.join.At, held.code)
	}
}

// TestAPasteReachesThePasswordWhenTheCursorIsOnIt is the other lobby field, and
// the one with something to lose.
//
// ⚠️ **A paste must not defeat the masking.** EchoMode is applied when the field
// is *drawn* rather than when it is filled, so a pasted password is starred like
// a typed one — asserted off the rendered screen rather than reasoned about,
// because "the library handles it" is what the whole of this file is about not
// assuming.
//
// *Sees:* the paste routed to the code field regardless of the cursor; a masking
// that only covers typed characters.
// *Cannot see:* wire.Password's redaction under fmt verbs, which covers the value
// travelling rather than the value drawn. That is internal/wire's.
func TestAPasteReachesThePasswordWhenTheCursorIsOnIt(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenJoin)
	m = key(t, m, "tab")
	if m.join.Field != joinFieldPassword {
		t.Fatalf("tab landed on field %d, want the password", m.join.Field)
	}

	const secret = "nhaminh"
	m = send(t, m, pasteOf(secret))
	if got := m.join.Password.Value(); got != secret {
		t.Fatalf("the password field holds %q after a paste", got)
	}
	if got := m.join.Code.Value(); got != "" {
		t.Errorf("the code field took %q from a paste aimed at the password", got)
	}
	body, _ := m.join.View(m.ctx())
	if strings.Contains(body, secret) {
		t.Error("the pasted password is drawn in the clear on the join screen")
	}
	if !strings.Contains(body, "*******") {
		t.Error("the pasted password is not masked on the join screen")
	}
}

// TestAPasteReachesTheSkillFilterOnThisClient is the one text target this client
// has on a screen out of internal/screen.
//
// *Sees:* the skills arm dropped from the route.
// *Cannot see:* what the narrowed listing draws. That is the listing's own tests.
func TestAPasteReachesTheSkillFilterOnThisClient(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSkills)
	m = key(t, m, "/")
	if !m.skills.Filtering {
		t.Fatal("`/` did not open the filter on this client")
	}
	m = send(t, m, pasteOf("long"))
	if m.skills.Query != "long" {
		t.Errorf("the filter holds %q after a paste", m.skills.Query)
	}
}

// TestAPasteWithNoFieldInFrontChangesNothingHere is the vacuity guard.
//
// ⚠️ **It is aimed at the squad catalogue and not at a screen with no fields.**
// A screen owning nothing would satisfy this with the whole route deleted. The
// squad catalogue owns three text fields and its level field is a NumberField,
// which focuses itself at construction — so this client really is drawing a
// catalogue with a focused text field behind it, and a route that asked only "is
// a field focused" would fill it. The battle is checked beside it as the other
// shape: a screen with no field at all, where the claim is that the model comes
// back unchanged rather than that some hidden value did not move.
//
// *Sees:* SquadsScreen.Pasting losing its mode check; a route that named the
// catalogue; a route that pasted into whatever screen was in front.
// *Cannot see:* a fifth screen growing a field — that is internal/screen's walk.
func TestAPasteWithNoFieldInFrontChangesNothingHere(t *testing.T) {
	m, _, _ := start(t, i18n.Vi)

	t.Run("the squad catalogue", func(t *testing.T) {
		at := m.enter(screenSquads)
		if !at.squads.LevelInput.Focused() {
			t.Fatal("the catalogue's level field is not focused, so this test measures nothing")
		}
		before := at.screenContent()
		after := send(t, at, pasteOf("42"))
		if got := after.squads.LevelInput.Value(); got != "" {
			t.Errorf("a paste on the catalogue filled the hidden level field with %q", got)
		}
		if got := after.squads.IDInput.Value(); got != "" {
			t.Errorf("a paste on the catalogue filled the hidden id field with %q", got)
		}
		if got := after.screenContent(); got != before {
			t.Error("a paste on the squad catalogue changed what it draws")
		}
	})

	t.Run("the battle", func(t *testing.T) {
		at := m.enter(screenBattle)
		before := at.screenContent()
		after := send(t, at, pasteOf(aPastedCode))
		if got := after.screenContent(); got != before {
			t.Error("a paste during a battle changed what it draws")
		}
	})

	t.Run("the menu", func(t *testing.T) {
		before := m.screenContent()
		after := send(t, m, pasteOf(aPastedCode))
		if after.menu != m.menu {
			t.Errorf("a paste on the menu moved the cursor to %d", after.menu)
		}
		if got := after.screenContent(); got != before {
			t.Error("a paste on the menu changed what it draws")
		}
	})
}

// TestNoPasteReachesAnAuthoringFieldInThisClient is the leak guard, and it is the
// paste-shaped twin of readonly_test.go's four.
//
// A paste is not a keystroke, so none of those four sees it: they press keys. The
// claim here is the same one they make — this client cannot reach a field that
// writes a data file — restated for the second input there now is.
//
// It walks **every** screen this client draws rather than the three that own a
// form, because the point is that no screen is the way in. Each is entered, the
// authoring keys are pressed at it (they do nothing, which is what those four
// tests are about), and a paste is driven at it; then every form field on all
// three shared screens is read and has to be empty.
//
// *Sees:* a route arm added for the works or squad forms; SkillsScreen.Paste
// reordered so its form arm is reached before FormInFront is consulted; a future
// screen that opened a form without Context.Authoring.
// *Cannot see:* a write that did not go through a field — there is none, because
// every save on those three screens reads the fields.
func TestNoPasteReachesAnAuthoringFieldInThisClient(t *testing.T) {
	reference, _, _ := start(t, i18n.Vi)
	// ⚠️ **A baseline rather than "every field is empty", and it had to be.**
	// SkillsScreen.ResetForm writes three defaults — a range, a strike count and a
	// cooldown — at construction, so an emptiness assertion failed on a client
	// that had done nothing at all. What is claimed is that a paste moves none of
	// them, which is the claim that was wanted and is the stronger one: a paste
	// that overwrote a default would show up here and an emptiness check would
	// have called it correct.
	want := authoringFields(reference)

	for at := screen(0); at < screenCount; at++ {
		m, _, _ := start(t, i18n.Vi)
		m = m.enter(at)
		for _, name := range []string{"a", "e", "n", "d", "enter"} {
			m = key(t, m, name)
			m = m.enter(at)
		}
		m = send(t, m, pasteOf(aPastedCode))

		for name, got := range authoringFields(m) {
			if got != want[name] {
				t.Errorf("a paste on screen %v changed %s from %q to %q", at, name, want[name], got)
			}
		}
		if m.skills.FormInFront() || m.works.Adding || m.squads.Mode != draw.SquadList {
			t.Errorf("a form is open on screen %v in a client that cannot author", at)
		}
	}
}

// authoringFields is every text field on this client that a save would read,
// named so a failure says which one moved.
//
// It walks the two slices rather than naming their entries: a form that grows a
// field is then covered without anybody remembering to add it here, which is the
// same reason the walk in internal/screen parses rather than lists.
func authoringFields(m model) map[string]string {
	held := map[string]string{}
	for field := range m.skills.Inputs {
		held[fmt.Sprintf("skill form field %d", field)] = m.skills.Inputs[field].Value()
	}
	for field := range m.works.Inputs {
		held[fmt.Sprintf("work form field %d", field)] = m.works.Inputs[field].Value()
	}
	held["the squad id"] = m.squads.IDInput.Value()
	held["the squad name"] = m.squads.NameInput.Value()
	held["the member level"] = m.squads.LevelInput.Value()
	// ⚠️ **The skill filter is deliberately NOT here.** It is a search rather than
	// an answer — no save reads it, and this client is *supposed* to be able to
	// paste into it. Listing it would turn the one text target a game client has
	// into a leak the day somebody opened the filter before pasting.
	return held
}

// TestCtrlVPastesTheClipboardIntoAFocusedField is the second route: the key a
// terminal does **not** intercept.
//
// ⌘V, ctrl+shift+V and right-click paste are all injected by the terminal as
// bracketed paste and are covered by the tests above. Plain ctrl+v reaches the
// program as an ordinary keystroke, so this client reads the clipboard itself and
// says the answer as the same tea.PasteMsg — which is what makes this test's
// second half a re-entry into the route already measured rather than a second
// insert path.
//
// ⚠️ **The reader is injected and the real clipboard is never driven.** A test
// that shelled out to pbpaste would fail on a machine with no helper and pass for
// the wrong reason on one whose clipboard happened to hold something. What that
// **cannot** see is that the real reader is still what is wired in;
// internal/clipboard's own test holds that by identity.
//
// *Sees:* the ctrl+v arm deleted; a ctrl+v that produced no command; a command
// whose message the route cannot name.
// *Cannot see:* that a terminal delivers ctrl+v as a key rather than swallowing
// it. Terminals that do swallow it send bracketed paste instead, which the tests
// above cover.
func TestCtrlVPastesTheClipboardIntoAFocusedField(t *testing.T) {
	handClipboard(t, aPastedCode)

	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenJoin)
	next, command := m.Update(press(t, "ctrl+v"))
	m = next.(model)
	if command == nil {
		t.Fatal("ctrl+v asked for no command, so no clipboard was ever read")
	}
	message := command()
	if _, named := message.(tea.PasteMsg); !named {
		t.Fatalf("ctrl+v produced a %T, which this model cannot name and would drop", message)
	}
	m = send(t, m, message)
	if got := m.join.Code.Value(); got != aPastedCode {
		t.Errorf("the code field holds %q after ctrl+v", got)
	}
}

// TestCtrlVWithNoFieldFocusedChangesNothing is the vacuity guard for the second
// route, aimed the same way the first one is: at a screen that owns a focused
// field it is not drawing.
//
// *Sees:* a ctrl+v that bypassed the route and wrote a field directly; the
// catalogue's mode check dropped.
// *Cannot see:* whether the clipboard was read. It is, deliberately — the read is
// unconditional and the landing is what refuses, so that where a paste may go is
// declared in one place.
func TestCtrlVWithNoFieldFocusedChangesNothing(t *testing.T) {
	handClipboard(t, aPastedCode)

	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenSquads)
	if !m.squads.LevelInput.Focused() {
		t.Fatal("the catalogue's level field is not focused, so this test measures nothing")
	}
	before := m.screenContent()

	next, command := m.Update(press(t, "ctrl+v"))
	m = next.(model)
	if command == nil {
		t.Fatal("ctrl+v asked for no command at all, so the route below is never reached")
	}
	m = send(t, m, command())
	if got := m.squads.LevelInput.Value(); got != "" {
		t.Errorf("ctrl+v on the catalogue filled the hidden level field with %q", got)
	}
	if got := m.screenContent(); got != before {
		t.Error("ctrl+v on the squad catalogue changed what it draws")
	}
}

// TestCtrlVWithNoClipboardHelperDoesNothingAtAll is the Linux box with no xclip.
//
// ⚠️ **The outcome has to be nothing — not a crash, not a hang, not a diagnostic
// on a game screen.** There is deliberately no wording for it: a keystroke
// somebody pressed by habit is not the place to explain a packaging decision, and
// ⌘V still works on that machine because that route never reads a clipboard.
//
// *Sees:* an error turned into a message the model would try to draw; a panic on
// a failed read; a clipboard error reaching a field as text.
// *Cannot see:* that atotto/clipboard really fails fast rather than blocking. It
// probes for its helpers at init, which is a dependency's behaviour and is
// recorded in internal/clipboard's doc comment rather than measured here.
func TestCtrlVWithNoClipboardHelperDoesNothingAtAll(t *testing.T) {
	restore := clipboard.Read
	t.Cleanup(func() { clipboard.Read = restore })
	clipboard.Read = func() (string, error) {
		return "", errors.New("no clipboard utilities available")
	}

	m, _, _ := start(t, i18n.Vi)
	m = m.enter(screenJoin)
	before := m.screenContent()

	next, command := m.Update(press(t, "ctrl+v"))
	m = next.(model)
	if command == nil {
		t.Fatal("ctrl+v asked for no command, so the failing read is never reached")
	}
	if message := command(); message != nil {
		t.Fatalf("a failed clipboard read produced %#v, want no message at all", message)
	}
	if got := m.join.Code.Value(); got != "" {
		t.Errorf("a failed clipboard read left %q in the code field", got)
	}
	if got := m.screenContent(); got != before {
		t.Error("a failed clipboard read changed what the join screen draws")
	}
}

// TestAnEmptyClipboardIsTheSameAsAFailedOne pins the one arm the two share.
//
// *Sees:* an empty read turned into a message, which would cost a redraw and a
// route walk for nothing.
func TestAnEmptyClipboardIsTheSameAsAFailedOne(t *testing.T) {
	handClipboard(t, "")
	if message := clipboard.Paste(); message != nil {
		t.Fatalf("an empty clipboard produced %#v, want no message at all", message)
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

// aPastedCode is a string of the shape this feature was reported against: a
// twelve-character room code. It is not a real room's — the tests that need one
// take it off the room they opened.
const aPastedCode = "7QK4M2XZ9BTF"

// joiningWithNothingTyped is `joining` with the code left out, because these
// tests put it there by pasting rather than by typing.
func joiningWithNothingTyped(t *testing.T, held *aRoom, library *forge.Library, lang i18n.Lang) (model, *fakeSender) {
	t.Helper()
	fake := newFakeSender()
	sess := newSession()
	sess.attach(fake)
	m := newModel(library, lang, sess)
	m.width, m.height = 120, 44
	m = m.enter(screenJoin)
	if len(m.join.Squads) == 0 {
		t.Fatal("the join screen found no side to bring")
	}
	if m.join.Code.Value() != "" {
		t.Fatalf("the code field opened holding %q, so a paste into it measures nothing",
			m.join.Code.Value())
	}
	return m, fake
}

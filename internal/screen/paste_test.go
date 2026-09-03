package screen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/vukyn/hexarena/internal/i18n"
)

// What a pasted string does to the four screens in this package that own a text
// field, and — much more of the work — what it does to the ones that do not.
//
// The clients' halves stay in their own packages: which screen a paste is routed
// to, and that ctrl+v reads a clipboard at all, are decisions a model makes.

// pasted is one bracketed paste as a terminal delivers it.
//
// ⚠️ **One message carrying the whole string**, which is the difference this
// whole feature turns on: a helper that sent a key per character would be
// measuring the typed path under another name, and every assertion below would
// pass with the paste route deleted.
func pasted(text string) tea.PasteMsg { return tea.PasteMsg{Content: text} }

// aRoomCode is a twelve-character room code, which is the string this feature
// was reported against.
const aRoomCode = "7QK4M2XZ9BTF"

// TestATextFieldTakesAPasteWhereItTakesAKeystroke is the positive claim, over
// every field-bearing screen in this package at once.
//
// *Sees:* the message never reaching the field, the wrong field being chosen,
// the form's dirty flag not being set by a paste, the number rule refusing a
// number.
// *Cannot see:* which screen a client routes a paste to, or that a terminal
// really sends one. Both are the clients' own suites.
func TestATextFieldTakesAPasteWhereItTakesAKeystroke(t *testing.T) {
	c, _ := start(t, i18n.Vi)

	t.Run("the origin form's id", func(t *testing.T) {
		o := onOrigins(t, c, NewOriginsScreen(c), "a")
		if !o.Inputs[OriginFieldID].Focused() {
			t.Fatal("the form opened with no field focused")
		}
		o, _ = o.Paste(c, "mot-tac-pham")
		if got := o.Inputs[OriginFieldID].Value(); got != "mot-tac-pham" {
			t.Errorf("the id field holds %q after a paste, want the pasted id", got)
		}
		if !o.Touched {
			t.Error("a paste left the form untouched, so escape would throw it away in silence")
		}
	})

	t.Run("the skill form's id", func(t *testing.T) {
		s := pressOn(t, c, NewSkillsScreen(c), "a")
		if !s.Inputs[SkillFieldID].Focused() {
			t.Fatal("the form opened with no field focused")
		}
		s, _ = s.Paste(c, "mot-chieu")
		if got := s.Inputs[SkillFieldID].Value(); got != "mot-chieu" {
			t.Errorf("the id field holds %q after a paste, want the pasted id", got)
		}
		if !s.Touched {
			t.Error("a paste left the form untouched, so escape would throw it away in silence")
		}
	})

	t.Run("the skill listing's typed filter", func(t *testing.T) {
		// ⚠️ The one text target in this package a **game** client reaches: the
		// filter opens on `/` and is guarded by nothing, while the form above it
		// opens behind Context.Authoring.
		s := pressOn(t, c, NewSkillsScreen(c), "/")
		if !s.Filtering {
			t.Fatal("`/` did not open the filter")
		}
		s, _ = s.Paste(c, "lua")
		if s.Query != "lua" {
			t.Errorf("the query is %q after a paste, want the pasted word", s.Query)
		}
	})

	t.Run("the squad builder's id and then its name", func(t *testing.T) {
		// ⚠️ **Both, because EditingID decides which of the two a paste reaches
		// and Begin opens on the id.** A test that pasted once would pass with
		// editField hard-wired to whichever field it happened to name.
		s := onSquads(t, c, NewSquadsScreen(c), "n")
		if s.Mode != SquadEdit {
			t.Fatalf("`n` landed on mode %v, want the builder", s.Mode)
		}
		if !s.IDInput.Focused() {
			t.Fatal("the builder opened with no field focused")
		}
		s, _ = s.Paste(c, "phe-dan")
		if s.IDInput.Value() != "phe-dan" {
			t.Errorf("the id field holds %q after a paste", s.IDInput.Value())
		}
		if s.Editing.ID != "phe-dan" {
			t.Errorf("the squad in hand has id %q, so the paste reached the field and "+
				"not the squad", s.Editing.ID)
		}
		s = onSquads(t, c, s, "tab")
		if s.EditingID {
			t.Fatal("tab did not move the keyboard off the id")
		}
		s, _ = s.Paste(c, "đội dán")
		if s.NameInput.Value() != "đội dán" {
			t.Errorf("the name field holds %q after a paste", s.NameInput.Value())
		}
		if s.Editing.Name != "đội dán" {
			t.Errorf("the squad in hand is named %q, so the paste reached the field and "+
				"not the squad", s.Editing.Name)
		}
		if s.IDInput.Value() != "phe-dan" {
			t.Errorf("the second paste changed the id to %q, so both landed in one field",
				s.IDInput.Value())
		}
	})

	t.Run("a member's level", func(t *testing.T) {
		s := aMemberUnderEdit(t, c)
		s = s.moveField(0)
		for s.Field != SquadUnitLevel {
			s = s.moveField(1)
		}
		if !s.LevelInput.Focused() {
			t.Fatal("the level row is not focused")
		}
		s.LevelInput.SetValue("")
		s, _ = s.Paste(c, "42")
		if s.LevelInput.Value() != "42" {
			t.Errorf("the level field holds %q after a paste", s.LevelInput.Value())
		}
		if s.Unit.Level != 42 {
			t.Errorf("the member is level %d, so the paste reached the field and not the "+
				"member", s.Unit.Level)
		}
	})

	t.Run("a picker's chance", func(t *testing.T) {
		s := pressOn(t, c, NewSkillsScreen(c), "a")
		for s.Field != SkillFieldInflicts {
			s = s.moveTo(s.Field + 1)
		}
		_, action, _ := s.Update(c, press(t, "space"))
		picker := action.Picker
		if picker == nil || picker.Typed == nil {
			t.Fatal("the status picker came up without its chance field")
		}
		picker.Typed.SetValue("")
		picker.Paste(c, "250")
		if picker.Typed.Value() != "250" {
			t.Errorf("the chance field holds %q after a paste", picker.Typed.Value())
		}
	})
}

// TestAPasteOnAScreenWithNoFieldInFrontChangesNothing is the vacuity guard, and
// it is aimed at the two screens that carry a **focused** field while drawing
// something else entirely.
//
// ⚠️ **That is the whole point of where it is aimed.** A screen with no fields at
// all would satisfy this test with the paste route deleted, with `Pasting`
// returning true always, and with every mode check removed — it would measure
// nothing. OriginsScreen and SkillsScreen focus `Inputs[0]` in ResetForm, which
// runs at construction, and SquadsScreen's level field is a NumberField, which
// focuses itself: so all three of these listings really do have a focused text
// field behind them, and a route that asked only "is a field focused" would fill
// it.
//
// *Sees:* the mode check dropped from any of the three Pasting methods, a client
// that routed a paste at a focused field regardless of what is drawn.
// *Cannot see:* a fifth screen growing a field — that is the walk at the bottom
// of this file.
func TestAPasteOnAScreenWithNoFieldInFrontChangesNothing(t *testing.T) {
	c, _ := start(t, i18n.Vi)

	t.Run("the works catalogue", func(t *testing.T) {
		o := NewOriginsScreen(c)
		if !o.Inputs[OriginFieldID].Focused() {
			t.Fatal("the hidden form's id field is not focused, so this test measures nothing")
		}
		next, command := o.Paste(c, aRoomCode)
		if got := next.Inputs[OriginFieldID].Value(); got != "" {
			t.Errorf("a paste on the catalogue filled the hidden form's id with %q", got)
		}
		if next.Touched || command != nil {
			t.Error("a paste on the catalogue moved the form's state or asked for a command")
		}
	})

	t.Run("the skill listing", func(t *testing.T) {
		s := NewSkillsScreen(c)
		if !s.Inputs[SkillFieldID].Focused() {
			t.Fatal("the hidden form's id field is not focused, so this test measures nothing")
		}
		next, command := s.Paste(c, aRoomCode)
		if got := next.Inputs[SkillFieldID].Value(); got != "" {
			t.Errorf("a paste on the listing filled the hidden form's id with %q", got)
		}
		if next.Query != "" {
			t.Errorf("a paste on the listing filled a filter that is not open with %q", next.Query)
		}
		if next.Touched || command != nil {
			t.Error("a paste on the listing moved the form's state or asked for a command")
		}
	})

	t.Run("the squad catalogue", func(t *testing.T) {
		s := NewSquadsScreen(c)
		if !s.LevelInput.Focused() {
			t.Fatal("the level field is not focused on a fresh catalogue, so this test " +
				"measures nothing")
		}
		next, command := s.Paste(c, "42")
		if got := next.LevelInput.Value(); got != "" {
			t.Errorf("a paste on the catalogue filled the level field with %q", got)
		}
		if command != nil {
			t.Error("a paste on the catalogue asked for a command")
		}
	})

	t.Run("a chooser row inside an open form", func(t *testing.T) {
		// The other half of the rule, and the one bubbles cannot help with: the
		// form **is** in front, and the keyboard is on a row that is not a field.
		o := onOrigins(t, c, NewOriginsScreen(c), "a")
		for o.Field != OriginFieldMedium {
			o = o.moveTo(o.Field + 1)
		}
		if o.Inputs[o.Field].Focused() {
			t.Fatal("the medium chooser row is focused, so this measures the wrong thing")
		}
		before := o.Inputs[OriginFieldID].Value()
		next, command := o.Paste(c, aRoomCode)
		for field := range next.Inputs {
			if next.Inputs[field].Value() != "" {
				t.Errorf("a paste on the medium chooser filled field %d with %q",
					field, next.Inputs[field].Value())
			}
		}
		if next.Inputs[OriginFieldID].Value() != before || command != nil {
			t.Error("a paste on a chooser row moved a field or asked for a command")
		}
	})

	t.Run("a picker with a description in front of the list", func(t *testing.T) {
		s := pressOn(t, c, NewSkillsScreen(c), "a")
		for s.Field != SkillFieldInflicts {
			s = s.moveTo(s.Field + 1)
		}
		_, action, _ := s.Update(c, press(t, "space"))
		picker := action.Picker
		if picker == nil || picker.Typed == nil {
			t.Fatal("the status picker came up without its chance field")
		}
		picker.Typed.SetValue("")
		picker.Reading = true
		if command := picker.Paste(c, "250"); command != nil {
			t.Error("a paste behind a description asked for a command")
		}
		if got := picker.Typed.Value(); got != "" {
			t.Errorf("a paste behind a description filled the chance field with %q", got)
		}
	})
}

// TestANumberFieldRefusesAPasteThatIsNotAllDigits is the rule the level and the
// chance are under, and the reason it is a rule at all.
//
// *Sees:* PasteDigits swapped for PasteInto on either field, a stripping
// implementation that pulled the digits out of "1,200".
// *Cannot see:* whether a reader is told the paste was refused. They are not,
// deliberately — a keystroke that does nothing is what a number field does with
// a letter today.
func TestANumberFieldRefusesAPasteThatIsNotAllDigits(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	for _, text := range []string{"abc", "1,200", "42 ", "-1", "4.2", ""} {
		t.Run(text, func(t *testing.T) {
			s := aMemberUnderEdit(t, c)
			for s.Field != SquadUnitLevel {
				s = s.moveField(1)
			}
			s.LevelInput.SetValue("")
			s.Unit.Level = 7
			next, command := s.Paste(c, text)
			if got := next.LevelInput.Value(); got != "" {
				t.Errorf("the level field took %q from a paste of %q", got, text)
			}
			if next.Unit.Level != 7 {
				t.Errorf("the member became level %d from a paste of %q", next.Unit.Level, text)
			}
			if command != nil {
				t.Errorf("a refused paste of %q asked for a command", text)
			}
		})
	}
}

// TestATextFieldTurnsANewlineInAPasteIntoASpace pins what bubbles does with the
// thing a pasted room code carries most often.
//
// ⚠️ **It is a measurement of a dependency and that is the point.** The join
// screen's submit trims before it measures a code's length *because* of this, and
// PasteInto's comment quotes these exact figures; a bubbles release that changed
// the sanitiser would otherwise show up as a room code that would not decode,
// three layers away from the line that changed.
//
// *Sees:* the sanitiser changing its replacement, a paste route that trimmed on
// the way in and made the comment above a lie.
// *Cannot see:* what a real terminal puts in the message. A terminal's bracketed
// paste carries the clipboard's bytes and the newline is the clipboard's.
func TestATextFieldTurnsANewlineInAPasteIntoASpace(t *testing.T) {
	for _, one := range []struct{ pasted, want string }{
		{aRoomCode, aRoomCode},
		{aRoomCode + "\n", aRoomCode + " "},
		{aRoomCode + "\r\n", aRoomCode + "  "},
		{"AB\nCD", "AB CD"},
		{"a\tb", "a b"},
		{"a\x1bb", "ab"},
	} {
		field := NewInput(true)
		field.CharLimit = 64
		field.Focus()
		PasteInto(&field, one.pasted)
		if got := field.Value(); got != one.want {
			t.Errorf("a paste of %q left the field holding %q, want %q", one.pasted, got, one.want)
		}
	}
}

// TestATextFieldRefusesAPasteWhenItIsNotFocused pins the backstop the four
// Pasting methods lean on.
//
// It is bubbles' own early return, not this package's rule — which is exactly why
// it is written down: it is relied on, it is a dependency's behaviour, and it is
// the reason none of the code here re-checks Focused after choosing a field.
//
// *Sees:* a bubbles release that dropped the focus guard, which would turn every
// "chooses the wrong field" bug in this package from harmless into a silent
// write.
// *Cannot see:* a screen handing over the wrong **focused** field. Nothing in
// bubbles can see that, which is why the mode checks are ours.
func TestATextFieldRefusesAPasteWhenItIsNotFocused(t *testing.T) {
	// ⚠️ **Dressed for a COLOURED terminal, and that is what makes the command
	// half of this able to fail.** NewInput(true) turns the virtual cursor off, so
	// a field on a plain terminal answers every paste with a nil command whether
	// it took the text or not — the assertion below would then hold for a field
	// that pasted happily. Every other test in this package runs plain.
	focused := NewInput(false)
	focused.CharLimit = 64
	focused.Focus()
	if command := PasteInto(&focused, aRoomCode); command == nil {
		t.Fatal("a focused field with a cursor asked for no blink, so the command " +
			"assertion below measures nothing")
	}

	blurred := NewInput(false)
	blurred.CharLimit = 64
	blurred.Blur()
	command := PasteInto(&blurred, aRoomCode)
	if got := blurred.Value(); got != "" {
		t.Errorf("a blurred field took %q from a paste", got)
	}
	if command != nil {
		t.Error("a blurred field asked for a cursor blink it has no cursor for")
	}
}

// TestTheFilterSanitisesAPasteExactlyAsATextFieldDoes holds the one place this
// package sanitises text itself equal to the library that sanitises everywhere
// else.
//
// The filter is a plain string rather than a textinput — SkillsScreen.Query says
// why — so it is the one target PasteInto cannot serve, and PasteText is
// therefore a second implementation of one rule. This is what stops the two
// drifting: the same strings through both, held equal.
//
// *Sees:* either side changing — a bubbles upgrade that replaced newlines with
// something else, or a hand-rolled predicate here that missed a control range.
// *Cannot see:* the CharLimit half. A textinput truncates by its own limit and
// the filter by FilterLimit, so the strings below are kept short enough that
// neither bites, and the filter's own truncation is measured separately.
func TestTheFilterSanitisesAPasteExactlyAsATextFieldDoes(t *testing.T) {
	for _, text := range []string{
		aRoomCode, aRoomCode + "\n", "AB\nCD", "a\tb", "a\x1bb", "a\rb",
		"tên có dấu", "a\x00b", "a\x7fb", "a\x9bb", "", "  spaced  ",
	} {
		field := NewInput(true)
		field.CharLimit = 200
		field.Focus()
		PasteInto(&field, text)
		if got, want := PasteText(text), field.Value(); got != want {
			t.Errorf("PasteText(%q) is %q and a text field makes it %q", text, got, want)
		}
	}
}

// TestTheFilterActuallySanitisesWhatIsPastedIntoIt is the gap the test above
// leaves, and it was found by mutation rather than by reading.
//
// ⚠️ **Holding PasteText equal to a text field says nothing about whether
// pasteFilter calls it.** Replacing `PasteText(text)` with a bare `text` left the
// whole suite green: the equality test still passed, because it exercises
// PasteText directly, and every other filter test pastes a string with nothing to
// sanitise in it. A query holding a real newline would then be drawn as two rows
// on a screen that budgets one.
//
// *Sees:* the sanitiser dropped from the filter's paste arm — which is precisely
// what nothing else here could.
// *Cannot see:* whether the sanitising rule is the right one. That is the
// equality test above.
func TestTheFilterActuallySanitisesWhatIsPastedIntoIt(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	for _, one := range []struct{ pasted, want string }{
		{"lua\n", "lua "},
		{"lua\tdao", "lua dao"},
		{"lua\x1bdao", "luadao"},
		{"lua\r\ndao", "lua  dao"},
	} {
		s := pressOn(t, c, NewSkillsScreen(c), "/")
		s, _ = s.Paste(c, one.pasted)
		if s.Query != one.want {
			t.Errorf("a filter paste of %q left the query %q, want %q",
				one.pasted, s.Query, one.want)
		}
		if strings.ContainsAny(s.Query, "\n\r\t\x1b") {
			t.Errorf("the query holds a control character after a paste of %q: %q",
				one.pasted, s.Query)
		}
	}
}

// TestAPasteIntoTheFilterIsTruncatedRatherThanRefused is the other half of the
// number field's rule, and the opposite answer for a stated reason.
//
// *Sees:* the truncation dropped, which would let a paste push the query past a
// limit the typed path enforces; and a refusal put in its place, which would make
// a long clipboard unsearchable.
// *Cannot see:* how the narrowed listing draws. That is the listing's own tests.
func TestAPasteIntoTheFilterIsTruncatedRatherThanRefused(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	s := pressOn(t, c, NewSkillsScreen(c), "/")
	long := strings.Repeat("x", FilterLimit+10)
	s, _ = s.Paste(c, long)
	if got := len([]rune(s.Query)); got != FilterLimit {
		t.Errorf("a %d-letter paste left a %d-letter query, want it held at %d",
			len(long), got, FilterLimit)
	}
	if s.Cursor < 0 {
		t.Errorf("the cursor is %d after a query that matches nothing", s.Cursor)
	}
}

// TestEveryTypeThatOwnsATextFieldTakesAPaste is the walk, and it is the answer
// to "a fifth screen grows a field and nobody notices".
//
// ⚠️ **A per-screen test is four tests a fifth screen slips past**, which
// TODO.md records having happened five separate times to screens missing from a
// sweep. So this does not read a written list: it parses **every non-test source
// file in the module**, finds every type declaring a field of bubbles'
// textinput.Model — as a value, a pointer or a slice — and holds that set equal
// to the set of types with a Paste method. A screen that grows a field and no
// route fails here, in the commit that adds the field.
//
// It walks the module rather than this package for the reason
// internal/socket's clock allowlist does: three of these types live in the two
// clients, a directory-scoped walk would see none of them, and a walk per package
// is three walks that can each go stale on their own.
//
// ⚠️ **Dot-directories are skipped**, copied from that walk: .claude/worktrees
// holds other checkouts of this repository, and a field on another branch is not
// a field here.
//
// ⚠️ **Types are keyed by DIRECTORY rather than by package name.** Both clients
// are `package main`, so a name-keyed walk collapses cmd/hexforge-tui's model and
// cmd/hexarena-tui's into one entry — and an exception written for one of them
// would silently excuse the other.
//
// *Sees:* a new field-bearing screen with no Paste; a Paste deleted from one that
// has a field; a type whose fields were removed and whose Paste was left behind.
// *Cannot see:* whether a Paste is **routed** — a method nothing calls satisfies
// this. Each client's own suite drives the route through the real model, and the
// count assertion below is what stops this walk quietly matching nothing.
func TestEveryTypeThatOwnsATextFieldTakesAPaste(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("find the module root: %v", err)
	}
	scanned := 0
	fields := map[string]string{}
	pastes := map[string]bool{}
	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		for _, declared := range file.Decls {
			switch typed := declared.(type) {
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					named, isType := spec.(*ast.TypeSpec)
					if !isType {
						continue
					}
					structure, isStruct := named.Type.(*ast.StructType)
					if !isStruct {
						continue
					}
					if holdsATextField(structure) {
						fields[filepath.ToSlash(filepath.Dir(relative))+"."+named.Name.Name] = relative
					}
				}
			case *ast.FuncDecl:
				if typed.Recv == nil || len(typed.Recv.List) == 0 {
					continue
				}
				if !strings.EqualFold(typed.Name.Name, "paste") {
					continue
				}
				if owner := receiverName(typed.Recv.List[0].Type); owner != "" {
					pastes[filepath.ToSlash(filepath.Dir(relative))+"."+owner] = true
				}
			}
		}
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatalf("walk the module: %v", err)
	}
	// A walk that read nothing, or whose predicate has stopped matching, agrees
	// with any claim at all.
	if scanned == 0 {
		t.Fatal("the walk read no source files, so it measures nothing")
	}
	if len(fields) == 0 {
		t.Fatal("the walk found no type holding a textinput in the whole module, and there " +
			"are six: it is measuring nothing")
	}
	for _, owner := range slices.Sorted(maps.Keys(fields)) {
		if !pastes[owner] {
			t.Errorf("%s (%s) holds a text field and has no Paste method, so every paste "+
				"into it is dropped in the model. That is the defect this file exists for",
				owner, fields[owner])
		}
	}
	for _, owner := range slices.Sorted(maps.Keys(pastes)) {
		if _, holds := fields[owner]; holds {
			continue
		}
		if because, routes := pasteRouters[owner]; routes {
			t.Logf("%s pastes and holds no field: %s", owner, because)
			continue
		}
		t.Errorf("%s has a Paste method and holds no text field; a paste with nowhere "+
			"to go is a route nobody will ever fail on. If it is a router rather "+
			"than a target, put it on pasteRouters with the reason", owner)
	}
	t.Logf("walked %d source files, %d types hold a text field", scanned, len(fields))
}

// pasteRouters is every type that answers a paste and holds no field of its own,
// with the reason. There are exactly two and they are the same thing twice: a
// client's model owns no text field — every one of them is on a screen — and what
// its paste method does is decide which screen the string is for. That decision
// has to live somewhere and a model is where every other routing decision in
// these two binaries lives.
//
// An entry here is a deliberate act with a red test in the way, which is the same
// shape internal/socket's clock allowlist is written in.
var pasteRouters = map[string]string{
	"cmd/hexarena-tui.model": "the game client's route: the lobby's two fields and " +
		"the skill filter, and nothing else — it holds no field itself",
	"cmd/hexforge-tui.model": "the authoring client's route: the guard, the picker " +
		"and four screens — it holds no field itself",
}

// holdsATextField reports whether a struct declares a bubbles textinput, however
// it is held: a value, a pointer, or a slice of them. All three are in the tree.
func holdsATextField(structure *ast.StructType) bool {
	held := false
	for _, field := range structure.Fields.List {
		ast.Inspect(field.Type, func(node ast.Node) bool {
			selector, isSelector := node.(*ast.SelectorExpr)
			if !isSelector || selector.Sel == nil || selector.Sel.Name != "Model" {
				return true
			}
			from, named := selector.X.(*ast.Ident)
			if named && from.Name == "textinput" {
				held = true
			}
			return true
		})
	}
	return held
}

// receiverName is the type a method hangs off, with any pointer taken away.
func receiverName(receiver ast.Expr) string {
	if star, isPointer := receiver.(*ast.StarExpr); isPointer {
		receiver = star.X
	}
	if named, isName := receiver.(*ast.Ident); isName {
		return named.Name
	}
	return ""
}

// aMemberUnderEdit is the squad builder three depths down, with one member open.
func aMemberUnderEdit(t *testing.T, c Context) SquadsScreen {
	t.Helper()
	s := onSquads(t, c, NewSquadsScreen(c), "n")
	next, _, _ := s.Update(c, press(t, "enter"))
	if next.Mode != SquadUnit {
		t.Fatalf("enter on an empty squad landed on mode %v, want a member under edit", next.Mode)
	}
	return next
}

// A compile-time check that the field the fixtures reach for really is bubbles'.
var _ textinput.Model = NewInput(true)

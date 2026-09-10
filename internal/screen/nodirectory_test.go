package screen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// A library with **no directory at all** is a state that did not exist until
// cmd/hexarena-tui learned to run off the embedded books, and it is not the same
// state as a directory whose files are absent. The screens here have to tell the
// two apart, because the sentence each deserves is different in kind:
//
//	a directory, no picture   an author forgot to draw one   MISSING, in red
//	no directory at all       the binary carries no art      a sentence, dimmed
//
// ⚠️ **The second is the easy half and the first is the one that rots.** A fix
// that draws the player's sentence in *both* states is a screen that reads
// perfectly, ships, and quietly deletes the author's warning — every test that
// only checks "the player is not accused" stays green through it. So each test
// below is a table over both states and asserts, in each, both what is said and
// what is **not**.

// overTheEmbeddedBooks is the same drawing context reading the copy go:embed
// baked into the binary: every book present, no directory anywhere.
//
// It asserts the premise rather than assuming it. A library that turned out to
// have a directory would make every arm below a second copy of the on-disk arm,
// which is the shape that agrees with any claim at all.
func overTheEmbeddedBooks(t *testing.T, c Context) Context {
	t.Helper()
	embedded, err := forge.LoadEmbedded()
	if err != nil {
		t.Fatalf("load the embedded copy: %v", err)
	}
	if embedded.HasDataDirectory() {
		t.Fatal("the embedded library reports a data directory, so nothing below is " +
			"measuring the state it names")
	}
	c.Lib = embedded
	return c
}

// booksWithoutArt is a real data directory holding the books and no pictures,
// which is the author's half of the pair: art that has not been drawn yet.
//
// The books are copied out of the shipped directory by hand rather than through
// testfixture.Data, and the assets are simply never brought over. Two reasons,
// and the second is the one that decided it: a fixture directory **shares** the
// shipped pictures rather than copying them, so producing this state by deleting
// them afterwards would be a test unlinking names inside the tree five other
// suites are sharing from — safe as written and one refactor away from not
// being. Not copying something cannot damage it.
func booksWithoutArt(t *testing.T) *forge.Library {
	t.Helper()
	target := t.TempDir()
	entries, err := os.ReadDir(shippedDataDir)
	if err != nil {
		t.Fatalf("read %s: %v", shippedDataDir, err)
	}
	books := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(shippedDataDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", entry.Name(), err)
		}
		books++
	}
	if books == 0 {
		t.Fatalf("%s handed over no books, so this library is not the state it claims",
			shippedDataDir)
	}
	lib, err := forge.Load(target)
	if err != nil {
		t.Fatalf("load a books-only directory: %v", err)
	}
	if !lib.HasDataDirectory() {
		t.Fatal("a library loaded from a real directory says it has none")
	}
	return lib
}

// firstCharacterOf is whoever the browser opens on, at the level cap, which is
// the row every other record in this package is taken at.
func firstCharacterOf(t *testing.T, c Context) Subject {
	t.Helper()
	browser := NewBrowseScreen(c.Lib)
	browser.Level = progression.LevelCap
	subject := browser.Subject()
	if subject.Of == 0 {
		t.Fatal("the cast has no rows, so there is nobody to draw")
	}
	return subject
}

// TestTheArtPreviewSaysMissingOnlyWhereThereIsADirectoryToBeMissingFrom is the
// wording pair on the screen whose whole subject is the picture.
//
// *Sees:* a player told MISSING; an author's warning replaced by the player's
// sentence; either wording drawn in the wrong language; a preview that stopped
// drawing the character at all in the state with no picture.
// *Cannot see:* the styling. Both lines are rendered under NO_COLOR here, so
// "dim rather than bad" is a claim this file cannot make — what it can make is
// that the two states say different things, and the style follows the wording.
func TestTheArtPreviewSaysMissingOnlyWhereThereIsADirectoryToBeMissingFrom(t *testing.T) {
	for _, lang := range i18n.Langs() {
		base, _ := start(t, lang)
		author := base
		author.Lib = booksWithoutArt(t)
		player := overTheEmbeddedBooks(t, base)

		for name, c := range map[string]Context{"an author": author, "a player": player} {
			preview := NewPreviewScreen()
			preview.Subject = firstCharacterOf(t, c)
			drawn, _ := preview.View(c)

			said, refused := c.Text(i18n.ArtMissing), c.Text(i18n.PreviewArtNotShipped)
			if name == "a player" {
				said, refused = refused, said
			}
			if !strings.Contains(drawn, said) {
				t.Errorf("%s | %s: the preview does not say %q:\n%s", lang, name, said, drawn)
			}
			if strings.Contains(drawn, refused) {
				t.Errorf("%s | %s: the preview says %q, which belongs to the other state:\n%s",
					lang, name, refused, drawn)
			}
			// Still the ordinary screen rather than an error page, in both
			// states: without this the two arms above would be satisfied by a
			// screen holding nothing but its own verdict.
			if id := preview.Subject.ID; !strings.Contains(drawn, id) {
				t.Errorf("%s | %s: the preview does not name %s:\n%s", lang, name, id, drawn)
			}
		}
	}
}

// TestTheCastRowSaysMissingOnlyWhereThereIsADirectoryToBeMissingFrom is the same
// pair on the browser's detail pane, which is the **fourth** site rather than
// one of the three the art step set out to fix.
//
// ⚠️ It was missed because the row reads as the authoring tool's: `artLine` is
// the tool's own art verdict and its wording is shared with the check screen.
// But internal/screen has one browser and both clients draw it, so a player
// walking the cast sees this row one keystroke before they see the preview — and
// it was saying MISSING to them in red too.
//
// Three arms rather than two, because this row has a third answer nothing else
// has: a picture that really is on disk.
func TestTheCastRowSaysMissingOnlyWhereThereIsADirectoryToBeMissingFrom(t *testing.T) {
	for _, lang := range i18n.Langs() {
		drawnArt, _ := start(t, lang)
		author := drawnArt
		author.Lib = booksWithoutArt(t)
		player := overTheEmbeddedBooks(t, drawnArt)

		arms := map[string]struct {
			c    Context
			says i18n.Key
		}{
			"art on disk": {drawnArt, i18n.ArtPresent},
			"an author":   {author, i18n.ArtMissing},
			"a player":    {player, i18n.ArtNotShipped},
		}
		for name, arm := range arms {
			browser := NewBrowseScreen(arm.c.Lib)
			browser.Level = progression.LevelCap
			drawn, _ := browser.View(arm.c)
			row := theArtRow(t, arm.c, drawn)
			if want := arm.c.Text(arm.says); !strings.Contains(row, want) {
				t.Errorf("%s | %s: the cast row's art verdict is not %q: %q",
					lang, name, want, row)
			}
			for other, verdict := range map[string]i18n.Key{
				"art on disk": i18n.ArtPresent,
				"an author":   i18n.ArtMissing,
				"a player":    i18n.ArtNotShipped,
			} {
				if other == name {
					continue
				}
				if word := arm.c.Text(verdict); strings.Contains(row, word) {
					t.Errorf("%s | %s: the row also says %q, which is %s's answer: %q",
						lang, name, word, other, row)
				}
			}
		}
	}
}

// theArtRow is the one line of a cast detail pane that carries the picture and
// the verdict on it.
//
// ⚠️ **The verdicts have to be matched on that line and not on the page**, and
// the shortest of them is why: `ArtPresent` in Vietnamese is `có`, which is two
// letters of ordinary prose and appears in most biographies in the book. Read
// against the whole pane, every arm of the table above says every verdict, and
// the sweep that is supposed to catch a state borrowing another state's word
// catches nothing at all. Measured — it is how this helper came to exist.
func theArtRow(t *testing.T, c Context, drawn string) string {
	t.Helper()
	label := c.Text(i18n.LabelArt)
	for _, line := range strings.Split(drawn, "\n") {
		if strings.Contains(line, label) && strings.Contains(line, "/") {
			return line
		}
	}
	t.Fatalf("the detail pane holds no %q row at all:\n%s", label, drawn)
	return ""
}

// TestABattleOffersNoSaveWhereThereIsNowhereToWriteOne is the decision on the
// battle log: a player on the embedded books is not offered the save at all.
//
// An offer that can only fail is worse than no offer. forge.Library.SaveBattleLog
// refuses a library with no directory, so the whole of what pressing the key
// could achieve was ErrNoDataDirectory drawn in red — *"this library was built
// from the copy embedded in the binary"* — which is a sentence about how the
// program was compiled, shown to somebody who pressed a key the footer had just
// named.
//
// *Sees:* a footer promising a key the screen ignores; a key still writing an
// error line; either half fixed without the other; the offer withdrawn from
// everybody.
func TestABattleOffersNoSaveWhereThereIsNowhereToWriteOne(t *testing.T) {
	for _, lang := range i18n.Langs() {
		onDisk, _ := start(t, lang)
		embedded := overTheEmbeddedBooks(t, onDisk)

		// The offer, in the two states, on both footers that carry it. The
		// finished battle has a footer of its own — it names the save key in a
		// different clause — so a fix applied to one of them is a fix applied to
		// half the screen's life.
		for name, build := range map[string]func(Context) PlayScreen{
			"a turn":  func(c Context) PlayScreen { return atABattleOf(t, c, 1) },
			"over":    func(c Context) PlayScreen { return playedOut(t, c, atABattleOf(t, c, 1)) },
			"the end": func(c Context) PlayScreen { return playedOut(t, c, atABattleOf(t, c, 3)) },
		} {
			_, offered := build(onDisk).View(onDisk)
			if !strings.Contains(offered, SaveKeyLabel()) {
				t.Errorf("%s | %s: a battle over a real directory does not offer the save "+
					"key, so the arm below measures nothing: %q", lang, name, offered)
			}
			_, withheld := build(embedded).View(embedded)
			if strings.Contains(withheld, SaveKeyLabel()) {
				t.Errorf("%s | %s: a battle with nowhere to write still names %s in its "+
					"footer: %q", lang, name, SaveKeyLabel(), withheld)
			}
		}

		// And the keyboard agrees with the footer. A withdrawn offer that still
		// answers the key is the same defect with the evidence moved one
		// keystroke away.
		played := atABattleOf(t, embedded, 1)
		pressed := playing(t, embedded, played, "ctrl+s")
		if pressed.Err != nil {
			t.Errorf("%s: pressing the save key with nowhere to write drew the error %q",
				lang, embedded.Lang.Error(pressed.Err))
		}
		if len(pressed.Notes) != 0 {
			t.Errorf("%s: pressing the save key with nowhere to write left %d notes behind",
				lang, len(pressed.Notes))
		}
		// The control, and it is what makes the two lines above a measurement:
		// the same keystroke on the same screen over a directory really does
		// write, so the silence is the library and not a broken save.
		wrote := playing(t, onDisk, atABattleOf(t, onDisk, 1), "ctrl+s")
		if wrote.Err != nil {
			t.Errorf("%s: the save over a real directory failed: %v", lang, wrote.Err)
		}
		if len(wrote.Notes) == 0 {
			t.Errorf("%s: the save over a real directory left no note, so the refusal "+
				"above is not measured against anything", lang)
		}
	}
}

// TestTheReplayNoteIsNeverBuiltWithoutADataDirectory is the decision on
// PlayScreen.save's filepath.Rel: it needs nothing, because it cannot be
// reached with an empty first argument — and if it were, the err check already
// carries it.
//
// Both halves are here because either alone is an argument rather than a
// measurement. Unreachability is a fact about today's control flow and the
// fallback is what makes it safe to be wrong about that.
func TestTheReplayNoteIsNeverBuiltWithoutADataDirectory(t *testing.T) {
	c, _ := start(t, i18n.Vi)
	embedded := overTheEmbeddedBooks(t, c)

	// Reached only past a successful SaveBattleLog, which a library with no
	// directory refuses — so the note pair the Rel builds is never built at all,
	// and the empty first argument never happens.
	saved := playing(t, embedded, atABattleOf(t, embedded, 1), "ctrl+s")
	if len(saved.Notes) != 0 {
		t.Fatalf("a save with no directory produced %d notes, so filepath.Rel was reached "+
			"with an empty directory after all", len(saved.Notes))
	}

	// What the call would answer if it ever were reached, measured rather than
	// reasoned about. The rooted case is the one SaveBattleLog produces whenever
	// the directory it was given was rooted, and it **errors** — which is why
	// the `err == nil` at the call site is a safety net and not a bug: relative
	// keeps the whole path. The relative case comes back untouched, which is
	// already the answer that site wants.
	rooted := filepath.Join(string(filepath.Separator), "somewhere", "battles", "a.json")
	if shortened, err := filepath.Rel("", rooted); err == nil {
		t.Errorf("filepath.Rel(\"\", %q) answered %q rather than refusing, so the call "+
			"site's err check no longer keeps the whole path", rooted, shortened)
	}
	const relative = "battles/a.json"
	shortened, err := filepath.Rel("", filepath.FromSlash(relative))
	if err != nil || shortened != filepath.FromSlash(relative) {
		t.Errorf("filepath.Rel(\"\", %q) answered %q (err %v), want it handed back "+
			"unchanged", relative, shortened, err)
	}
}

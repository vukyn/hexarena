package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/socket"
)

// # A player's own squads, measured through the client that offers them
//
// The file is forge's — read, validated and merged there — and what this suite
// is about is the half forge cannot see: that the sides in it reach the two
// places a player picks one, that the side a picker *shows* is the side that
// takes the field, and that none of it made this client able to write anything.
//
// ⚠️ **The vacuous version of every test here is "the list got longer".** A file
// that was ignored while something else grew satisfies that, so every claim
// below names the player's side **by id and by units** and asserts the count as
// well.

// aPlayerFile writes squads out in the shape a player's own file has, and hands
// back the path.
//
// A temporary directory rather than anywhere near a real configuration
// directory: os.UserConfigDir is resolved once at the binary's edge precisely so
// that nothing under it has to be told which platform it is on, and a test that
// wrote into the real one would be a test that measured the machine.
func aPlayerFile(t *testing.T, squads ...placement.Squad) string {
	t.Helper()
	raw, err := placement.Marshal(squads)
	if err != nil {
		t.Fatalf("write the player's squads out: %v", err)
	}
	return aPlayerFileHolding(t, string(raw))
}

// aPlayerFileHolding is the same file with its bytes given verbatim, which is
// what the malformed and hand-edited cases need.
func aPlayerFileHolding(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "squads.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// aPlayerSide is a legal side built around a character none of the sides already
// on the catalogue fields, with ids of its own so a roster can be told apart
// from the fixture's.
//
// ⚠️ **A different character AND different unit ids**, and both halves earn
// their place: shared ids would make "the player's side took the field" a claim
// no roster could settle, and a shared character would let a client that fielded
// the wrong side draw a board that looks right.
func aPlayerSide(t *testing.T, lib *forge.Library, id string) placement.Squad {
	t.Helper()
	taken := make(map[string]bool)
	for _, squad := range lib.Squads() {
		for _, unit := range squad.Units {
			taken[unit.Character] = true
		}
	}
	for _, character := range lib.Characters().All() {
		if taken[character.ID] {
			continue
		}
		squad := aSideOf(t, character, 0)
		squad.ID, squad.Name = id, "đội của tôi"
		for index := range squad.Units {
			squad.Units[index].ID = "cua-toi-" + strconv.Itoa(index)
		}
		// The same call the game makes of a side before it fields one. A fixture
		// that skipped it would let an illegal side into a test about legal ones.
		if _, err := squad.Take(hex.SideAlly, lib.Characters()); err != nil {
			continue
		}
		return squad
	}
	t.Fatalf("no character outside the %d already fielded can be built into a side, so a "+
		"player's own side cannot be told from the catalogue's", len(taken))
	return placement.Squad{}
}

// startCarrying is start with a player's own squad file read the way the binary
// reads one: through Library.PlayerSquads, off a path handed in.
func startCarrying(t *testing.T, lang i18n.Lang, path string) (model, *forge.Library) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	dir := scratchData(t)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	twoSidesSaved(t, lib)
	player, err := lib.PlayerSquads(path)
	if err != nil {
		t.Fatalf("read the player's squads from %s: %v", path, err)
	}
	m := newModel(lib, lang, newSession(), player)
	m.width, m.height = 120, 44
	return m, lib
}

// idsOf and unitIDsOf are the two readings every assertion here is written
// against: which sides are offered, and who is in one.
func idsOf(squads []placement.Squad) []string {
	out := make([]string, 0, len(squads))
	for _, squad := range squads {
		out = append(out, squad.ID)
	}
	return out
}

func unitIDsOf(squad placement.Squad) []string {
	out := make([]string, 0, len(squad.Units))
	for _, unit := range squad.Units {
		out = append(out, unit.ID)
	}
	return out
}

// TestAPlayerSquadReachesTheJoinScreensChooser is the first claim: a side out of
// a player's own file is offered beside the shipped ones and can be chosen.
//
// ⚠️ **Driven through the real model rather than through joinScreen.Refresh**,
// because the claim is about the client: a join screen handed a list is half of
// it, and the other half is that this client builds the list out of the player's
// file at all. The chooser is walked with the key a player presses.
//
// Both languages, because the row a chooser draws is wording.
func TestAPlayerSquadReachesTheJoinScreensChooser(t *testing.T) {
	for _, lang := range i18n.Langs() {
		plain, lib := startCarrying(t, lang, "")
		mine := aPlayerSide(t, lib, "doi-cua-toi")
		shipped := plain.enter(screenJoin).join.Squads
		if len(shipped) == 0 {
			t.Fatalf("the fixture catalogue is empty in %s, so nothing below is measuring a "+
				"list that grew", lang)
		}

		carrying, _ := startCarrying(t, lang, aPlayerFile(t, mine))
		offered := carrying.enter(screenJoin).join.Squads
		if len(offered) != len(shipped)+1 {
			t.Fatalf("%d sides are offered in %s with a player file holding one, over %d "+
				"without it: %v", len(offered), lang, len(shipped), idsOf(offered))
		}
		// By id AND by units, which is what stops this passing on a list that grew
		// for some other reason.
		at := slices.IndexFunc(offered, func(squad placement.Squad) bool { return squad.ID == mine.ID })
		if at < 0 {
			t.Fatalf("the player's side %q is not among the %v offered in %s",
				mine.ID, idsOf(offered), lang)
		}
		if !offered[at].Equal(mine) {
			t.Errorf("the side offered as %q in %s fields %v, and the player's file says %v",
				mine.ID, lang, unitIDsOf(offered[at]), unitIDsOf(mine))
		}
		// And every shipped side is still there, because "offered alongside" is
		// the claim rather than "offered instead of".
		for _, was := range shipped {
			if !slices.Contains(idsOf(offered), was.ID) {
				t.Errorf("the shipped side %q is gone from the chooser in %s: %v",
					was.ID, lang, idsOf(offered))
			}
		}

		// Pickable, through the key: the chooser is walked until it lands on the
		// player's side, and the bound is the whole cycle plus the "bring none"
		// position, so a chooser that never reaches it fails rather than spinning.
		walked := carrying.enter(screenJoin)
		landed := false
		for range len(offered) + 1 {
			if chosen, have := walked.join.Chosen(); have && chosen.ID == mine.ID {
				if !chosen.Equal(mine) {
					t.Errorf("the chooser in %s lands on %q holding %v, and the file says %v",
						mine.ID, lang, unitIDsOf(chosen), unitIDsOf(mine))
				}
				landed = true
				break
			}
			walked = key(t, walked, "right")
		}
		if !landed {
			t.Errorf("→ never reaches the player's side %q in %s over %d positions",
				mine.ID, lang, len(offered)+1)
		}
		// The row a player reads names it, and says whose it is.
		if body := drawnBody(walked); !strings.Contains(body, mine.Name) ||
			!strings.Contains(body, walked.ctx().Text(i18n.SquadMine)) {
			t.Errorf("the join screen in %s does not name the player's side and mark it as "+
				"theirs:\n%s", lang, body)
		}
	}
}

// TestAPlayerSquadIsTheRosterThatTakesTheField is the one a picker that shows a
// name and sends something else cannot pass.
//
// A real registry, a real listener, a real opponent: the side is chosen with the
// key a player presses, the room is dialled with enter, and what is asserted is
// the roster the **mirror** holds once the battle starts — the ids Squad.Take
// prefixed with the side, which is the only place a squad becomes units.
//
// ⚠️ **The room runs on the EMBEDDED books**, so the player's side is built out
// of the shipped cast rather than out of the fixture's injected one — a side
// naming a character only the scratch directory has would be refused at the gate,
// correctly, and this test would be measuring that instead.
func TestAPlayerSquadIsTheRosterThatTakesTheField(t *testing.T) {
	held, library := openARoom(t, 1)
	characters, err := seed.Cast()
	if err != nil {
		t.Fatalf("load the embedded cast: %v", err)
	}
	// The same three characters the fixture's own side brings — deliberately,
	// because the match has a budget and those three are the ones measured to
	// fit it — with **ids of its own**, which is what makes the roster below able
	// to say which of the two sides was fielded.
	mine := aShippedSide(t, characters, "doi-cua-toi",
		"pokemon.bulbasaur", "pokemon.poliwag", "pokemon.gastly")
	mine.Name = "đội của tôi"
	for index := range mine.Units {
		mine.Units[index].ID = "cua-toi-" + strconv.Itoa(index)
	}
	player, err := library.PlayerSquads(aPlayerFile(t, mine))
	if err != nil {
		t.Fatalf("read the player's squads: %v", err)
	}
	if len(player) != 1 {
		t.Fatalf("the player's file read as %d sides, want 1", len(player))
	}

	fake := newFakeSender()
	sess := newSession()
	sess.attach(fake)
	m := newModel(library, i18n.Vi, sess, player)
	m.width, m.height = 120, 44
	m = m.enter(screenJoin)
	// The player's side, chosen with the key. Asserted rather than assigned: a
	// hand-set cursor would measure this test's idea of the chooser.
	landed := false
	for range len(m.join.Squads) + 1 {
		if chosen, have := m.join.Chosen(); have && chosen.ID == mine.ID {
			landed = true
			break
		}
		m = key(t, m, "right")
	}
	if !landed {
		t.Fatalf("→ never reaches the player's side %q over %v", mine.ID, idsOf(m.join.Squads))
	}
	if len(m.join.Squads) < 2 {
		t.Fatalf("only %d side is offered, so choosing the player's one is not a choice",
			len(m.join.Squads))
	}
	m = typeText(t, m, string(held.code))

	next, command := m.Update(press(t, "enter"))
	m = next.(model)
	if command == nil {
		t.Fatal("enter on the join screen asked for no command, so no room was called")
	}
	joined := command()
	if failure, refused := joined.(matchFailedMsg); refused {
		t.Fatalf("the dial was turned away: %v", failure.err)
	}
	m = send(t, m, joined)
	theOpponent(t, held)

	// Driven until the mirror holds a battle, which is the first moment a squad
	// has become a roster.
	var fielded []string
	deadline := time.Now().Add(theWholeMatch)
	for len(fielded) == 0 && time.Now().Before(deadline) {
		if fake.awaits(time.Second) {
			for _, message := range fake.take() {
				m = send(t, m, message)
			}
		}
		m.session.read(func(sight socket.Sight) {
			if sight.Fight == nil {
				return
			}
			for _, unit := range sight.Fight.Units() {
				if unit.Side == sight.Side {
					fielded = append(fielded, unit.ID)
				}
			}
		})
	}
	if len(fielded) == 0 {
		t.Fatalf("no battle reached the mirror inside %s, so nothing was fielded", theWholeMatch)
	}
	want := make([]string, 0, len(mine.Units))
	for _, unit := range mine.Units {
		want = append(want, hex.SideAlly.String()+"."+unit.ID)
	}
	slices.Sort(fielded)
	slices.Sort(want)
	if !slices.Equal(fielded, want) {
		t.Errorf("the side that took the field is %v, and the player picked %v", fielded, want)
	}
}

// TestAnAbsentPlayerFileIsSilentAndTheShippedSidesStillWork is the ordinary
// case, and it is worth a test because the ordinary case is the one a refusal
// written slightly too widely breaks.
//
// Three readings of "silent": a path that names nothing answers no error and no
// sides; the empty path — what a machine whose configuration directory could not
// be resolved hands down — answers the same; and the screens a client draws are
// **byte-identical** to the ones it draws having been told about no file at all.
func TestAnAbsentPlayerFileIsSilentAndTheShippedSidesStillWork(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "there-is-no-such-file.json")
	for _, path := range []string{absent, ""} {
		about := path
		if about == "" {
			about = "the empty path"
		}
		m, lib := startCarrying(t, i18n.Vi, path)
		squads, err := lib.PlayerSquads(path)
		if err != nil {
			t.Fatalf("%s was refused: %v", about, err)
		}
		if len(squads) != 0 {
			t.Errorf("%s read as %d sides", about, len(squads))
		}
		// The shipped sides are all of the catalogue and all of the chooser.
		shipped := idsOf(lib.Squads())
		if len(shipped) == 0 {
			t.Fatal("the fixture saved no sides, so this measures nothing")
		}
		if got := idsOf(m.enter(screenSquads).squads.Saved); !slices.Equal(got, shipped) {
			t.Errorf("the catalogue draws %v over the shipped %v with %s", got, shipped, about)
		}
		if got := idsOf(m.enter(screenJoin).join.Squads); !slices.Equal(got, shipped) {
			t.Errorf("the chooser offers %v over the shipped %v with %s", got, shipped, about)
		}
		// And nothing on the drawing changed either, which is the half an id
		// comparison cannot see: a mark drawn on a side nobody owns would pass
		// every assertion above.
		plain, _ := startCarrying(t, i18n.Vi, "")
		for name, view := range map[string]screen{"the catalogue": screenSquads, "the join screen": screenJoin} {
			// Below the header, because the two models are built by two calls to
			// startCarrying and therefore stand in two scratch directories — and
			// the header names the data directory. Whether the digit that differs
			// is on screen comes down to whether the path clears the clip at
			// minWidth, which the random component of a temp directory decides,
			// so comparing the whole drawing compared the fixture rather than the
			// player file. Both goldens drop that line for the same reason.
			if with, without := belowHeader(m.enter(view)), belowHeader(plain.enter(view)); with != without {
				t.Errorf("%s is drawn differently with %s:\nwith\n%s\nwithout\n%s",
					name, about, with, without)
			}
		}
	}
}

// TestAMalformedPlayerFileIsRefusedAndNamesItself is the case that may not be
// swallowed.
//
// A player hand-edits this file, so a missing comma is a thing that happens, and
// the outcome that must not follow is their squads quietly not being there. Two
// halves: the refusal names the path, and it comes back with **no** sides — a
// partial list would be the swallow wearing an error's clothes.
//
// ⚠️ **The list is asserted empty as well as the error non-nil**, because those
// are separately breakable: a loader that returned what it had parsed so far
// beside an error would hand a client half a file and let the caller decide,
// which is exactly the decision this refusal exists to take away.
func TestAMalformedPlayerFileIsRefusedAndNamesItself(t *testing.T) {
	dir := scratchData(t)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	good := aPlayerSide(t, lib, "doi-cua-toi")
	whole, err := placement.Marshal([]placement.Squad{good})
	if err != nil {
		t.Fatalf("write the player's squads out: %v", err)
	}
	// Two shapes of broken, because they fail in different places: bytes that are
	// not JSON at all, and JSON whose squad list is one side short of legal.
	broken := map[string]string{
		"a missing comma":    strings.Replace(string(whole), `",`, `"`, 1),
		"not a squad at all": `{"squads": [{"id": "doi-cua-toi", "units": []}]}`,
	}
	for about, raw := range broken {
		path := aPlayerFileHolding(t, raw)
		squads, err := lib.PlayerSquads(path)
		if err == nil {
			t.Errorf("%s was accepted, and read as %d sides", about, len(squads))
			continue
		}
		if len(squads) != 0 {
			t.Errorf("%s was refused and still handed back %d sides: %v",
				about, len(squads), idsOf(squads))
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("the refusal of %s does not name the file it is about: %v", about, err)
		}
	}
}

// TestAnIllegalPlayerSquadIsRefusedByTheSameValidatorTheGameUses is question 4,
// and the assertion is deliberately stronger than "it was refused".
//
// A hand-edited file can name a character that does not exist, a level off the
// table, a skill the unit cannot carry or two units on one cell — and every one
// of those is a question **Squad.Take against the cast book** already answers:
// the call Library.SaveSquad makes before it writes, and the call room's gate
// makes before it seats anybody. So what is asserted is that the refusal a
// player's file gets **ends with the refusal Take gives**, which a second
// validator written here could not satisfy.
//
// Four cases, one per way a hand-edited file goes wrong, listed so a fifth kind
// of illegality added to the file's shape has somewhere obvious to go.
func TestAnIllegalPlayerSquadIsRefusedByTheSameValidatorTheGameUses(t *testing.T) {
	dir := scratchData(t)
	lib, err := forge.Load(dir)
	if err != nil {
		t.Fatalf("load %s: %v", dir, err)
	}
	legal := aPlayerSide(t, lib, "doi-cua-toi")
	if _, err := legal.Take(hex.SideAlly, lib.Characters()); err != nil {
		t.Fatalf("the fixture side is already illegal, so nothing below is a change: %v", err)
	}
	spoil := map[string]func(placement.Squad) placement.Squad{
		"a character nobody wrote": func(squad placement.Squad) placement.Squad {
			squad.Units[0].Character = "khong-co-nhan-vat-nay"
			return squad
		},
		"a skill the unit cannot carry": func(squad placement.Squad) placement.Squad {
			squad.Units[0].Skills = []string{"khong-co-chieu-nay"}
			return squad
		},
		"two units on one cell": func(squad placement.Squad) placement.Squad {
			squad.Units[1].Slot = squad.Units[0].Slot
			return squad
		},
		"a stage off the line": func(squad placement.Squad) placement.Squad {
			squad.Units[0].Stage = "khong-co-dang-nay"
			return squad
		},
	}
	if len(spoil) != 4 {
		t.Fatalf("this walk has %d cases and its comment says four", len(spoil))
	}
	for about, spoilt := range spoil {
		squad := spoilt(legal.Clone())
		_, want := squad.Take(hex.SideAlly, lib.Characters())
		if want == nil {
			t.Errorf("%s is legal to Take, so this row measures nothing", about)
			continue
		}
		path := aPlayerFile(t, squad)
		squads, err := lib.PlayerSquads(path)
		if err == nil {
			t.Errorf("%s was accepted out of a player's file, and read as %d sides",
				about, len(squads))
			continue
		}
		if len(squads) != 0 {
			t.Errorf("%s was refused and still handed back %d sides", about, len(squads))
		}
		// The same sentence, from the same validator, behind the path this file
		// is at. Equality of the whole string would only say the wrapping is what
		// it is today; the suffix says the answer came from Take.
		if !strings.HasSuffix(err.Error(), want.Error()) {
			t.Errorf("a player's file refuses %s with\n\t%v\nand Take refuses it with\n\t%v",
				about, err, want)
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("the refusal of %s does not name the file it is about: %v", about, err)
		}
	}
}

// TestAPlayerSideWinsTheIdItSharesAndTheRowSaysSo is the collision rule, both
// halves.
//
// The rule is that a player's side wins the id it shares with one of the game's,
// and the reason a rule is needed at all is that a side is chosen by id: two
// rows under one id would be a reader pointing at the second and fielding the
// first. So the count may not grow, the survivor has to be the player's **by
// units**, and — the half that stops this being the silent collision no answer
// may be — the row has to say whose side it is, in both places a player picks
// one and in both languages.
func TestAPlayerSideWinsTheIdItSharesAndTheRowSaysSo(t *testing.T) {
	for _, lang := range i18n.Langs() {
		plain, lib := startCarrying(t, lang, "")
		shipped := lib.Squads()
		if len(shipped) < 2 {
			t.Fatalf("the fixture saved %d sides, so a collision cannot be told from a "+
				"replacement of the whole catalogue", len(shipped))
		}
		shadowed := shipped[0]
		mine := aPlayerSide(t, lib, shadowed.ID)
		if mine.Equal(shadowed) {
			t.Fatalf("the player's side and the shipped %q are the same squad, so nothing "+
				"below can tell which one won", shadowed.ID)
		}

		carrying, _ := startCarrying(t, lang, aPlayerFile(t, mine))
		offered := carrying.enter(screenSquads).squads.Saved
		if len(offered) != len(shipped) {
			t.Errorf("the catalogue holds %d sides in %s over the shipped %d, so a shared id "+
				"grew the list: %v", len(offered), lang, len(shipped), idsOf(offered))
		}
		if got := slices.Contains(idsOf(offered[1:]), shadowed.ID); got {
			t.Errorf("%q appears twice in %s: %v", shadowed.ID, lang, idsOf(offered))
		}
		at := slices.IndexFunc(offered, func(squad placement.Squad) bool { return squad.ID == shadowed.ID })
		if at < 0 {
			t.Fatalf("%q is on neither list in %s: %v", shadowed.ID, lang, idsOf(offered))
		}
		if !offered[at].Equal(mine) {
			t.Errorf("%q is drawn in %s fielding %v; the player's file says %v and the game's "+
				"data says %v", shadowed.ID, lang, unitIDsOf(offered[at]),
				unitIDsOf(mine), unitIDsOf(shadowed))
		}
		// The row a player reads says whose it is — and the shipped row beside it
		// does not, which is what stops a mark drawn on everything from passing.
		mark := carrying.ctx().Text(i18n.SquadMine)
		rows := strings.Split(drawnBody(carrying.enter(screenSquads)), "\n")
		marked, mine1 := 0, 0
		for _, row := range rows {
			if !strings.Contains(row, mark) {
				continue
			}
			marked++
			if strings.Contains(row, shadowed.ID) {
				mine1++
			}
		}
		if marked != 1 || mine1 != 1 {
			t.Errorf("%d rows of the %s catalogue are marked as the player's and %d of those "+
				"is %q; want exactly one, and it:\n%s", marked, lang, mine1, shadowed.ID,
				drawnBody(carrying.enter(screenSquads)))
		}
		// The other place a side is picked, on the row the chooser lands on.
		joined := carrying.enter(screenJoin)
		landed := false
		for range len(offered) + 1 {
			if chosen, have := joined.join.Chosen(); have && chosen.ID == shadowed.ID {
				landed = true
				break
			}
			joined = key(t, joined, "right")
		}
		if !landed {
			t.Fatalf("→ never reaches %q on the %s join screen", shadowed.ID, lang)
		}
		if body := drawnBody(joined); !strings.Contains(body, mark) {
			t.Errorf("the %s join screen does not say the side under the chooser is the "+
				"player's:\n%s", lang, body)
		}
		// And the shipped side of that id is drawn nowhere, which is the whole of
		// what "the player's wins" costs.
		if plainBody := drawnBody(plain.enter(screenSquads)); strings.Contains(plainBody, mark) {
			t.Errorf("the %s catalogue marks a side as the player's with no player file "+
				"at all:\n%s", lang, plainBody)
		}
	}
}

// TestAPlayerFileTurnsNoAuthoringKeyOn is the half readonly_test.go cannot see.
//
// ⚠️ **Its four tests run on a client with no player file**, so every one of
// them is blind to a client that started authoring the day one arrived — which
// is exactly the change this work was closest to making by accident, since the
// whole point of a player's own file is that a player may eventually write in
// it. That is the *next* item in TODO.md; nothing here may bring it forward.
//
// Three claims: the reading is still read-only, the authoring keys still do
// nothing on the screens that have them, and the file on disk is **byte for byte
// what it was** after every key this suite can send has been pressed on the
// squad catalogue.
func TestAPlayerFileTurnsNoAuthoringKeyOn(t *testing.T) {
	_, lib := startCarrying(t, i18n.Vi, "")
	mine := aPlayerSide(t, lib, "doi-cua-toi")
	path := aPlayerFile(t, mine)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for _, lang := range i18n.Langs() {
		m, _ := startCarrying(t, lang, path)
		if m.ctx().Authoring {
			t.Fatalf("this client's Context reads as authoring in %s", lang)
		}
		if !slices.Contains(idsOf(m.enter(screenSquads).squads.Saved), mine.ID) {
			t.Fatalf("the player's side is not on the %s catalogue, so pressing keys on it "+
				"measures nothing", lang)
		}
		for view, keys := range authoringKeys {
			at := m.enter(view)
			was := at.screenContent()
			for _, name := range keys {
				after := key(t, at, name)
				if after.screen != view {
					t.Errorf("%q on screen %v in %s moved to screen %v with a player file "+
						"loaded", name, view, lang, after.screen)
				}
				if got := after.screenContent(); got != was {
					t.Errorf("%q on screen %v in %s changed the screen with a player file "+
						"loaded:\nbefore\n%s\nafter\n%s", name, view, lang, was, got)
				}
			}
		}
		// Every key, not only the authoring ones: what is being measured here is
		// that a file arrived, not that five particular letters are guarded.
		for _, name := range everyKeyPressed() {
			m = key(t, m.enter(screenSquads), name)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(after) != string(before) {
		t.Errorf("the player's file was written to:\nbefore\n%s\nafter\n%s", before, after)
	}
	// And the footers still name none of the keys this client ignores, which is
	// readonly_test.go's second claim asked again with a file loaded.
	m, _ := startCarrying(t, i18n.Vi, path)
	for name, at := range everyScreen(t, m) {
		ignored := authoringKeys[at.screen]
		if len(ignored) == 0 {
			continue
		}
		_, footer := at.parts()
		for _, named := range keysNamed(footer) {
			if slices.Contains(ignored, named) {
				t.Errorf("the %s screen's footer names %q with a player file loaded:\n%s",
					name, named, footer)
			}
		}
	}
}

// TestThePlayerSquadPathIsBuiltRatherThanRead holds the one decision the flag
// carries: the configuration directory is a parameter.
//
// ⚠️ **Nothing here calls os.UserConfigDir, and that is the point.** It reads the
// environment and resolves differently on every platform, so a test that drove it
// would be measuring the machine it ran on rather than this code — the mistake
// memory/windows-sets-no-term.md records. What is measured instead is the
// function that takes the answer: it puts the file under the directory it was
// handed, and it answers **nothing** rather than a path relative to the working
// directory when there is no directory to put it under.
func TestThePlayerSquadPathIsBuiltRatherThanRead(t *testing.T) {
	if got := forge.PlayerSquadsPath(""); got != "" {
		t.Errorf("with no configuration directory the path is %q, want none", got)
	}
	// ⚠️ **Built with filepath.Join rather than written as a POSIX literal.**
	// PlayerSquadsPath joins, and Join normalises to the platform's own
	// separator — so a literal `/somewhere/…` comes back `\somewhere\…` on
	// Windows and the prefix below never matches, which is this test failing on
	// a fact about the machine rather than about the code. It is the same shape
	// as memory/windows-sets-no-term.md, one directory over. The path is
	// fictional either way; nothing here touches a disk.
	anywhere := filepath.Join(string(filepath.Separator), "somewhere", "a-machine-keeps-configuration")
	got := forge.PlayerSquadsPath(anywhere)
	if !strings.HasPrefix(got, anywhere+string(filepath.Separator)) {
		t.Errorf("the player's file is at %q, which is not under %q", got, anywhere)
	}
	if filepath.Base(got) != "squads.json" {
		t.Errorf("the player's file is called %q", filepath.Base(got))
	}
	// The game's own data directory is a different place entirely, which is the
	// whole distinction this file rests on.
	if strings.Contains(got, forge.DefaultDataDir) {
		t.Errorf("the player's file at %q is inside the game's own data directory", got)
	}
}

// TestTheSquadsFlagOverridesTheResolvedDefault is the flag's own shape, which is
// --data's: a default worked out by the binary, overridden by what was typed.
func TestTheSquadsFlagOverridesTheResolvedDefault(t *testing.T) {
	const resolved = "/somewhere/hexarena/squads.json"
	const asked = "./mine.json"
	chosen, err := parseOptions(nil, "", resolved, discardOutput())
	if err != nil {
		t.Fatalf("parse no flags: %v", err)
	}
	if chosen.squads != resolved {
		t.Errorf("with no flag the player's file is %q, want the resolved %q",
			chosen.squads, resolved)
	}
	chosen, err = parseOptions([]string{"-squads", asked}, "", resolved, discardOutput())
	if err != nil {
		t.Fatalf("parse -squads: %v", err)
	}
	if chosen.squads != asked {
		t.Errorf("-squads %q was read as %q", asked, chosen.squads)
	}
	// And it is a different flag from --data, which names the game's own books.
	if chosen.dir != forge.DefaultDataDir {
		t.Errorf("-squads moved the data directory to %q", chosen.dir)
	}
}

// discardOutput is where the flag package's own writing goes in these two tests.
func discardOutput() *strings.Builder { return &strings.Builder{} }

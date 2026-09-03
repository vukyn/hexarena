package screen

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/placement"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
	"github.com/vukyn/hexarena/internal/i18n"
)

// The squad builder is three views of one thing, and they are one screen rather
// than three because they are one decision taken at three depths: which squads
// exist, who is in this one, and what that unit brings. Splitting them would put
// the half-built squad somewhere both of the others could reach, which is the
// shape that lets two screens disagree about what is being edited.
//
// ⚠️ **It is the one screen in this package that writes the author's own file**,
// and that is a constraint on the tests rather than on the code: `SaveSquad` and
// `DeleteSquad` go through internal/forge like every other write here, but
// screens.golden loads the books straight out of ../seed/data with no temp copy,
// so a golden entry has to be a value built by hand rather than a file written.
type SquadMode int

const (
	// SquadList is the catalogue: which squads are on the file.
	SquadList SquadMode = iota
	// SquadEdit is one squad in hand: its id, its name and its members.
	SquadEdit
	// SquadUnit is one member of that squad: the six facts a roster entry
	// carries.
	SquadUnit
)

// The fields of one unit, in the order they are asked.
//
// Character first because everything under it is read against the character:
// the level bounds the forms, the form bounds the learnset, and the kit is
// chosen out of that. Slot last of the settled facts because it is the only one
// that says nothing about the unit itself — where it stands is a fact about the
// squad.
const (
	SquadUnitCharacter = iota
	SquadUnitLevel
	SquadUnitStage
	SquadUnitSlot
	SquadUnitSkills
	SquadUnitPassives
	// SquadUnitFieldCount is how many fields a member has, which is what the
	// cursor wraps against.
	SquadUnitFieldCount
)

// SquadsAsk is what one of the builder's two questions is about.
//
// ⚠️ **This is the vocabulary Action.About exists to carry**, and it is the
// screen's own rather than a client's. The delete needs a value the screen does
// not hold — the id under the catalogue's cursor when `d` was pressed — and the
// two questions need telling apart by something other than what mode the screen
// happens to be in, which would be state read to recover what the question could
// simply have said.
//
// A comparable struct, so an Action carrying one stays a comparable value a test
// can write out as a literal.
type SquadsAsk struct {
	Kind SquadsAskKind

	// ID is a squad id, and only SquadsAskSavedSquad spends it.
	ID string
}

// SquadsAskKind is which of the builder's questions was put.
type SquadsAskKind uint8

const (
	// SquadsAskNothing is the zero value, and it is what a question this screen
	// did not ask comes back as: About is an `any`, so a value out of another
	// screen's vocabulary lands here through a failed type assertion and is
	// declined rather than guessed at.
	SquadsAskNothing SquadsAskKind = iota
	// SquadsAskSavedSquad names a squad on the file, by the id that was under
	// the catalogue's cursor when the question was asked.
	SquadsAskSavedSquad
	// SquadsAskSquadInHand is the half-built squad under edit, which is the
	// other question and the reason a kind is carried at all.
	SquadsAskSquadInHand
)

// SquadsPick is where one of the builder's two lists lands.
//
// ⚠️ **A destination names a field of one screen**, which is why it lives beside
// that screen and not in the client: the two halves of a loadout are one
// keystroke apart and differ in nothing else, so a screen alone cannot say which
// was being filled. It is the same shape SkillsPick has, and PickState carries
// either as the `any` it always was.
type SquadsPick uint8

const (
	// SquadsPickNothing is the zero value: a destination that names no field.
	// No raise here uses it, and Picked declines it.
	SquadsPickNothing SquadsPick = iota
	// SquadsPickKit is the four skills a member brings.
	SquadsPickKit
	// SquadsPickTrait is the one trait it brings.
	SquadsPickTrait
	// SquadsPickCount is the count a client's dispatch is held total against, in
	// the shape SkillsPickCount, SubjectKindCount and TargetCount already have.
	SquadsPickCount
)

// SquadsScreen is the squad builder: the catalogue, one squad under edit, and
// one member of it.
type SquadsScreen struct {
	Mode SquadMode
	// Saved is the catalogue as it is on disk, refreshed when the screen is
	// entered so a squad written by the other front-end is not invisible here.
	Saved  []placement.Squad
	Cursor int

	// Characters is the WHOLE cast, held rather than looked up per keystroke
	// because cycling walks it.
	//
	// The whole cast rather than the offered part of it, because this slice
	// answers two different questions and only one of them is about choosing.
	// Character() looks a member's character up here to read its forms, its
	// learnset and its traits, and a squad on the file may name a character that
	// has since been hidden — so a filtered slice would leave that member with no
	// forms and an empty kit picker. What may be *chosen* is offeredCharacters,
	// asked of this one at each of the two sites that choose.
	Characters []cast.Character

	// Editing is the squad in hand. It is a whole squad rather than an index
	// into Saved, because a squad being built has not been saved yet and an
	// index would have nothing to point at.
	Editing placement.Squad
	// Baseline is the squad as it was last written down: what Open read off
	// the file, what Save put back, or the empty squad Begin started from.
	// The guard on leaving compares against it.
	//
	// It is a reading rather than a latched flag because a latch cannot tell a
	// squad that changed from one that was merely touched. Commit writes a
	// member back whether or not anything moved, and it runs on the way out of
	// every member, so opening one and pressing escape used to be
	// indistinguishable from editing it — the question was asked over changes
	// nobody had made, which is how a question stops being read.
	//
	// It is a Clone, not the value Editing was set from: Editing is mutated in
	// place, and a baseline sharing its slices would compare equal to itself
	// for ever.
	Baseline placement.Squad
	// Units is the cursor over the squad's members, and it may sit one past the
	// last, which is the row that adds another.
	Units int

	IDInput   textinput.Model
	NameInput textinput.Model
	// EditingID is true while the id field is the one being typed into. An id is
	// asked once, when a squad is created: changing it later would write a
	// second squad rather than rename this one, and a field that silently makes
	// a copy is worse than one that is not offered.
	EditingID bool

	// Unit is the member under edit, and UnitIndex is where it sits in the squad.
	Unit      placement.Placement
	UnitIndex int
	// UnitOpenedAs is the character this member named when it was opened, and it
	// is what the character chooser keeps offering however held back that
	// character is. See offeredCharacters.
	//
	// Read from here rather than from Unit.Character — the obvious spelling —
	// because the obvious spelling makes the list change shape while it is being
	// walked: stepping off a held-back character drops it, so `right` then
	// `left` lands one row short of where it started and the character is
	// unreachable for the rest of the edit. A chooser is a fixed list the arrows
	// walk, so what may be chosen has to be settled when the member is opened
	// rather than recomputed from the answer in hand.
	UnitOpenedAs string
	Field        int
	LevelInput   textinput.Model

	Err   error
	Notes []forge.Note
}

// NewSquadsScreen builds the builder over a library.
//
// ⚠️ **It takes a Context rather than a library**, for the reason
// NewSkillsScreen does: three text fields have to be dressed, NewInput wants to
// know whether colour would be noise, this package may not read an environment,
// and Palette already carries the answer the binary was handed.
func NewSquadsScreen(c Context) SquadsScreen {
	s := SquadsScreen{
		IDInput:    NewInput(c.Style.Plain),
		NameInput:  NewInput(c.Style.Plain),
		LevelInput: *NumberField(c.Style.Plain, ""),
	}
	s.IDInput.Prompt, s.NameInput.Prompt = "", ""
	s.IDInput.CharLimit, s.NameInput.CharLimit = 32, 48
	s.IDInput.SetWidth(34)
	s.NameInput.SetWidth(40)
	return s.Refresh(c)
}

// Refresh re-reads the catalogue, and leaves a squad under edit alone: entering
// the screen while building one is the ordinary way back from a picker.
func (s SquadsScreen) Refresh(c Context) SquadsScreen {
	s.Saved = c.Lib.Squads()
	s.Characters = c.Lib.Characters().All()
	s.Cursor = Clamp(s.Cursor, 0, len(s.Saved)-1)
	return s
}

// offeredCharacters is who the builder proposes: every character not held back,
// plus — always — the one named by `keep`.
//
// keep is SquadsScreen.UnitOpenedAs, the character the member under edit named
// when it was opened, and the empty string when there is none, which is what a
// brand new member passes. It is deliberately not "whatever is chosen right
// now" — see the field's own comment for why a list that moves while it is
// walked is a chooser the arrows cannot bring back.
//
// ⚠️ **That second half is the whole reason this is a function and not an `if`
// at the call site.** A squad on the file may name a character that has since
// been hidden, and a chooser that simply dropped it would step off it the first
// time an author touched the row: cycle finds no match, falls back to index 0
// and writes somebody else into a member nobody asked to change — a silent edit
// to the author's own saved file, in the one screen here that writes it. Hidden
// therefore means *not offered for a new choice*, never *taken away from a
// choice already made*, and declaration order is preserved so keeping the one
// named costs the list nothing but its own row.
//
// It filters at the call site rather than through a `cast.Book` accessor, and
// that is a judgement rather than an accident. The rule this applies is not a
// fact about the cast — "the one this member already holds" is a fact about the
// squad being edited, which no book can know — so an accessor could answer only
// the first half and this screen would still have to union the second back in.
// That is a second vocabulary for "the cast" that no caller could use unaided,
// while Book.All() stays the honest answer to "the cast" for the browser, the
// builds screen, `hexforge list`, the spar, the roster and the restriction
// picker, none of which are choosing a side to fight with. The day a second
// screen wants the offered list, the argument reverses and the accessor is the
// change to make.
func offeredCharacters(all []cast.Character, keep string) []cast.Character {
	out := make([]cast.Character, 0, len(all))
	for _, character := range all {
		if character.Hidden && character.ID != keep {
			continue
		}
		out = append(out, character)
	}
	return out
}

// Update routes a keystroke to whichever of the three depths is in front, and
// says what the client owes it.
//
// ⚠️ **Three returns rather than two, and the third is a tea.Cmd**, for the
// reason SkillsScreen.Update has three: this screen has text fields — the id,
// the name and the level — and a bubbles textinput answers an Update with a
// command, the cursor's blink, which has to come out or the field loses its
// cursor.
func (s SquadsScreen) Update(c Context, message tea.KeyPressMsg) (SquadsScreen, Action, tea.Cmd) {
	switch s.Mode {
	case SquadEdit:
		return s.updateEdit(c, message)
	case SquadUnit:
		return s.updateUnit(c, message)
	}
	return s.updateList(c, message)
}

func (s SquadsScreen) updateList(c Context, message tea.KeyPressMsg) (SquadsScreen, Action, tea.Cmd) {
	switch message.String() {
	case "esc", "q":
		return s, Action{Kind: Back}, nil
	case "up", "k":
		s.Cursor = Clamp(s.Cursor-1, 0, len(s.Saved)-1)
	case "down", "j":
		s.Cursor = Clamp(s.Cursor+1, 0, len(s.Saved)-1)
	// The three keys that reach the builder, all guarded on the client being
	// able to write — see Context.Authoring. A game client draws this catalogue
	// so a player can see which sides have been built and take one into a
	// battle, and every one of these three ends in squads.json: `n` and `enter`
	// open the two depths under this one and `d` deletes a file outright.
	//
	// ⚠️ **`f` is not among them**, which is the whole reason this screen is on a
	// game client's menu at all: fighting a side is what a side is for. It raises
	// a Target and the client turns that into one of its own screens, which is
	// the seam the two clients answer differently.
	case "n":
		if c.Authoring {
			s = s.Begin()
		}
	case "enter":
		if len(s.Saved) > 0 && c.Authoring {
			s = s.Open(s.Saved[Clamp(s.Cursor, 0, len(s.Saved)-1)])
		}
	case "f":
		// The fight carries its own two sides, so this names one: the squad
		// under the cursor becomes the home side, which is what f has always
		// meant here. Naming it is the whole of the difference — the fight reads
		// its own field afterwards, so walking the catalogue behind it cannot
		// change what is being measured.
		//
		// ⚠️ It goes by **id** rather than by the row it sits on, which is the
		// Subject idiom: a raise names what it wants and asks nobody where it
		// lives, so turning that back into whichever row of the catalogue the
		// fight keeps is the client's job.
		if len(s.Saved) > 0 {
			return s, Action{Kind: Raise, Target: Fight, Subject: Subject{
				Kind: SquadSubject,
				ID:   s.Saved[Clamp(s.Cursor, 0, len(s.Saved)-1)].ID,
			}}, nil
		}
	case "d":
		if len(s.Saved) > 0 && c.Authoring {
			return s, Action{
				Kind:     Ask,
				Question: i18n.SquadDiscardSaved,
				About: SquadsAsk{
					Kind: SquadsAskSavedSquad,
					ID:   s.Saved[Clamp(s.Cursor, 0, len(s.Saved)-1)].ID,
				},
			}, nil
		}
	}
	return s, Action{}, nil
}

// Confirmed answers whichever of the builder's two questions was asked, told
// apart by what the question was about rather than by what mode the screen is
// in.
//
// ⚠️ **This is the one screen with two confirms**, and it is why Action.About
// exists. The mode would answer it — the guard freezes every other key, so the
// screen cannot have changed depth between the question and the answer — and
// that is state being read to recover something the question could simply have
// said. The delete also needs a value the screen does not hold: the id under the
// catalogue's cursor when `d` was pressed.
//
// Both stay on this screen, so both hand back the zero action.
func (s SquadsScreen) Confirmed(c Context, about any) (SquadsScreen, Action) {
	// A value out of another screen's vocabulary comes back as the zero ask and
	// falls through below, which is the same answer a question this screen did
	// not put would get.
	asked, _ := about.(SquadsAsk)
	switch asked.Kind {
	case SquadsAskSavedSquad:
		// The refusal is kept on the screen rather than thrown, because a file
		// that would not go is something the reader has to be told about where
		// they asked — and a failed delete leaves the catalogue exactly as it
		// was, so nothing is refreshed.
		if err := c.Lib.DeleteSquad(asked.ID); err != nil {
			s.Err = err
			return s, Action{}
		}
		s.Err = nil
		s.Notes = nil
		return s.Refresh(c), Action{}
	case SquadsAskSquadInHand:
		s.Mode = SquadList
		// Discarding is what the question said, so what is in hand goes back to
		// the squad last written rather than being left changed behind a mode
		// that no longer reads it.
		s.Editing = s.Baseline.Clone()
		return s.Refresh(c), Action{}
	}
	// SquadsAskNothing, which the builder never asks. Named by falling through
	// rather than by a default that does something: a question this screen did
	// not ask has no answer here, and guessing at one would be the delete taking
	// whatever id happened to be around.
	return s, Action{}
}

// Begin starts a squad nobody has written yet.
func (s SquadsScreen) Begin() SquadsScreen {
	s.Mode = SquadEdit
	s.Editing = placement.Squad{}
	s.Baseline = s.Editing.Clone()
	s.Units = 0
	s.EditingID = true
	s.Err, s.Notes = nil, nil
	s.IDInput.SetValue("")
	s.NameInput.SetValue("")
	s.IDInput.Focus()
	s.NameInput.Blur()
	return s
}

// Open takes a saved squad up for editing, on its own copy.
func (s SquadsScreen) Open(squad placement.Squad) SquadsScreen {
	s.Mode = SquadEdit
	s.Editing = squad.Clone()
	s.Baseline = s.Editing.Clone()
	s.Units = 0
	s.EditingID = false
	s.Err, s.Notes = nil, nil
	s.IDInput.SetValue(squad.ID)
	s.NameInput.SetValue(squad.Name)
	s.IDInput.Blur()
	s.NameInput.Focus()
	return s
}

// Dirty reports whether the squad in hand differs from the one last written
// down, which is the whole of what the guard on leaving asks.
//
// It is a comparison rather than a flag, so a round trip that changes nothing —
// opening a member and leaving it, stepping the cell chooser onto another cell
// and back — leaves the question unasked. Everything under edit has reached
// s.Editing by the time this is read: the only route out of a member commits
// first, and so does opening either of its lists, while the id and the name are
// written on every keypress that reaches them.
func (s SquadsScreen) Dirty() bool {
	return !s.Editing.Equal(s.Baseline)
}

func (s SquadsScreen) updateEdit(c Context, message tea.KeyPressMsg) (SquadsScreen, Action, tea.Cmd) {
	if IsSaveKey(message) {
		return s.Save(c), Action{}, nil
	}
	switch message.String() {
	case "esc":
		if s.Dirty() {
			return s, Action{
				Kind:     Ask,
				Question: i18n.SquadDiscard,
				About:    SquadsAsk{Kind: SquadsAskSquadInHand},
			}, nil
		}
		s.Mode = SquadList
		return s.Refresh(c), Action{}, nil
	case "tab":
		s.EditingID = !s.EditingID && s.fresh()
		s = s.focus()
	case "up":
		s.Units = Clamp(s.Units-1, 0, len(s.Editing.Units))
	case "down":
		s.Units = Clamp(s.Units+1, 0, len(s.Editing.Units))
	case "enter":
		if s.Units >= len(s.Editing.Units) {
			return s.addUnit(), Action{}, nil
		}
		s = s.EditUnit(s.Units)
	case "ctrl+x":
		if s.Units < len(s.Editing.Units) {
			s.Editing.Units = append(s.Editing.Units[:s.Units], s.Editing.Units[s.Units+1:]...)
			s.Units = Clamp(s.Units, 0, len(s.Editing.Units))
			s.Err, s.Notes = nil, nil
		}
	default:
		field := &s.NameInput
		if s.EditingID {
			field = &s.IDInput
		}
		updated, command := field.Update(message)
		if updated.Value() != field.Value() {
			s.Err, s.Notes = nil, nil
		}
		*field = updated
		s.Editing.ID = strings.TrimSpace(s.IDInput.Value())
		s.Editing.Name = strings.TrimSpace(s.NameInput.Value())
		return s, Action{}, command
	}
	return s, Action{}, nil
}

// Paste puts a pasted string where a typed one would go, and nowhere when the
// keyboard is not on a field.
//
// ⚠️ **The SquadList fall-through is the load-bearing half of that.** The level
// field is a NumberField, which focuses itself at construction, and the id and
// the name stay focused after the builder is escaped — so the **catalogue**,
// which is the screen a game client draws and the one this screen opens on, has a
// focused text field on it at all times. A paste routed at "whichever field is
// focused" would go into a squad nobody is building. The switch is what says so,
// and there is no second predicate beside it: this method has to find the field
// anyway, so a `Pasting` restating these three facts would be the same rule
// declared twice, and only one of the two could be wrong at a time.
//
// The two depths want different things of it and both are the key path's own
// rule: the id and the name are free text and take a paste as they take a
// keystroke, while the level is a number field, so PasteDigits refuses anything
// else for the reason spelled out beside it. The bookkeeping either side is
// updateEdit's and updateUnit's, on the same condition each of them uses.
func (s SquadsScreen) Paste(_ Context, text string) (SquadsScreen, tea.Cmd) {
	switch s.Mode {
	case SquadEdit:
		field := s.editField()
		if field == nil {
			return s, nil
		}
		before := field.Value()
		command := PasteInto(field, text)
		if field.Value() != before {
			s.Err, s.Notes = nil, nil
		}
		s.Editing.ID = strings.TrimSpace(s.IDInput.Value())
		s.Editing.Name = strings.TrimSpace(s.NameInput.Value())
		return s, command
	case SquadUnit:
		if !s.LevelInput.Focused() {
			return s, nil
		}
		// ⚠️ **Whether the paste landed is read off the VALUE and never off the
		// command.** PasteInto's command is the cursor's blink, and a field on a
		// plain terminal has no virtual cursor — NewInput turns it off under
		// NO_COLOR — so bubbles hands back a nil command for a paste that
		// succeeded perfectly well. Reading the nil as a refusal is exactly what
		// this arm did first: the field took "42" and the member stayed at sixty,
		// and every assertion about the field passed.
		before := s.LevelInput.Value()
		command := PasteDigits(&s.LevelInput, text)
		if s.LevelInput.Value() == before {
			// Refused for not being a number, so nothing about the member moved.
			return s, nil
		}
		if level, err := strconv.Atoi(strings.TrimSpace(s.LevelInput.Value())); err == nil {
			s.Unit.Level = level
			s = s.settleStage()
		}
		s.Err = nil
		return s, command
	}
	return s, nil
}

// editField is the one of the two the builder's keyboard is on, and nil when
// neither has it.
//
// Which of them is `s.EditingID`'s answer, exactly as it is on the key path;
// whether either of them has the keyboard at all is Focused's, which is what
// makes the same method usable from the catalogue where neither does.
func (s *SquadsScreen) editField() *textinput.Model {
	field := &s.NameInput
	if s.EditingID {
		field = &s.IDInput
	}
	if !field.Focused() {
		return nil
	}
	return field
}

// fresh reports whether the squad in hand is one that has never been saved,
// which is the only time its id may still be typed.
func (s SquadsScreen) fresh() bool {
	for _, saved := range s.Saved {
		if saved.ID == s.Editing.ID && s.Editing.ID != "" {
			return false
		}
	}
	return true
}

func (s SquadsScreen) focus() SquadsScreen {
	if s.EditingID {
		s.IDInput.Focus()
		s.NameInput.Blur()
		return s
	}
	s.IDInput.Blur()
	s.NameInput.Focus()
	return s
}

// addUnit puts the first character the builder offers in the first free slot,
// which is a unit an author then edits rather than a blank one they have to fill
// from nothing.
//
// It asks for the offered list with nothing held, because a member being added
// has chosen nobody yet: a hidden character is never what a new member starts
// as. A cast that is hidden all the way down therefore adds nobody, the way an
// empty one does.
func (s SquadsScreen) addUnit() SquadsScreen {
	offered := offeredCharacters(s.Characters, "")
	if len(s.Editing.Units) >= hex.MaxTeamSize || len(offered) == 0 {
		return s
	}
	character := offered[0]
	unit := placement.Placement{
		Character: character.ID,
		Level:     progression.LevelCap,
		Slot:      s.freeSlot(-1),
	}
	unit.ID = s.freeID(character.ID, -1)
	s.Editing.Units = append(s.Editing.Units, unit)
	return s.EditUnit(len(s.Editing.Units) - 1)
}

// freeSlot is the first formation cell nobody else in the squad stands on.
func (s SquadsScreen) freeSlot(except int) hex.Offset {
	taken := map[hex.Offset]bool{}
	for i, unit := range s.Editing.Units {
		if i != except {
			taken[unit.Slot] = true
		}
	}
	for _, slot := range FormationSlots() {
		if !taken[slot] {
			return slot
		}
	}
	return hex.Offset{}
}

// freeID is a unit id nobody in the squad answers to, built from the
// character's own name so a log reads as the cast rather than as unit1, unit2.
func (s SquadsScreen) freeID(character string, except int) string {
	base := character
	if cut := strings.LastIndex(base, "."); cut >= 0 {
		base = base[cut+1:]
	}
	taken := map[string]bool{}
	for i, unit := range s.Editing.Units {
		if i != except {
			taken[unit.ID] = true
		}
	}
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := base + strconv.Itoa(n)
		if !taken[candidate] {
			return candidate
		}
	}
}

// FormationSlots is every cell of one side's formation, front column first: a
// squad is read from the rank that meets the enemy backwards, because that is
// the order an attack arrives in.
func FormationSlots() []hex.Offset {
	out := make([]hex.Offset, 0, hex.FormationCols*hex.FormationRows)
	for _, col := range formationColumns() {
		for row := 0; row < hex.FormationRows; row++ {
			out = append(out, hex.Offset{Col: col, Row: row})
		}
	}
	return out
}

// formationColumns is the authoring columns in the order a rank is met: the
// entry at index 0 is the column that stands in front of the rest.
//
// It is read off hex.Ranks rather than counted down from FormationCols, for the
// reason the shape chooser draws its cells from pattern.Targets: an authoring
// column is not a depth. The ally half counts down from its own frontline and
// the enemy half counts up, so a drawing that decided which end was the front by
// looking at the column number would be right for whichever side whoever wrote
// it had in mind — which is the mistake hex.Ranks exists to have made once.
func formationColumns() []int {
	out := make([]int, hex.FormationCols)
	for col := range hex.FormationCols {
		if depth := rankOf(hex.Offset{Col: col}); depth >= 0 {
			out[depth] = col
		}
	}
	return out
}

// rankOf is how deep into its own half a formation cell stands, counted from the
// rank that meets the enemy first, or -1 for a cell that is on no formation.
//
// A squad carries no side, so the question is asked of one and the answer holds
// for both: hex.Place maps an enemy formation through a 180 degree rotation, so
// a cell authored at a given depth is at that depth whichever half it is fielded
// as. TestARankIsTheSameDepthOnEitherSide is what says so.
func rankOf(slot hex.Offset) int {
	placed := hex.Place(hex.SideAlly, slot)
	for depth, rank := range hex.Ranks(hex.SideAlly) {
		for _, cell := range rank {
			if cell == placed {
				return depth
			}
		}
	}
	return -1
}

// rankNames is one wording per rank, front first. It is an array of exactly the
// ranks there are, so a board given a fourth column is a compile error here
// rather than a fourth rank quietly drawing no name.
var rankNames = [hex.FormationCols]i18n.Key{
	i18n.SquadRankFront, i18n.SquadRankMiddle, i18n.SquadRankBack,
}

// RankLabel is what to call the rank a cell stands in, and empty for a cell that
// is on no formation — a slot a hand-edited file put off the grid draws its
// coordinate alone rather than borrowing a rank it is not in.
func RankLabel(c Context, slot hex.Offset) string {
	depth := rankOf(slot)
	if depth < 0 || depth >= len(rankNames) {
		return ""
	}
	return c.Text(rankNames[depth])
}

// EditUnit takes one member of the squad in hand up for editing.
func (s SquadsScreen) EditUnit(index int) SquadsScreen {
	s.Mode = SquadUnit
	s.UnitIndex = index
	// The cursor on the squad behind follows, so coming back out lands on the
	// member that was open rather than wherever it was before.
	s.Units = index
	s.Unit = s.Editing.Units[index].Clone()
	s.UnitOpenedAs = s.Unit.Character
	s.Field = SquadUnitCharacter
	s.LevelInput.SetValue(strconv.Itoa(s.Unit.Level))
	s.LevelInput.Blur()
	s.Err, s.Notes = nil, nil
	return s
}

func (s SquadsScreen) updateUnit(c Context, message tea.KeyPressMsg) (SquadsScreen, Action, tea.Cmd) {
	switch message.String() {
	case "esc":
		s = s.Commit()
		s.Mode = SquadEdit
		return s.focus(), Action{}, nil
	case "up":
		s = s.moveField(-1)
	case "down":
		s = s.moveField(1)
	case "left":
		s = s.cycle(-1)
	case "right":
		s = s.cycle(1)
	case "enter":
		switch s.Field {
		case SquadUnitSkills:
			s = s.Commit()
			if picker := s.OpenSkills(); picker != nil {
				return s, Action{Kind: Pick, Picker: picker}, nil
			}
		case SquadUnitPassives:
			s = s.Commit()
			if picker := s.OpenPassives(); picker != nil {
				return s, Action{Kind: Pick, Picker: picker}, nil
			}
		}
	default:
		if s.Field == SquadUnitLevel && NumberKey(message) {
			updated, command := s.LevelInput.Update(message)
			s.LevelInput = updated
			if level, err := strconv.Atoi(strings.TrimSpace(updated.Value())); err == nil {
				s.Unit.Level = level
				s = s.settleStage()
			}
			s.Err = nil
			return s, Action{}, command
		}
	}
	return s, Action{}, nil
}

func (s SquadsScreen) moveField(by int) SquadsScreen {
	s.Field = (s.Field + by + SquadUnitFieldCount) % SquadUnitFieldCount
	if s.Field == SquadUnitLevel {
		s.LevelInput.Focus()
		return s
	}
	s.LevelInput.Blur()
	return s
}

// Commit writes the unit under edit back into the squad.
func (s SquadsScreen) Commit() SquadsScreen {
	if s.UnitIndex >= 0 && s.UnitIndex < len(s.Editing.Units) {
		s.Editing.Units[s.UnitIndex] = s.Unit.Clone()
	}
	return s
}

func (s SquadsScreen) cycle(by int) SquadsScreen {
	switch s.Field {
	case SquadUnitCharacter:
		// The character this member was opened with is on this list whether or
		// not it is held back, so stepping off it is a decision an author took
		// rather than one the filter took for them — and stepping back onto it
		// works, because the list does not move while it is being walked.
		offered := offeredCharacters(s.Characters, s.UnitOpenedAs)
		if len(offered) == 0 {
			return s
		}
		at := 0
		for i, character := range offered {
			if character.ID == s.Unit.Character {
				at = i
			}
		}
		chosen := offered[(at+by+len(offered))%len(offered)]
		// A kit belongs to the character it was chosen from, so changing the
		// character empties it rather than carrying names the new one has never
		// heard of into a refusal at save time.
		s.Unit.Character = chosen.ID
		s.Unit.Skills, s.Unit.Passives = nil, nil
		s.Unit.ID = s.freeID(chosen.ID, s.UnitIndex)
		s = s.settleStage()
	case SquadUnitSlot:
		slots := FormationSlots()
		at := 0
		for i, slot := range slots {
			if slot == s.Unit.Slot {
				at = i
			}
		}
		// Cells somebody else stands on are stepped over rather than refused:
		// two units on one cell is not a squad, and a chooser that stops on one
		// is a chooser that can be left in a state the save has to reject.
		for n := 1; n <= len(slots); n++ {
			candidate := slots[(at+by*n+len(slots)*len(slots))%len(slots)]
			if s.occupant(candidate) < 0 {
				s.Unit.Slot = candidate
				break
			}
		}
	case SquadUnitStage:
		forms := s.StageChoices()
		at := 0
		for i, form := range forms {
			if form == s.Unit.Stage {
				at = i
			}
		}
		s.Unit.Stage = forms[(at+by+len(forms))%len(forms)]
		s.Unit.Skills, s.Unit.Passives = nil, nil
	default:
		return s
	}
	s.Err = nil
	return s
}

// occupant is who else in the squad stands on a cell, or -1.
func (s SquadsScreen) occupant(slot hex.Offset) int {
	for i, unit := range s.Editing.Units {
		if i != s.UnitIndex && unit.Slot == slot {
			return i
		}
	}
	return -1
}

// StageChoices is the forms this unit may be fielded as: the empty string, which
// is the furthest the level reaches, and then every form by name.
//
// The empty option is offered first because it is what a placement means when it
// says nothing — and it is deliberately not the only option, because a line that
// forks has no furthest and the arms have to be nameable.
func (s SquadsScreen) StageChoices() []string {
	out := []string{""}
	character, known := s.Character()
	if !known {
		return out
	}
	stages, err := character.StagesAt(s.Unit.Level)
	if err != nil {
		return out
	}
	return append(out, progression.StageNames(stages)...)
}

// settleStage drops a chosen form the new character or level no longer offers,
// so the chooser never shows a name that is not on the list under it.
func (s SquadsScreen) settleStage() SquadsScreen {
	if s.Unit.Stage == "" {
		return s
	}
	for _, form := range s.StageChoices() {
		if form == s.Unit.Stage {
			return s
		}
	}
	s.Unit.Stage = ""
	return s
}

// holdsAHiddenCharacter is whether the member under edit names a character the
// cast has held back.
//
// A member naming nobody the book knows is not this: that is a squad pointing at
// a deleted character, which is a different fault with a different answer, and
// saying "held back" about it would be a guess.
func (s SquadsScreen) holdsAHiddenCharacter() bool {
	character, known := s.Character()
	return known && character.Hidden
}

// Character is the member under edit's own character, looked up in the whole
// cast this screen holds.
func (s SquadsScreen) Character() (cast.Character, bool) {
	return s.characterOf(s.Unit.Character)
}

// characterOf is the same lookup for a member that is not the one under edit,
// which is what the squad's own rows are: viewEdit draws every member and only
// one of them is open.
func (s SquadsScreen) characterOf(id string) (cast.Character, bool) {
	for _, character := range s.Characters {
		if character.ID == id {
			return character, true
		}
	}
	return cast.Character{}, false
}

// unnamedArms is the arms of a member's line when the placement has named none
// of them: empty on a line that does not fork, empty on a member that has
// chosen, and both ends otherwise.
//
// It is FormArms — the read-only views' own helper — rather than a second
// reading of a fork, because "which grown forms does this level reach" is one
// question and two answers to it is how two screens come to disagree about what
// a fork is. What is different here is only what the emptiness means: those
// views field the first arm, and a placement may not be fielded at all until its
// author names one.
func (s SquadsScreen) unnamedArms(unit placement.Placement) []progression.Stage {
	if unit.Stage != "" {
		return nil
	}
	character, known := s.characterOf(unit.Character)
	if !known {
		return nil
	}
	arms := FormArms(character, unit.Level)
	if len(arms) < 2 {
		return nil
	}
	return arms
}

// Form is the stage name the unit's learnset is read against: the one chosen, or
// the furthest the level reaches when nothing is.
//
// ⚠️ **On a fork the placement has not named it is the empty string, and that is
// a refusal rather than an answer.** progression.Line.StageAt will not pick an
// arm — a wrong arm is a wrong learnset written into the author's own file — so
// there is nothing to resolve to, and cast.SkillsAt/PassivesAt read an empty
// form as "no gate is held", which leaves the two pickers offering only what
// every arm learns. That narrowing is silent by construction: a list cannot say
// why a row is not on it. What says so is the screen — stageLabel stops calling
// this state *furthest*, a word with no meaning on a line with two ends, and the
// SquadForkArms line under the fields names the arms, the key that chooses one,
// and both consequences. The choice itself needs nothing new: the form field is
// already a chooser and StageChoices already lists both arms by name.
func (s SquadsScreen) Form() string {
	if s.Unit.Stage != "" {
		return s.Unit.Stage
	}
	character, known := s.Character()
	if !known {
		return ""
	}
	_, stage, err := character.Resolve(s.Unit.Level, progression.Furthest)
	if err != nil {
		return ""
	}
	return stage.Name
}

// OpenSkills and OpenPassives are two pickers over one rule: what this character
// knows at this level as this form. They are separate methods rather than one
// taking a field because each writes a different half of the loadout back, and a
// picker that decided which half it was would be a switch inside a callback.
//
// Each names its own hint for the reason the allowlists do: the picker's default
// is the form's, and the form is choosing out of the whole skill book, where a
// row may carry a refusal. Here the options are the learnset already, so a mark
// for what cannot be taken names something no row can ever draw — and on the
// trait list an order says nothing either, since there is one slot to fill.
//
// These two are also the only pickers that carry slots, and they carry the
// constants the write reads rather than numbers of their own. The cap stops an
// over-full answer being built; Refuse below and the save still ask the rule
// through cast.ChooseFrom, because the cap is a courtesy on top of the rule and
// never a replacement for it — a loadout hand-edited into squads.json never came
// through a keystroke at all.
//
// Nil is a member naming nobody the book knows, which is the one state that has
// no learnset to offer; the key that raises it declines rather than opening an
// empty list.
func (s SquadsScreen) OpenSkills() *PickState {
	character, known := s.Character()
	if !known {
		return nil
	}
	return &PickState{
		Title:   i18n.SquadPickSkills,
		Hint:    i18n.SquadKitHint,
		Kind:    PickSkills,
		Slots:   cast.SkillSlots,
		Options: IDOptions(character.SkillsAt(s.Unit.Level, s.Form())),
		Chosen:  append([]string(nil), s.Unit.Skills...),
		Into:    SquadsPickKit,
	}
}

func (s SquadsScreen) OpenPassives() *PickState {
	character, known := s.Character()
	if !known {
		return nil
	}
	return &PickState{
		Title:   i18n.SquadPickPassives,
		Hint:    i18n.SquadTraitHint,
		Kind:    PickPassives,
		Slots:   cast.TraitSlots,
		Options: IDOptions(character.PassivesAt(s.Unit.Level, s.Form())),
		Chosen:  append([]string(nil), s.Unit.Passives...),
		Into:    SquadsPickTrait,
	}
}

// Picked is what the builder's two lists write back: one half of the member's
// loadout each, judged by the rule the write uses and committed to the squad in
// hand.
//
// ⚠️ **The character is looked up here rather than carried from the raise.** The
// two closures this replaces captured it at the moment the list opened, and the
// capture was doing nothing a lookup does not: the picker takes every key while
// it is up, so the member under edit cannot have changed character between the
// question and the answer — and the screen already knows how to find its own,
// because every other row on it is read the same way.
//
// ⚠️ **The destination arrives as an `any`** — see SkillsScreen.Picked for why
// there is more than one destination vocabulary to tell apart.
func (s SquadsScreen) Picked(_ Context, into any, answer PickAnswer) (SquadsScreen, Action) {
	character, known := s.Character()
	if !known {
		// The raise declines on the same reading, so this is unreachable from a
		// keystroke; it is here because the learnset cannot be read without one.
		return s, Action{}
	}
	switch into {
	case SquadsPickKit:
		s.Unit.Skills = answer.Chosen
		s.Err = s.Refuse(cast.SkillSlots, answer.Chosen, "skill",
			character.SkillsAt(s.Unit.Level, s.Form()), cast.Required)
	case SquadsPickTrait:
		s.Unit.Passives = answer.Chosen
		s.Err = s.Refuse(cast.TraitSlots, answer.Chosen, "trait",
			character.PassivesAt(s.Unit.Level, s.Form()), cast.Optional)
	default:
		// A destination this screen does not own; see SkillsScreen.Picked.
		return s, Action{}
	}
	// Both stay on the member being edited, so both hand back the zero action.
	return s.Commit(), Action{}
}

// Refuse is the loadout rule asked early, so an over-filled slot is a line under
// the picker rather than a surprise at the save.
//
// It is the same call the write makes — cast.ChooseFrom — because a builder that
// judged a choice for itself would be a second copy of the rule, and the copy
// would be the one an author trusted.
func (s SquadsScreen) Refuse(slots int, chosen []string, kind string,
	available []string, insist bool) error {
	_, err := cast.ChooseFrom(fmt.Sprintf("unit %q", s.Unit.ID), kind, chosen,
		available, slots, s.Unit.Level, insist)
	return err
}

// Save writes the squad, and refuses exactly as a battle would.
func (s SquadsScreen) Save(c Context) SquadsScreen {
	s.Editing.ID = strings.TrimSpace(s.IDInput.Value())
	s.Editing.Name = strings.TrimSpace(s.NameInput.Value())
	if err := c.Lib.SaveSquad(s.Editing); err != nil {
		s.Err = err
		s.Notes = nil
		return s
	}
	s.Err = nil
	s.Notes = []forge.Note{
		{Kind: forge.NoteWrote, ID: s.Editing.ID, Path: c.Lib.SquadsPath()},
		{Kind: forge.NoteRebuild},
	}
	s.Baseline = s.Editing.Clone()
	return s.Refresh(c)
}

// View draws whichever of the three depths is in front, body and footer apart.
func (s SquadsScreen) View(c Context) (string, string) {
	switch s.Mode {
	case SquadEdit:
		return s.viewEdit(c), c.Text(i18n.SquadEditFooter, SaveKeyLabel())
	case SquadUnit:
		return s.viewUnit(c), c.Text(i18n.SquadUnitFooter)
	}
	return s.viewList(c), c.Footer(i18n.SquadsFooter, i18n.SquadsReadFooter)
}

func (s SquadsScreen) viewList(c Context) string {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SquadsHeading)) + "  " +
		c.Style.Dim.Render(c.Text(i18n.SquadsSubtitle, len(s.Saved), hex.MaxTeamSize)) + "\n\n")
	if len(s.Saved) == 0 {
		out.WriteString(c.Style.Dim.Render(c.Text(i18n.SquadsEmpty)) + "\n")
		return strings.TrimRight(out.String(), "\n")
	}
	// The id column is measured over what is drawn, header included, because a
	// squad is named by whoever built it and nothing bounds how long that is.
	width := lipgloss.Width(c.Text(i18n.SquadColumnID))
	for _, squad := range s.Saved {
		if drawn := lipgloss.Width(squad.ID); drawn > width {
			width = drawn
		}
	}
	out.WriteString("  " + c.Style.Dim.Render(Pad(c.Text(i18n.SquadColumnID), width)+" "+
		c.Text(i18n.SquadColumnMembers)) + "\n")
	from, to := Window(len(s.Saved), s.Cursor, squadRoom(c))
	for index := from; index < to; index++ {
		squad := s.Saved[index]
		marker := "  "
		line := Pad(squad.ID, width) + " " +
			c.Text(i18n.SquadMemberCount, len(squad.Units))
		if squad.Name != "" {
			line += "  " + squad.Name
		}
		if index == s.Cursor {
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + Clip(line, c.UsableWidth()-2) + "\n")
	}
	if s.Err != nil {
		out.WriteString("\n" + c.Style.Bad.Render(c.Lang.Error(s.Err)) + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

// squadRoom is how many rows the listing has, measured the way every listing
// here measures: the window less the frame, the heading, the column row and
// whatever is drawn under the list.
func squadRoom(c Context) int {
	const frame, heading, columns = 4, 2, 1
	room := c.Height - frame - heading - columns - 2
	if room < 3 {
		return 3
	}
	return room
}

func (s SquadsScreen) viewEdit(c Context) string {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SquadHeading)) + "\n\n")
	width := squadLabelWidth(c)
	id := s.IDInput.View()
	if !s.fresh() {
		id = s.Editing.ID
	}
	out.WriteString(c.LabelAt(c.Text(i18n.SquadFieldID), width, "%s", id))
	out.WriteString(c.LabelAt(c.Text(i18n.SquadFieldName), width, "%s", s.NameInput.View()))
	out.WriteString("\n")
	for index, unit := range s.Editing.Units {
		marker := "  "
		line := s.unitLine(c, index, unit)
		if index == s.Units {
			marker = "> "
			line = c.Style.Selected.Render(line)
		}
		out.WriteString(marker + Clip(line, c.UsableWidth()-2) + "\n")
	}
	add := c.Text(i18n.SquadAddMember)
	if len(s.Editing.Units) >= hex.MaxTeamSize {
		add = c.Text(i18n.SquadFull, hex.MaxTeamSize)
	}
	if s.Units >= len(s.Editing.Units) {
		out.WriteString("> " + c.Style.Selected.Render(add) + "\n")
	} else {
		out.WriteString("  " + c.Style.Dim.Render(add) + "\n")
	}
	out.WriteString("\n" + s.formation(c, -1))
	out.WriteString(s.report(c))
	return strings.TrimRight(out.String(), "\n")
}

// unitLine is one member of the squad as a row: who it is, how grown, where it
// stands and what it brings.
func (s SquadsScreen) unitLine(c Context, index int, unit placement.Placement) string {
	form := s.formLabel(c, unit)
	return fmt.Sprintf("%d %s %s@%d %s %s %s",
		index+1, unit.ID, unit.Character, unit.Level, form, unit.Slot,
		c.Text(i18n.SquadLoadoutCount, len(unit.Skills), cast.SkillSlots,
			len(unit.Passives), cast.TraitSlots))
}

func (s SquadsScreen) viewUnit(c Context) string {
	var out strings.Builder
	out.WriteString(c.Style.Heading.Render(c.Text(i18n.SquadUnitHeading, s.Unit.ID)) + "\n\n")
	width := squadLabelWidth(c)
	rows := []struct {
		label i18n.Key
		value string
	}{
		{i18n.SquadFieldCharacter, s.chooserValue(s.Unit.Character)},
		{i18n.SquadFieldLevel, s.LevelInput.View()},
		{i18n.SquadFieldStage, s.chooserValue(s.stageLabel(c))},
		{i18n.SquadFieldSlot, s.chooserValue(s.slotLabel(c))},
		{i18n.SquadFieldSkills, s.listValue(c, s.Unit.Skills, cast.SkillSlots)},
		{i18n.SquadFieldPassives, s.listValue(c, s.Unit.Passives, cast.TraitSlots)},
	}
	for field, row := range rows {
		marker := "  "
		value := row.value
		if field == s.Field {
			marker = "> "
			value = c.Style.Selected.Render(value)
		}
		out.WriteString(marker + c.LabelAt(c.Text(row.label), width-2, "%s", value))
	}
	// Drawn only while the member holds a character the cast has held back, which
	// is the one case offeredCharacters keeps something on the list that nothing
	// else offers. Without it the row is a character id like any other and the
	// flag reads as not working.
	if s.holdsAHiddenCharacter() {
		out.WriteString("  " + c.Style.Dim.Render(c.Text(i18n.SquadHeldBack)) + "\n")
	}
	// Drawn only while the member's line forks at its level and the placement has
	// named no arm, which is the one state the form row above cannot say enough
	// about by itself: the chooser has room for a value and this needs three
	// facts — which arms there are, that ←/→ picks one, and what not picking one
	// already costs. Both costs are otherwise invisible. The two loadout lists
	// quietly drop every arm-gated entry (see Form), and a list has no way to
	// mention a row it is not showing; and the save refuses the member with a
	// sentence about a fork that nothing before it had mentioned.
	//
	// Unaligned and under the rows for the reason the held-back line above is:
	// the label column is for facts about the unit, and this is a note about the
	// state of the form the row above it draws.
	//
	// ⚠️ **Wrapped at MinWidth rather than clipped like the held-back line**, and
	// that is the width rule rather than a preference: this is the only note on
	// the screen carrying a value out of the data — the arms, which are authored
	// stage names with no promised length — so a wording that fits the floor on
	// its own does not fit it beside them, and the held-back line's single clip
	// would take the consequences off the end. Wrapped at the floor and not at
	// the window because it is prose: a sentence measured against the terminal in
	// hand has one shape per window and TestEveryWordingFitsTheMinimumWidth has
	// nothing to hold.
	if arms := s.unnamedArms(s.Unit); len(arms) > 1 {
		note := c.Text(i18n.SquadForkArms, strings.Join(progression.StageNames(arms), " / "))
		for _, line := range WrapWords(note, MinWidth-3) {
			out.WriteString("  " + c.Style.Dim.Render(line) + "\n")
		}
	}
	out.WriteString("\n" + s.formation(c, s.UnitIndex))
	out.WriteString(s.report(c))
	return strings.TrimRight(out.String(), "\n")
}

// chooserValue draws a value that is stepped through with the arrow keys, in the
// decoration the character form's choosers already use.
func (s SquadsScreen) chooserValue(value string) string {
	return fmt.Sprintf("< %s >", value)
}

// slotLabel is where the unit stands, said twice: as the cell the data writes
// and as the rank a formation is imagined in.
//
// Both rather than the reading alone. The coordinate is what squads.json holds
// and what a roster beside it is authored in, so an author matching the screen
// against a file needs it — and the rank is the half that means something,
// because reach is counted in ranks and a coordinate does not say which end of
// the grid is met first.
func (s SquadsScreen) slotLabel(c Context) string {
	rank := RankLabel(c, s.Unit.Slot)
	if rank == "" {
		return s.Unit.Slot.String()
	}
	return fmt.Sprintf("%s  %s", s.Unit.Slot, rank)
}

// stageLabel is the value in the form chooser, and formLabel is the same value
// for any member of the squad — the row in viewEdit is the field in viewUnit for
// a member that is not open, so they are one wording read twice rather than two.
//
// ⚠️ **An empty stage is two different things and used to be worded as one.** On
// a line that does not fork it is the furthest form the level reaches, which is
// what SquadFurthest says and what it goes on saying — that reading is unchanged
// here, and so is every record of it. On a line that forks it is a form nobody
// has named, and calling *that* furthest names nothing: the level has two ends,
// StageAt refuses to choose between them, and the placement is not fieldable
// until its author does. See Form for what else that state costs.
func (s SquadsScreen) stageLabel(c Context) string {
	return s.formLabel(c, s.Unit)
}

func (s SquadsScreen) formLabel(c Context, unit placement.Placement) string {
	if unit.Stage != "" {
		return unit.Stage
	}
	if len(s.unnamedArms(unit)) > 1 {
		return c.Text(i18n.SquadForkUnnamed)
	}
	return c.Text(i18n.SquadFurthest)
}

// listValue is a chosen list with how full its slots are, so an unfinished kit
// says so before the save does.
func (s SquadsScreen) listValue(c Context, chosen []string, slots int) string {
	if len(chosen) == 0 {
		return c.Style.Dim.Render(c.Text(i18n.SquadNothingChosen, slots))
	}
	return fmt.Sprintf("%s %s", strings.Join(chosen, " "),
		c.Style.Dim.Render(c.Text(i18n.ChoicePosition, len(chosen), slots)))
}

// The grid's own measurements. A cell is three characters wide because that is
// what "[n]" is, and n is a member number of a squad that fields hex.MaxTeamSize
// — one digit — so nothing widens it. The marker under the front column is built
// from the same figure rather than written out, so the two cannot part company.
const (
	formationIndent = "    "
	formationCell   = 3
)

// formation draws the squad's own 3x3, front column first, with the unit under
// edit marked and the front rank called out under it.
//
// editing is the member the arrows are moving, or -1 where there is none. It is
// what makes the picture live: see unitsDrawn.
//
// ASCII and nothing else: the box-drawing and arrow glyphs that would make it
// prettier are East-Asian-Ambiguous, which lipgloss measures as one cell and
// half the terminals draw as two — so the grid would overlap whatever is beside
// it on exactly the machines the client is used on. That is why the front rank
// is marked with carets rather than with an arrow.
func (s SquadsScreen) formation(c Context, editing int) string {
	var out strings.Builder
	out.WriteString(c.Style.Dim.Render(c.Text(i18n.SquadFormation)) + "\n")
	units := s.unitsDrawn(editing)
	columns := formationColumns()
	for row := 0; row < hex.FormationRows; row++ {
		line := formationIndent
		for _, col := range columns {
			slot := hex.Offset{Col: col, Row: row}
			cell := " . "
			for index, unit := range units {
				if unit.Slot != slot {
					continue
				}
				cell = fmt.Sprintf("[%d]", index+1)
				if index == editing {
					cell = fmt.Sprintf("(%d)", index+1)
				}
			}
			line += cell + " "
		}
		out.WriteString(line + "\n")
	}
	// The marker sits under the first column drawn, which is the front rank by
	// construction: formationColumns orders them by depth.
	out.WriteString(c.Style.Dim.Render(formationIndent+strings.Repeat("^", formationCell)+
		" "+c.Text(i18n.SquadFormationFront)) + "\n")
	out.WriteString(c.Style.Dim.Render(formationIndent+c.Text(i18n.SquadFormationLegend)) + "\n")
	return out.String()
}

// unitsDrawn is the squad as the formation should picture it: every committed
// member, with the one under edit replaced by the copy the arrows are moving.
//
// This is the whole of what makes the grid live. ←/→ steps s.Unit and Commit is
// what writes that back, and Commit runs only when the member is left or a
// picker is opened — so a drawing off the committed list alone stays on the old
// cell for the entire time the cell is being chosen, which is exactly when the
// picture is being looked at.
//
// It is not fixed by committing on every keypress either. s.Editing.Units is
// shared with every copy of this value, so a write from inside a drawing reaches
// all of them — which is what a value receiver looks like it prevents and does
// not. Reading the unit under edit costs nothing and writes nothing.
//
// The guard on leaving no longer rides on that: it compares the squad in hand
// against the one last written (see Dirty), so a cursor that passed over a cell
// and came back leaves it down by arithmetic rather than by nobody having
// called Commit.
//
// editing is -1 from the squad view, where there is no member under edit and the
// committed list is the whole truth. The substitution replaces rather than
// appends, so the cell a member steps off empties in the same draw the cell it
// steps onto fills — and the copy is into a new slice, because s.Editing.Units
// is shared with every copy of this value and a write into it would reach them
// all from inside a drawing.
func (s SquadsScreen) unitsDrawn(editing int) []placement.Placement {
	if editing < 0 || editing >= len(s.Editing.Units) {
		return s.Editing.Units
	}
	out := append([]placement.Placement(nil), s.Editing.Units...)
	out[editing] = s.Unit
	return out
}

// report is the refusal or the confirmation under whichever view is in front.
func (s SquadsScreen) report(c Context) string {
	if s.Err != nil {
		return "\n" + c.Style.Bad.Render(c.Text(i18n.WriteRefused, c.Lang.Error(s.Err))) + "\n"
	}
	if len(s.Notes) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	for index, line := range c.Lang.Notes(s.Notes) {
		style := c.Style.Dim
		if index == 0 {
			style = c.Style.Good
		}
		out.WriteString(style.Render(line) + "\n")
	}
	return out.String()
}

// squadLabelWidth is measured rather than fixed, like every other label column
// in this program: the two languages word these differently and a constant would
// be right for one of them.
func squadLabelWidth(c Context) int {
	width := 0
	for _, key := range []i18n.Key{
		i18n.SquadFieldID, i18n.SquadFieldName, i18n.SquadFieldCharacter,
		i18n.SquadFieldLevel, i18n.SquadFieldStage, i18n.SquadFieldSlot,
		i18n.SquadFieldSkills, i18n.SquadFieldPassives,
	} {
		if drawn := lipgloss.Width(c.Text(key)); drawn > width {
			width = drawn
		}
	}
	return width + 3
}

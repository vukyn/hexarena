package battle_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
)

// callersOwnRow is a value no battle emits, so finding it inside the record —
// or failing to find it in the caller's own slice — names an aliasing fault
// exactly rather than reporting two events that merely differ.
var callersOwnRow = battle.Event{Kind: battle.Started, Actor: "the-callers-own-row"}

// oneTurn drives exactly one turn with the rating that ships, which is the
// smallest thing that makes the battle emit.
func oneTurn(t *testing.T, fight *battle.Battle) {
	t.Helper()
	if _, err := fight.RunToEndWith(1, fight.Suggest); err != nil {
		t.Fatalf("one turn: %v", err)
	}
}

// TestAViewAndTheRecordSurviveEachOthersAppends is the load-bearing test of the
// record, because the fault it holds off corrupts BOTH sides and neither side
// can see the other.
//
// A view into an append-only record shares the record's backing array. If that
// view carried the array's spare capacity, a caller's own `append` would write
// into the slot the battle's next `emit` is about to use — so the battle would
// overwrite what the caller appended, and a caller appending after the battle
// had already written there would destroy an event out of the record. Both
// directions are driven here, in that order, off one view.
//
// Both clients hit this: internal/screen/play.go takes a view and then appends
// to it, and cmd/hexarena/main.go does the same. So the view is three-index
// capped (`b.events[from:n:n]`), which makes a caller's append reallocate.
func TestAViewAndTheRecordSurviveEachOthersAppends(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)
	fight.Begin()

	// A sweep over ten record lengths rather than one reading, because the
	// fault is only *reachable* while the record's array has spare capacity and
	// a caller cannot ask a slice about somebody else's capacity. Append growth
	// is amortised, so only a minority of lengths have none; ten readings at
	// ten lengths is what stops this being the vacuous fixture. `checked`
	// counts the readings that could have seen something at all.
	checked := 0
	for turn := 0; turn < 10 && !fight.Finished(); turn++ {
		before := fight.Recorded()
		view, cursor := fight.Since(0)
		if len(view) != before || cursor != before {
			t.Fatalf("Since(0) answered %d events and cursor %d against a record of %d",
				len(view), cursor, before)
		}
		// The property stated directly, which is the sharpest form of it: a
		// view's capacity is its length, so a caller's append cannot reach the
		// record's next slot however the runtime grew the array.
		// Reported and not fatal, so the two corruption readings below still
		// run and say what the spare capacity actually costs: a bare capacity
		// figure is the cause and they are the damage.
		if cap(view) != len(view) {
			t.Errorf("at %d events the view carries %d of spare capacity",
				before, cap(view)-len(view))
		}

		// One: the caller appends, then the battle emits.
		early := append(view, callersOwnRow) //nolint:gocritic // the append is the measurement
		oneTurn(t, fight)
		after := fight.Recorded()
		if after == before {
			continue
		}
		checked++
		if early[before] != callersOwnRow {
			t.Errorf("at %d events the battle's next emit overwrote the caller's own row with %+v",
				before, early[before])
		}

		// Two: the same view, appended to now the record has grown past it.
		late := append(view, callersOwnRow) //nolint:gocritic // the append is the measurement
		if late[before] != callersOwnRow {
			t.Errorf("at %d events a late append did not keep the caller's own row, got %+v",
				before, late[before])
		}
		record, _ := fight.Since(0)
		for index, event := range record {
			if event == callersOwnRow {
				t.Errorf("the caller's own row reached the record at %d of %d, over the event that belonged there",
					index, len(record))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no turn in the sweep emitted anything, so this fixture measured nothing")
	}
}

// TestTwoCursorsDoNotDisturbEachOther is the whole point of a cursor per
// consumer: a room has two players, spectators and a log, and a read is a
// position rather than a consumption.
func TestTwoCursorsDoNotDisturbEachOther(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)
	fight.Begin()

	// One consumer reads the opening board and keeps a cursor past it. The
	// other is a spectator joining mid-battle that wants the lot, which is
	// what Since(0) is for.
	opening, player := fight.Since(0)
	if len(opening) == 0 {
		t.Fatal("the opening board recorded nothing")
	}
	const spectator = 0

	if _, err := fight.RunToEndWith(3, fight.Suggest); err != nil {
		t.Fatalf("run: %v", err)
	}

	fresh, playerNext := fight.Since(player)
	if len(fresh) == 0 {
		t.Fatal("three turns recorded nothing, so the two cursors are at the same place")
	}
	whole, spectatorNext := fight.Since(spectator)

	if playerNext != spectatorNext {
		t.Errorf("the two consumers came back holding cursors %d and %d", playerNext, spectatorNext)
	}
	if len(whole) != len(opening)+len(fresh) {
		t.Fatalf("the spectator read %d events against the player's %d + %d",
			len(whole), len(opening), len(fresh))
	}
	for index, event := range opening {
		if whole[index] != event {
			t.Errorf("event %d differs:\n%+v\n%+v", index, whole[index], event)
		}
	}
	for index, event := range fresh {
		if whole[len(opening)+index] != event {
			t.Errorf("event %d differs:\n%+v\n%+v", len(opening)+index, whole[len(opening)+index], event)
		}
	}

	// And the spectator's read did not move the player: the same cursor answers
	// the same events a second time.
	again, againNext := fight.Since(player)
	if len(again) != len(fresh) || againNext != playerNext {
		t.Errorf("reading cursor %d twice answered %d then %d events", player, len(fresh), len(again))
	}
}

// TestADrainAndACursorDoNotMoveEachOther is the property that lets the several
// hundred existing Drain callers stay exactly as they are while a room reads
// the same battle three other ways. Drain is a consumer holding a cursor the
// battle keeps for it, and it is not privileged.
func TestADrainAndACursorDoNotMoveEachOther(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)
	fight.Begin()
	const watcher = 0

	drained := fight.Drain()
	if len(drained) == 0 {
		t.Fatal("the opening board drained nothing")
	}

	// The Drain took the opening board and did not move the watcher, which
	// still answers all of it.
	whole, _ := fight.Since(watcher)
	if len(whole) != len(drained) {
		t.Fatalf("the watcher reads %d events where the drain took %d", len(whole), len(drained))
	}
	for index, event := range drained {
		if whole[index] != event {
			t.Errorf("event %d differs:\n%+v\n%+v", index, whole[index], event)
		}
	}

	// And the watcher's read did not move the Drain: nothing new has been
	// emitted, so the next Drain is empty even though a cursor just read
	// everything the first one took.
	if again := fight.Drain(); again != nil {
		t.Errorf("a drain after a cursor read the same events answered %d events, want nil", len(again))
	}

	oneTurn(t, fight)

	// One turn on: the watcher still counts from the top and the Drain only
	// takes what is new.
	grown, _ := fight.Since(watcher)
	taken := fight.Drain()
	if len(taken) == 0 {
		t.Fatal("a turn recorded nothing, so this measures nothing")
	}
	// Fatal rather than Error: the elementwise comparison below indexes
	// grown at len(whole)+index, which cannot be in range once the lengths
	// disagree. Reporting and carrying on would panic the test instead of
	// failing it, and a panicking test takes the rest of the package's
	// tests down with it — so a real regression here would hide whatever
	// else was about to catch it.
	if len(grown) != len(whole)+len(taken) {
		t.Fatalf("the watcher reads %d events against %d already drained + %d newly drained",
			len(grown), len(whole), len(taken))
	}
	for index, event := range taken {
		if grown[len(whole)+index] != event {
			t.Errorf("event %d differs:\n%+v\n%+v", len(whole)+index, grown[len(whole)+index], event)
		}
	}
}

// TestTheRecordHoldsEveryEventEveryDrainTook is the reconnect and spectate
// property, and it is the one that would rot silently: if Drain ever dropped an
// event, or the record ever forgot one, a player who reconnected would be
// handed a different battle from the one that was fought and nothing else here
// would say so.
func TestTheRecordHoldsEveryEventEveryDrainTook(t *testing.T) {
	fight := duel(t, []string{"strike", "pop", "brace"}, []string{"strike", "envenom"}, 120, 100)
	fight.Begin()

	var taken []battle.Event
	taken = append(taken, fight.Drain()...)
	for turn := 0; turn < 2000 && !fight.Finished(); turn++ {
		oneTurn(t, fight)
		taken = append(taken, fight.Drain()...)
	}
	if !fight.Finished() {
		t.Fatalf("the battle did not finish inside the turn limit, %d events in", len(taken))
	}
	if len(taken) < 40 {
		t.Fatalf("the battle produced only %d events, too short to be a real check", len(taken))
	}

	whole, cursor := fight.Since(0)
	if cursor != fight.Recorded() {
		t.Errorf("Since(0) came back holding cursor %d against a record of %d", cursor, fight.Recorded())
	}
	if len(whole) != len(taken) {
		t.Fatalf("the record holds %d events against the %d every drain took", len(whole), len(taken))
	}
	for index := range taken {
		if whole[index] != taken[index] {
			t.Fatalf("event %d differs:\n%+v\n%+v", index, whole[index], taken[index])
		}
	}
}

// TestDrainAnswersNilWhenNothingIsNew pins the one part of Drain's observable
// behaviour that the record could most easily have changed. `out := b.events`
// answered nil for an empty record because the previous Drain had set the field
// to nil; a cursor into a record answers a slice, and a zero-length view of a
// full array is not nil. Several hundred call sites read this answer.
func TestDrainAnswersNilWhenNothingIsNew(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)

	if drained := fight.Drain(); drained != nil {
		t.Errorf("a battle that has recorded nothing drained %d events, want nil", len(drained))
	}
	fight.Begin()
	if drained := fight.Drain(); len(drained) == 0 {
		t.Fatal("the opening board drained nothing")
	}
	if drained := fight.Drain(); drained != nil {
		t.Errorf("a second drain with nothing new answered %d events, want nil", len(drained))
	}

	// Since answers the same way at the end of the record, so a consumer that
	// is up to date and one that has just drained cannot disagree about what
	// "nothing new" looks like.
	recorded := fight.Recorded()
	if fresh, cursor := fight.Since(recorded); fresh != nil || cursor != recorded {
		t.Errorf("Since(%d) at the end of the record answered %d events and cursor %d",
			recorded, len(fresh), cursor)
	}
}

// TestACursorOutsideTheRecordIsLoud holds the refusal rather than the
// arithmetic. A consumer that has somehow got ahead of the record and is handed
// an empty slice looks exactly like one that is up to date, which is the silent
// desync a cursor exists to prevent — so it panics, on the same reading
// rng.Intn takes of a non-positive bound: a cursor is a number Since handed the
// caller itself, so a bad one is a programming error and not a runtime
// condition.
func TestACursorOutsideTheRecordIsLoud(t *testing.T) {
	fight := duel(t, []string{"strike"}, []string{"strike"}, 100, 100)
	fight.Begin()
	recorded := fight.Recorded()
	if recorded == 0 {
		t.Fatal("the opening board recorded nothing, so there is no end to be past")
	}

	cases := []struct {
		name   string
		cursor int
	}{
		{"negative", -1},
		{"far negative", -recorded - 1000},
		{"one past the end", recorded + 1},
		{"far past the end", recorded + 1000},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Since(%d) against a record of %d returned instead of refusing",
						one.cursor, recorded)
				}
			}()
			events, cursor := fight.Since(one.cursor)
			t.Errorf("Since(%d) answered %d events and cursor %d", one.cursor, len(events), cursor)
		})
	}

	// The end of the record is IN range and must not be loud: it is exactly
	// what a consumer with nothing new to read holds.
	t.Run("the end of the record", func(t *testing.T) {
		if events, cursor := fight.Since(recorded); events != nil || cursor != recorded {
			t.Errorf("Since(%d) answered %d events and cursor %d", recorded, len(events), cursor)
		}
	})
}

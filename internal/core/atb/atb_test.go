package atb_test

import (
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/atb"
)

func queueOf(t *testing.T, units ...struct {
	id    string
	speed int64
}) *atb.Queue {
	t.Helper()
	queue := atb.New()
	for _, unit := range units {
		if err := queue.Add(unit.id, unit.speed); err != nil {
			t.Fatalf("add %s: %v", unit.id, err)
		}
	}
	return queue
}

type unit = struct {
	id    string
	speed int64
}

func TestWaitIsScaleOverSpeed(t *testing.T) {
	cases := []struct {
		speed, want int64
	}{
		{1, atb.Scale},
		{80, 12500},
		{100, 10_000},
		{150, 6666},
		{200, 5000},
		// Speed below one is treated as one rather than dividing by zero.
		{0, atb.Scale},
		{-40, atb.Scale},
		// The wait never falls below one, so no unit acts infinitely often.
		{atb.Scale * 4, 1},
	}
	for _, testCase := range cases {
		if got := atb.Wait(testCase.speed); got != testCase.want {
			t.Errorf("Wait(%d) = %d, want %d", testCase.speed, got, testCase.want)
		}
	}
}

// TestSpeedIsTurnsPerCycle is the property that makes speed legible: over one
// cycle of action value a unit takes as many turns as it has speed.
func TestSpeedIsTurnsPerCycle(t *testing.T) {
	for _, speed := range []int64{80, 90, 110, 150, 200} {
		queue := queueOf(t, unit{"solo", speed})
		turns := 0
		for {
			preview := queue.Preview(1)
			if len(preview) == 0 || preview[0].At > atb.Scale {
				break
			}
			queue.Next()
			turns++
		}
		// Truncation of the integer wait can cost at most one turn per cycle.
		if int64(turns) < speed-1 || int64(turns) > speed {
			t.Errorf("speed %d took %d turns in one cycle, want about %d", speed, turns, speed)
		}
	}
}

func TestTurnShareMatchesTheSpeedRatio(t *testing.T) {
	queue := queueOf(t, unit{"swift", 200}, unit{"slow", 80})
	counts := map[string]int{}
	for i := 0; i < 2800; i++ {
		turn, ok := queue.Next()
		if !ok {
			t.Fatal("the queue emptied")
		}
		counts[turn.ID]++
	}
	// 200 against 80 is a ratio of 2.5, so the share should be 2000 per mille.
	ratio := counts["swift"] * 1000 / counts["slow"]
	if ratio < 2450 || ratio > 2550 {
		t.Errorf("swift took %d turns and slow %d, a ratio of %d per mille, want about 2500",
			counts["swift"], counts["slow"], ratio)
	}
}

func TestEqualSpeedsRotateInJoinOrder(t *testing.T) {
	queue := queueOf(t, unit{"a", 100}, unit{"b", 100}, unit{"c", 100})
	want := []string{"a", "b", "c", "a", "b", "c", "a", "b", "c"}
	for i, expected := range want {
		turn, ok := queue.Next()
		if !ok {
			t.Fatalf("turn %d: the queue emptied", i)
		}
		if turn.ID != expected {
			t.Errorf("turn %d went to %s, want %s", i, turn.ID, expected)
		}
	}
}

// TestTheFasterUnitWinsATie keeps the tie-break from being arbitrary in a way a
// player would notice: when two units come due at the same instant, the faster
// one goes first.
func TestTheFasterUnitWinsATie(t *testing.T) {
	queue := atb.New()
	// Speeds of 100 and 200 both divide the scale exactly, so they collide on
	// every second turn of the slower one.
	if err := queue.Add("slow", 100); err != nil {
		t.Fatalf("add slow: %v", err)
	}
	if err := queue.Add("swift", 200); err != nil {
		t.Fatalf("add swift: %v", err)
	}
	order := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		turn, _ := queue.Next()
		order = append(order, turn.ID)
	}
	if got, want := strings.Join(order, ","), "swift,swift,slow,swift,swift,slow"; got != want {
		t.Errorf("the order was %q, want %q", got, want)
	}
}

func TestTurnNumberAndCycleAdvance(t *testing.T) {
	queue := queueOf(t, unit{"solo", 100})
	for expected := 1; expected <= 250; expected++ {
		turn, ok := queue.Next()
		if !ok {
			t.Fatalf("turn %d: the queue emptied", expected)
		}
		if turn.Number != expected {
			t.Errorf("turn %d reported number %d", expected, turn.Number)
		}
		if turn.ID != "solo" || turn.Speed != 100 {
			t.Errorf("turn %d reported %s at speed %d", expected, turn.ID, turn.Speed)
		}
		wantCycle := int(turn.At / atb.Scale)
		if turn.Cycle != wantCycle {
			t.Errorf("turn %d reported cycle %d, want %d", expected, turn.Cycle, wantCycle)
		}
	}
	if got, want := queue.Turns("solo"), 250; got != want {
		t.Errorf("the unit has taken %d turns, want %d", got, want)
	}
	if got, want := queue.Cycle(), 2; got != want {
		t.Errorf("the queue is on cycle %d, want %d", got, want)
	}
}

// TestRescheduleKeepsTheServedFraction is the rule that stops a speed buff from
// handing out a partial turn, and stops alternating a buff and a debuff from
// stalling a unit for ever.
func TestRescheduleKeepsTheServedFraction(t *testing.T) {
	queue := queueOf(t, unit{"slow", 100}, unit{"tick", 1})
	// Let the slow unit serve part of its wait, then double its speed.
	queue.Next()
	before := queue.Pending("slow")
	if before != atb.Wait(100) {
		t.Fatalf("the unit has %d pending, want a full wait of %d", before, atb.Wait(100))
	}
	if err := queue.Reschedule("slow", 200); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if got, want := queue.Pending("slow"), before/2; got != want {
		t.Errorf("after doubling the speed there is %d pending, want %d", got, want)
	}
	if got, want := queue.Speed("slow"), int64(200); got != want {
		t.Errorf("the unit is scheduled at speed %d, want %d", got, want)
	}

	// Halving it again puts the pending value back where it started.
	if err := queue.Reschedule("slow", 100); err != nil {
		t.Fatalf("reschedule back: %v", err)
	}
	if got := queue.Pending("slow"); got != before {
		t.Errorf("after halving the speed again there is %d pending, want the original %d", got, before)
	}
}

// TestRescheduleDriftIsBounded covers the one place the integer scaling is not
// exact. Each change truncates, so speeds that do not divide evenly lose up to
// one action value per change and up to two per buff-and-removal pair. The loss
// must always favour the unit, because drifting the other way would mean a buff
// and a debuff alternated on a target could stall it for ever, and it must stay
// at two per pair so that eating a wait of thousands takes more pairs than a
// battle contains.
func TestRescheduleDriftIsBounded(t *testing.T) {
	const (
		baseSpeed   = 80
		buffedSpeed = 117
		pairs       = 300
	)
	queue := queueOf(t, unit{"target", baseSpeed}, unit{"tick", 1})
	queue.Next()
	start := queue.Pending("target")
	worstPair := int64(0)
	for i := 0; i < pairs; i++ {
		before := queue.Pending("target")
		if err := queue.Reschedule("target", buffedSpeed); err != nil {
			t.Fatalf("buff %d: %v", i, err)
		}
		if err := queue.Reschedule("target", baseSpeed); err != nil {
			t.Fatalf("debuff %d: %v", i, err)
		}
		after := queue.Pending("target")
		if after > before {
			t.Fatalf("pair %d moved the unit backwards, from %d pending to %d", i, before, after)
		}
		if drift := before - after; drift > worstPair {
			worstPair = drift
		}
	}
	// One truncation per change, so at most two per pair.
	if worstPair > 2 {
		t.Errorf("a single buff and debuff pair drifted by %d action value, want at most 2", worstPair)
	}
	if got, want := start-queue.Pending("target"), int64(2*pairs); got > want {
		t.Errorf("%d pairs drifted by %d in total, want at most %d", pairs, got, want)
	}
	// The drift is small enough that the pairs needed to consume a whole wait
	// exceed anything a battle could contain.
	if drift := start - queue.Pending("target"); drift > 0 {
		pairsToConsume := start / (drift/int64(pairs) + 1)
		if pairsToConsume < 1000 {
			t.Errorf("a wait of %d would be consumed in %d pairs, want far more", start, pairsToConsume)
		}
	}
	// Speeds that divide evenly are exact, so the drift is not inherent to the
	// rule, only to the rounding.
	exact := queueOf(t, unit{"even", 100})
	exact.Next()
	before := exact.Pending("even")
	for i := 0; i < 500; i++ {
		if err := exact.Reschedule("even", 200); err != nil {
			t.Fatalf("buff %d: %v", i, err)
		}
		if err := exact.Reschedule("even", 100); err != nil {
			t.Fatalf("debuff %d: %v", i, err)
		}
	}
	if got := exact.Pending("even"); got != before {
		t.Errorf("500 pairs at evenly dividing speeds drifted from %d to %d", before, got)
	}
}

func TestRescheduleEdges(t *testing.T) {
	queue := queueOf(t, unit{"solo", 100})
	if err := queue.Reschedule("nobody", 200); err == nil {
		t.Error("rescheduling a unit that is not there was accepted")
	}
	if err := queue.Reschedule("solo", 100); err != nil {
		t.Errorf("rescheduling to the same speed failed: %v", err)
	}
	if err := queue.Reschedule("solo", -50); err != nil {
		t.Errorf("rescheduling to a negative speed failed: %v", err)
	}
	if got, want := queue.Speed("solo"), int64(1); got != want {
		t.Errorf("a negative speed became %d, want %d", got, want)
	}
}

func TestGrantTurnPutsAUnitNext(t *testing.T) {
	queue := queueOf(t, unit{"swift", 200}, unit{"slow", 80})
	queue.Next()
	if !queue.GrantTurn("slow") {
		t.Fatal("granting a turn to a unit in the queue failed")
	}
	turn, ok := queue.Next()
	if !ok || turn.ID != "slow" {
		t.Errorf("the granted turn went to %s", turn.ID)
	}
	if turn.At != queue.Now() {
		t.Errorf("the granted turn is at %d but the clock is at %d", turn.At, queue.Now())
	}
	if queue.GrantTurn("nobody") {
		t.Error("granting a turn to a unit that is not there succeeded")
	}
}

func TestDelayPushesATurnBack(t *testing.T) {
	queue := queueOf(t, unit{"target", 100}, unit{"other", 100})
	queue.Next()
	before := queue.Pending("target")
	added := queue.Delay("target", 500)
	if want := atb.Wait(100) / 2; added != want {
		t.Errorf("a half wait delay added %d, want %d", added, want)
	}
	if got, want := queue.Pending("target"), before+added; got != want {
		t.Errorf("there is %d pending, want %d", got, want)
	}
	if got := queue.Delay("nobody", 500); got != 0 {
		t.Errorf("delaying a unit that is not there added %d", got)
	}
	if got := queue.Delay("target", 0); got != 0 {
		t.Errorf("a delay of nothing added %d", got)
	}
	// A tiny share still costs at least one action value, so a delay is never
	// silently free.
	if got := queue.Delay("target", 1); got < 1 {
		t.Errorf("a one per mille delay added %d, want at least 1", got)
	}
}

func TestPreviewDoesNotMutate(t *testing.T) {
	queue := queueOf(t, unit{"a", 200}, unit{"b", 110}, unit{"c", 80})
	before := queue.Standings()
	preview := queue.Preview(12)
	if len(preview) != 12 {
		t.Fatalf("the preview holds %d turns, want 12", len(preview))
	}
	after := queue.Standings()
	if len(before) != len(after) {
		t.Fatalf("the queue changed size")
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("standing %d changed from %+v to %+v", i, before[i], after[i])
		}
	}
	if got := queue.Now(); got != 0 {
		t.Errorf("the clock advanced to %d", got)
	}
	// Taking the turns for real must match what the preview promised.
	for i, expected := range preview {
		actual, ok := queue.Next()
		if !ok {
			t.Fatalf("turn %d: the queue emptied", i)
		}
		if actual != expected {
			t.Errorf("turn %d was %+v, the preview said %+v", i, actual, expected)
		}
	}
	if got := queue.Preview(0); got != nil {
		t.Errorf("a preview of nothing returned %v", got)
	}
}

func TestAddAndRemove(t *testing.T) {
	queue := atb.New()
	if err := queue.Add("", 100); err == nil {
		t.Error("a unit with no id was accepted")
	}
	if err := queue.Add("a", 100); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := queue.Add("a", 200); err == nil {
		t.Error("a duplicate id was accepted")
	}
	if !queue.Has("a") || queue.Len() != 1 {
		t.Errorf("the queue holds %d units and has(a) is %v", queue.Len(), queue.Has("a"))
	}
	if !queue.Remove("a") {
		t.Error("removing a unit in the queue failed")
	}
	if queue.Remove("a") {
		t.Error("removing a unit twice succeeded")
	}
	if _, ok := queue.Next(); ok {
		t.Error("an empty queue produced a turn")
	}
	if got := queue.Turns("a"); got != 0 {
		t.Errorf("a unit that is gone reports %d turns", got)
	}
	if got := queue.Speed("a"); got != 0 {
		t.Errorf("a unit that is gone reports speed %d", got)
	}
	if got := queue.Pending("a"); got != 0 {
		t.Errorf("a unit that is gone reports %d pending", got)
	}
}

// TestAUnitJoiningMidBattleWaitsItsFullTurn keeps a summon from acting the
// instant it arrives.
func TestAUnitJoiningMidBattleWaitsItsFullTurn(t *testing.T) {
	queue := queueOf(t, unit{"host", 100})
	queue.Next()
	queue.Next()
	if err := queue.Add("summon", 150); err != nil {
		t.Fatalf("add: %v", err)
	}
	if got, want := queue.Pending("summon"), atb.Wait(150); got != want {
		t.Errorf("the summon has %d pending, want a full wait of %d", got, want)
	}
	if got := queue.Turns("summon"); got != 0 {
		t.Errorf("the summon has already taken %d turns", got)
	}
}

func TestRemovingTheActingUnitDoesNotSkipTheRest(t *testing.T) {
	queue := queueOf(t, unit{"a", 200}, unit{"b", 100}, unit{"c", 100})
	turn, _ := queue.Next()
	if !queue.Remove(turn.ID) {
		t.Fatalf("removing %s failed", turn.ID)
	}
	seen := map[string]bool{}
	for i := 0; i < 6; i++ {
		next, ok := queue.Next()
		if !ok {
			t.Fatalf("turn %d: the queue emptied", i)
		}
		if next.ID == turn.ID {
			t.Errorf("the removed unit %s acted again", next.ID)
		}
		seen[next.ID] = true
	}
	if len(seen) != 2 {
		t.Errorf("%d units acted after the removal, want 2", len(seen))
	}
}

// TestTwoQueuesAgree is the replay guarantee: the same units in the same order
// produce the same sequence, with no dependence on map iteration.
func TestTwoQueuesAgree(t *testing.T) {
	build := func() *atb.Queue {
		queue := atb.New()
		for _, u := range []unit{{"bulwark", 80}, {"sentinel", 90}, {"vanguard", 110}, {"duelist", 150}, {"skirmisher", 200}} {
			if err := queue.Add(u.id, u.speed); err != nil {
				t.Fatalf("add %s: %v", u.id, err)
			}
		}
		return queue
	}
	first, second := build(), build()
	for i := 0; i < 400; i++ {
		if i == 137 {
			if err := first.Reschedule("bulwark", 160); err != nil {
				t.Fatalf("reschedule: %v", err)
			}
			if err := second.Reschedule("bulwark", 160); err != nil {
				t.Fatalf("reschedule: %v", err)
			}
		}
		if i == 201 {
			first.Remove("duelist")
			second.Remove("duelist")
		}
		a, okA := first.Next()
		b, okB := second.Next()
		if okA != okB || a != b {
			t.Fatalf("turn %d differs: %+v against %+v", i, a, b)
		}
	}
}

func TestStandingsAreInTurnOrder(t *testing.T) {
	queue := queueOf(t, unit{"slow", 80}, unit{"swift", 200}, unit{"mid", 110})
	standings := queue.Standings()
	if len(standings) != 3 {
		t.Fatalf("there are %d standings, want 3", len(standings))
	}
	previous := int64(-1)
	for i, standing := range standings {
		if standing.Pending < previous {
			t.Errorf("standing %d has %d pending, less than the one before it", i, standing.Pending)
		}
		previous = standing.Pending
	}
	if standings[0].ID != "swift" {
		t.Errorf("the first standing is %s, want swift", standings[0].ID)
	}
	if standings[2].ID != "slow" {
		t.Errorf("the last standing is %s, want slow", standings[2].ID)
	}
}

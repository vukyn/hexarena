package socket

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/draft"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// TestTwoRealClientsDraftAndFightOverALoopbackListener is the test this step
// exists for, and it is the first thing in the repository that proves a drafting
// room is joinable by a real client at all.
//
// ⚠️ **The gap it closes was measured and is worse than it looks.**
// Mirror.Receive had no Drafted arm, so a wire.Drafted errored the client and
// tore the connection down — and every test of the room's draft passed anyway,
// because they drive the room with in-process fakes rather than through
// socket.Client. A fake that mirrors a draft measures internal/draft and
// internal/room and says nothing at all about the transport.
//
// Four claims, each of which could be quietly wrong while a drafting match still
// ran to completion:
//
//  1. **Both clients computed the same draft from the record alone.** Neither was
//     sent a pool, a squad or a turn — only the decisions — so the two Squads()
//     agreeing is the mirror proven end to end over a socket.
//  2. **The battle was opened on those squads**, by value: the unit ids of the
//     roster each client's own engine built are the drafted pair, resolved, home
//     enlisted first — which is the sixty-point fact, since roster order decides
//     a speed tie.
//  3. **Nothing downstream of the draft changed**: the battle then plays out
//     exactly as an undrafted one does, digest against digest on every turn.
//  4. **Nothing was reported and nobody was refused**, which for a draft is
//     sharper than for a battle: every legality refusal in a draft travels as
//     wire.CodeIllegalAction, so a client whose pool or whose sequence had parted
//     company from the room's would show up here as a code rather than as a hang.
func TestTwoRealClientsDraftAndFightOverALoopbackListener(t *testing.T) {
	dependencies := deps(t)
	configuration := draftingConfig(11, room.DefaultAllowance)
	held := listening(t, Timings{})
	code := held.open(t, configuration, dependencies)

	host := held.drafter(t, code, "Host", dependencies)
	if host.Seat() != wire.SeatHost {
		t.Fatalf("the first client took the %q seat, want the host's", host.Seat())
	}
	guest := held.drafter(t, code, "Guest", dependencies)
	if guest.Seat() != wire.SeatGuest {
		t.Fatalf("the second client took the %q seat, want the guest's", guest.Seat())
	}

	// ⚠️ **Both are dialled before either plays, and that is the opposite of the
	// undrafted end-to-end test, which starts the host's loop first because the
	// second join is what produces its wire.Start.** A drafting room produces
	// nothing at all when it fills, so the host's first ban is not triggered by a
	// message — it is sent by Play before its first read — and the room refuses a
	// decision until **both** seats are taken. Measured with the loops started the
	// other way round: the host was refused wire.CodeNotYourTurn **five** times
	// before the guest arrived, and each refusal is what made it try again. → the
	// wrinkle in TestTheMirrorHasTheHostDecidingFirst, and TODO.md, which records
	// it rather than working around it: a client whose chooser waits for a player
	// does not do that, and a player who bans before an opponent arrives has the
	// ban thrown away instead.
	ctx := context.Background()
	hostPlay := play(ctx, host.Client, rating(host.Client))
	guestPlay := play(ctx, guest.Client, rating(guest.Client))

	if err := hostPlay.wait(t, "the host"); err != nil {
		t.Fatalf("the host's drafted match: %v", err)
	}
	if err := guestPlay.wait(t, "the guest"); err != nil {
		t.Fatalf("the guest's drafted match: %v", err)
	}
	done := held.finished(t)
	result, played := done.reading.Result, done.reading.Played
	if !result.Verdict.Over() {
		t.Fatalf("the transport reported a finished match with the verdict %q", result.Verdict)
	}
	if len(played) != 1 {
		t.Fatalf("a drafting room is a bo1 and this one played %d battles", len(played))
	}

	// Both clients were told the room drafts and both built a draft off it —
	// which is the whole of what lets a client mirror one, since nothing else on
	// the wire says a draft is coming.
	for _, client := range []*drafter{host, guest} {
		welcome, seated := client.Mirror().Welcome()
		if !seated || !welcome.Drafts {
			t.Fatalf("%s played a drafted match with drafts=%v seated=%v",
				client.Seat(), welcome.Drafts, seated)
		}
		if !client.Mirror().Draft().Mirrored {
			t.Fatalf("%s built no draft of its own", client.Seat())
		}
	}

	// Claim 1. The decision count is derived arithmetic rather than a magic
	// number, and it is this run's vacuity guard for the draft: a run that
	// replayed a handful of entries did not draft a side.
	hostDraft, guestDraft := host.Mirror().Draft(), guest.Mirror().Draft()
	for _, client := range []*drafter{host, guest} {
		its := client.Mirror().Draft()
		if !its.Done {
			t.Fatalf("%s's own draft is not finished after replaying %d entries",
				client.Seat(), its.Replayed)
		}
		if want := draftDecisions(configuration.Format); its.Replayed != want {
			t.Errorf("%s replayed %d decisions, want %d — %d ban slots, %d picks with a "+
				"loadout each, and one arrangement a side", client.Seat(), its.Replayed, want,
				2*draft.BansPerSide(configuration.Format), 2*draft.PicksPerSide(configuration.Format))
		}
		if stale := client.Mirror().Stale(); stale != 0 {
			t.Errorf("%s refused %d of its own answers as stale in a draft nothing went wrong in",
				client.Seat(), stale)
		}
	}
	if !reflect.DeepEqual(hostDraft.Squads, guestDraft.Squads) {
		t.Fatal("the two clients replayed the same record into different squads, so a draft " +
			"is not a pure function of the decisions taken")
	}
	// The pool is exclusive, so the twelve — six — units are that many different
	// characters, across both sides as well as within one: both spend out of one
	// pool. That is a property of the pool rather than of any refusal.
	characters := map[string]int{}
	for _, side := range hostDraft.Squads {
		for _, unit := range side.Units {
			characters[unit.Character]++
		}
	}
	if units := 2 * draft.PicksPerSide(configuration.Format); len(characters) != units {
		t.Errorf("%d units were drafted out of %d distinct characters: the pool is exclusive, "+
			"so neither side can double up and neither can the pair of them",
			units, len(characters))
	}

	// Claim 2, by value: the roster each client's own engine opened the battle on
	// is the two drafted squads resolved, home first. Read at the moment the
	// battle arrived, because a finished battle's units may include summons.
	homeSeat := homeOf(t, host, guest)
	homeIndex, _ := seatIndex(homeSeat)
	awayIndex := 1 - homeIndex
	home, err := hostDraft.Squads[homeIndex].Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the drafted home squad: %v", err)
	}
	away, err := hostDraft.Squads[awayIndex].Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the drafted away squad: %v", err)
	}
	wanted := make([]string, 0, len(home)+len(away))
	for _, one := range append(home, away...) {
		wanted = append(wanted, one.ID)
	}
	for _, client := range []*drafter{host, guest} {
		if !reflect.DeepEqual(client.opened, wanted) {
			t.Errorf("%s opened its battle on %v, and the drafted pair resolves to %v",
				client.Seat(), client.opened, wanted)
		}
	}

	// Claim 3, which is the undrafted match's own claim unchanged: every turn was
	// checked, by both clients, and there were enough of them to have been a
	// battle. A 3v3 of the shipped cast takes 34 to 55 decisions.
	if host.Mirror().Compared() != guest.Mirror().Compared() {
		t.Errorf("the host checked %d digests and the guest %d; every turn goes to both",
			host.Mirror().Compared(), guest.Mirror().Compared())
	}
	compared := host.Mirror().Compared()
	if compared < 30 {
		t.Errorf("each client checked only %d digests, which is too few for a real battle", compared)
	}
	fought := host.Mirror().Fought()
	if len(fought) != 1 {
		t.Fatalf("the host settled %d battles in a bo1", len(fought))
	}
	// ⚠️ The vacuity guard that makes "it finished" mean something. A capped
	// battle is a hang detector firing rather than an ending: the engine concluded
	// nothing about it, so a run that reached the cap fought nobody to a finish.
	if fought[0].Capped {
		t.Fatalf("the battle hit the %d-turn cap, so this measures a battle that was stopped "+
			"rather than one that ended", configuration.TurnCap)
	}
	if !fought[0].Decided && result.Verdict == room.VerdictWon {
		t.Errorf("the room declared %q the winner of a battle the host's own engine did not decide",
			result.Winner)
	}

	// Claim 4.
	if said := held.failures.everything(); len(said) != 0 {
		t.Errorf("the transport reported %d errors over a whole drafted match: %q", len(said), said)
	}
	for _, client := range []*drafter{host, guest} {
		if refusals := client.Mirror().Refusals(); len(refusals) != 0 {
			t.Errorf("%s was refused %v over a drafted match nothing went wrong in",
				client.Seat(), refusals)
		}
	}
	held.emptied(t)

	t.Logf("a %s draft in %d decisions over a socket; %s drafted %v against %s's %v; "+
		"home %s, %d turns checked by each client, outcome %q, verdict %q, winner %q",
		configuration.Format, hostDraft.Replayed,
		wire.SeatHost, squadIDs(hostDraft.Squads[0]),
		wire.SeatGuest, squadIDs(hostDraft.Squads[1]),
		homeSeat, compared, fought[0].Outcome, result.Verdict, result.Winner)
}

// TestTheMirrorHasTheHostDecidingFirst is the constant that is deliberately not
// on the wire, and it is asserted from **both** seats because a client that
// computed the other one is refused on its very first ban.
//
// ⚠️ **It also records the wrinkle this step does not solve.** A client's draft
// is due its first decision the moment the welcome arrives, and a welcome arrives
// when *this* seat is seated rather than when the room is full — so the host
// believes it is being asked before any opponent exists, and the room refuses a
// decision until both seats are taken. The host is the only seat that can be
// wrong about this, because the guest's own welcome is the room filling up. →
// TODO.md, which names the two-phase handshake that would close it.
func TestTheMirrorHasTheHostDecidingFirst(t *testing.T) {
	for _, seat := range []wire.Seat{wire.SeatHost, wire.SeatGuest} {
		mirror, _ := aMirrorInADraftingRoom(t, seat)
		sight := mirror.Draft()
		if !sight.Mirrored || !sight.Open {
			t.Fatalf("%s's mirror of a drafting room reports mirrored=%v open=%v",
				seat, sight.Mirrored, sight.Open)
		}
		if sight.OnTurn != wire.SeatHost || sight.Step != wire.StepBan {
			t.Errorf("%s's own draft opens waiting on %q to %q, want the host to ban",
				seat, sight.OnTurn, sight.Step)
		}
		prompt, asked := mirror.DraftAsking()
		if asked != (seat == wire.SeatHost) {
			t.Errorf("%s is asked=%v for the opening ban", seat, asked)
		}
		if asked && prompt.Due != (DraftDue{Seat: seat, Step: wire.StepBan}) {
			t.Errorf("%s's opening decision is %+v, want the first ban of the record",
				seat, prompt.Due)
		}
	}
}

// TestTheMirrorsPoolLeavesOutEveryHeldBackCharacter is draft.NewPool being the
// single declaration of what a draft may seat, applied at this end too.
//
// ⚠️ A client that built its pool from the whole cast book would offer a
// held-back character its own screen could ban and the room would refuse with
// wire.CodeIllegalAction — a divergence the join's data digest cannot catch,
// because both peers hold the same cast and only one of them read the flag.
//
// The vacuity guard is that the shipped cast really holds one: with nobody held
// back the two lists are equal and this measures nothing.
func TestTheMirrorsPoolLeavesOutEveryHeldBackCharacter(t *testing.T) {
	mirror, dependencies := aMirrorInADraftingRoom(t, wire.SeatHost)
	whole := dependencies.Characters.All()
	held := 0
	for _, character := range whole {
		if character.Hidden {
			held++
		}
	}
	if held == 0 {
		t.Fatal("no shipped character is held back, so a pool built from the whole cast would " +
			"be the same list and this test measures nothing")
	}
	candidates := mirror.Draft().Candidates
	if want := draft.NewPool(whole).Len(); len(candidates) != want {
		t.Errorf("the mirror's opening ban chooses from %d characters, and the draftable pool "+
			"holds %d of the %d in the book", len(candidates), want, len(whole))
	}
	for _, character := range candidates {
		if character.Hidden {
			t.Errorf("%q is held back and the mirror offers it to a ban", character.ID)
		}
	}
	t.Logf("%d of %d shipped characters are draftable, %d held back",
		len(candidates), len(whole), held)
}

// TestADraftedIsAppliedInTheOrderItCarries is the order being load-bearing rather
// than incidental.
//
// ⚠️ **It needs a batch of more than one entry, and a room sends one only in the
// arrange phase** — where both entries are legal in either order, so nothing
// there can see a reversal. What can see it is a batch whose entries alternate
// seats, which is what a spectator joining at cursor nought is handed and what
// this builds: applied forwards it is two bans, and applied backwards the guest
// is banning out of turn.
func TestADraftedIsAppliedInTheOrderItCarries(t *testing.T) {
	mirror, dependencies := aMirrorInADraftingRoom(t, wire.SeatHost)
	pool := draft.NewPool(dependencies.Characters.All()).All()
	if len(pool) < 2 {
		t.Fatalf("the draftable pool holds %d characters and this needs two", len(pool))
	}
	forwards := []wire.DraftEntry{
		{Seat: wire.SeatHost, Step: wire.StepBan, Character: pool[0].ID},
		{Seat: wire.SeatGuest, Step: wire.StepBan, Character: pool[1].ID},
	}
	if err := mirror.Receive(wire.Drafted{Decisions: forwards}); err != nil {
		t.Fatalf("a mirror replays two bans in the order they were recorded: %v", err)
	}
	sight := mirror.Draft()
	if sight.Replayed != len(forwards) {
		t.Errorf("the mirror replayed %d of %d entries", sight.Replayed, len(forwards))
	}
	if sight.OnTurn != wire.SeatHost || sight.Step != wire.StepBan {
		t.Errorf("after both sides' first ban the draft waits on %q to %q, want the host's "+
			"second ban", sight.OnTurn, sight.Step)
	}

	backwards, _ := aMirrorInADraftingRoom(t, wire.SeatHost)
	reversed := []wire.DraftEntry{forwards[1], forwards[0]}
	err := backwards.Receive(wire.Drafted{Decisions: reversed})
	if err == nil {
		t.Fatal("a mirror took a record backwards and reported nothing, so it is computing a " +
			"draft that is no longer the room's")
	}
	// Nothing was applied, so the failure is on the first entry rather than
	// halfway through a batch.
	if replayed := backwards.Draft().Replayed; replayed != 0 {
		t.Errorf("a reversed batch applied %d entries before it was refused", replayed)
	}
	t.Logf("a reversed record is refused: %v", err)
}

// TestADraftedNothingCanReplayIsRefused is the two ways a wire.Drafted is not a
// record this client can take in, and neither is a hypothetical.
//
// A batch with no draft open is a client that was never told the room drafts,
// which is what a room sending a record to the wrong seat would look like. An
// **empty** batch is a room saying nothing happened, and it is a real path: the
// first of the two arrangements records nothing, so exactly one decision of a
// draft is answered with no message at all — and a room that sent one anyway
// would be a room whose guard had gone.
func TestADraftedNothingCanReplayIsRefused(t *testing.T) {
	dependencies := deps(t)
	plain := NewMirror(wire.SeatHost, dependencies.Books, dependencies.Characters)
	if err := plain.Receive(wire.Welcome{
		Format: wire.Format3v3, Battles: 1, Allowance: room.DefaultAllowance,
		TurnCap: room.DefaultTurnCap, Seat: wire.SeatHost,
	}); err != nil {
		t.Fatalf("welcome a mirror into a room that does not draft: %v", err)
	}
	if err := plain.Receive(wire.Drafted{Decisions: []wire.DraftEntry{
		{Seat: wire.SeatHost, Step: wire.StepBan, Character: "pokemon.bulbasaur"},
	}}); err == nil {
		t.Error("a mirror in a room that does not draft took a draft record in")
	}

	drafting, _ := aMirrorInADraftingRoom(t, wire.SeatHost)
	if err := drafting.Receive(wire.Drafted{}); err == nil {
		t.Error("a mirror took in a wire.Drafted carrying no decisions, which is a room saying " +
			"nothing happened")
	}
}

// TestASightsDraftIsASnapshotAndNotTheDraft is the rule DraftSight exists under,
// measured as the thing a caller can observe: a reading kept past the callback
// does not move when the draft does.
//
// ⚠️ **The type-level mutation does not compile**, which is why this measures the
// aliasing instead. Putting a *draft.Draft on Sight in place of this makes every
// field read below a method value, so the mutation the plan named cannot be run —
// and that is a stronger guarantee than a test. What a test can still see is the
// value-level half: a snapshot that shared the draft's own slices would show a
// banned character leaving the candidate list it was handed.
func TestASightsDraftIsASnapshotAndNotTheDraft(t *testing.T) {
	mirror, dependencies := aMirrorInADraftingRoom(t, wire.SeatHost)
	pool := draft.NewPool(dependencies.Characters.All()).All()
	var kept DraftSight
	mirror.Read(func(sight Sight) { kept = sight.Draft })
	if len(kept.Candidates) != len(pool) || kept.Replayed != 0 {
		t.Fatalf("the reading holds %d candidates and %d replayed decisions against a pool of %d",
			len(kept.Candidates), kept.Replayed, len(pool))
	}

	banned := pool[0].ID
	if err := mirror.Receive(wire.Drafted{Decisions: []wire.DraftEntry{
		{Seat: wire.SeatHost, Step: wire.StepBan, Character: banned},
	}}); err != nil {
		t.Fatalf("the mirror replays a ban: %v", err)
	}
	// The draft moved, or the comparison below is between two readings of one
	// state and says nothing.
	if now := mirror.Draft(); now.Replayed != 1 || holds(now.Candidates, banned) {
		t.Fatalf("after a ban the live reading holds %d replayed decisions and %v for %q",
			now.Replayed, holds(now.Candidates, banned), banned)
	}
	if kept.Replayed != 0 {
		t.Errorf("a reading kept past the callback now counts %d replayed decisions", kept.Replayed)
	}
	if !holds(kept.Candidates, banned) {
		t.Errorf("a reading kept past the callback lost %q from its candidates, so it was a "+
			"view of the draft rather than a snapshot of it", banned)
	}
}

// holds reports whether a candidate list names a character.
func holds(candidates []cast.Character, id string) bool {
	return slices.ContainsFunc(candidates, func(one cast.Character) bool { return one.ID == id })
}

// TestDecideDraftRefusesAnAnswerForAnotherDecision is the hazard the step alone
// cannot see, built at the one place it bites.
//
// ⚠️ **The two decisions it tells apart are both the host's, and both are bans.**
// A seat's ban slots have the other seat's between them, so an answer given for
// the first ban and delivered while the second is open carries the *same* step and
// the same absent character — everything wire.Decide has. What moved is the
// record, which is why DraftDue carries a count of it.
func TestDecideDraftRefusesAnAnswerForAnotherDecision(t *testing.T) {
	mirror, dependencies := aMirrorInADraftingRoom(t, wire.SeatHost)
	pool := draft.NewPool(dependencies.Characters.All()).All()

	// The answer as it would have been given for the opening ban.
	opening, asked := mirror.DraftAsking()
	if !asked {
		t.Fatal("the host is not asked for the opening ban")
	}
	stale := DraftAnswer{
		For:      opening.Due,
		Decision: wire.DraftDecision{Step: wire.StepBan, Character: pool[2].ID},
	}

	// Two bans go by, so the host's *second* ban is what is open now.
	if err := mirror.Receive(wire.Drafted{Decisions: []wire.DraftEntry{
		{Seat: wire.SeatHost, Step: wire.StepBan, Character: pool[0].ID},
		{Seat: wire.SeatGuest, Step: wire.StepBan, Character: pool[1].ID},
	}}); err != nil {
		t.Fatalf("the mirror replays two bans: %v", err)
	}
	live, asked := mirror.DraftAsking()
	if !asked || live.Due.Step != wire.StepBan {
		t.Fatalf("the host is asked=%v for a %q, want its second ban", asked, live.Due.Step)
	}
	// The premise: the two decisions are indistinguishable by everything the wire
	// carries, so a check on the step alone would let the stale answer through.
	if stale.For.Step != live.Due.Step || stale.For.Character != live.Due.Character {
		t.Fatalf("the two bans differ in step or subject (%+v against %+v), so this measures "+
			"the easy case rather than the one the count is for", stale.For, live.Due)
	}
	if stale.For == live.Due {
		t.Fatal("the two bans are the same decision, so nothing here is stale")
	}

	if body, decided := mirror.DecideDraft(func(DraftPrompt) (DraftAnswer, bool) {
		return stale, true
	}); decided {
		t.Errorf("a stale answer was sent as %v; an answer for a decision already spent must "+
			"be refused rather than applied to the next one", body)
	}
	if mirror.Stale() != 1 {
		t.Errorf("the mirror counted %d stale answers, want one — without the count the "+
			"refusal leaves no trace at all", mirror.Stale())
	}
	// And the honest answer for the open decision goes, or the check above is
	// refusing everything.
	body, decided := mirror.DecideDraft(drafting(t, dependencies.Characters))
	if !decided {
		t.Fatal("the open decision's own answer was refused as well, so the check refuses " +
			"everything and proves nothing")
	}
	if _, sends := body.(wire.Decide); !sends {
		t.Errorf("a draft answer travels as a %T, want a wire.Decide — which is the shape with "+
			"nowhere to put a seat", body)
	}
	if mirror.Stale() != 1 {
		t.Errorf("the honest answer was counted stale as well (%d)", mirror.Stale())
	}
}

// TestDecideDraftDoesNotHoldTheLockAcrossTheChooser is Decide's ordering held for
// the draft, and it is the same three-way for the same measured reason.
//
// ⚠️ **The obvious test does not catch it.** sync.RWMutex admits several readers,
// so a chooser that only reads sits happily beside a renderer that only reads and
// holding the lock across choose passes. What makes the hold fatal is a **writer
// arriving while the chooser waits**: Go queues a waiting writer ahead of new
// readers, so the next Receive blocks behind the held read lock and the renderer
// then blocks behind the writer — and the renderer is what the player is waiting
// to see before they can choose a ban. → Mirror.Decide, which carries the
// measurement, and TestDecideDoesNotHoldTheLockAcrossTheChooser, which is this
// test for a battle.
func TestDecideDraftDoesNotHoldTheLockAcrossTheChooser(t *testing.T) {
	mirror, _ := aMirrorInADraftingRoom(t, wire.SeatHost)
	if _, asked := mirror.DraftAsking(); !asked {
		t.Fatal("the host is not asked for the opening ban, so there is no chooser to block")
	}

	inChooser := make(chan struct{})
	release := make(chan struct{})
	decided := make(chan struct{})
	go func() {
		defer close(decided)
		mirror.DecideDraft(func(DraftPrompt) (DraftAnswer, bool) {
			close(inChooser)
			<-release
			return DraftAnswer{}, false
		})
	}()
	<-inChooser

	// A writer, which is what turns a held read lock into a queue.
	written := make(chan struct{})
	go func() {
		defer close(written)
		if err := mirror.Receive(wire.Refused{Code: wire.CodeNotYourTurn}); err != nil {
			t.Errorf("the mirror refused a refusal: %v", err)
		}
	}()

	// And the renderer, which is what the player is waiting to see.
	drawn := make(chan struct{})
	go func() {
		defer close(drawn)
		mirror.Read(func(Sight) {})
	}()

	const bounded = 5 * time.Second
	select {
	case <-drawn:
	case <-time.After(bounded):
		close(release)
		t.Fatalf("a renderer could not read the mirror within %s while a draft chooser was "+
			"waiting for the player, which is the client deadlocked against itself: the read "+
			"lock is being held across the chooser", bounded)
	}
	select {
	case <-written:
	case <-time.After(bounded):
		close(release)
		t.Fatalf("a message could not be taken in within %s while a draft chooser was waiting",
			bounded)
	}
	close(release)
	select {
	case <-decided:
	case <-time.After(bounded):
		t.Fatalf("the draft chooser never returned within %s", bounded)
	}
	if refusals := mirror.Refusals(); len(refusals) != 1 {
		t.Errorf("the mirror recorded %d refusals, want the one the writer sent — without it "+
			"no writer was ever queued and this measures nothing", len(refusals))
	}
}

// TestAClientWithNoDraftChooserFailsRatherThanWaiting is the loud ending
// ClientOptions.Draft promises.
//
// ⚠️ **A nil chooser that quietly answered nothing would be a client sitting for
// ever on a decision**, which is the hang the whole timeout mechanism exists to
// prevent — and it would sit there with the room's allowance running, so the
// ending a player saw would be a cancelled draft blaming nobody. The failure is
// at the point of being **asked** rather than at the point of joining, so that a
// caller with nothing to decide can still dial a room that drafts.
func TestAClientWithNoDraftChooserFailsRatherThanWaiting(t *testing.T) {
	dependencies := deps(t)
	held := listening(t, Timings{})
	code := held.open(t, draftingConfig(11, room.DefaultAllowance), dependencies)

	client, err := Dial(context.Background(), code, helloWithNoSquad(t, "Host"),
		dependencies.Books, ClientOptions{
			Timings:    held.timings,
			Characters: dependencies.Characters,
		})
	if err != nil {
		t.Fatalf("dial a drafting room with no draft chooser: %v", err)
	}
	t.Cleanup(client.Close)
	// The dial itself is deliberately fine — → the ⚠️ above.
	if _, asked := client.Mirror().DraftAsking(); !asked {
		t.Fatal("the host was welcomed into a drafting room and is asked nothing")
	}
	err = play(context.Background(), client, rating(client)).wait(t, "a client with no chooser")
	if err == nil {
		t.Fatal("a client with no draft chooser played a drafting room without complaint, so " +
			"it would have sat on the decision for as long as the room let it")
	}
	if !strings.Contains(err.Error(), "no draft chooser") {
		t.Errorf("the failure is %q, which does not say what is missing", err)
	}
	t.Logf("refused loudly: %v", err)
}

// TestAMirrorWithNoCastCannotMirrorADraft is the other half of the same rule: the
// pool is built out of a cast book, and a client that has none cannot compute the
// draft the room is running.
//
// It fails at the **welcome** rather than at the first decision, which is the one
// place a client learns a draft is coming at all — so the refusal reaches Dial and
// a client that could never take part never joins.
func TestAMirrorWithNoCastCannotMirrorADraft(t *testing.T) {
	dependencies := deps(t)
	mirror := NewMirror(wire.SeatHost, dependencies.Books, nil)
	err := mirror.Receive(wire.Welcome{
		Format: wire.Format3v3, Battles: 1, Allowance: room.DefaultAllowance,
		TurnCap: room.DefaultTurnCap, Drafts: true, Seat: wire.SeatHost,
	})
	if err == nil {
		t.Fatal("a mirror with no cast book was welcomed into a drafting room without complaint")
	}
	if mirror.Draft().Mirrored {
		t.Error("it built a draft anyway")
	}
	// And the same mirror in a room that does not draft is welcomed perfectly:
	// the cast is wanted for the draft and for nothing else.
	if err := mirror.Receive(wire.Welcome{
		Format: wire.Format3v3, Battles: 1, Allowance: room.DefaultAllowance,
		TurnCap: room.DefaultTurnCap, Seat: wire.SeatHost,
	}); err != nil {
		t.Errorf("a mirror with no cast book cannot join a room that does not draft either: %v", err)
	}
	t.Logf("refused at the welcome: %v", err)
}

// aMirrorInADraftingRoom is a mirror welcomed into a room that drafts, built out
// of a message rather than over a socket: what the tests above are about is the
// mirror, and a listener would put a network's timing into them.
func aMirrorInADraftingRoom(t *testing.T, seat wire.Seat) (*Mirror, room.Deps) {
	t.Helper()
	dependencies := deps(t)
	mirror := NewMirror(seat, dependencies.Books, dependencies.Characters)
	if err := mirror.Receive(wire.Welcome{
		Format:    wire.Format3v3,
		Battles:   1,
		Allowance: room.DefaultAllowance,
		TurnCap:   room.DefaultTurnCap,
		Drafts:    true,
		Seat:      seat,
	}); err != nil {
		t.Fatalf("welcome %s into a drafting room: %v", seat, err)
	}
	return mirror, dependencies
}

// homeOf is the seat the battle was fought home from, read off the side each
// client was given on its own wire.Start rather than off Config.HomeFor — a bo1's
// home is picked by the low bit of the derived seed, and asking the rule what the
// rule says would agree with any rule at all.
func homeOf(t *testing.T, clients ...*drafter) wire.Seat {
	t.Helper()
	for _, client := range clients {
		fought := client.Mirror().Fought()
		if len(fought) > 0 && fought[0].Side == hex.SideAlly {
			return client.Seat()
		}
	}
	t.Fatal("neither client played the ally side, so no seat was enlisted first")
	return ""
}

// TestNoFieldOfADraftSightCanAliasTheDraft is the mechanical half of the rule
// DraftSight's own comment states, and it exists because the obvious mutation is
// not the reachable one.
//
// ⚠️ **Replacing the snapshot with a *draft.Draft does not compile, but adding
// one BESIDE it does — and that passed this whole package.** Measured while this
// step was reviewed: a `Live *draft.Draft` field filled in by draftSight builds
// clean and reddens nothing, in socket or in the client. So "it would not
// compile" is a guarantee about one shape of the mistake and not about the
// mistake, which is what this walk is for.
//
// It bans four kinds of field rather than only a pointer, because each of them
// can reach the draft the Play goroutine is stepping *after* the lock has been
// let go: a pointer and a map alias it directly, a channel hands a reader a way
// to be handed one, and a func can close over it. Everything a screen needs is a
// value, a slice internal/draft already clones, or an array of those — which is
// what the type holds today, with no exception to write down.
func TestNoFieldOfADraftSightCanAliasTheDraft(t *testing.T) {
	const subject = "DraftSight"
	set := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read this package's own directory: %v", err)
	}
	var fields, scanned int
	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(set, entry.Name(), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		scanned++
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok || spec.Name.Name != subject {
				return true
			}
			shape, ok := spec.Type.(*ast.StructType)
			if !ok {
				return true
			}
			found = true
			for _, field := range shape.Fields.List {
				fields++
				because := ""
				switch field.Type.(type) {
				case *ast.StarExpr:
					because = "a pointer aliases whatever it points at, so a reading kept past " +
						"the lock would move when the Play goroutine steps the draft"
				case *ast.MapType:
					because = "a map is a reference, so this would alias the draft's own state — " +
						"and a range over one reaches an output in whatever order Go feels like"
				case *ast.ChanType:
					because = "a channel is a way to be handed something later, which is a pointer " +
						"with extra steps"
				case *ast.FuncType:
					because = "a func can close over the draft, so calling it after the lock is " +
						"let go reads state nobody is holding still"
				}
				if because == "" {
					continue
				}
				names := "an embedded field"
				if len(field.Names) > 0 {
					names = field.Names[0].Name
				}
				t.Errorf("%s.%s is not a value: %s. Everything a screen needs off a draft is a "+
					"value, a slice internal/draft already clones, or an array of those — see the "+
					"type's own comment for why the import graph settles this before the lock does",
					subject, names, because)
			}
			return false
		})
	}
	if scanned == 0 {
		t.Fatal("no non-test source file was scanned, so this walk measured nothing")
	}
	if !found {
		t.Fatalf("no declaration of %s was found in %d source files, so this walk measured "+
			"nothing — it was renamed or moved and this test has to follow it", subject, scanned)
	}
	if fields == 0 {
		t.Fatalf("%s was found holding no fields at all, so this walk measured nothing", subject)
	}
	t.Logf("scanned %d source files; every one of %s's %d fields is a value", scanned, subject, fields)
}

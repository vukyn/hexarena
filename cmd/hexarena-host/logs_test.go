package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/room"
	"github.com/vukyn/hexarena/internal/wire"
)

// The -logs flag: where a finished battle goes, and the two states that are not
// "a file appeared".

// TestWithNoLogsFlagNothingIsWritten is the default, and it is asserted against
// an existing empty directory rather than a missing one — a writer that ignored
// the flag and used the working directory would pass a test that only checked
// the named place stayed absent.
func TestWithNoLogsFlagNothingIsWritten(t *testing.T) {
	where := t.TempDir()
	held := &hosted{code: aCode(t)}
	if err := held.write(oneBattleReading(t), &strings.Builder{}); err != nil {
		t.Fatalf("write with no directory named: %v", err)
	}
	left, err := os.ReadDir(where)
	if err != nil {
		t.Fatalf("read %s: %v", where, err)
	}
	if len(left) != 0 {
		t.Errorf("%d file(s) appeared with no -logs given: %v", len(left), left)
	}
}

// TestEachFinishedBattleIsWrittenAndReplays is the flag doing its job, and the
// assertion is the one that matters: the file **re-runs**. A log that lands on
// disk and cannot be replayed is a file rather than a record.
func TestEachFinishedBattleIsWrittenAndReplays(t *testing.T) {
	where := filepath.Join(t.TempDir(), "not", "made", "yet")
	code := aCode(t)
	held := &hosted{code: code, logs: where}
	var said strings.Builder
	reading := oneBattleReading(t)
	if err := held.write(reading, &said); err != nil {
		t.Fatalf("write into %s: %v", where, err)
	}

	// The directory is created rather than required, because -logs names where
	// files go and a host who typed a path should not have to make it first.
	written, err := os.ReadDir(where)
	if err != nil {
		t.Fatalf("read %s: %v", where, err)
	}
	if len(written) != len(reading.Played) {
		t.Fatalf("%d battle(s) played and %d file(s) written", len(reading.Played), len(written))
	}
	// The name tells one match's files from another's: the seats are "host" and
	// "guest" in every match this binary will ever serve, so the code and the
	// battle number are the only things that distinguish them.
	name := written[0].Name()
	for _, part := range []string{string(code), "battle1", "seed"} {
		if !strings.Contains(name, part) {
			t.Errorf("the file is called %q, which does not carry %q", name, part)
		}
	}
	if said := said.String(); !strings.Contains(said, "--replay") {
		t.Errorf("the host wrote a log and did not say how to replay one:\n%s", said)
	}

	raw, err := os.ReadFile(filepath.Join(where, name))
	if err != nil {
		t.Fatalf("read the log back: %v", err)
	}
	log, err := battle.ParseLog(raw)
	if err != nil {
		t.Fatalf("parse the log this binary just wrote: %v", err)
	}
	if !log.Replayable() {
		t.Fatal("the log carries no placement, so `--replay --verify` could not re-run it")
	}
	if log.Seed != reading.Played[0].Seed {
		t.Errorf("the log says seed %d and the battle was fought from %d",
			log.Seed, reading.Played[0].Seed)
	}
	if len(log.Events) != len(reading.Played[0].Log.Events) {
		t.Errorf("the room recorded %d events and %d came back off disk",
			len(reading.Played[0].Log.Events), len(log.Events))
	}
}

// TestADirectoryThatWillNotTakeAFileIsReportedAndNotFatal holds the shape of the
// failure: a host is told, and the match result it has already been given stands.
//
// ⚠️ The path names an existing **file**, so MkdirAll refuses it — which is a
// real thing to mistype and the one filesystem refusal that does not need a
// permission trick to provoke on every platform.
func TestADirectoryThatWillNotTakeAFileIsReportedAndNotFatal(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "in-the-way")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("make the obstruction: %v", err)
	}
	held := &hosted{code: aCode(t), logs: blocked}
	err := held.write(oneBattleReading(t), &strings.Builder{})
	if err == nil {
		t.Fatal("writing into a path that is a file came back with no error")
	}
	if !strings.Contains(err.Error(), blocked) {
		t.Errorf("the refusal does not name the path a host typed: %v", err)
	}
}

// aCode is a room code to name files after. Any valid one does: what is being
// measured is that the name carries it.
func aCode(t *testing.T) wire.RoomCode {
	t.Helper()
	held := hosting(t, aRoom(), dialableAt, newPaper().on, newPaper().on)
	return held.code
}

// oneBattleReading is a room.Reading holding one battle that really was fought,
// so the log in it is a log rather than a struct.
func oneBattleReading(t *testing.T) room.Reading {
	t.Helper()
	dependencies, err := dependenciesOf("host-test")
	if err != nil {
		t.Fatalf("load the data: %v", err)
	}
	squad := squadOf(t, dependencies.Characters, "logger")
	roster, err := squad.Take(hex.SideAlly, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the squad: %v", err)
	}
	facing, err := squad.Take(hex.SideEnemy, dependencies.Characters)
	if err != nil {
		t.Fatalf("field the other squad: %v", err)
	}
	roster = append(roster, facing...)
	const seed = 4242
	fight, err := battle.New(dependencies.Books, seed, roster)
	if err != nil {
		t.Fatalf("open the battle: %v", err)
	}
	fight.Begin()
	if _, err := fight.RunToEnd(room.DefaultTurnCap); err != nil {
		t.Fatalf("play the battle out: %v", err)
	}
	events := fight.Drain()
	if len(events) == 0 {
		t.Fatal("the fixture battle produced no events, so the log would measure nothing")
	}
	return room.Reading{
		Finished: true,
		Result:   room.Result{Verdict: room.VerdictDrawn, Battles: 1},
		Played: []room.BattleResult{{
			Battle: 1, Home: wire.SeatHost, Seed: seed,
			Log: battle.Log{Seed: seed, Roster: roster, Events: events},
		}},
	}
}

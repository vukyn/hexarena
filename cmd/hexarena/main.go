// Command hexarena plays a battle in the terminal.
//
// It is a renderer and a prompt, nothing more. Every rule lives in the engine and
// every line it prints is built from the battle's event log, so a graphical
// client can be written against the same log without either one becoming a second
// copy of the rules.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/i18n"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

type config struct {
	seed    uint64
	auto    bool
	side    hex.Side
	limit   int
	preview int
	logPath string
	replay  string
	verify  bool
}

func main() {
	var cfg config
	sideName := flag.String("side", "ally", "which side you play, ally or enemy")
	flag.Uint64Var(&cfg.seed, "seed", 1, "seed the battle's rolls come from; the same seed replays exactly")
	flag.BoolVar(&cfg.auto, "auto", false, "let both sides pick their own actions and print the whole battle")
	flag.IntVar(&cfg.limit, "turns", 4000, "give up after this many turns rather than hang on a stalemate")
	flag.IntVar(&cfg.preview, "preview", 6, "how many upcoming turns to show")
	flag.StringVar(&cfg.logPath, "log", "", "write the battle to this file as json when it ends")
	flag.StringVar(&cfg.replay, "replay", "", "read a saved battle from this file and print it instead of playing")
	flag.BoolVar(&cfg.verify, "verify", false, "with -replay, re-run the battle from its seed and check the log matches")
	flag.Parse()

	side, err := parseSide(*sideName)
	if err != nil {
		fail(err)
	}
	cfg.side = side

	if cfg.replay != "" {
		if err := replay(cfg); err != nil {
			fail(err)
		}
		return
	}
	if err := play(cfg); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "hexarena: %v\n", err)
	os.Exit(1)
}

func parseSide(name string) (hex.Side, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ally":
		return hex.SideAlly, nil
	case "enemy":
		return hex.SideEnemy, nil
	default:
		return 0, fmt.Errorf("side %q is neither ally nor enemy", name)
	}
}

// session holds a battle and the decisions that produced it. Keeping the
// decisions is what makes an undo possible without cloning any state: because
// the engine is deterministic, dropping the last decision and replaying the rest
// rebuilds exactly the position the battle was in.
type session struct {
	cfg   config
	fight *battle.Battle
	// roster is the placement this battle was fought with, kept because the log
	// records it and because a rewind rebuilds from it. Reading the data again
	// would work today and would stop working the moment a placement is chosen
	// at a prompt rather than read out of a file.
	roster []battle.Roster
	tags   map[string]string
	names  map[string]string
	script battle.Script
	// events is everything that has happened, kept so the log and the summary
	// can be written from the whole battle rather than the last turn.
	events []battle.Event
	// pending is a turn a rebuild already opened, so the loop uses it instead of
	// advancing a turn that has begun.
	pending *battle.Prompt
}

func newSession(cfg config) (*session, error) {
	books, err := seed.Books()
	if err != nil {
		return nil, err
	}
	roster, err := seed.Roster()
	if err != nil {
		return nil, err
	}
	fight, err := battle.New(books, cfg.seed, roster)
	if err != nil {
		return nil, err
	}
	fight.Begin()
	current := &session{
		cfg:    cfg,
		fight:  fight,
		roster: roster,
		tags:   tui.Tags(fight.Units()),
		names:  tui.Names(fight.Units()),
	}
	current.collect()
	return current, nil
}

// collect drains the battle and both prints and keeps what it recorded.
func (s *session) collect() {
	drained := s.fight.Drain()
	if len(drained) == 0 {
		return
	}
	s.events = append(s.events, drained...)
	fmt.Println(tui.Log(drained, s.tags))
}

// rewind rebuilds the battle from its seed and replays a shortened script. It is
// the whole of undo: no snapshot, no deep copy, just the seed and the decisions.
func (s *session) rewind(script battle.Script) error {
	books, err := seed.Books()
	if err != nil {
		return err
	}
	// The roster this battle was fought with, not the one on disk. An undo that
	// re-read the data would quietly field a different squad the moment the data
	// changed underneath a running game.
	fight, err := battle.New(books, s.cfg.seed, s.roster)
	if err != nil {
		return err
	}
	fight.Begin()
	replayed, pending, err := fight.Replay(script, s.cfg.limit, nil)
	if err != nil {
		return err
	}
	s.fight = fight
	s.script = replayed
	s.pending = pending
	s.events = fight.Drain()
	return nil
}

// undo drops the player's last decision, along with anything that followed it,
// and rebuilds. It reports whether there was anything to take back.
func (s *session) undo() (bool, error) {
	cut := -1
	for i, decision := range slices.Backward(s.script) {
		unit, ok := s.fight.Unit(decision.Unit)
		if ok && unit.Side == s.cfg.side {
			cut = i
			break
		}
	}
	if cut < 0 {
		return false, nil
	}
	shortened := make(battle.Script, cut)
	copy(shortened, s.script[:cut])
	if err := s.rewind(shortened); err != nil {
		return false, err
	}
	return true, nil
}

func (s *session) record(decision battle.Decision) { s.script = append(s.script, decision) }

// suggest asks the engine what it would do, and turns the answer into a decision
// so it lands in the script alongside the player's own.
func (s *session) suggest(prompt *battle.Prompt) battle.Decision {
	choice, ok := s.fight.Suggest(prompt)
	decision := battle.Decision{
		Unit: prompt.Unit, Turn: prompt.Turn,
		Skill: choice.Skill, Passed: !ok,
	}
	if ok {
		decision.Aim = hex.At(choice.Aim)
	} else {
		decision.Reason = battle.NoActionReason
	}
	return decision
}

func (s *session) apply(decision battle.Decision) error {
	if decision.Passed {
		if err := s.fight.Pass(decision.PassReason()); err != nil {
			return err
		}
	} else {
		aim, aimed := decision.Aim.Offset()
		if !aimed {
			return fmt.Errorf("%q was taken with %q but aims nowhere", decision.Unit, decision.Skill)
		}
		if err := s.fight.Act(decision.Skill, aim); err != nil {
			return err
		}
	}
	s.record(decision)
	return nil
}

func play(cfg config) error {
	current, err := newSession(cfg)
	if err != nil {
		return err
	}
	input := bufio.NewScanner(os.Stdin)
	fmt.Printf("hexarena, seed %d\n\n", cfg.seed)
	show(current)

	for turns := 0; !current.fight.Finished() && turns < cfg.limit; turns++ {
		prompt := current.pending
		current.pending = nil
		if prompt == nil {
			opened, err := current.fight.Advance()
			if err != nil {
				return err
			}
			prompt = opened
		}
		current.collect()
		if prompt.Skipped {
			continue
		}
		unit, ok := current.fight.Unit(prompt.Unit)
		if !ok {
			return fmt.Errorf("the battle offered a turn to %q, which is not fighting", prompt.Unit)
		}

		if cfg.auto || unit.Side != cfg.side {
			if err := current.apply(current.suggest(prompt)); err != nil {
				return err
			}
			current.collect()
			continue
		}

		fmt.Println()
		show(current)
		fmt.Printf("\n%s %s, turn %d at %s\n", current.tags[unit.ID], unit.Name, prompt.Turn, unit.Cell)
		outcome, err := choose(current, prompt, input)
		if err != nil {
			return err
		}
		switch outcome {
		case quit:
			fmt.Println("\nleaving the battle unfinished")
			return finish(current)
		case rewound:
			// The rebuild already opened the turn again, so the loop picks it up
			// from pending rather than advancing past it.
			turns--
			continue
		}
		current.collect()
	}

	if !current.fight.Finished() {
		fmt.Printf("\nstopped after %d turns without a decision\n", cfg.limit)
	} else {
		switch outcome := current.fight.Outcome(); outcome {
		case battle.Victory:
			winner, _ := current.fight.Winner()
			fmt.Printf("\nthe %s side wins\n", winner)
		case battle.Stalemate:
			fmt.Println("\nthe battle ends in a draw: both sides are still standing and neither can reach the other")
		default:
			fmt.Println("\nthe battle ends with nobody standing")
		}
	}
	return finish(current)
}

// finish prints the closing board and summary, and writes the log if asked.
func finish(current *session) error {
	fmt.Println()
	fmt.Println(tui.Roster(current.fight, current.tags))
	fmt.Println()
	fmt.Println("== summary ==")
	fmt.Println(tui.Summary(current.events, current.tags, current.names))
	if current.cfg.logPath == "" {
		return nil
	}
	raw, err := battle.MarshalLog(battle.Log{
		Seed: current.cfg.seed, Roster: current.roster,
		Choices: current.script, Events: current.events,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(current.cfg.logPath, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", current.cfg.logPath, err)
	}
	fmt.Printf("\nwrote %s (%d events, %d choices)\n",
		current.cfg.logPath, len(current.events), len(current.script))
	return nil
}

func show(current *session) {
	fmt.Println(tui.Board(current.fight, current.tags))
	fmt.Println()
	fmt.Println(tui.Roster(current.fight, current.tags))
}

type outcome uint8

const (
	acted outcome = iota
	rewound
	quit
)

// choose runs the two questions a turn needs: which skill, then where. It keeps
// asking rather than giving up on a bad answer, because a mistyped number should
// not cost a turn.
// describe answers a question about the turn in front of the player: a number is
// one of the offered skills, a star is the whole status reference, and anything
// else is a unit's tag or the name of a status.
//
// It refuses rather than guessing when it recognises none of them. A question
// that silently printed the wrong thing would be worse than one that says it did
// not understand, because the player would read a description and act on it.
//
// The tags are tried before the statuses and they cannot collide: a tag is a
// side letter and a number, and no status is named "a1".
func describe(current *session, prompt *battle.Prompt, question string) {
	if question == "" {
		fmt.Println("  ask about a skill by its number, a unit by its tag, or a status by its name")
		return
	}
	books := current.fight.Books()
	if question == "*" {
		fmt.Println()
		fmt.Println(tui.DetailStatuses(i18n.Vi, books.Statuses.Grouped()))
		return
	}
	if index, err := strconv.Atoi(question); err == nil {
		if index < 1 || index > len(prompt.Options) {
			fmt.Printf("  there is no skill %d on this turn\n", index)
			return
		}
		declared, lookupErr := books.Skills.Lookup(prompt.Options[index-1].Skill)
		if lookupErr != nil {
			fmt.Printf("  %v\n", lookupErr)
			return
		}
		fmt.Println()
		fmt.Println(tui.Detail(i18n.Vi, declared, books.Patterns))
		return
	}
	for id, tag := range current.tags {
		if !strings.EqualFold(tag, question) {
			continue
		}
		unit, ok := current.fight.Unit(id)
		if !ok {
			continue
		}
		held := make([]passive.Passive, 0, len(unit.Passives))
		for _, name := range unit.Passives {
			// A trait the book has lost is skipped rather than reported: the unit
			// is carrying it either way, and this is a reading aid, not a place
			// to discover that the data is broken.
			if found, lookupErr := books.Passives.Lookup(name); lookupErr == nil {
				held = append(held, found)
			}
		}
		fmt.Println()
		fmt.Println(tui.DetailPassives(i18n.Vi, fmt.Sprintf("%s %s", tag, unit.Name), held))
		return
	}
	// By id or by the name it is printed under, because the player reading a
	// Vietnamese unit table has only ever seen the second one.
	for _, kind := range books.Statuses.Kinds() {
		if !strings.EqualFold(kind.ID, question) &&
			!strings.EqualFold(i18n.Vi.Gloss(kind.ID), question) {
			continue
		}
		fmt.Println()
		fmt.Println(tui.DetailStatus(i18n.Vi, kind))
		return
	}
	fmt.Printf("  %q is not a skill number, a unit tag or a status; ?* lists the statuses\n", question)
}

func choose(current *session, prompt *battle.Prompt, input *bufio.Scanner) (outcome, error) {
	for {
		fmt.Println(tui.Order(current.fight.Queue(), current.tags, current.cfg.preview))
		fmt.Println(tui.Menu(current.fight, prompt, current.tags))
		fmt.Println("  ?N) what a skill does    ?TAG) a unit's traits    ?NAME or ?*) a status")
		fmt.Println("  a) let the engine pick    p) pass    u) undo    q) quit")
		answer, ok := ask(input, "> ")
		if !ok {
			return quit, nil
		}
		// A question is answered and then the menu is drawn again, deliberately
		// costing nothing: reading what a skill does is part of deciding, so it
		// cannot be a move, and a player who asks about the wrong one must be
		// able to ask about the next without spending anything.
		if question, asked := strings.CutPrefix(answer, "?"); asked {
			describe(current, prompt, strings.TrimSpace(question))
			continue
		}
		switch answer {
		case "q":
			return quit, nil
		case "a":
			return acted, current.apply(current.suggest(prompt))
		case "p":
			return acted, current.apply(battle.Decision{
				Unit: prompt.Unit, Turn: prompt.Turn, Passed: true,
			})
		case "u":
			taken, err := current.undo()
			if err != nil {
				return quit, err
			}
			if !taken {
				fmt.Println("  there is nothing of yours to take back")
				continue
			}
			fmt.Println("\ntook back your last action")
			return rewound, nil
		}
		index, convErr := strconv.Atoi(answer)
		if convErr != nil || index < 1 || index > len(prompt.Options) {
			fmt.Println("  pick one of the numbers, or a, p, u or q")
			continue
		}
		option := prompt.Options[index-1]
		if !option.Available() {
			fmt.Printf("  %s cannot be used: %s\n", option.Skill, option.Reason)
			continue
		}
		aim, chosen, wantsOut := chooseAim(current, option, input)
		if wantsOut {
			return quit, nil
		}
		if !chosen {
			continue
		}
		if err := current.apply(battle.Decision{
			Unit: prompt.Unit, Turn: prompt.Turn, Skill: option.Skill, Aim: hex.At(aim),
		}); err != nil {
			fmt.Printf("  %v\n", err)
			continue
		}
		return acted, nil
	}
}

// chooseAim asks where a skill is pointed. A skill with one legal cell does not
// ask at all, because a question with one answer is not a decision.
func chooseAim(current *session, option battle.Option, input *bufio.Scanner) (aim hex.Offset, chosen, wantsOut bool) {
	if len(option.Aims) == 1 {
		return option.Aims[0], true, false
	}
	for {
		fmt.Printf("\naim %s at:\n", option.Skill)
		fmt.Println(tui.Aims(current.fight, option, current.tags))
		fmt.Println("  b) go back")
		answer, ok := ask(input, "> ")
		if !ok {
			return hex.Offset{}, false, true
		}
		switch answer {
		case "q":
			return hex.Offset{}, false, true
		case "b":
			return hex.Offset{}, false, false
		}
		index, err := strconv.Atoi(answer)
		if err != nil || index < 1 || index > len(option.Aims) {
			fmt.Println("  pick one of the numbers, or b to go back")
			continue
		}
		return option.Aims[index-1], true, false
	}
}

// replay prints a saved battle, and with -verify re-runs it from its seed to
// prove the file is a faithful record rather than a story about one.
func replay(cfg config) error {
	raw, err := os.ReadFile(cfg.replay)
	if err != nil {
		return fmt.Errorf("read %s: %w", cfg.replay, err)
	}
	log, err := battle.ParseLog(raw)
	if err != nil {
		return err
	}
	// The tags come from the log's own opening records, so rendering a saved
	// battle needs nothing but the file.
	tags := tui.TagsFromLog(log.Events)
	fmt.Printf("replaying seed %d, %d events, %d choices\n\n", log.Seed, len(log.Events), len(log.Choices))
	fmt.Println(tui.Log(log.Events, tags))
	fmt.Println()
	fmt.Println("== summary ==")
	fmt.Println(tui.Summary(log.Events, tags, tui.NamesFromLog(log.Events)))
	if !cfg.verify {
		// Everything above was read straight out of the file. Nothing re-ran the
		// battle, so a hand-edited log renders exactly like an honest one; say so
		// rather than let the output pass for a verified record.
		fmt.Printf("\nunverified: this is what %s says happened, not a re-run of it; "+
			"add -verify to replay seed %d and check every event\n", cfg.replay, log.Seed)
		return nil
	}
	return verify(log, cfg.limit)
}

func verify(log battle.Log, limit int) error {
	if !log.Replayable() {
		return fmt.Errorf(
			"this log records no placement, so there is nothing to re-run it with: " +
				"it was written before a placement was a choice, and re-running the shipped " +
				"roster against it would compare two different battles")
	}
	books, err := seed.Books()
	if err != nil {
		return err
	}
	// From the log's own roster rather than the shipped one. That is the whole
	// point of recording it: the battle being checked is the battle that was
	// fought, not whatever the data says today.
	fight, err := battle.New(books, log.Seed, log.Roster)
	if err != nil {
		return err
	}
	fight.Begin()
	if _, _, err := fight.Replay(log.Choices, limit, nil); err != nil {
		return fmt.Errorf("re-running the battle: %w", err)
	}
	rerun := fight.Drain()
	if len(rerun) != len(log.Events) {
		return fmt.Errorf("the log records %d events but re-running produced %d",
			len(log.Events), len(rerun))
	}
	for i := range rerun {
		if rerun[i] != log.Events[i] {
			return fmt.Errorf("event %d differs from the log:\nlogged  %+v\nre-ran  %+v",
				i, log.Events[i], rerun[i])
		}
	}
	fmt.Printf("\nverified: re-running seed %d reproduced all %d events exactly\n",
		log.Seed, len(rerun))
	return nil
}

// ask reads one line. A closed input reports false rather than looping for ever,
// so piping a short script in ends cleanly instead of hanging.
func ask(input *bufio.Scanner, prompt string) (string, bool) {
	fmt.Print(prompt)
	if !input.Scan() {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(input.Text())), true
}

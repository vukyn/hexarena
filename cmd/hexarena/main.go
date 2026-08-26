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
	cfg    config
	fight  *battle.Battle
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
	fight, err := seed.NewBattle(cfg.seed)
	if err != nil {
		return nil, err
	}
	fight.Begin()
	current := &session{
		cfg:   cfg,
		fight: fight,
		tags:  tui.Tags(fight.Units()),
		names: tui.Names(fight.Units()),
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
	fight, err := seed.NewBattle(s.cfg.seed)
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
		Skill: choice.Skill, Aim: choice.Aim, Passed: !ok,
	}
	if !ok {
		decision.Reason = battle.NoActionReason
	}
	return decision
}

func (s *session) apply(decision battle.Decision) error {
	if decision.Passed {
		if err := s.fight.Pass(decision.PassReason()); err != nil {
			return err
		}
	} else if err := s.fight.Act(decision.Skill, decision.Aim); err != nil {
		return err
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
		Seed: current.cfg.seed, Choices: current.script, Events: current.events,
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
func choose(current *session, prompt *battle.Prompt, input *bufio.Scanner) (outcome, error) {
	for {
		fmt.Println(tui.Order(current.fight.Queue(), current.tags, current.cfg.preview))
		fmt.Println(tui.Menu(current.fight, prompt, current.tags))
		fmt.Println("  a) let the engine pick    p) pass    u) undo    q) quit")
		answer, ok := ask(input, "> ")
		if !ok {
			return quit, nil
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
			Unit: prompt.Unit, Turn: prompt.Turn, Skill: option.Skill, Aim: aim,
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
	fight, err := seed.NewBattle(log.Seed)
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

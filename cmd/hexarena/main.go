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
	"strconv"
	"strings"

	"github.com/vukyn/hexarena/internal/core/battle"
	"github.com/vukyn/hexarena/internal/core/hex"
	"github.com/vukyn/hexarena/internal/seed"
	"github.com/vukyn/hexarena/internal/tui"
)

func main() {
	seedValue := flag.Uint64("seed", 1, "seed the battle's rolls come from; the same seed replays exactly")
	auto := flag.Bool("auto", false, "let both sides pick their own actions and print the whole battle")
	sideName := flag.String("side", "ally", "which side you play, ally or enemy")
	limit := flag.Int("turns", 4000, "give up after this many turns rather than hang on a stalemate")
	preview := flag.Int("preview", 6, "how many upcoming turns to show")
	flag.Parse()

	side, err := parseSide(*sideName)
	if err != nil {
		fail(err)
	}
	if err := run(*seedValue, *auto, side, *limit, *preview); err != nil {
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

func run(seedValue uint64, auto bool, side hex.Side, limit, preview int) error {
	fight, err := seed.NewBattle(seedValue)
	if err != nil {
		return err
	}
	tags := tui.Tags(fight.Units())
	input := bufio.NewScanner(os.Stdin)

	fight.Begin()
	fmt.Printf("hexarena, seed %d\n\n", seedValue)
	show(fight, tags)
	flush(fight, tags)

	for turns := 0; !fight.Finished() && turns < limit; turns++ {
		prompt, err := fight.Advance()
		if err != nil {
			return err
		}
		flush(fight, tags)
		if prompt.Skipped {
			continue
		}
		unit, ok := fight.Unit(prompt.Unit)
		if !ok {
			return fmt.Errorf("the battle offered a turn to %q, which is not fighting", prompt.Unit)
		}

		if auto || unit.Side != side {
			if err := autoAct(fight, prompt); err != nil {
				return err
			}
			flush(fight, tags)
			continue
		}

		fmt.Println()
		show(fight, tags)
		fmt.Printf("\n%s %s, turn %d at %s\n", tags[unit.ID], unit.Name, prompt.Turn, unit.Cell)
		quit, err := choose(fight, prompt, tags, input)
		if err != nil {
			return err
		}
		if quit {
			fmt.Println("\nleaving the battle unfinished")
			return nil
		}
		flush(fight, tags)
	}

	if !fight.Finished() {
		fmt.Printf("\nstopped after %d turns without a decision\n", limit)
		return nil
	}
	if winner, decided := fight.Winner(); decided {
		fmt.Printf("\nthe %s side wins\n", winner)
	} else {
		fmt.Println("\nthe battle ends with nobody standing")
	}
	fmt.Println()
	fmt.Println(tui.Roster(fight, tags))
	return nil
}

// show draws the board, the roster and what is coming.
func show(fight *battle.Battle, tags map[string]string) {
	fmt.Println(tui.Board(fight, tags))
	fmt.Println()
	fmt.Println(tui.Roster(fight, tags))
}

// flush prints and clears whatever the battle has recorded.
func flush(fight *battle.Battle, tags map[string]string) {
	if rendered := tui.Log(fight.Drain(), tags); rendered != "" {
		fmt.Println(rendered)
	}
}

func autoAct(fight *battle.Battle, prompt *battle.Prompt) error {
	choice, ok := fight.Suggest(prompt)
	if !ok {
		return fight.Pass("nothing usable")
	}
	return fight.Act(choice.Skill, choice.Aim)
}

// choose runs the two questions a turn needs: which skill, then where. It keeps
// asking rather than giving up on a bad answer, because a mistyped number should
// not cost a turn.
func choose(fight *battle.Battle, prompt *battle.Prompt, tags map[string]string, input *bufio.Scanner) (quit bool, err error) {
	for {
		fmt.Println(tui.Order(fight.Queue(), tags, 6))
		fmt.Println(tui.Menu(fight, prompt, tags))
		fmt.Println("  a) let the engine pick    p) pass    q) quit")
		answer, ok := ask(input, "> ")
		if !ok {
			return true, nil
		}
		switch answer {
		case "q":
			return true, nil
		case "a":
			return false, autoAct(fight, prompt)
		case "p":
			return false, fight.Pass("passed")
		}
		index, convErr := strconv.Atoi(answer)
		if convErr != nil || index < 1 || index > len(prompt.Options) {
			fmt.Println("  pick one of the numbers, or a, p or q")
			continue
		}
		option := prompt.Options[index-1]
		if !option.Available() {
			fmt.Printf("  %s cannot be used: %s\n", option.Skill, option.Reason)
			continue
		}
		aim, chosen, quit := chooseAim(fight, option, tags, input)
		if quit {
			return true, nil
		}
		if !chosen {
			continue
		}
		if actErr := fight.Act(option.Skill, aim); actErr != nil {
			fmt.Printf("  %v\n", actErr)
			continue
		}
		return false, nil
	}
}

// chooseAim asks where a skill is pointed. A skill with one legal cell does not
// ask at all, because a question with one answer is not a decision.
func chooseAim(fight *battle.Battle, option battle.Option, tags map[string]string, input *bufio.Scanner) (aim hex.Offset, chosen, quit bool) {
	if len(option.Aims) == 1 {
		return option.Aims[0], true, false
	}
	for {
		fmt.Printf("\naim %s at:\n", option.Skill)
		fmt.Println(tui.Aims(fight, option, tags))
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

// ask reads one line. A closed input reports false rather than looping for ever,
// so piping a short script in ends cleanly instead of hanging.
func ask(input *bufio.Scanner, prompt string) (string, bool) {
	fmt.Print(prompt)
	if !input.Scan() {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(input.Text())), true
}

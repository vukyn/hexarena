package main

import (
	"fmt"
	"os"

	"github.com/vukyn/hexarena/internal/core/progression"
)

func runShow(args []string) error {
	set := newFlagSet("show")
	dir := dataFlag(set)
	level := set.Int("level", progression.LevelCap,
		"the level to resolve the character at; the cap is what the stat budget is written for")
	operands, err := parseArgs(set, args)
	if err != nil {
		return err
	}
	if len(operands) != 1 {
		return fmt.Errorf("usage: hexforge show <id> [--level N]")
	}
	if *level < 1 || *level > progression.LevelCap {
		return fmt.Errorf("level %d is outside 1..%d", *level, progression.LevelCap)
	}
	lib, err := load(*dir)
	if err != nil {
		return err
	}
	id := operands[0]
	character, known := lib.characters.Get(id)
	if !known {
		return fmt.Errorf("no character %q in %s; list them with: hexforge cast", id, lib.dir)
	}
	renderCharacter(os.Stdout, lib, character, *level)
	return nil
}

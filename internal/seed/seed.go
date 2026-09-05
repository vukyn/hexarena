// Package seed owns the embedded data files the game boots from. Keeping the
// embed here means the core packages stay pure: they parse bytes handed to
// them and never touch the filesystem.
package seed

import (
	"embed"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/combat"
	"github.com/vukyn/hexarena/internal/core/composition"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/modifier"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/pattern"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/core/skill"
	"github.com/vukyn/hexarena/internal/core/status"
)

//go:embed data/elements.json data/combat.json data/progression.json data/modifiers.json data/patterns.json data/statuses.json data/passives.json data/skills.json data/origins.json data/species.json data/archetypes.json data/cast.json data/roster.json data/builds.json data/squads.json data/bonuses.json
var files embed.FS

// ElementsFile is the raw affinity chart declaration.
func ElementsFile() ([]byte, error) { return files.ReadFile("data/elements.json") }

// ElementChart parses the embedded affinity chart.
func ElementChart() (*element.Chart, error) {
	raw, err := ElementsFile()
	if err != nil {
		return nil, err
	}
	return element.ParseChart(raw)
}

// CombatFile is the raw damage-formula declaration.
func CombatFile() ([]byte, error) { return files.ReadFile("data/combat.json") }

// CombatRules parses the embedded damage-formula constants.
func CombatRules() (combat.Rules, error) {
	raw, err := CombatFile()
	if err != nil {
		return combat.Rules{}, err
	}
	return combat.ParseRules(raw)
}

// ProgressionFile is the raw stat-budget declaration.
func ProgressionFile() ([]byte, error) { return files.ReadFile("data/progression.json") }

// ProgressionLimits parses the embedded stat budget.
func ProgressionLimits() (progression.Limits, error) {
	raw, err := ProgressionFile()
	if err != nil {
		return progression.Limits{}, err
	}
	return progression.ParseLimits(raw)
}

// ModifiersFile is the raw buff and debuff bounds declaration.
func ModifiersFile() ([]byte, error) { return files.ReadFile("data/modifiers.json") }

// ModifierBounds parses the embedded buff and debuff limits.
func ModifierBounds() (modifier.Bounds, error) {
	raw, err := ModifiersFile()
	if err != nil {
		return modifier.Bounds{}, err
	}
	return modifier.ParseBounds(raw)
}

// PatternsFile is the raw area-shape declaration.
func PatternsFile() ([]byte, error) { return files.ReadFile("data/patterns.json") }

// PatternBook parses the embedded area shapes.
func PatternBook() (*pattern.Book, error) {
	raw, err := PatternsFile()
	if err != nil {
		return nil, err
	}
	return pattern.ParseBook(raw)
}

// StatusesFile is the raw timed-effect declaration.
func StatusesFile() ([]byte, error) { return files.ReadFile("data/statuses.json") }

// StatusBook parses the embedded timed effects.
func StatusBook() (*status.Book, error) {
	raw, err := StatusesFile()
	if err != nil {
		return nil, err
	}
	return status.ParseBook(raw)
}

// PassivesFile is the raw passive declaration.
func PassivesFile() ([]byte, error) { return files.ReadFile("data/passives.json") }

// PassiveBook parses the embedded passives against the status book, which is
// what a granted status is checked against.
func PassiveBook() (*passive.Book, error) {
	raw, err := PassivesFile()
	if err != nil {
		return nil, err
	}
	statuses, err := StatusBook()
	if err != nil {
		return nil, err
	}
	return passive.ParseBook(raw, passive.Deps{Statuses: statuses})
}

// BonusesFile is the raw composition-bonus declaration.
func BonusesFile() ([]byte, error) { return files.ReadFile("data/bonuses.json") }

// BonusBook parses the embedded composition bonuses against the two books a
// bonus is checked with: the statuses it grants and the chart it counts.
func BonusBook() (*composition.Book, error) {
	raw, err := BonusesFile()
	if err != nil {
		return nil, err
	}
	statuses, err := StatusBook()
	if err != nil {
		return nil, err
	}
	chart, err := ElementChart()
	if err != nil {
		return nil, err
	}
	return composition.ParseBook(raw, composition.Deps{Statuses: statuses, Chart: chart})
}

// SkillsFile is the raw skill declaration.
func SkillsFile() ([]byte, error) { return files.ReadFile("data/skills.json") }

// SkillBook parses the embedded skills, checking every shape and status they
// name against the books that declare them.
func SkillBook() (*skill.Book, error) {
	raw, err := SkillsFile()
	if err != nil {
		return nil, err
	}
	patterns, err := PatternBook()
	if err != nil {
		return nil, err
	}
	statuses, err := StatusBook()
	if err != nil {
		return nil, err
	}
	return skill.ParseBook(raw, skill.Deps{Patterns: patterns, Statuses: statuses})
}

// OriginsFile is the raw catalog of works the cast is borrowed from.
func OriginsFile() ([]byte, error) { return files.ReadFile("data/origins.json") }

// Origins parses the embedded origin catalog.
func Origins() (*cast.OriginBook, error) {
	raw, err := OriginsFile()
	if err != nil {
		return nil, err
	}
	return cast.ParseOrigins(raw)
}

// SpeciesFile is the raw catalog of what a unit can be.
func SpeciesFile() ([]byte, error) { return files.ReadFile("data/species.json") }

// Species parses the embedded species catalog.
func Species() (*cast.SpeciesBook, error) {
	raw, err := SpeciesFile()
	if err != nil {
		return nil, err
	}
	return cast.ParseSpecies(raw)
}

// ArchetypesFile is the raw role-preset declaration.
func ArchetypesFile() ([]byte, error) { return files.ReadFile("data/archetypes.json") }

// Archetypes parses the embedded role presets, checking every skill they
// suggest and every curve they propose against the stat budget.
func Archetypes() (*cast.ArchetypeBook, error) {
	raw, err := ArchetypesFile()
	if err != nil {
		return nil, err
	}
	limits, err := ProgressionLimits()
	if err != nil {
		return nil, err
	}
	rules, err := CombatRules()
	if err != nil {
		return nil, err
	}
	skills, err := SkillBook()
	if err != nil {
		return nil, err
	}
	passives, err := PassiveBook()
	if err != nil {
		return nil, err
	}
	return cast.ParseArchetypes(raw, cast.ArchetypeDeps{
		Skills: skills, Passives: passives, Limits: limits, Rules: rules,
	})
}

// CastFile is the raw character declaration.
func CastFile() ([]byte, error) { return files.ReadFile("data/cast.json") }

// Cast parses the embedded characters, checking every origin, archetype and
// skill they name against the book that declares it.
func Cast() (*cast.Book, error) {
	raw, err := CastFile()
	if err != nil {
		return nil, err
	}
	deps, err := CastDeps()
	if err != nil {
		return nil, err
	}
	return cast.ParseBook(raw, deps)
}

// BuildsFile is the raw build-catalogue declaration.
func BuildsFile() ([]byte, error) { return files.ReadFile("data/builds.json") }

// Builds parses the embedded build catalogue against the cast, which is what
// says whether a build's kit is one its character could field.
func Builds() (*cast.BuildBook, error) {
	raw, err := BuildsFile()
	if err != nil {
		return nil, err
	}
	characters, err := Cast()
	if err != nil {
		return nil, err
	}
	return cast.ParseBuilds(raw, characters)
}

// CastDeps assembles everything a character is validated against. It is
// exported because an authoring tool builds its own cast book from a working
// directory and has to validate it against the same books.
func CastDeps() (cast.Deps, error) {
	origins, err := Origins()
	if err != nil {
		return cast.Deps{}, err
	}
	species, err := Species()
	if err != nil {
		return cast.Deps{}, err
	}
	archetypes, err := Archetypes()
	if err != nil {
		return cast.Deps{}, err
	}
	skills, err := SkillBook()
	if err != nil {
		return cast.Deps{}, err
	}
	passives, err := PassiveBook()
	if err != nil {
		return cast.Deps{}, err
	}
	chart, err := ElementChart()
	if err != nil {
		return cast.Deps{}, err
	}
	limits, err := ProgressionLimits()
	if err != nil {
		return cast.Deps{}, err
	}
	rules, err := CombatRules()
	if err != nil {
		return cast.Deps{}, err
	}
	return cast.Deps{
		Origins: origins, Species: species, Archetypes: archetypes,
		Skills: skills, Passives: passives,
		Chart: chart, Limits: limits, Rules: rules,
	}, nil
}

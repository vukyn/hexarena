// Package progression holds everything about a unit that is resolved before a
// battle starts: how its stats grow with level, and which evolution stage it
// has reached.
//
// Stats grow linearly. A curve declares where a stat starts at level 1 and
// where it lands at the level cap, and any level in between is interpolated
// with a single integer division. Authoring a start and an end reads better
// than authoring a per-level increment, and it makes both endpoints exact
// instead of accumulating rounding error over sixty steps.
//
// Evolution is resolved here too, never inside the battle engine: a level
// selects a stage, the stage carries a stat table, and the engine only ever
// receives the flat numbers that come out. That is why nothing downstream has
// to know evolution exists.
package progression

import (
	"encoding/json"
	"fmt"

	"github.com/vukyn/hexarena/internal/core/combat"
)

// LevelCap is the highest level a unit can reach. It is a fixed design
// decision rather than a tuning knob, so it lives in code; the data files
// declare it too and Limits.Validate rejects a mismatch.
const LevelCap = 60

// Kind identifies one of the four stats.
type Kind uint8

const (
	HP Kind = iota
	Attack
	Defense
	Speed
	// Accuracy raises the chance a skill connects. It is a stat rather than a
	// per-skill constant because two units with the same skill should not be
	// equally reliable with it.
	Accuracy
	// Dodge lowers the chance an incoming skill connects. It is the answer to
	// accuracy: without it, whether an attack lands would be entirely the
	// attacker's business and stacking accuracy would have no counterplay.
	Dodge
)

// KindCount is the number of stats.
const KindCount = int(Dodge) + 1

var kindNames = [KindCount]string{
	HP:       "hp",
	Attack:   "attack",
	Defense:  "defense",
	Speed:    "speed",
	Accuracy: "accuracy",
	Dodge:    "dodge",
}

func (k Kind) String() string {
	if int(k) >= KindCount {
		return fmt.Sprintf("stat(%d)", uint8(k))
	}
	return kindNames[k]
}

// Kinds returns every stat in declaration order.
func Kinds() []Kind {
	out := make([]Kind, 0, KindCount)
	for i := 0; i < KindCount; i++ {
		out = append(out, Kind(i))
	}
	return out
}

// Values is one number per stat. It serves both as a resolved stat line and as
// a set of ceilings.
type Values [KindCount]int64

// Get returns one stat, or zero for an unknown kind.
func (v Values) Get(kind Kind) int64 {
	if int(kind) >= KindCount {
		return 0
	}
	return v[kind]
}

func (v Values) String() string {
	return fmt.Sprintf("hp %d, atk %d, def %d, spd %d, acc %d, ddg %d",
		v[HP], v[Attack], v[Defense], v[Speed], v[Accuracy], v[Dodge])
}

type valuesFile struct {
	HP       *int64 `json:"hp"`
	Attack   *int64 `json:"attack"`
	Defense  *int64 `json:"defense"`
	Speed    *int64 `json:"speed"`
	Accuracy *int64 `json:"accuracy"`
	Dodge    *int64 `json:"dodge"`
}

// UnmarshalJSON requires all four stats to be present, so a typo in a data
// file cannot silently leave a stat at zero.
func (v *Values) UnmarshalJSON(raw []byte) error {
	var file valuesFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("decode stat values: %w", err)
	}
	fields := [KindCount]*int64{
		HP: file.HP, Attack: file.Attack, Defense: file.Defense,
		Speed: file.Speed, Accuracy: file.Accuracy, Dodge: file.Dodge,
	}
	for _, kind := range Kinds() {
		if fields[kind] == nil {
			return fmt.Errorf("stat values are missing %q", kind)
		}
		v[kind] = *fields[kind]
	}
	return nil
}

// MarshalJSON writes the stats by name.
func (v Values) MarshalJSON() ([]byte, error) {
	hp, attack, defense := v[HP], v[Attack], v[Defense]
	speed, accuracy, dodge := v[Speed], v[Accuracy], v[Dodge]
	return json.Marshal(valuesFile{
		HP: &hp, Attack: &attack, Defense: &defense,
		Speed: &speed, Accuracy: &accuracy, Dodge: &dodge,
	})
}

// Curve is a linear progression for one stat, from Base at level 1 to Max at
// LevelCap.
type Curve struct {
	Base int64 `json:"base"`
	Max  int64 `json:"max"`
}

// At returns the stat at a given level. Levels outside the range clamp to the
// nearest endpoint, and both endpoints come out exact.
func (c Curve) At(level int) int64 {
	switch {
	case level <= 1:
		return c.Base
	case level >= LevelCap:
		return c.Max
	}
	return c.Base + (c.Max-c.Base)*int64(level-1)/int64(LevelCap-1)
}

// Validate rejects a curve that starts at nothing or shrinks with level.
func (c Curve) Validate(kind Kind) error {
	switch {
	case c.Base <= 0:
		return fmt.Errorf("%s starts at %d, want a positive value", kind, c.Base)
	case c.Max < c.Base:
		return fmt.Errorf("%s ends at %d but starts at %d, stats may not shrink with level", kind, c.Max, c.Base)
	}
	return nil
}

// Table holds one curve per stat.
type Table [KindCount]Curve

type tableFile struct {
	HP       *Curve `json:"hp"`
	Attack   *Curve `json:"attack"`
	Defense  *Curve `json:"defense"`
	Speed    *Curve `json:"speed"`
	Accuracy *Curve `json:"accuracy"`
	Dodge    *Curve `json:"dodge"`
}

// UnmarshalJSON requires a curve for every stat.
func (t *Table) UnmarshalJSON(raw []byte) error {
	var file tableFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("decode stat table: %w", err)
	}
	fields := [KindCount]*Curve{
		HP: file.HP, Attack: file.Attack, Defense: file.Defense,
		Speed: file.Speed, Accuracy: file.Accuracy, Dodge: file.Dodge,
	}
	for _, kind := range Kinds() {
		if fields[kind] == nil {
			return fmt.Errorf("stat table is missing %q", kind)
		}
		t[kind] = *fields[kind]
	}
	return nil
}

// MarshalJSON writes the curves by stat name.
func (t Table) MarshalJSON() ([]byte, error) {
	hp, attack, defense := t[HP], t[Attack], t[Defense]
	speed, accuracy, dodge := t[Speed], t[Accuracy], t[Dodge]
	return json.Marshal(tableFile{
		HP: &hp, Attack: &attack, Defense: &defense,
		Speed: &speed, Accuracy: &accuracy, Dodge: &dodge,
	})
}

// At resolves every stat at a given level.
func (t Table) At(level int) Values {
	var out Values
	for _, kind := range Kinds() {
		out[kind] = t[kind].At(level)
	}
	return out
}

// Validate checks every curve on its own.
func (t Table) Validate() error {
	for _, kind := range Kinds() {
		if err := t[kind].Validate(kind); err != nil {
			return err
		}
	}
	return nil
}

// Limits are the design budget a unit's stats have to fit inside.
type Limits struct {
	// LevelCap must match the LevelCap constant; the field exists so a data
	// file that disagrees with the code fails loudly.
	LevelCap int `json:"level_cap"`
	// Ceilings is the highest each stat may reach at the level cap.
	Ceilings Values `json:"ceilings"`
	// MaxEffectiveHP bounds health and defence together.
	//
	// Health and damage reduction multiply, so a unit at both ceilings is not
	// merely durable, it is durable squared: 4800 health behind 800 defence
	// takes the same punishment as 17600 health behind none. Capping the
	// product is what stops a unit being accidentally unkillable, and it turns
	// the two stats into a trade rather than a stack.
	MaxEffectiveHP int64 `json:"max_effective_hp"`
}

// ParseLimits reads a limits declaration. It never touches the filesystem.
func ParseLimits(raw []byte) (Limits, error) {
	var limits Limits
	if err := json.Unmarshal(raw, &limits); err != nil {
		return Limits{}, fmt.Errorf("decode progression limits: %w", err)
	}
	if err := limits.Validate(); err != nil {
		return Limits{}, err
	}
	return limits, nil
}

// Validate checks the limits themselves are coherent.
func (l Limits) Validate() error {
	if l.LevelCap != LevelCap {
		return fmt.Errorf("level_cap is %d but the engine is built for %d", l.LevelCap, LevelCap)
	}
	for _, kind := range Kinds() {
		if l.Ceilings[kind] <= 0 {
			return fmt.Errorf("the %s ceiling is %d, want a positive value", kind, l.Ceilings[kind])
		}
	}
	if l.MaxEffectiveHP <= 0 {
		return fmt.Errorf("max_effective_hp is %d, want a positive value", l.MaxEffectiveHP)
	}
	return nil
}

// EffectiveHP returns how much raw damage a stat line absorbs: health scaled
// up by whatever its defence turns away.
func EffectiveHP(values Values, rules combat.Rules) int64 {
	reduction := int64(rules.DefenseReduction(values[Defense]))
	if reduction <= 0 {
		return values[HP]
	}
	return values[HP] * int64(combat.PermilleBase) / reduction
}

// CheckValues rejects a resolved stat line that breaks the budget.
func (l Limits) CheckValues(values Values, rules combat.Rules) error {
	for _, kind := range Kinds() {
		if values[kind] > l.Ceilings[kind] {
			return fmt.Errorf("%s is %d, over the ceiling of %d", kind, values[kind], l.Ceilings[kind])
		}
	}
	if effective := EffectiveHP(values, rules); effective > l.MaxEffectiveHP {
		return fmt.Errorf("%d health behind %d defence absorbs %d damage, over the budget of %d",
			values[HP], values[Defense], effective, l.MaxEffectiveHP)
	}
	return nil
}

// CheckTable rejects a stat table whose curve breaks the budget at any level.
//
// With linear curves and the current reduction formula the budget is worst at
// the cap, so checking the cap alone would be enough today. Walking every
// level costs sixty comparisons, does not depend on that property holding, and
// names the first level that breaks, which is what an author needs to see.
func (l Limits) CheckTable(table Table, rules combat.Rules) error {
	if err := table.Validate(); err != nil {
		return err
	}
	for level := 1; level <= LevelCap; level++ {
		if err := l.CheckValues(table.At(level), rules); err != nil {
			return fmt.Errorf("at level %d: %w", level, err)
		}
	}
	return nil
}

// Stage is one step of an evolution line.
type Stage struct {
	Name string `json:"name"`
	// MinLevel is the level at which this stage takes over. The first stage
	// must start at level 1.
	MinLevel int   `json:"min_level"`
	Stats    Table `json:"stats"`
}

// Line is a unit's full evolution line, ordered by MinLevel.
type Line []Stage

// Validate checks the line covers every level exactly once and that no stage
// breaks the stat budget.
func (l Line) Validate(limits Limits, rules combat.Rules) error {
	if len(l) == 0 {
		return fmt.Errorf("an evolution line needs at least one stage")
	}
	previous := 0
	for i, stage := range l {
		if stage.Name == "" {
			return fmt.Errorf("stage %d has no name", i)
		}
		switch {
		case i == 0 && stage.MinLevel != 1:
			return fmt.Errorf("the first stage %q starts at level %d, want 1", stage.Name, stage.MinLevel)
		case stage.MinLevel <= previous:
			return fmt.Errorf("stage %q starts at level %d, not after the previous stage's %d",
				stage.Name, stage.MinLevel, previous)
		case stage.MinLevel > LevelCap:
			return fmt.Errorf("stage %q starts at level %d, past the cap of %d",
				stage.Name, stage.MinLevel, LevelCap)
		}
		if err := limits.CheckTable(stage.Stats, rules); err != nil {
			return fmt.Errorf("stage %q: %w", stage.Name, err)
		}
		previous = stage.MinLevel
	}
	return nil
}

// StageAt returns the stage a unit of the given level has reached.
func (l Line) StageAt(level int) (Stage, error) {
	if level < 1 || level > LevelCap {
		return Stage{}, fmt.Errorf("level %d is outside 1..%d", level, LevelCap)
	}
	reached := -1
	for i, stage := range l {
		if stage.MinLevel > level {
			break
		}
		reached = i
	}
	if reached < 0 {
		return Stage{}, fmt.Errorf("no stage covers level %d", level)
	}
	return l[reached], nil
}

// Resolve flattens an evolution line and a level into the stat line the battle
// engine works with. Nothing downstream sees the line itself.
func (l Line) Resolve(level int) (Values, Stage, error) {
	stage, err := l.StageAt(level)
	if err != nil {
		return Values{}, Stage{}, err
	}
	return stage.Stats.At(level), stage, nil
}

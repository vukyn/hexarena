// Package element implements the elemental affinity chart.
//
// The chart is declared as data, not code: a set of cycles where each entry
// beats the next and the last beats the first, plus mutually-strong pairs and
// inert elements. Three short chains express every relation:
//
//	organic     water > fire > grass > ground > (water)
//	industrial  ice > metal > wind > electric > (ice)
//	cross       water > metal > grass > wind > fire > ice > ground > electric > (water)
//
// That layout gives each cycled element exactly two strengths and two
// weaknesses, so no element is structurally better than another. Validate
// enforces the property, which is what stops a later element being added in a
// way that quietly breaks the balance.
//
// An attacker and a defender each carry one element, so exactly one relation
// ever applies and multipliers never stack.
//
// Multipliers are integers in parts per thousand and damage scaling is integer
// division. There is no floating point anywhere in this package, so a battle
// replayed from the same seed produces the same numbers on every platform.
package element

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Element identifies one elemental affinity. The zero value is Neutral, which
// is inert in both directions and is the right default for a unit whose theme
// has no element.
type Element uint8

const (
	Neutral Element = iota
	Water
	Fire
	Grass
	Ground
	Ice
	Metal
	Wind
	Electric
	Light
	Dark
)

// Count is the number of declared elements.
const Count = int(Dark) + 1

var names = [Count]string{
	Neutral:  "neutral",
	Water:    "water",
	Fire:     "fire",
	Grass:    "grass",
	Ground:   "ground",
	Ice:      "ice",
	Metal:    "metal",
	Wind:     "wind",
	Electric: "electric",
	Light:    "light",
	Dark:     "dark",
}

func (e Element) String() string {
	if int(e) >= Count {
		return fmt.Sprintf("element(%d)", uint8(e))
	}
	return names[e]
}

// Valid reports whether the value is one of the declared elements.
func (e Element) Valid() bool { return int(e) < Count }

// Parse resolves an element name as written in the data files.
func Parse(name string) (Element, error) {
	for i, candidate := range names {
		if candidate == name {
			return Element(i), nil
		}
	}
	return Neutral, fmt.Errorf("unknown element %q", name)
}

// All returns every declared element in declaration order.
func All() []Element {
	out := make([]Element, 0, Count)
	for i := 0; i < Count; i++ {
		out = append(out, Element(i))
	}
	return out
}

// Multipliers holds the three affinity outcomes in parts per thousand.
type Multipliers struct {
	Advantage    int `json:"advantage"`
	Neutral      int `json:"neutral"`
	Disadvantage int `json:"disadvantage"`
}

// Chart is a resolved affinity chart. Build one with ParseChart; the zero
// value is not usable.
type Chart struct {
	multipliers Multipliers
	beats       [Count][Count]bool
	matrix      [Count][Count]int
	cycled      [Count]bool
	mutual      [Count]bool
	mutualPair  [Count][Count]bool
	inert       [Count]bool
}

type chartFile struct {
	Multipliers Multipliers `json:"multipliers"`
	Cycles      []struct {
		Name  string   `json:"name"`
		Chain []string `json:"chain"`
	} `json:"cycles"`
	Mutual [][]string `json:"mutual"`
	Inert  []string   `json:"inert"`
}

// ParseChart reads a chart declaration and resolves it into a lookup table.
// It never touches the filesystem; the caller supplies the bytes.
func ParseChart(raw []byte) (*Chart, error) {
	var file chartFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode element chart: %w", err)
	}

	chart := &Chart{multipliers: file.Multipliers}
	if err := chart.multipliers.validate(); err != nil {
		return nil, err
	}

	for _, cycle := range file.Cycles {
		if len(cycle.Chain) < 3 {
			return nil, fmt.Errorf("cycle %q has %d entries, a cycle needs at least 3", cycle.Name, len(cycle.Chain))
		}
		chain := make([]Element, 0, len(cycle.Chain))
		for _, name := range cycle.Chain {
			member, err := Parse(name)
			if err != nil {
				return nil, fmt.Errorf("cycle %q: %w", cycle.Name, err)
			}
			chain = append(chain, member)
		}
		for i, attacker := range chain {
			defender := chain[(i+1)%len(chain)]
			if err := chart.addEdge(attacker, defender, "cycle "+cycle.Name); err != nil {
				return nil, err
			}
			chart.cycled[attacker] = true
			chart.cycled[defender] = true
		}
	}

	for _, pair := range file.Mutual {
		if len(pair) != 2 {
			return nil, fmt.Errorf("mutual entry %v has %d members, want 2", pair, len(pair))
		}
		left, err := Parse(pair[0])
		if err != nil {
			return nil, fmt.Errorf("mutual pair: %w", err)
		}
		right, err := Parse(pair[1])
		if err != nil {
			return nil, fmt.Errorf("mutual pair: %w", err)
		}
		if err := chart.addEdge(left, right, "mutual pair"); err != nil {
			return nil, err
		}
		if err := chart.addEdge(right, left, "mutual pair"); err != nil {
			return nil, err
		}
		chart.mutual[left], chart.mutual[right] = true, true
		chart.mutualPair[left][right], chart.mutualPair[right][left] = true, true
	}

	for _, name := range file.Inert {
		member, err := Parse(name)
		if err != nil {
			return nil, fmt.Errorf("inert list: %w", err)
		}
		chart.inert[member] = true
	}

	chart.fillMatrix()
	if err := chart.Validate(); err != nil {
		return nil, err
	}
	return chart, nil
}

func (m Multipliers) validate() error {
	switch {
	case m.Neutral <= 0:
		return fmt.Errorf("neutral multiplier is %d, want a positive value in parts per thousand", m.Neutral)
	case m.Advantage <= m.Neutral:
		return fmt.Errorf("advantage multiplier %d does not exceed neutral %d", m.Advantage, m.Neutral)
	case m.Disadvantage <= 0 || m.Disadvantage >= m.Neutral:
		return fmt.Errorf("disadvantage multiplier %d is not between 0 and neutral %d", m.Disadvantage, m.Neutral)
	}
	return nil
}

func (c *Chart) addEdge(attacker, defender Element, source string) error {
	if attacker == defender {
		return fmt.Errorf("%s: %s beats itself", source, attacker)
	}
	if c.beats[attacker][defender] {
		return fmt.Errorf("%s: %s beats %s more than once", source, attacker, defender)
	}
	c.beats[attacker][defender] = true
	return nil
}

func (c *Chart) fillMatrix() {
	for attacker := 0; attacker < Count; attacker++ {
		for defender := 0; defender < Count; defender++ {
			switch {
			case c.beats[attacker][defender]:
				c.matrix[attacker][defender] = c.multipliers.Advantage
			case c.beats[defender][attacker]:
				c.matrix[attacker][defender] = c.multipliers.Disadvantage
			default:
				c.matrix[attacker][defender] = c.multipliers.Neutral
			}
		}
	}
}

// Validate enforces the structural invariants the chart's balance rests on:
// every element is classified exactly once, a pair is only strong in both
// directions when it was declared mutual, inert elements have no relations,
// members of a mutual pair have exactly one relation each way, and every
// cycled element has the same number of strengths as weaknesses.
func (c *Chart) Validate() error {
	for _, attacker := range All() {
		for _, defender := range All() {
			if !c.beats[attacker][defender] || !c.beats[defender][attacker] {
				continue
			}
			if !c.mutualPair[attacker][defender] {
				return fmt.Errorf("%s and %s beat each other but were not declared a mutual pair", attacker, defender)
			}
		}
	}

	for _, member := range All() {
		classes := 0
		for _, in := range []bool{c.cycled[member], c.mutual[member], c.inert[member]} {
			if in {
				classes++
			}
		}
		if classes != 1 {
			return fmt.Errorf("%s belongs to %d of {cycled, mutual, inert}, want exactly 1", member, classes)
		}
	}

	strengths, weaknesses := c.degrees()
	cycledDegree := -1
	for _, member := range All() {
		switch {
		case c.inert[member]:
			if strengths[member] != 0 || weaknesses[member] != 0 {
				return fmt.Errorf("inert element %s has %d strengths and %d weaknesses, want none",
					member, strengths[member], weaknesses[member])
			}
		case c.mutual[member]:
			if strengths[member] != 1 || weaknesses[member] != 1 {
				return fmt.Errorf("mutual element %s has %d strengths and %d weaknesses, want 1 and 1",
					member, strengths[member], weaknesses[member])
			}
		default:
			if strengths[member] != weaknesses[member] {
				return fmt.Errorf("%s has %d strengths but %d weaknesses, a cycled element must have the same of each",
					member, strengths[member], weaknesses[member])
			}
			if cycledDegree == -1 {
				cycledDegree = strengths[member]
			} else if strengths[member] != cycledDegree {
				return fmt.Errorf("%s has %d strengths, other cycled elements have %d; the cycles are uneven",
					member, strengths[member], cycledDegree)
			}
		}
	}
	return nil
}

func (c *Chart) degrees() (strengths, weaknesses [Count]int) {
	for attacker := 0; attacker < Count; attacker++ {
		for defender := 0; defender < Count; defender++ {
			if c.beats[attacker][defender] {
				strengths[attacker]++
				weaknesses[defender]++
			}
		}
	}
	return strengths, weaknesses
}

// Multiplier returns the affinity multiplier in parts per thousand for an
// attack of one element landing on a defender of another.
func (c *Chart) Multiplier(attacker, defender Element) int {
	if !attacker.Valid() || !defender.Valid() {
		return c.multipliers.Neutral
	}
	return c.matrix[attacker][defender]
}

// Scale applies the affinity multiplier to a damage figure using integer
// division, so the result is identical on every platform.
func (c *Chart) Scale(damage int64, attacker, defender Element) int64 {
	return damage * int64(c.Multiplier(attacker, defender)) / int64(c.multipliers.Neutral)
}

// Multipliers returns the three configured outcomes.
func (c *Chart) Multipliers() Multipliers { return c.multipliers }

// Strengths returns the elements the given element is strong against, in
// declaration order.
func (c *Chart) Strengths(member Element) []Element { return c.edgesFrom(member, true) }

// Weaknesses returns the elements the given element is weak to, in
// declaration order.
func (c *Chart) Weaknesses(member Element) []Element { return c.edgesFrom(member, false) }

func (c *Chart) edgesFrom(member Element, outgoing bool) []Element {
	if !member.Valid() {
		return nil
	}
	out := make([]Element, 0, 4)
	for _, other := range All() {
		hit := c.beats[member][other]
		if !outgoing {
			hit = c.beats[other][member]
		}
		if hit {
			out = append(out, other)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Affinity is a unit's elemental makeup: one element, or two.
//
// An attack always carries a single element — the one its skill declares — so
// a dual affinity is not a free offensive upgrade. What it changes is the
// defending side, where the two multipliers stack, and the set of skills the
// unit is allowed to learn.
//
// The zero value is a single neutral affinity, which is the right default for
// a unit whose theme has no element.
type Affinity struct {
	primary   Element
	secondary Element
	dual      bool
}

// Single builds a one-element affinity.
func Single(member Element) (Affinity, error) {
	if !member.Valid() {
		return Affinity{}, fmt.Errorf("affinity: %s is not a declared element", member)
	}
	return Affinity{primary: member}, nil
}

// Dual builds a two-element affinity. It rejects a repeated element and any
// pairing involving Neutral, which is inert and so adds nothing as a second
// element. Whether the two elements are allowed to sit together also depends
// on the chart; see Chart.ValidateAffinity.
func Dual(primary, secondary Element) (Affinity, error) {
	if !primary.Valid() || !secondary.Valid() {
		return Affinity{}, fmt.Errorf("affinity: %s/%s contains an undeclared element", primary, secondary)
	}
	if primary == secondary {
		return Affinity{}, fmt.Errorf("affinity: %s is listed twice", primary)
	}
	if primary == Neutral || secondary == Neutral {
		return Affinity{}, fmt.Errorf("affinity: %s/%s pairs with neutral, which is inert and adds nothing", primary, secondary)
	}
	return Affinity{primary: primary, secondary: secondary, dual: true}, nil
}

// Primary returns the first element, which is also the sole element of a
// single affinity.
func (a Affinity) Primary() Element { return a.primary }

// Secondary returns the second element and whether there is one.
func (a Affinity) Secondary() (Element, bool) { return a.secondary, a.dual }

// IsDual reports whether the affinity carries two elements.
func (a Affinity) IsDual() bool { return a.dual }

// Elements returns the one or two elements, primary first.
func (a Affinity) Elements() []Element {
	if a.dual {
		return []Element{a.primary, a.secondary}
	}
	return []Element{a.primary}
}

// Has reports whether the affinity includes the given element. This is the
// check that decides whether a unit may learn a skill of that element.
func (a Affinity) Has(member Element) bool {
	return a.primary == member || (a.dual && a.secondary == member)
}

func (a Affinity) String() string {
	if a.dual {
		return a.primary.String() + "/" + a.secondary.String()
	}
	return a.primary.String()
}

// UnmarshalJSON accepts either an element name or an array of one or two
// names, so seed data can write "fire" or ["fire", "wind"].
func (a *Affinity) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		member, err := Parse(name)
		if err != nil {
			return err
		}
		built, err := Single(member)
		if err != nil {
			return err
		}
		*a = built
		return nil
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("affinity must be an element name or an array of one or two names, got %s", raw)
	}
	members := make([]Element, 0, len(list))
	for _, name := range list {
		member, err := Parse(name)
		if err != nil {
			return err
		}
		members = append(members, member)
	}
	switch len(members) {
	case 1:
		built, err := Single(members[0])
		if err != nil {
			return err
		}
		*a = built
	case 2:
		built, err := Dual(members[0], members[1])
		if err != nil {
			return err
		}
		*a = built
	default:
		return fmt.Errorf("affinity lists %d elements, want 1 or 2", len(members))
	}
	return nil
}

// MarshalJSON writes a single affinity as a name and a dual affinity as an
// array, matching what UnmarshalJSON accepts.
func (a Affinity) MarshalJSON() ([]byte, error) {
	if a.dual {
		return json.Marshal([]string{a.primary.String(), a.secondary.String()})
	}
	return json.Marshal(a.primary.String())
}

// Related reports whether either element beats the other.
func (c *Chart) Related(left, right Element) bool {
	if !left.Valid() || !right.Valid() {
		return false
	}
	return c.beats[left][right] || c.beats[right][left]
}

// ValidateAffinity rejects a dual affinity whose two elements have a relation
// with each other.
//
// A unit that is both the counter and the victim of its own second element is
// incoherent, and the pairings that survive the rule turn out to be exactly
// the ones with a natural reading — water with ice, fire with metal, ground
// with wind. The rule still leaves plenty of duals with a stacked weakness, so
// it removes the nonsense without removing the risk.
func (c *Chart) ValidateAffinity(affinity Affinity) error {
	for _, member := range affinity.Elements() {
		if !member.Valid() {
			return fmt.Errorf("affinity %s contains an undeclared element", affinity)
		}
	}
	secondary, dual := affinity.Secondary()
	if !dual {
		return nil
	}
	if c.Related(affinity.Primary(), secondary) {
		return fmt.Errorf("affinity %s pairs elements that already counter each other", affinity)
	}
	return nil
}

// LegalPairs returns every dual affinity the chart allows, ordered by element
// declaration order.
func (c *Chart) LegalPairs() []Affinity {
	out := make([]Affinity, 0, Count*Count/2)
	for _, primary := range All() {
		for _, secondary := range All() {
			if secondary <= primary {
				continue
			}
			pair, err := Dual(primary, secondary)
			if err != nil {
				continue
			}
			if c.ValidateAffinity(pair) != nil {
				continue
			}
			out = append(out, pair)
		}
	}
	return out
}

// MultiplierAgainst returns the affinity multiplier in parts per thousand for
// an attack of one element landing on a defender's affinity. A dual defender
// stacks the two multipliers, so a hit that both elements are weak to lands at
// roughly 2.25x and one that both resist at roughly 0.44x, while a weakness
// and a resistance cancel back to neutral.
func (c *Chart) MultiplierAgainst(attacker Element, defender Affinity) int {
	multiplier := c.Multiplier(attacker, defender.Primary())
	if secondary, dual := defender.Secondary(); dual {
		multiplier = multiplier * c.Multiplier(attacker, secondary) / c.multipliers.Neutral
	}
	return multiplier
}

// ScaleAgainst applies MultiplierAgainst to a damage figure.
func (c *Chart) ScaleAgainst(damage int64, attacker Element, defender Affinity) int64 {
	return damage * int64(c.MultiplierAgainst(attacker, defender)) / int64(c.multipliers.Neutral)
}

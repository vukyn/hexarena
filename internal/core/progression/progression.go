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
	"strings"

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
//
// It describes damage that does not pierce. That was every damage source in the
// game until piercing arrived and it is still all but one of them, which is why
// this stays the plain name and the bound is measured against it — but it is no
// longer the worst case, and anything showing the figure to an author has to say
// which case it means. See EffectiveHPAgainst.
func EffectiveHP(values Values, rules combat.Rules) int64 {
	return EffectiveHPAgainst(values, rules, 0)
}

// EffectiveHPAgainst is the same figure against damage that ignores a share of
// the defence, in parts per thousand.
//
// Against full piercing it comes back as the raw health, because that is what a
// unit with no defence left absorbs. That end of the range is the honest floor
// of a stat line's durability and the number an armour-heavy build should be
// read against, since piercing is the counter it was given.
func EffectiveHPAgainst(values Values, rules combat.Rules, pierce int) int64 {
	reduction := int64(rules.DefenseReduction(combat.Pierced(values[Defense], pierce)))
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
	MinLevel int `json:"min_level"`
	// Image is this form's own art, and it is optional: a stage that names none
	// shows the character's. The fallback is not applied here — see
	// cast.Character.StageArt, which is the one place that decides it, because
	// this package has no character to fall back to.
	//
	// It sits beside Name rather than one layer up because it is the same kind
	// of fact: what this form is called and what it looks like both belong to
	// the form, and a parallel list keyed by stage name would be a second thing
	// to keep in step — one that goes stale exactly when a stage is renamed or
	// removed. What this package does not do is check the path: that is
	// cast.ValidateImagePath's job, the same division a skill's pattern name
	// follows.
	Image string `json:"image,omitempty"`
	// After is the stage this one grows out of, by name, and it is what lets a
	// line fork: two stages naming the same predecessor are alternatives, and a
	// placement picks which arm it fielded.
	//
	// It is optional, and its absence means "the stage declared before this one"
	// — which is what every line meant before forks existed, so a linear line
	// writes nothing and reads as it always did. ⚠️ **A line may not mix the
	// two.** The moment any stage names a predecessor, every stage but the root
	// has to, because otherwise the *order of the file* would decide who a stage
	// grows out of in a file that also says so explicitly — which is a silent
	// wrong answer rather than an error, and the exact failure this whole change
	// is written to avoid. Line.Validate refuses the mixture.
	After string `json:"after,omitempty"`
	Stats Table  `json:"stats"`
}

// ValidateStageName checks the shape of a stage name.
//
// A stage name is an **identifier**, not display text, and this is the rule that
// keeps it one. Four things in this repository say so:
//
//   - Line.Resolve looks a stage up by comparing candidate.Name == stage, so the
//     name is the lookup key rather than a caption beside one.
//   - Stage.After names a predecessor **by name** ("after": "Ivysaur"), which is
//     a second hand-typed spelling of the same string.
//   - A roster entry or a squad placement writes "stage": "Ivysaur", and a
//     learnset entry gates itself on a list of stage names — both spelled by hand
//     in files the line itself never sees.
//   - Line.Validate refuses two stages sharing a name, "so naming one of them
//     chooses neither". Only a key has a uniqueness constraint.
//
// A string spelled by hand in four places and compared with == has to be
// typeable and has to have exactly one spelling, and both fail outside ASCII:
// "Tiên" has two Unicode encodings that draw identically — the composed ê and an
// e followed by a combining circumflex — and == calls them different names, so
// the visibly-correct key silently misses. Printable ASCII is how that is
// checked; it is the mechanical form of the rule rather than the rule itself.
//
// The other half is that a stage name reaches the screen **unglossed, in both
// languages** — there is nothing to translate it to, so it is drawn exactly as
// the data writes it. A name authored in one language's own script is therefore
// that language's word on the other language's screen. Refusing anything outside
// ASCII refuses that too. There is deliberately no Lang.StageName accessor: the
// house rule is that an id is shown as the data writes it.
//
// The rule is no tighter than that on purpose. Spaces, digits, hyphens,
// apostrophes and periods stay legal, because "Mega Charizard X", "Ho-Oh" and
// "Porygon2" are all plausible forms and none of them is a phrase in a language.
// What is refused is a name that is empty, carries a byte outside printable
// ASCII, carries no letter at all, or has a space at either end or two in a row —
// the last because a difference between two spellings of a key that nobody can
// see is exactly the failure the ASCII rule exists to prevent.
//
// It is exported for the same reason cast.ValidateID and cast.ValidateImagePath
// are: an authoring tool has to be able to reject an answer as it is typed,
// rather than at the end of a wizard nobody wants to fill in twice.
func ValidateStageName(name string) error {
	if name == "" {
		return fmt.Errorf("has no name")
	}
	letters := 0
	for i := 0; i < len(name); i++ {
		letter := name[i]
		switch {
		case letter >= 'a' && letter <= 'z', letter >= 'A' && letter <= 'Z':
			letters++
		case letter >= ' ' && letter <= '~':
		default:
			return fmt.Errorf(
				"is called %q, which is not written in ASCII; a stage name is the key an after, a placement and a learnset gate each spell by hand, so it has to be typeable and to have one spelling",
				name)
		}
	}
	if letters == 0 {
		return fmt.Errorf("is called %q, which has no letter in it", name)
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("is called %q, which has a space at one end that no other spelling of the key will have", name)
	}
	if strings.Contains(name, "  ") {
		return fmt.Errorf("is called %q, which has two spaces in a row — a difference between two spellings of the key that nobody can see", name)
	}
	return nil
}

// Line is a unit's full evolution line, ordered by MinLevel.
type Line []Stage

// Validate checks the line is a tree rooted at level one, that every stage
// starts after the one it grows out of, and that no stage breaks the stat
// budget.
func (l Line) Validate(limits Limits, rules combat.Rules) error {
	if len(l) == 0 {
		return fmt.Errorf("an evolution line needs at least one stage")
	}
	parents, err := l.Parents()
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(l))
	for i, stage := range l {
		if err := ValidateStageName(stage.Name); err != nil {
			return fmt.Errorf("stage %d %w", i, err)
		}
		if seen[stage.Name] {
			return fmt.Errorf("two stages are called %q, so naming one of them chooses neither", stage.Name)
		}
		seen[stage.Name] = true
		after := 0
		if parents[i] >= 0 {
			after = l[parents[i]].MinLevel
		}
		switch {
		case i == 0 && stage.MinLevel != 1:
			return fmt.Errorf("the first stage %q starts at level %d, want 1", stage.Name, stage.MinLevel)
		case stage.MinLevel <= after:
			// The message names the predecessor rather than "the previous
			// stage", because with a fork the two are not the same thing and a
			// reader chasing the wrong one finds nothing wrong with it.
			return fmt.Errorf("stage %q starts at level %d, not after %q's %d",
				stage.Name, stage.MinLevel, l[parents[i]].Name, after)
		case stage.MinLevel > LevelCap:
			return fmt.Errorf("stage %q starts at level %d, past the cap of %d",
				stage.Name, stage.MinLevel, LevelCap)
		}
		if err := limits.CheckTable(stage.Stats, rules); err != nil {
			return fmt.Errorf("stage %q: %w", stage.Name, err)
		}
	}
	return nil
}

// Parents is each stage's predecessor by index, and -1 for the root.
//
// It is exported because a summary line has to draw the shape of a line and
// cannot work the parentage out a second time: two readings of the same rule are
// how a screen comes to disagree with the engine about what a file says.
//
// It is the one place the two spellings of a line meet. A line where nothing
// names an After is read by order — stage i grows out of stage i-1, which is
// what a line meant before it could fork. A line where anything names one is
// read by name only, and every stage but the first has to name a predecessor
// **declared before it**: that ordering requirement is what makes a cycle
// unwritable rather than something to go looking for, and it keeps a file
// readable top to bottom.
//
// ⚠️ Mixing the two is refused rather than resolved. A file that names some
// edges and leaves others to the order would have the order deciding parentage
// in a file that also states it, and the wrong answer would be a stat line
// rather than an error.
func (l Line) Parents() ([]int, error) {
	explicit := false
	for _, stage := range l {
		if stage.After != "" {
			explicit = true
			break
		}
	}
	out := make([]int, len(l))
	for i := range l {
		out[i] = i - 1
	}
	if !explicit {
		return out, nil
	}
	at := make(map[string]int, len(l))
	for i, stage := range l {
		if i > 0 && stage.After == "" {
			return nil, fmt.Errorf("stage %q names no predecessor while %q does: "+
				"a line that forks has to say what every stage grows out of, "+
				"because the order of the file cannot say it for one stage and not another",
				stage.Name, l[firstNamed(l)].Name)
		}
		if i == 0 && stage.After != "" {
			return nil, fmt.Errorf("the first stage %q grows out of %q, and nothing comes before the first",
				stage.Name, stage.After)
		}
		if i > 0 {
			parent, known := at[stage.After]
			if !known {
				return nil, fmt.Errorf("stage %q grows out of %q, which is not declared before it",
					stage.Name, stage.After)
			}
			out[i] = parent
		}
		at[stage.Name] = i
	}
	return out, nil
}

// firstNamed is the index of the first stage that names a predecessor, for a
// refusal that can point at the stage the file's own rule came from.
func firstNamed(l Line) int {
	for i, stage := range l {
		if stage.After != "" {
			return i
		}
	}
	return 0
}

// reached reports, for each stage, whether a level has grown into it: the stage
// itself and every stage it grows out of, all the way to the root.
//
// A stage past the level is not reached, and neither is one whose predecessor is
// past it.
//
// ⚠️ The second half is redundant on a **validated** line and always will be: a
// stage has to start after what it grows out of, so a level reaching the child
// has reached the parent already. Mutating it to plain `true` survives the whole
// suite. It is kept because this is asked of lines nobody has validated — a
// fixture, a file half-edited by a tool — and there the difference is a stat line
// rather than an error. Written down so the next person to mutate it stops
// looking for the missing test.
func (l Line) reached(level int) ([]bool, error) {
	parents, err := l.Parents()
	if err != nil {
		return nil, err
	}
	out := make([]bool, len(l))
	for i, stage := range l {
		// Parents are declared before their children, so a single pass in file
		// order always has the answer it needs.
		if stage.MinLevel > level {
			continue
		}
		if parents[i] < 0 {
			out[i] = true
			continue
		}
		out[i] = out[parents[i]]
	}
	return out, nil
}

// Furthest is the stage a level reaches on its own, and it is what a caller
// passes when it is not choosing.
//
// It is the empty string because "which form is this" and "the newest form this
// level allows" are the same answer whenever nobody has decided otherwise, and a
// second spelling would let a placement mean the default in two ways. Naming it
// is what keeps `Resolve(level, "")` from reading as a caller that forgot.
const Furthest = ""

// StageAt returns the furthest stage a unit of the given level has reached, and
// refuses when there is more than one.
//
// This is what a level *allows*, not what a unit *is*: a placement may field an
// earlier form, and Allowed is the list it may choose from. What this answers is
// the question a browser asks — "show me this character at level 30" — where
// there is no placement and therefore nobody to have chosen.
//
// ⚠️ **A line that forks has no single furthest, and that is an error rather
// than a pick.** The furthest stages are the reached ones with no reached stage
// growing out of them, and on a linear line there is exactly one — so nothing
// about a line without forks changes. Where two arms are reachable, taking
// "whichever the file lists last" would hand a browser, a budget row or a
// balance table the wrong form's stat line with nothing anywhere saying so. The
// refusal names both arms, because the caller's fix is to choose one.
func (l Line) StageAt(level int) (Stage, error) {
	furthest, err := l.Furthest(level)
	if err != nil {
		return Stage{}, err
	}
	if len(furthest) > 1 {
		return Stage{}, fmt.Errorf("level %d reaches %v, which are alternatives: name the one being fielded",
			level, StageNames(furthest))
	}
	return furthest[0], nil
}

// Furthest is every stage a level has reached that nothing reached grows out of:
// one stage on a line that does not fork, and one per arm on a line that does.
//
// It is exported because "what are the alternatives" is a question a front-end
// asks in its own right — a picker offering the arms, a refusal listing them —
// and working it out from Allowed a second time is how two screens come to
// disagree about what a fork is.
func (l Line) Furthest(level int) ([]Stage, error) {
	if level < 1 || level > LevelCap {
		return nil, fmt.Errorf("level %d is outside 1..%d", level, LevelCap)
	}
	reached, err := l.reached(level)
	if err != nil {
		return nil, err
	}
	parents, err := l.Parents()
	if err != nil {
		return nil, err
	}
	grown := make([]bool, len(l))
	for i := range l {
		if reached[i] && parents[i] >= 0 {
			grown[parents[i]] = true
		}
	}
	out := make([]Stage, 0, len(l))
	for i, stage := range l {
		if reached[i] && !grown[i] {
			out = append(out, stage)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no stage covers level %d", level)
	}
	return out, nil
}

// Leaves is every stage of the line that nothing grows out of: the tip of a
// line that does not fork, and one per arm of a line that does.
//
// ⚠️ **It is not Furthest, and the difference is the whole reason it exists.**
// Furthest is a fact about a *level* — the grown end of everything that level
// has reached — so it moves as the level moves and answers "what has this unit
// become". This is a fact about the *line* and moves only when the line is
// edited: it answers "is there anything after this form at all". A caller that
// wanted the second question and asked the first would get the right answer at
// the cap and a different one everywhere else, which is a silent wrong answer
// rather than an error.
//
// The caller it was added for is a PvP room's join gate, which insists a squad
// field fully grown units. ⚠️ **Written as Furthest(LevelCap) that gate would be
// right — provably, not by luck — and substituting it reddens nothing in this
// repository.** Validate refuses a stage whose MinLevel is past the cap, so every
// stage of every line it accepts is reachable at the cap, and there the grown end
// of what the cap reaches *is* the tip of each arm. That was measured rather than
// reasoned: the substitution passes all twenty-one tests over this predicate and
// its gate, because no legal line can tell the two apart at that level.
//
// So this exists to **name the question**, not to answer it differently, and the
// value of the name is the level that is no longer in it: the two answers diverge
// at every level below the cap, so a caller that wanted "is anything after this
// form" and reached for Furthest would have to supply a level, and the wrong
// level is a silent wrong answer rather than an error. The gate happens to ask
// only at 60 — it insists on level 60 before it asks — so it is the *next*
// caller this protects, and the difference that is real today is the error on a
// name the line does not answer to, below.
//
// An earlier draft of this comment justified it as "would start refusing a legal
// squad the day a stage was authored above the cap". That day cannot come:
// Validate is what refuses it. The conclusion survived the correction and the
// reason did not.
//
// It cannot be StageAt either, which refuses a fork outright rather than
// reporting both arms as tips.
//
// Parentage is read through Parents and never worked out here, because Parents
// is the one place the two spellings of a line meet: a line where nothing names
// an After is read by order, and one where anything does is read by name only.
// A second reading of that rule is how a gate comes to disagree with the engine
// about what a file says.
func (l Line) Leaves() ([]Stage, error) {
	parents, err := l.Parents()
	if err != nil {
		return nil, err
	}
	grown := make([]bool, len(l))
	for index := range l {
		if parents[index] >= 0 {
			grown[parents[index]] = true
		}
	}
	out := make([]Stage, 0, len(l))
	for index, stage := range l {
		if !grown[index] {
			out = append(out, stage)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no stage of this line is a tip, so every stage grows out of another")
	}
	return out, nil
}

// IsLeaf reports whether the named stage is a tip of the line: nothing in the
// line grows out of it.
//
// A name the line does not answer to is an **error** rather than a false. The
// two are different mistakes — a typo and a form with something after it — and
// answering false to both would let a misspelled stage be refused for the wrong
// reason, in a gate whose whole job is to say which rule a squad broke.
func (l Line) IsLeaf(name string) (bool, error) {
	leaves, err := l.Leaves()
	if err != nil {
		return false, err
	}
	found := false
	for _, stage := range l {
		if stage.Name == name {
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("no stage of this line is called %q; it has %v", name, StageNames(l))
	}
	for _, stage := range leaves {
		if stage.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// Allowed is every stage a level may be fielded as, in line order.
//
// A level reaching Venusaur reaches Ivysaur and Bulbasaur too, so a line that
// does not fork answers with a prefix of itself — which is what makes "evolving"
// a threshold that is passed rather than a door that closes behind the unit.
//
// A fork answers with **both arms** and everything before them, because each is
// a form the level may legally be fielded as. Fielding one is what makes the
// other moot, and that choice lives in the placement rather than here: nothing
// in the line marks an arm as taken, because nothing in a line knows about a
// unit.
func (l Line) Allowed(level int) ([]Stage, error) {
	if level < 1 || level > LevelCap {
		return nil, fmt.Errorf("level %d is outside 1..%d", level, LevelCap)
	}
	reached, err := l.reached(level)
	if err != nil {
		return nil, err
	}
	out := make([]Stage, 0, len(l))
	for i, stage := range l {
		if reached[i] {
			out = append(out, stage)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no stage covers level %d", level)
	}
	return out, nil
}

// StageNames is the names of a list of stages, which is what a refusal offers
// somebody who has just named one that is not there.
func StageNames(stages []Stage) []string {
	out := make([]string, 0, len(stages))
	for _, stage := range stages {
		out = append(out, stage.Name)
	}
	return out
}

// Resolve flattens an evolution line, a level and a chosen stage into the stat
// line the battle engine works with. Nothing downstream sees the line itself.
//
// # Why the stage is a parameter
//
// It used to be derived: the stage was whatever the level reached, and there was
// no decision anywhere in it. A level should instead *allow* a form, with
// whoever is fielding the unit saying which one it fielded — otherwise "may
// evolve" and "does evolve" are the same sentence, and a learnset entry gated on
// a stage would be a second spelling of a level.
//
// Furthest is the answer for a caller that is not choosing, and it is the
// behaviour every caller had before this: a browser showing a character at level
// 30 has no placement and therefore nobody to have chosen for it.
//
// Naming a stage the level has not reached is refused rather than clamped. A
// clamp would field a different unit from the one that was written down, which
// is the one outcome worse than saying no.
func (l Line) Resolve(level int, stage string) (Values, Stage, error) {
	if stage == Furthest {
		reached, err := l.StageAt(level)
		if err != nil {
			return Values{}, Stage{}, err
		}
		return reached.Stats.At(level), reached, nil
	}
	allowed, err := l.Allowed(level)
	if err != nil {
		return Values{}, Stage{}, err
	}
	for _, candidate := range allowed {
		if candidate.Name == stage {
			return candidate.Stats.At(level), candidate, nil
		}
	}
	// Told apart, because the two are different mistakes: a name nobody in the
	// line answers to is a typo, and a name that is simply ahead of the level is
	// a placement that has not grown into it yet.
	for _, candidate := range l {
		if candidate.Name == stage {
			return Values{}, Stage{}, fmt.Errorf(
				"stage %q begins at level %d, and this is level %d",
				stage, candidate.MinLevel, level)
		}
	}
	return Values{}, Stage{}, fmt.Errorf("no stage of this line is called %q; it has %v",
		stage, StageNames(l))
}

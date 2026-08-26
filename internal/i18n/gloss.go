package i18n

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// A gloss is the Vietnamese name of something a data file names: an element, an
// archetype, a skill. It is shown beside the id rather than instead of it —
// grass/electric (cỏ/điện) — so that a reader can check the translation against
// the thing it names, and still type, grep and edit the id the data holds.
//
// # Why this is not the catalog, and must not become it
//
// keys.go is the client's own wording. That set is finite, authored in Go, and
// strict on purpose: a key with no entry in a language is a bug, and
// TestEveryKeyIsWordedInEveryLanguage fails loudly on it, because the person a
// blank line lands on is the one least likely to report it.
//
// A gloss is the opposite kind of thing. It names an id in
// internal/seed/data — and skills and archetypes are added by editing JSON,
// which is the whole point of declaring them as data. So this table is
// open-ended by construction and a miss in it is **normal**: an id nobody has
// glossed yet renders as the bare id, exactly as it did before this file
// existed. Never "bolt ()", never a placeholder, never an error.
//
// Wiring these into the catalog would turn every new skill in a JSON file into
// a build-breaking gap in Go, which is precisely the cost that keeping balance
// out of Go was meant to avoid. So the two mechanisms stay apart: strict for
// wording, lenient for data.
//
// Elements and targeting sides are the exceptions, and only because they are not
// data in the same sense: element.Element and skill.Side are Go enums, adding to
// either is a Go change, and element.Chart.Validate already refuses a twelfth
// element that was not classified. TestEveryElementIsGlossed and
// TestEverySideIsGlossed hold those two sets complete for the same reason the
// catalog is held complete.
var (
	// The eleven declared elements. Fixed, and checked for completeness.
	elementGloss = map[string]string{
		"fire":     "lửa",
		"water":    "nước",
		"grass":    "cỏ",
		"ground":   "đất",
		"wind":     "gió",
		"ice":      "băng",
		"metal":    "kim loại",
		"electric": "điện",
		"light":    "sáng",
		"dark":     "tối",
		"neutral":  "trung tính",
	}

	// The four sides a skill can aim at. A Go enum like the elements, so this is
	// held complete too.
	//
	// They were bare ids on screen until "all" existed, and that is what made
	// the gap worth closing rather than a general tidy: three of the four read
	// as English words an author could guess at, and the fourth is the answer to
	// "can a skill hit both sides", which is the question that produced it. A
	// chooser offering "all" and nothing else would be a fourth option nobody
	// could tell from the first three.
	sideGloss = map[string]string{
		"enemy": "đối phương",
		"ally":  "bên mình",
		"self":  "bản thân",
		"all":   "cả hai bên",
	}

	// The statuses in statuses.json. Authored data like the skills, so this is
	// the lenient kind of table and a miss is a bare id.
	//
	// It exists because the field that names a status is a text field with a
	// syntax: an author picking one out of the book is choosing between "mire"
	// and "expose", and an id is not much to choose on.
	statusGloss = map[string]string{
		"poison": "trúng độc",
		"burn":   "bỏng",
		"weaken": "yếu đòn",
		"expose": "hở sườn",
		"blind":  "mù",
		"mire":   "sa lầy",
		"stun":   "choáng",
		"rally":  "sung sức",
		"haste":  "nhanh chân",
		"focus":  "tập trung",
		"veil":   "mờ ảo",
		"block":  "đỡ đòn",
	}

	// The role presets in archetypes.json. Hand-authored data, so this is a
	// lookup that may miss.
	archetypeGloss = map[string]string{
		"bulwark":    "lá chắn",
		"vanguard":   "tiên phong",
		"sentinel":   "trấn thủ",
		"duelist":    "đấu sĩ",
		"skirmisher": "du kích",
		"blighter":   "kẻ gieo độc",
	}

	// The nineteen skills that shipped before skill.Skill carried a name of its
	// own. This is a **fallback** now, not the place a skill's name is
	// authored — see SkillName.
	//
	// It is kept rather than migrated into skills.json on purpose. Moving it
	// would edit nineteen lines of a balance file to change nothing anybody can
	// observe, and the goldens measured from that file are the design record;
	// leaving it is why the field arrived without moving a single number. A skill
	// authored from here on carries its own name and never reaches this table,
	// and a shipped skill given one overrides its entry here.
	skillGloss = map[string]string{
		"strike":       "đòn đánh",
		"bolt":         "tia bắn",
		"ember_lance":  "thương lửa",
		"venom_fang":   "nanh độc",
		"creeping_rot": "mục rữa",
		"cinder_burst": "bùng than",
		"detonate":     "kích nổ",
		"flurry":       "đòn liên hoàn",
		"gale_slash":   "chém lốc",
		"sever":        "cắt lìa",
		"hex_curse":    "lời nguyền",
		"riptide":      "sóng ngầm",
		"arc_bolt":     "hồ quang",
		"swift_edge":   "lưỡi chớp",
		"guard_wall":   "tường chắn",
		"war_cry":      "tiếng hô xung trận",
		"quickstep":    "bước nhanh",
		"purify":       "thanh tẩy",
		"unmake":       "tước phép",
	}
)

// glossaries is the tables a lookup walks, in a fixed order.
//
// A slice of the five rather than one merged map, so each kind of id keeps its
// own list to read and to complete, and TestNoIDIsGlossedTwice holds them
// disjoint — with an ordered walk, an id in two tables would silently take the
// first, which is a wrong name rather than a missing one and therefore the
// worse of the two failures.
var glossaries = []map[string]string{
	elementGloss, sideGloss, statusGloss, archetypeGloss, skillGloss,
}

// glossBracket is how a gloss sits beside the id it explains. It is punctuation
// rather than wording, which is why it is here and not a Key: there is no
// language in which the brackets are words.
const glossBracket = "%s (%s)"

// affinityJoin is how the two halves of a dual affinity's gloss are separated.
// It is element.Affinity.String's own separator, so that the ids and their
// names read as the same pair; TestADualAffinityGlossesAsOnePair pins the two
// together on the exact format.
const affinityJoin = "/"

// kitJoin separates the names on a kit's gloss line. A skill's name is two or
// three words in Vietnamese, so a space alone would not say where one name ends
// and the next begins.
const kitJoin = " · "

// Gloss is the Vietnamese name for a data id, or empty when there is none.
//
// Empty in English too: there, a name is shown exactly as the data writes it,
// with no bracket and nothing added.
func (l Lang) Gloss(id string) string {
	if l != Vi {
		return ""
	}
	for _, table := range glossaries {
		if name := table[id]; name != "" {
			return name
		}
	}
	return ""
}

// Glossed is a data id with its Vietnamese name beside it, or the bare id when
// the table has no name for it.
func (l Lang) Glossed(id string) string {
	name := l.Gloss(id)
	if name == "" {
		return id
	}
	return fmt.Sprintf(glossBracket, id, name)
}

// SkillName is a skill's Vietnamese name: the one authored on the skill itself
// if it has one, otherwise the compiled table's, otherwise nothing.
//
// The order is the whole point and it goes one way only. A skill authored through
// the tool carries its name in skills.json, which is where a name belongs once it
// can be edited at all; the table is what the nineteen skills that shipped before
// the field existed still fall back on. So an authored name **wins**, including
// over a table entry for the same id — a shipped skill given a name through the
// form or `hexforge skills edit --name` renders under the new one, and the entry
// in skillGloss stops being reachable for it rather than fighting it.
//
// Empty in English, exactly as Gloss is: skill.Skill.Name is opaque text as far
// as internal/core is concerned, and it is *this* package that decides which
// slot it fills — the same one the Vietnamese table fills. An English screen
// shows a data id as the data writes it, and a column of Vietnamese appearing
// there would be a worse surprise than no column.
func (l Lang) SkillName(carried skill.Skill) string {
	if l != Vi {
		return ""
	}
	if authored := strings.TrimSpace(carried.Name); authored != "" {
		return authored
	}
	return l.Gloss(carried.ID)
}

// GlossedSkill is a skill's id with its Vietnamese name beside it, or the bare id
// when nothing has one — the same shape Glossed gives any other data id, over the
// authored-then-compiled order SkillName applies.
func (l Lang) GlossedSkill(carried skill.Skill) string {
	name := l.SkillName(carried)
	if name == "" {
		return carried.ID
	}
	return fmt.Sprintf(glossBracket, carried.ID, name)
}

// GlossedAffinity is an affinity — one element or two — with one bracket for
// the whole of it: grass/electric (cỏ/điện).
//
// One bracket rather than one per element, because the affinity is a single
// fact. "grass (cỏ)/electric (điện)" reads as two things a unit has, which is
// the one thing about a dual affinity an author must not come away believing.
//
// A half with no name of its own keeps its id inside the bracket, so the two
// sides stay positional and it is still visible which half is unglossed. With
// the element table held complete that is unreachable through real data, and it
// is written down anyway because "unreachable" is a property of today's enum.
func (l Lang) GlossedAffinity(affinity element.Affinity) string {
	members := affinity.Elements()
	names := make([]string, 0, len(members))
	glossed := false
	for _, member := range members {
		id := member.String()
		name := l.Gloss(id)
		if name == "" {
			name = id
		} else {
			glossed = true
		}
		names = append(names, name)
	}
	if !glossed {
		return affinity.String()
	}
	return fmt.Sprintf(glossBracket, affinity, strings.Join(names, affinityJoin))
}

// GlossedKit is a kit's Vietnamese names, in the kit's own order, for the
// dimmed row a caller draws under the ids themselves.
//
// A kit is up to five skills and five brackets do not fit in 80 columns — that
// was measured against the shipped presets, not judged by eye — so the names go
// on their own line under the ids rather than inline beside them.
//
// Empty when nothing in the kit is glossed, and in English, so that a caller
// draws no row at all instead of an empty one. A skill with no name keeps its
// id in place, which is what holds the line in the same order as the one above
// it.
//
// It takes the resolved skills rather than their ids, and that is what a name
// living on the declaration costs: an id is no longer enough to look a name up,
// because the first place to look is the skill itself. A caller with ids in hand
// resolves them — forge.Library.KitSkills is that, and it stands an id the book
// has lost in as a skill with nothing but that id, so this line stays one entry
// per id and in step with the ids above it.
func (l Lang) GlossedKit(carried []skill.Skill) string {
	names := make([]string, 0, len(carried))
	glossed := false
	for _, one := range carried {
		name := l.SkillName(one)
		if name == "" {
			name = one.ID
		} else {
			glossed = true
		}
		names = append(names, name)
	}
	if !glossed {
		return ""
	}
	return strings.Join(names, kitJoin)
}

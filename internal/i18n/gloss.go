package i18n

import (
	"fmt"
	"strings"

	"github.com/vukyn/hexarena/internal/core/cast"
	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/passive"
	"github.com/vukyn/hexarena/internal/core/skill"
)

// A gloss is the Vietnamese name of something a data file names: an element, an
// archetype, a skill. It is shown beside the id rather than instead of it —
// grass/electric <cỏ/điện> — so that a reader can check the translation against
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
		"light":    "ánh sáng",
		"dark":     "bóng tối",
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
		"poison":   "trúng độc",
		"burn":     "bỏng",
		"weaken":   "suy yếu",
		"expose":   "phá giáp",
		"blind":    "mù",
		"mire":     "sa lầy",
		"stun":     "choáng",
		"taunting": "bị khiêu khích",
		"fury":     "cuồng nộ",
		"haste":    "nhanh nhẹn",
		"focus":    "tập trung",
		"veil":     "mờ ảo",
		"block":    "đỡ đòn",
		// A counter rather than an effect: it is a noun because it is a THING the
		// holder is carrying, where every other harmful status here is a verb for
		// something being done to them. Nothing is being done — that is the point of
		// the category — and a verb would promise otherwise.
		"charge": "nhiễm điện",
		// A noun for the same reason the counter above is one, and a different
		// word from the block charge two lines up on purpose: `block` is "đỡ đòn",
		// what a unit DOES with a charge, while this is a thing standing in front
		// of it. A player who read the two as one word would be reading the wrong
		// half of the only choice the two guards offer.
		"aegis": "lá chắn",
		// The granted twin of the one above, and a different word because they
		// are a different bargain: `aegis` is a turn somebody spent and runs out,
		// `bastion` is what a unit simply walked in wearing and never comes back
		// once it is gone. A player reading both as "lá chắn" would think the
		// first one could be put back up.
		"bastion": "thành trì",
		// The three that shipped without one. A regen, and the two permanent
		// statuses a trait grants -- which are the ones a reader is least able
		// to guess at, because nothing else on screen says what they do.
		"unleashed": "bộc phát",
		"bare":      "trần trụi",
		"regrowth":  "tái sinh",
		"toughened": "kiên cường",
		"kindled":   "bùng cháy",
		"quickened": "gia tốc",
		"fortified": "kiên cố",
		"encumber":  "nặng nề",
		// A debuff, so a verb, like every other one here. It names the wound
		// rather than the mechanism: what the holder has is a sore that will not
		// close, and "less healing received" is the arithmetic under it.
		"fester": "lở loét",
		// A buff, so a noun. It is not haste ("nhanh nhẹn") and not veil
		// ("mờ ảo"): those two are speed and a timed blur, and this is the
		// standing quality of being hard to land a blow on.
		"evasive": "khinh công",
		// The three reserves. Nouns, for the reason `charge` two dozen lines up is
		// one — a counter is a THING its holder is carrying and nothing is being
		// done to anybody — and each is the weather its element leaves behind
		// rather than a word for fuel, so a reader who has never seen the category
		// still reads "there is a lot of heat about" off the row.
		//
		// ⚠️ `swelter` is "sóng nhiệt" and the fire skill that used to carry that
		// name is now "luồng nhiệt". A skill and a status sharing a name is two
		// different sentences in the log reading identically — "dùng sóng nhiệt"
		// against "sóng nhiệt ×3" — and the id collision that forced the rename
		// here was the same fact arriving through the gloss table.
		"swelter":  "sóng nhiệt",
		"verdure":  "xum xuê",
		"moisture": "ẩm ướt",
		// The fourth, and the one whose element has no weather. Ground leaves
		// behind what it always leaves behind: mass, sitting where it was put. A
		// noun like its three neighbours, and a plain one on purpose — a reader
		// meeting "sức nặng ×3" on a row is told the holder is carrying weight
		// and nothing about what it will do with it, which is exactly what a
		// counter is.
		"heft": "sức nặng",
	}

	// The status categories were glossed here and are not any more. A gloss
	// answers for a **data id**, and English deliberately has none — an id is
	// shown exactly as the data writes it — but a status.Category is a Go enum,
	// so the English half of a cleanse's sentence fell through to "stat_debuff"
	// while Vietnamese read correctly. They are keys now, in both languages, at
	// CategoryNounDot and its seven neighbours; Lang.StatusCategoryNoun is the
	// lookup. ⚠️ Removing the table also stopped it answering for the **skill**
	// `taunt`, whose id collides with the category's: nothing showed it, because
	// that skill carries an authored name and SkillName prefers one, but a
	// caller glossing the bare id would have been given "hiệu ứng khiêu khích".
	//
	// The role presets in archetypes.json.
	//
	// Unlike the other authored tables this one may **not** miss for a shipped
	// preset, and TestEveryShippedArchetypeIsGlossed says so. The character
	// browser prints the preset id under "lối chơi", so a preset with no entry
	// here reads as an English word in the middle of a Vietnamese screen -- and
	// it is the one field on that screen a reader cannot work out from the
	// numbers beside it. A preset that is not shipped is still free to miss.
	archetypeGloss = map[string]string{
		"bulwark":    "lá chắn",
		"vanguard":   "tiên phong",
		"sentinel":   "trấn thủ",
		"duelist":    "đấu sĩ",
		"skirmisher": "du kích",
		"blighter":   "kẻ gieo độc",
		"scorcher":   "kẻ thiêu đốt",
		"warden":     "người gác cổng",
		"summoner":   "người triệu hồi",
		"bruiser":    "kẻ áp sát",
		"slugger":    "kẻ giáng đòn",
		"mender":     "người chữa lành",
		"bombardier": "kẻ oanh tạc",
		"shifter":    "kẻ vạn biến",
		"breaker":    "kẻ phá giáp",
		"hexer":      "kẻ phá phép",
		"glacier":    "kẻ chịu đòn",
		"diehard":    "kẻ tử chiến",
		"sapper":     "kẻ bào mòn",
		"cleanser":   "người gột rửa",
		"tempest":    "kẻ gọi bão",
		"predator":   "kẻ săn mồi",
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
//
// ⚠️ **Angle brackets, and round ones would nest.** A gloss is drawn in places
// that already carry a parenthetical of their own — the battle log names the trait
// a status came from as `(virulence)`, so a round gloss inside it read
// `(virulence (độc lực))`, a bracket inside a bracket for the reader to unpick.
// The gloss is also the *inner* thing wherever the two meet, so it is the one that
// changes shape. Same two cells either way, so nothing measured moved.
const glossBracket = "%s <%s>"

// GlossBracket puts a name beside the id it explains, in the one shape this
// package uses everywhere — skirmisher <du kích>.
//
// It is exported because internal/tui draws data ids too, on the battle log, and
// the format has to have exactly one definition: a second spelling of "%s <%s>"
// somewhere else is a second thing to change the day the brackets become
// something else, and nothing would report the disagreement. The name is the
// caller's to find (a log reads a map, a screen reads SkillName); the punctuation
// is this package's.
func GlossBracket(id, name string) string {
	if name == "" || name == id {
		return id
	}
	return fmt.Sprintf(glossBracket, id, name)
}

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

// PassiveName is the name a trait is called by in this language, or nothing.
//
// There is no gloss table for traits, unlike skills and statuses: a trait's name
// is authored in passives.json and nowhere else, so this is the authored field
// and a trim, with none of SkillName's fallback to a compiled table. Vietnamese
// only, the same trade every authored name here makes — an English reader gets
// the id, which in English *is* the name.
func (l Lang) PassiveName(held passive.Passive) string {
	if l != Vi {
		return ""
	}
	return strings.TrimSpace(held.Name)
}

// SpeciesName is a kind's authored name in this language, or nothing.
//
// English gets nothing, the same trade PassiveName makes: the word beside the id
// is authored once and in Vietnamese, and a Vietnamese name on an English screen
// is a leak rather than a translation. TestTheScreensGlossEveryDataName is what
// caught the distinction being missed — a data name is a field on the
// declaration and is Vietnamese whoever asks, unlike a compiled gloss, which is
// empty in English by construction.
func (l Lang) SpeciesName(kind cast.Species) string {
	if l != Vi {
		return ""
	}
	return strings.TrimSpace(kind.Name)
}

// GlossedPassive is a trait's id with its Vietnamese name beside it, or the bare
// id when it has none — the same shape GlossedSkill gives a skill, so a screen
// showing one beside the other names them the same way.
func (l Lang) GlossedPassive(held passive.Passive) string {
	name := l.PassiveName(held)
	if name == "" {
		return held.ID
	}
	return fmt.Sprintf(glossBracket, held.ID, name)
}

// BuildName is what a build is called in this language, or nothing.
//
// The same trade PassiveName and SpeciesName make, and the reason is worth
// repeating for a build because a build is *mostly* its name: the field is
// authored in builds.json, in the one language the data is written in, so
// showing it on an English screen would be showing Vietnamese rather than
// translating anything. An English reader gets the id, which is a build's own
// summary — bulbasaur.poison says what "rải độc" says.
//
// What is *not* dropped in English is the intent beside it: a name is a label an
// id can stand in for, and an intent is prose with nothing behind it, which is
// the division an origin's note and a species' note already sit on.
func (l Lang) BuildName(built cast.Build) string {
	if l != Vi {
		return ""
	}
	return strings.TrimSpace(built.Name)
}

// GlossedBuild is a build's id with its Vietnamese name beside it, or the bare id
// when it has none — the shape GlossedPassive gives a trait, so a pane naming a
// build above a trait names both the same way.
func (l Lang) GlossedBuild(built cast.Build) string {
	name := l.BuildName(built)
	if name == "" {
		return built.ID
	}
	return fmt.Sprintf(glossBracket, built.ID, name)
}

// GlossedAffinity is an affinity — one element or two — with one bracket for
// the whole of it: grass/electric <cỏ/điện>.
//
// One bracket rather than one per element, because the affinity is a single
// fact. "grass <cỏ>/electric <điện>" reads as two things a unit has, which is
// the one thing about a dual affinity an author must not come away believing.
//
// A half with no name of its own keeps its id inside the bracket, so the two
// sides stay positional and it is still visible which half is unglossed. With
// the element table held complete that is unreachable through real data, and it
// is written down anyway because "unreachable" is a property of today's enum.
func (l Lang) GlossedAffinity(affinity element.Affinity) string {
	names := l.AffinityNames(affinity)
	if names == "" {
		return affinity.String()
	}
	return fmt.Sprintf(glossBracket, affinity, names)
}

// AffinityNames is the affinity's names without its ids, for the dimmed row a
// caller draws under the ids themselves — the shape GlossedKit, GlossedPassives
// and GlossedSpecies all hand back.
//
// It is the derivation and GlossedAffinity is the bracketed reading of it, so a
// screen that puts the names under the ids and one that puts them beside cannot
// disagree about what the names are. Empty when nothing is glossed, which is what
// makes an English screen draw no second row at all rather than an empty one.
func (l Lang) AffinityNames(affinity element.Affinity) string {
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
		return ""
	}
	return strings.Join(names, affinityJoin)
}

// GlossedKit is a kit's Vietnamese names, in the kit's own order, for the
// dimmed row a caller draws under the ids themselves.
//
// A kit is up to five skills and five brackets do not fit in 80 columns — that
// was measured against the shipped presets, not judged by eye — so the names go
// on their own line under the ids rather than inline beside them.
//
// ⚠️ **That measurement was taken at the old floor and has not been re-taken.**
// The floor is 120 now, so five brackets may well fit inline and the row under
// the ids may be a row this no longer has to spend. It is left alone rather than
// changed on the arithmetic: the reason to draw the names apart was a width that
// is gone, but whether they *read* better inline is a different question from
// whether they fit, and the answer to it was never written down. Re-measure
// against the shipped presets before moving them, the way this was measured
// before it was written.
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
// GlossedPassives is GlossedKit for traits: the authored names, or nothing at all
// when none of them has one.
//
// A trait's name is authored in the passive book and there is no compiled table
// behind it, unlike a skill's — the traits arrived after names were data, so
// there was never a version of them that needed one.
//
// Nothing in English, exactly as SkillName gives nothing: a data name is
// Vietnamese because that is what the data files hold, and an English screen shows a
// data id as the data writes it. Reading the field raw put "bền bỉ · máu độc"
// under an English traits row for as long as nobody looked at one.
func (l Lang) GlossedPassives(held []passive.Passive) string {
	if l != Vi {
		return ""
	}
	names := make([]string, 0, len(held))
	glossed := false
	for _, one := range held {
		name := l.PassiveName(one)
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

// GlossedSpecies is what a character is, in words rather than ids.
//
// The same shape GlossedPassives has and for the same reason: a species is
// authored with its name beside it, so there is no compiled table behind it and
// nothing to fall back to. A kind the catalog has lost keeps its id, which is
// what every id with no name does here.
func (l Lang) GlossedSpecies(kinds []cast.Species) string {
	if l != Vi {
		return ""
	}
	names := make([]string, 0, len(kinds))
	named := false
	for _, kind := range kinds {
		name := strings.TrimSpace(kind.Name)
		if name == "" {
			name = kind.ID
		} else {
			named = true
		}
		names = append(names, name)
	}
	if !named {
		return ""
	}
	return strings.Join(names, kitJoin)
}

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

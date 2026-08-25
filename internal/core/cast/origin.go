package cast

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Medium is the kind of work a character was borrowed from.
type Medium uint8

const (
	Anime Medium = iota
	Film
	Series
	Game
	Comic
	Novel
)

// MediumCount is the number of declared media.
const MediumCount = int(Novel) + 1

var mediumNames = [MediumCount]string{
	Anime:  "anime",
	Film:   "film",
	Series: "series",
	Game:   "game",
	Comic:  "comic",
	Novel:  "novel",
}

func (m Medium) String() string {
	if int(m) >= MediumCount {
		return fmt.Sprintf("medium(%d)", uint8(m))
	}
	return mediumNames[m]
}

// Valid reports whether the value is one of the declared media.
func (m Medium) Valid() bool { return int(m) < MediumCount }

// ParseMedium resolves a medium name as written in the data files.
func ParseMedium(name string) (Medium, error) {
	for i, candidate := range mediumNames {
		if candidate == name {
			return Medium(i), nil
		}
	}
	return 0, fmt.Errorf("unknown medium %q, want one of %s", name, strings.Join(MediumNames(), ", "))
}

// Mediums returns every declared medium in declaration order.
func Mediums() []Medium {
	out := make([]Medium, 0, MediumCount)
	for i := range MediumCount {
		out = append(out, Medium(i))
	}
	return out
}

// MediumNames returns every medium's name in declaration order, which is what
// a usage message wants.
func MediumNames() []string {
	out := make([]string, 0, MediumCount)
	for _, medium := range Mediums() {
		out = append(out, medium.String())
	}
	return out
}

// MarshalJSON writes the medium by name. Names are the wire format for every
// enum in this engine, for the same reason event kinds are: inserting a
// constant must not silently reinterpret a file that was already written.
func (m Medium) MarshalJSON() ([]byte, error) {
	if !m.Valid() {
		return nil, fmt.Errorf("cannot encode %s", m)
	}
	return json.Marshal(m.String())
}

// UnmarshalJSON reads a medium by name.
func (m *Medium) UnmarshalJSON(raw []byte) error {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return fmt.Errorf("medium must be a name, got %s", raw)
	}
	parsed, err := ParseMedium(name)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// earliestPlausibleYear is a sanity bound rather than a rule about art. It is
// there to catch a mistyped year, not to decide what counts as a work.
const earliestPlausibleYear = 1850

// Origin is a work a character was borrowed from.
type Origin struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Medium Medium `json:"medium"`
	// Year is optional. Zero means it was not recorded.
	Year int    `json:"year"`
	Note string `json:"note,omitempty"`
}

// OriginBook is the catalog of works the cast is drawn from.
type OriginBook struct {
	origins []Origin
	byID    map[string]Origin
}

type originFile struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Medium is a pointer so an omitted field is an error rather than a silent
	// fall back to the zero medium.
	Medium *string `json:"medium"`
	Year   int     `json:"year"`
	Note   string  `json:"note"`
}

type originBookFile struct {
	Origins []originFile `json:"origins"`
}

type originMarshalFile struct {
	Origins []Origin `json:"origins"`
}

// ParseOrigins reads an origin catalog. It never touches the filesystem.
//
// An empty catalog is allowed, because a project that has not borrowed from
// anything yet is a starting point rather than a data error.
func ParseOrigins(raw []byte) (*OriginBook, error) {
	var file originBookFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("decode origin book: %w", err)
	}
	book := &OriginBook{byID: make(map[string]Origin, len(file.Origins))}
	for _, declared := range file.Origins {
		built, err := resolveOrigin(declared)
		if err != nil {
			return nil, err
		}
		if _, clash := book.byID[built.ID]; clash {
			return nil, fmt.Errorf("origin %q is declared twice", built.ID)
		}
		book.byID[built.ID] = built
		book.origins = append(book.origins, built)
	}
	return book, nil
}

func resolveOrigin(declared originFile) (Origin, error) {
	if err := checkSlug("origin", declared.ID); err != nil {
		return Origin{}, err
	}
	fail := func(format string, args ...any) (Origin, error) {
		return Origin{}, fmt.Errorf("origin %q: "+format, append([]any{declared.ID}, args...)...)
	}
	if strings.TrimSpace(declared.Title) == "" {
		return fail("has no title")
	}
	if declared.Medium == nil {
		return fail("does not say which medium it is")
	}
	medium, err := ParseMedium(*declared.Medium)
	if err != nil {
		return fail("%w", err)
	}
	if declared.Year != 0 && declared.Year <= earliestPlausibleYear {
		return fail("is dated %d, which is not a plausible year; leave it out if it is unknown", declared.Year)
	}
	return Origin{
		ID: declared.ID, Title: declared.Title, Medium: medium,
		Year: declared.Year, Note: declared.Note,
	}, nil
}

// Get returns an origin by id.
func (b *OriginBook) Get(id string) (Origin, bool) {
	found, ok := b.byID[id]
	return found, ok
}

// All returns every origin in declaration order. It never ranges over the
// lookup map, because Go randomises that and the order reaches an output.
func (b *OriginBook) All() []Origin {
	out := make([]Origin, len(b.origins))
	copy(out, b.origins)
	return out
}

// Marshal writes the catalog as a data file: two-space indented JSON with the
// origins sorted by id, for the same reason Book.Marshal sorts.
func (b *OriginBook) Marshal() ([]byte, error) {
	sorted := b.All()
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	out, err := json.MarshalIndent(originMarshalFile{Origins: sorted}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode origin book: %w", err)
	}
	return append(out, '\n'), nil
}

// Append returns a new catalog holding the existing origins plus the extra
// ones, validated exactly as a parse would validate them.
func (b *OriginBook) Append(extra ...Origin) (*OriginBook, error) {
	combined := originMarshalFile{Origins: append(b.All(), extra...)}
	raw, err := json.Marshal(combined)
	if err != nil {
		return nil, fmt.Errorf("encode origin book: %w", err)
	}
	return ParseOrigins(raw)
}

package i18n

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/vukyn/hexarena/internal/core/element"
	"github.com/vukyn/hexarena/internal/core/progression"
	"github.com/vukyn/hexarena/internal/forge"
)

// The catalog's own guarantees are what this file holds: that both languages
// are complete, that neither has an entry nobody asks for, that a line means
// the same thing in both, and that a Vietnamese letter takes one terminal cell
// like a Latin one. A gap in any of those shows up on somebody's screen as a
// blank line, a %!s(MISSING) or a column that has drifted a character to the
// right, and all three are the kind of thing nobody reports.

// TestEveryKeyIsWordedInEveryLanguage is the completeness rule.
//
// An empty entry is a failure rather than a fall back to English, and the
// reason is worth stating: the person a missing Vietnamese line lands on is
// exactly the person least able to tell it was supposed to be Vietnamese. A
// loud failure here is cheaper than a quiet one there.
func TestEveryKeyIsWordedInEveryLanguage(t *testing.T) {
	for _, key := range Keys() {
		for _, lang := range Langs() {
			if strings.TrimSpace(lang.Text(key)) == "" {
				t.Errorf("key %d has no %s wording", key, lang)
			}
		}
	}
	if got := len(Keys()); got != int(keyCount) {
		t.Errorf("Keys returned %d entries, want %d", got, keyCount)
	}
}

// TestTheSameBlanksInEveryLanguage catches the mistake a completeness check
// alone would miss: a wording that is present but takes a different number of
// values than its counterpart. A caller passes one set of arguments for both,
// so a mismatch renders as %!d(MISSING) in one language and as nothing at all
// in the other.
func TestTheSameBlanksInEveryLanguage(t *testing.T) {
	for _, key := range Keys() {
		want := verbs(Default.Text(key))
		for _, lang := range Langs() {
			if lang == Default {
				continue
			}
			got := verbs(lang.Text(key))
			if len(got) != len(want) {
				t.Errorf("key %d takes %v in %s and %v in %s", key, want, Default, got, lang)
				continue
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("key %d's blank %d is %q in %s and %q in %s",
						key, i, want[i], Default, got[i], lang)
				}
			}
		}
	}
}

// verbs pulls the formatting blanks out of a wording, keeping their flags: %2d
// and %d are not interchangeable in a column.
func verbs(text string) []string {
	var found []string
	for i := 0; i < len(text); i++ {
		if text[i] != '%' {
			continue
		}
		if i+1 < len(text) && text[i+1] == '%' {
			i++
			continue
		}
		end := i + 1
		for end < len(text) && !unicode.IsLetter(rune(text[end])) {
			end++
		}
		if end < len(text) {
			found = append(found, text[i:end+1])
			i = end
		}
	}
	return found
}

// TestEveryWordingIsOneCellPerLetter is the layout measurement, made against
// the same width calculation the screens are clipped and padded with.
//
// Vietnamese is written here in its composed form, so ế is one rune and one
// cell. Written decomposed — an e followed by a combining acute — it would be
// two runes, the combining one measuring zero, and every fixed-width column on
// a screen holding it would drift by a character. That is invisible in a diff
// and obvious on a terminal, which is the wrong way round, so it is checked
// here instead.
func TestEveryWordingIsOneCellPerLetter(t *testing.T) {
	for _, key := range Keys() {
		for _, lang := range Langs() {
			for _, line := range strings.Split(lang.Text(key), "\n") {
				if got, want := lipgloss.Width(line), utf8.RuneCountInString(line); got != want {
					t.Errorf("key %d in %s measures %d cells over %d runes: %q",
						key, lang, got, want, line)
				}
				for _, letter := range line {
					if unicode.Is(unicode.Mn, letter) {
						t.Errorf("key %d in %s holds the combining mark %U; "+
							"write Vietnamese composed", key, lang, letter)
					}
				}
			}
		}
	}
}

// TestNoKeyIsOrphaned fails on a wording nobody can ever see.
//
// A key that no screen asks for is two translations to keep up to date for no
// reader, and it is the residue a reworked screen leaves behind. The scan is
// over the module's own source: every constant declared in keys.go has to be
// named somewhere other than its own declaration.
func TestNoKeyIsOrphaned(t *testing.T) {
	declared := declaredKeys(t)
	if len(declared) != int(keyCount) {
		t.Fatalf("keys.go declares %d keys, but keyCount is %d", len(declared), keyCount)
	}
	used := usedIdentifiers(t)
	for _, name := range declared {
		if !used[name] {
			t.Errorf("nothing outside its declaration asks for %s", name)
		}
	}
}

// declaredKeys reads the key names out of keys.go rather than a hand-kept list,
// so adding a key to the const block is the whole of adding a key.
func declaredKeys(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "keys.go", nil, 0)
	if err != nil {
		t.Fatalf("parse keys.go: %v", err)
	}
	var names []string
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			for _, name := range value.Names {
				if name.Name == "keyCount" || name.Name == "_" {
					continue
				}
				names = append(names, name.Name)
			}
		}
	}
	return names
}

// usedIdentifiers is every identifier the module mentions, apart from the key
// declarations and the two catalogs.
//
// The catalogs have to be skipped or this proves nothing: every key is named in
// both of them by construction, so counting those as uses would let an orphan
// through with two translations attached.
func usedIdentifiers(t *testing.T) map[string]bool {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("find the module root: %v", err)
	}
	declarations := make(map[string]bool)
	for _, name := range []string{"keys.go", "vietnamese.go", "english.go"} {
		absolute, err := filepath.Abs(name)
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		declarations[absolute] = true
	}
	used := make(map[string]bool)
	walk := func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || declarations[path] {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if name, isName := node.(*ast.Ident); isName {
				used[name.Name] = true
			}
			return true
		})
		return nil
	}
	if err := filepath.WalkDir(root, walk); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return used
}

// TestALanguageIsParsedByName covers the enum's own contract, which is the one
// every other enum in this module keeps: names are the wire format, and an
// unknown one is an error.
func TestALanguageIsParsedByName(t *testing.T) {
	for _, lang := range Langs() {
		parsed, err := Parse(lang.String())
		if err != nil {
			t.Errorf("parse %q: %v", lang, err)
		}
		if parsed != lang {
			t.Errorf("%q parsed to %q", lang, parsed)
		}
	}
	if Default != Vi {
		t.Errorf("the default is %q, want Vietnamese", Default)
	}
	if Vi.Other() != En || En.Other() != Vi {
		t.Error("the toggle does not swap the two languages")
	}

	// A near miss names both spellings that work, in both languages, because
	// this is the one refusal raised before the language is known.
	_, err := Parse("vn")
	if err == nil {
		t.Fatal("\"vn\" was accepted as a language")
	}
	for _, want := range []string{"vn", "vi", "en", "không rõ ngôn ngữ", "unknown language"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not mention %q", err, want)
		}
	}
}

// TestTheFlagBeatsTheEnvironment is the precedence rule: a variable is a
// standing preference, a flag is this run.
func TestTheFlagBeatsTheEnvironment(t *testing.T) {
	cases := []struct {
		flag, environment string
		want              Lang
	}{
		{"", "", Vi},
		{"en", "", En},
		{"", "en", En},
		{"vi", "en", Vi},
		{"en", "vi", En},
	}
	for _, test := range cases {
		got, err := Resolve(test.flag, test.environment)
		if err != nil {
			t.Errorf("Resolve(%q, %q): %v", test.flag, test.environment, err)
			continue
		}
		if got != test.want {
			t.Errorf("Resolve(%q, %q) is %q, want %q", test.flag, test.environment, got, test.want)
		}
	}

	// Either source being unreadable names where it came from, so a typo in a
	// shell profile does not read as a typo on the command line.
	if _, err := Resolve("nope", ""); err == nil || !strings.Contains(err.Error(), "--"+FlagName) {
		t.Errorf("a bad flag gave %v, want it to name the flag", err)
	}
	if _, err := Resolve("", "nope"); err == nil || !strings.Contains(err.Error(), EnvVar) {
		t.Errorf("a bad environment value gave %v, want it to name the variable", err)
	}
	// A bad flag is still an error when the environment holds something usable:
	// silently taking the other one would hide the typo.
	if _, err := Resolve("nope", "en"); err == nil {
		t.Error("a bad flag was excused by a good environment value")
	}
	if got := Prefer("nope", ""); got != Default {
		t.Errorf("Prefer fell back to %q, want the default", got)
	}
}

// TestARefusalIsWordedFromItsFacts is the point of the typed errors: the same
// value becomes a different sentence in each language, and neither language
// takes the other apart to get there.
func TestARefusalIsWordedFromItsFacts(t *testing.T) {
	fire, err := element.Single(element.Fire)
	if err != nil {
		t.Fatalf("the fire affinity: %v", err)
	}
	carry := &forge.CarryError{Affinity: fire, Skill: "sever", Element: element.Metal}

	if got, want := Vi.Error(carry), `hệ fire không mang được chiêu "sever" (hệ metal)`; got != want {
		t.Errorf("the carry refusal reads\n %q\nwant\n %q", got, want)
	}
	if got, want := En.Error(carry), `fire cannot carry the skill "sever", which is metal`; got != want {
		t.Errorf("the carry refusal reads\n %q\nwant\n %q", got, want)
	}
	// The command line's wording is a third thing, and unchanged: a script that
	// grepped it before still finds it.
	if got, want := carry.Error(), `fire cannot carry "sever", which is metal`; got != want {
		t.Errorf("the command line's wording is %q, want %q", got, want)
	}

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"an id already taken", &forge.IDTakenError{ID: "example-film.tester"},
			`nhân vật "example-film.tester" đã có trong danh sách rồi`},
		{"a work that is not catalogued", &forge.UnknownOriginError{ID: "nowhere"},
			`không có nguồn "nowhere"; thêm bằng lệnh "hexforge origins add nowhere"`},
		{"a skill named twice", &forge.DuplicateSkillError{ID: "strike"},
			`chiêu "strike" bị ghi hai lần`},
		{"an answer that is not a curve", &forge.CurveShapeError{Raw: "120"},
			`"120" chưa đúng dạng; viết theo kiểu base:max`},
		{"a curve that shrinks", &forge.CurveRefusedError{
			Kind:   progression.HP,
			Curve:  progression.Curve{Base: 900, Max: 400},
			Reason: forge.CurveReasonShrinks,
			Err:    errors.New("unused"),
		}, "hp kết thúc ở 400 nhưng bắt đầu từ 900; chỉ số không tụt khi lên cấp"},
		{"a stat naming its own refusal", &forge.StatFieldError{
			Kind: progression.Attack, Err: &forge.CurveShapeError{Raw: "x"},
		}, `atk: "x" chưa đúng dạng; viết theo kiểu base:max`},
		{"an element that does not exist", &forge.UnknownElementError{
			Name: "flame", Err: errors.New("unused"),
		}, `không có hệ nào tên "flame"`},
	}
	for _, test := range cases {
		if got := Vi.Error(test.err); got != test.want {
			t.Errorf("%s reads\n %q\nwant\n %q", test.name, got, test.want)
		}
	}

	// Anything internal/core refused about the shape of a value keeps that
	// package's own English, behind a lead-in in the reader's language.
	fromTheParser := &forge.FieldRefusedError{
		Field: forge.FieldID, Value: "Ab", Err: errors.New("character id is empty"),
	}
	if got, want := Vi.Error(fromTheParser),
		"id này không dùng được: character id is empty"; got != want {
		t.Errorf("a parser's refusal reads %q, want %q", got, want)
	}
	if got := Vi.Error(nil); got != "" {
		t.Errorf("no error rendered as %q", got)
	}
}

// TestAWriteIsReportedInBothLanguages covers the notes a save leaves behind,
// which are the other thing internal/forge hands over as facts.
func TestAWriteIsReportedInBothLanguages(t *testing.T) {
	notes := []forge.Note{
		{Kind: forge.NoteWrote, ID: "example-film.tester", Path: "data/cast.json"},
		{Kind: forge.NoteArtMissing, Path: "data/assets/tester.svg"},
		{Kind: forge.NoteRebuild},
	}
	vietnamese := Vi.Notes(notes)
	if got, want := vietnamese[0], "đã ghi example-film.tester vào data/cast.json"; got != want {
		t.Errorf("the write reads %q, want %q", got, want)
	}
	if !strings.Contains(vietnamese[1], "data/assets/tester.svg") {
		t.Errorf("the warning does not name the art: %q", vietnamese[1])
	}
	english := En.Notes(notes)
	if len(english) != len(vietnamese) {
		t.Fatalf("%d notes in English against %d in Vietnamese", len(english), len(vietnamese))
	}
	for i := range english {
		if english[i] == vietnamese[i] {
			t.Errorf("note %d is the same line in both languages: %q", i, english[i])
		}
	}
}

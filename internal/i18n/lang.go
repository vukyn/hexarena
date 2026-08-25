// Package i18n is the wording of the full-screen authoring client, in
// Vietnamese and in English.
//
// # What is in here and what is not
//
// cmd/hexforge-tui holds no user-visible sentence of its own: it decides where
// something goes on screen and asks this package what it says. cmd/hexforge is
// deliberately not a caller — it is what a script and a pipe read, so its
// output stays English and is not routed through a catalog that could change
// under it.
//
// The facts stay in internal/forge. A refused kit, a spent budget, an
// evolution line and a write's confirmation all arrive here as values, never as
// sentences, because a sentence cannot be translated after it has been built
// without taking it apart again. That is why forge.CheckCarry returns a
// *forge.CarryError with the affinity, the skill and the skill's element on it
// rather than a formatted line.
//
// This package therefore depends on internal/forge, and forge must never depend
// on it: there is exactly one place that knows a rule, and two places that know
// how to say so.
//
// # What is deliberately not translated
//
// Element ids (fire, metal, water), skill ids, character ids, origin ids and
// the six stat labels hp atk def spd acc ddg stay as they are in both
// languages. They are what an author types, what --element takes and what the
// data files store; a translated column heading would be a value nobody can
// match back to the file they are editing. See forge.ShortStat for the same
// note on the stat labels.
//
// Diagnostics from internal/core stay English too, and get a lead-in in the
// author's language instead. They describe the shape of a data file, and a
// second rendering of them here would be a second declaration of a rule this
// module keeps in exactly one place.
package i18n

import "fmt"

// Lang is a language the client can be read in.
//
// Like every other enum in this module it is parsed and written by name rather
// than by number, and an unknown name is an error rather than a silent fall
// back to a default: a person who typed "vn" is better served by being told the
// two spellings that work than by a screen that quietly ignores them.
type Lang int

const (
	// Vi is Vietnamese, and the default.
	Vi Lang = iota
	// En is English.
	En
	langCount
)

// Default is what an author gets without asking. Vietnamese: this client is
// authored and read here, and English is one flag away.
const Default = Vi

// EnvVar is the environment variable a language may be set in, for a shell that
// wants it every time without a flag on every invocation.
const EnvVar = "HEXARENA_LANG"

// FlagName is the flag that overrides it, named here so a message about a bad
// value and the flag itself cannot disagree.
const FlagName = "lang"

var langNames = [langCount]string{Vi: "vi", En: "en"}

func (l Lang) String() string {
	if !l.Valid() {
		return fmt.Sprintf("lang(%d)", int(l))
	}
	return langNames[l]
}

// Valid reports whether the value is one of the declared languages.
func (l Lang) Valid() bool { return l >= 0 && l < langCount }

// Other is the language the toggle switches to. With two languages that is the
// one that is not in front; a third would make this a rotation.
func (l Lang) Other() Lang {
	if l == Vi {
		return En
	}
	return Vi
}

// Langs returns every declared language in declaration order.
func Langs() []Lang {
	out := make([]Lang, 0, langCount)
	for i := range langCount {
		out = append(out, Lang(i))
	}
	return out
}

// Names returns the accepted spellings, which is what a refusal has to list.
func Names() []string {
	out := make([]string, 0, langCount)
	for _, lang := range Langs() {
		out = append(out, lang.String())
	}
	return out
}

// Parse reads a language by name.
//
// The refusal is the one message in this package that is written in both
// languages at once, and that is not an oversight: it is raised precisely when
// the language is not yet known, so choosing either one would be a guess about
// the person who just mistyped it.
func Parse(name string) (Lang, error) {
	for i, candidate := range langNames {
		if candidate == name {
			return Lang(i), nil
		}
	}
	accepted := ""
	for i, candidate := range langNames {
		if i > 0 {
			accepted += " "
		}
		accepted += candidate
	}
	return Default, fmt.Errorf(
		"không rõ ngôn ngữ %q, chọn một trong: %s / unknown language %q, want one of: %s",
		name, accepted, name, accepted)
}

// Resolve picks the language for a run: the flag if it was given, otherwise the
// environment, otherwise the default.
//
// The flag wins because it is the more specific of the two — an environment
// variable is a standing preference and a flag is this invocation. Either one
// being unreadable is an error naming where it came from, so a typo in a shell
// profile does not look like a typo on the command line.
func Resolve(flagValue, environment string) (Lang, error) {
	switch {
	case flagValue != "":
		lang, err := Parse(flagValue)
		if err != nil {
			return Default, fmt.Errorf("--%s: %w", FlagName, err)
		}
		return lang, nil
	case environment != "":
		lang, err := Parse(environment)
		if err != nil {
			return Default, fmt.Errorf("%s: %w", EnvVar, err)
		}
		return lang, nil
	default:
		return Default, nil
	}
}

// Prefer is Resolve without the refusal, for the one thing that has to be
// worded before a refusal can be raised: the flag descriptions the flag package
// prints while it is still parsing. An unusable value falls back to the default
// here and is reported properly by Resolve a moment later.
func Prefer(flagValue, environment string) Lang {
	lang, err := Resolve(flagValue, environment)
	if err != nil {
		return Default
	}
	return lang
}

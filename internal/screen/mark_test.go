package screen

import "testing"

// TestATraitsStatusNamesAreMarkedWordByWord is the marking, and the word by word
// part is the whole of it.
//
// The sentences are wrapped after they are marked, and the wrap splits on
// spaces. A style spanning "kiên cường" survives that only until the two words
// land on different lines, and then the first line opens a sequence the second
// one closes — so every word is marked whole instead.
func TestATraitsStatusNamesAreMarkedWordByWord(t *testing.T) {
	got := Marked("Luôn mang kiên cường.", []string{"kiên cường"},
		func(word string) string { return "<" + word + ">" })
	if want := "Luôn mang <kiên> <cường>."; got != want {
		t.Errorf("Marked %q, want %q", got, want)
	}
}

// TestAMarkedNameNeverTakesAnotherNamesOpening is the trap a replacement per
// name falls into whichever order the names are tried in.
//
// Longest first and "bỏng" matches inside the output "bỏng nặng" just produced;
// shortest first and "bỏng nặng" never matches at all. Only one left-to-right
// pass, taking the longest name that starts at each position, gets both right -
// and the first version of this marked the first word twice.
func TestAMarkedNameNeverTakesAnotherNamesOpening(t *testing.T) {
	got := Marked("bỏng nặng và bỏng.", []string{"bỏng", "bỏng nặng"},
		func(word string) string { return "<" + word + ">" })
	if want := "<bỏng> <nặng> và <bỏng>."; got != want {
		t.Errorf("Marked %q, want %q — one pass, longest match wins", got, want)
	}
}

// TestNothingIsMarkedInASentenceThatNamesNothing keeps the marking from being a
// search-and-replace over prose: only the names the trait actually declares are
// looked for, so a flavour clause carrying a status name by coincidence is left
// alone unless that trait names it.
func TestNothingIsMarkedInASentenceThatNamesNothing(t *testing.T) {
	sentence := "Chịu đòn quen rồi, đau tới đâu cũng đứng vững tới đó."
	if got := Marked(sentence, nil, func(word string) string { return "<" + word + ">" }); got != sentence {
		t.Errorf("Marked %q with no names to mark", got)
	}
}

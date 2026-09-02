package wire

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// TestARoomPasswordIsNeverPrinted measures the half of the design record's
// promise that can be measured.
//
// The record owes a password two things: a constant-time comparison and never
// being logged. **Constant-time-ness is not directly testable** — a timing
// assertion on a test runner is a flaky test wearing a security claim — so what
// is held here is that the comparison goes through crypto/subtle (read off the
// source, below) and that the secret is absent from a formatted struct, which is
// what a log line actually is.
//
// It walks the message bodies by **reflection** rather than naming Hello, so a
// second message that grows a Password field is covered the day it is written
// rather than the day somebody remembers this test exists.
func TestARoomPasswordIsNeverPrinted(t *testing.T) {
	const secret = fixturePassword
	holders := 0
	passwordType := reflect.TypeFor[Password]()
	for kind, fixture := range messageFixtures(t) {
		structure := reflect.TypeOf(fixture).Elem()
		fields := make([]int, 0, 1)
		for index := range structure.NumField() {
			if structure.Field(index).Type == passwordType {
				fields = append(fields, index)
			}
		}
		if len(fields) == 0 {
			continue
		}
		holders++
		t.Run(kind.String(), func(t *testing.T) {
			// Set every password field to the secret, so a message that holds
			// two is measured on both.
			value := reflect.New(structure).Elem()
			value.Set(reflect.ValueOf(fixture).Elem())
			for _, index := range fields {
				value.Field(index).Set(reflect.ValueOf(Password(secret)))
			}
			held := value.Addr().Interface()
			for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
				line := fmt.Sprintf(format, held)
				if strings.Contains(line, secret) {
					t.Errorf("%s of a %s prints the password: %s", format, kind, line)
				}
			}
		})
	}
	// Non-vacuity: a walk that found no password field would pass this test
	// whether or not a password could be printed, which is the shape of pass
	// this repository has recorded several times.
	if holders == 0 {
		t.Fatal("no message body holds a Password, so this test measured nothing")
	}
	// And the type itself, which is what a caller logging one field reaches.
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		if line := fmt.Sprintf(format, Password(secret)); strings.Contains(line, secret) {
			t.Errorf("%s of a Password prints it: %s", format, line)
		}
	}
	// The redaction still says the one thing that is safe to say, or a room with
	// no password and a room with one would read identically in a log.
	if Password("").String() == Password(secret).String() {
		t.Error("a room with no password and a room with one redact to the same words")
	}
	t.Logf("%d of %d message bodies hold a password", holders, KindCount)
}

// TestARoomPasswordIsComparedInConstantTime holds the half that cannot be timed,
// by reading the source: the comparison goes through crypto/subtle and never
// through ==.
//
// ⚠️ This is a weaker test than it looks and is written down as such. It proves
// the call is *there*, not that no caller compares two Passwords with == of its
// own — nothing here can prove that, since == on a named string type is legal Go
// and there is no room, no server and no handler yet to check. What it does catch
// is the tidy-up: somebody simplifying Equal into a string comparison because the
// password "is not security anyway", which is true and is not the point.
func TestARoomPasswordIsComparedInConstantTime(t *testing.T) {
	source := readPackageSource(t)
	if !strings.Contains(source, "subtle.ConstantTimeCompare") {
		t.Error("nothing in this package calls subtle.ConstantTimeCompare, so Password.Equal is not the comparison the record promises")
	}
	// And it answers correctly, which a constant-time comparison still has to.
	cases := []struct {
		left, right Password
		want        bool
	}{
		{fixturePassword, fixturePassword, true},
		{fixturePassword, fixturePassword + "x", false},
		{fixturePassword, Password(fixturePassword[:len(fixturePassword)-1]), false},
		{fixturePassword, "", false},
		{"", "", true},
		{"a", "b", false},
	}
	for _, one := range cases {
		if got := one.left.Equal(one.right); got != one.want {
			t.Errorf("%q equals %q reported %v", one.left.String(), one.right.String(), got)
		}
	}
	if Password("").Set() {
		t.Error("an empty password reports itself as set")
	}
	if !Password(fixturePassword).Set() {
		t.Error("a password reports itself as unset")
	}
}

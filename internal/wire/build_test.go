package wire

import (
	"runtime/debug"
	"testing"
)

// TestTheBuildStringFallsBackInOneOrder holds the three sources and the order
// they are trusted in.
//
// ⚠️ It drives the **pure** function and not the linker, which is the whole
// reason BuildOf takes its two inputs as parameters: a test that asserted on a
// real -ldflags stamp would need to shell out to `go build`, and one that
// asserted on this process's own build info would assert a different thing under
// `go test` than under `go run`. Neither would be measuring the ordering, which
// is the only decision here.
//
// ⚠️ **What it cannot see** is that a binary actually calls this rather than
// spelling the fallback out again — that is held by there being no second copy
// of it in the module, which is why the function moved here in the first place.
func TestTheBuildStringFallsBackInOneOrder(t *testing.T) {
	stamped := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.modified", Value: "false"},
	}}
	dirty := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
		{Key: "vcs.modified", Value: "true"},
	}}
	silent := &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "GOARCH", Value: "arm64"}}}

	for _, each := range []struct {
		what    string
		stamp   string
		info    *debug.BuildInfo
		want    string
		because string
	}{
		{
			what: "a release, stamped by -ldflags", stamp: "v0.4.0", info: stamped, want: "v0.4.0",
			because: "a stamp is a deliberate act by whoever produced the binary, so nothing may override it",
		},
		{
			what: "a stamped binary built from a dirty tree", stamp: "v0.4.0", info: dirty, want: "v0.4.0",
			because: "the stamp still wins; the person who stamped it said what it is",
		},
		{
			what: "go build with no stamp", stamp: "", info: stamped, want: "0123456789ab",
			because: "the revision is what a person compares across two machines, abbreviated to twelve like the data digest beside it",
		},
		{
			what: "go build from a dirty tree", stamp: "", info: dirty, want: "0123456789ab+dirty",
			because: "two people comparing builds are usually comparing a release against a working copy, and the working copy has to say so",
		},
		{
			what: "go run, which records no revision", stamp: "", info: silent, want: Unstamped,
			because: "the ordinary case during development, so it is a word rather than an empty string beside the digest",
		},
		{
			what: "a binary with no build info at all", stamp: "", info: nil, want: Unstamped,
			because: "a blank on that line reads as a bug in the printing rather than a fact about the binary",
		},
	} {
		if got := BuildOf(each.stamp, each.info); got != each.want {
			t.Errorf("%s: build string %q, want %q — %s", each.what, got, each.want, each.because)
		}
	}
}

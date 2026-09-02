package seed

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"
)

// Digest is the fingerprint of the embedded data. It answers exactly one
// question, at a room's gate: **will these two binaries simulate the same
// battle?**
//
// It is not a version number. Nothing about it is meant to be stable across
// commits — this repository ships data changes constantly, and every one of them
// is *supposed* to move the digest, because every one of them changes the
// battles. The only property that matters is equality between two peers built
// from the same data.
type Digest [32]byte

// String is the whole digest, sixty-four lowercase hex characters. This is the
// form that travels and the form that gets compared.
func (d Digest) String() string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 0, len(d)*2)
	for _, b := range d {
		out = append(out, hexDigits[b>>4], hexDigits[b&0x0f])
	}
	return string(out)
}

// Short is the first twelve hex characters, for a human reading two screens side
// by side.
//
// ⚠️ Never compare on this. Twelve hex characters is forty-eight bits, which is
// plenty for a person telling two builds apart at a glance and nowhere near
// enough to be the check that decides whether a battle is joinable. The gate
// compares String; Short exists so that a mismatch can be *reported* in
// something a person can read out loud.
func (d Digest) Short() string { return d.String()[:12] }

// dataFiles names the fifteen embedded data files in the order the go:embed
// directive in seed.go declares them.
//
// The order is part of the digest (see digest), so this list is not sorted and
// must not be: it mirrors the directive, and the directive is what the design
// record names as the order. It is also the third independent copy of these
// fifteen names in the package — the directive is the first, the fifteen
// ReadFile calls are the second — which is why digest_test.go walks the embedded
// FS in both directions rather than trusting any one of the three.
var dataFiles = []string{
	"data/elements.json",
	"data/combat.json",
	"data/progression.json",
	"data/modifiers.json",
	"data/patterns.json",
	"data/statuses.json",
	"data/passives.json",
	"data/skills.json",
	"data/origins.json",
	"data/species.json",
	"data/archetypes.json",
	"data/cast.json",
	"data/roster.json",
	"data/builds.json",
	"data/squads.json",
}

// DataDigest is the digest of the fifteen embedded data files.
//
// Art is not in it. assets/ cannot reach the simulation, so a peer with a newer
// picture is not a peer that fights a different battle, and refusing it at the
// gate would refuse a match that would have been identical.
func DataDigest() (Digest, error) { return digest(files, dataFiles) }

// digest hashes the named files out of fsys, in the order given, feeding sha256
// three things per file: the **name**, the **byte length**, then the **bytes**.
//
// The name and the length are the whole reason this is not a one-liner: a plain
// sha256 over the concatenated bytes cannot say **which file** any of those bytes
// came out of, only that the run of them is the same. Framing each file closes
// that, costs nothing, and touches nothing of the record's rule that the digest
// never parses the data — a name is not in the data at all and a length is a
// property of the bytes rather than a reading of them.
//
// ⚠️ The obvious justification for framing — **two files exchanging contents** —
// is not the one, and it was believed here before it was measured. The design
// record asks only that the bytes be hashed unparsed and names no failure case
// at all; the swap is the case that suggests itself, and it is wrong. Measured
// (see TestTheNameAndLengthFrameEveryFileInTheDigest): the files are read in list
// order, so a swap moves those bytes to different offsets and a content-only
// hash sees it too. What a
// content-only hash is genuinely blind to is a **boundary that moved**: two
// adjacent files holding the same total bytes split differently, where the
// concatenation is identical and only the framing can tell the two apart. It is
// also blind to a **rename**, since the same bytes under another name concatenate
// to the same stream.
//
// The two halves of the framing are not equally load-bearing, and that was
// measured one at a time rather than assumed:
//
//   - The **name** is what does nearly all of it. It catches the rename, and
//     because it sits between two files' bytes it doubles as a separator, so it
//     catches an ordinary moved boundary as well.
//   - The **length** is what makes the framing unambiguous *unconditionally*
//     rather than "unless a file's bytes happen to contain another file's name".
//     With the length dropped, a boundary can move and the stream stay
//     byte-identical — a file whose contents hold the next name is the case, and
//     it is the only assertion here that reddens for the length alone. The gate
//     compares peers rather than adversaries, so this is defence in depth; it
//     costs eight bytes a file and it is the difference between a framing that
//     cannot be confused and one that is merely hard to confuse.
//
// The length is big-endian on a fixed-width uint64 rather than a formatted
// number, so the encoding cannot drift with a locale, a width, or a change of
// mind about how to print an integer.
//
// A missing or unreadable file is an error and never a partial digest. Two peers
// agreeing on a partial read is the worst outcome this gate can produce: they
// would both be confident, and only one of them would be right about the fight.
func digest(fsys fs.FS, names []string) (Digest, error) {
	hasher := sha256.New()
	var length [8]byte
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return Digest{}, fmt.Errorf("digest %s: %w", name, err)
		}
		hasher.Write([]byte(name))
		binary.BigEndian.PutUint64(length[:], uint64(len(raw)))
		hasher.Write(length[:])
		hasher.Write(raw)
	}
	var out Digest
	copy(out[:], hasher.Sum(nil))
	return out, nil
}

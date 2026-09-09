package wire

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/hex"
)

// TestTheLargestFormatIsTheSquadCap holds hex.MaxSquadSize in step with the
// formats, from the side that knows what a format is.
//
// ⚠️ **The check has to live here and cannot live in hex.** internal/core/hex is
// the leaf: it declares how many units a side is ever fielded with, because four
// packages need that number and it is the only place all four can see. It does
// not and must not know what a format is — the formats are declared in this
// package, which imports hex, so the correspondence can only be checked from
// here. It is the same shape skill/cast already use for a name check: declare
// what the package can see, verify the agreement one layer up.
//
// **Both directions, because each fails differently.**
//
//   - A format above the cap is a room a host can open and placement.Squad,
//     the squad builder, the draft and the room gate would all refuse to fill.
//   - A cap above every format bounds nothing. It would sit there looking like a
//     rule while no squad in the game could reach it, and composition's rung
//     ceiling — which reads the same constant — would declare rungs legal that no
//     side can ever satisfy. That is the vacuous row DAT-002's decision 6 refuses.
//
// ⚠️ **Walked as an explicit slice, not a map and not a range over the values
// between the two.** A map's order is not stable and would reach an output here
// (the failure message); a numeric range would silently start passing for a
// format that had been deleted. A format added to this package has to be added
// to this list, and that is the point.
func TestTheLargestFormatIsTheSquadCap(t *testing.T) {
	formats := []Format{Format3v3, Format5v5}
	largest := 0
	for _, format := range formats {
		if !format.Valid() {
			t.Errorf("%s is listed here and the protocol does not offer it", format)
		}
		if format.Units() > hex.MaxSquadSize {
			t.Errorf("%s fields %d units against a squad cap of %d: a host could open a "+
				"room no squad may be built for", format, format.Units(), hex.MaxSquadSize)
		}
		if format.Units() > largest {
			largest = format.Units()
		}
	}
	if largest != hex.MaxSquadSize {
		t.Errorf("the largest format fields %d units and hex.MaxSquadSize is %d: a cap "+
			"above every format bounds nothing, and composition's rung ceiling reads it",
			largest, hex.MaxSquadSize)
	}
}

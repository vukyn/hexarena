// Package scale holds the proportional arithmetic the whole engine shares.
//
// Every ratio in the game — a skill's power, an elemental multiplier, a buff, a
// hit chance — is an integer in parts per thousand against Base. Keeping one
// denominator means a value can be handed from one system to another without a
// conversion step, and a mismatch cannot silently scale a number by a thousand.
package scale

// Base is the denominator every proportional value is expressed against.
const Base = 1000

// Apply returns value scaled by a proportional factor.
func Apply(value, permille int64) int64 { return value * permille / Base }

// Saturate applies a change to a base value with diminishing returns, so the
// result approaches a limit without ever reaching it.
//
//	result = base + gap * delta / (gap + delta)
//
// The gap is the distance from base to the limit in the direction of the
// change. A small delta relative to that gap is worth almost its face value; a
// delta the size of the gap covers exactly half of it; an unbounded delta
// converges on the limit. Because saturation depends only on the ratio of delta
// to gap, stats of completely different magnitudes saturate at the same rate.
//
// A hard clamp would do the same job at the edges and nothing in between, which
// is what makes stacking buffs feel either free or worthless with no middle
// ground. This way each additional buff is worth measurably less than the last,
// and no stack of them can push a stat past its limit.
func Saturate(base, delta, ceiling, floor int64) int64 {
	switch {
	case delta > 0:
		gap := ceiling - base
		if gap <= 0 {
			return base
		}
		return base + gap*delta/(gap+delta)
	case delta < 0:
		gap := base - floor
		if gap <= 0 {
			return base
		}
		penalty := -delta
		return base - gap*penalty/(gap+penalty)
	}
	return base
}

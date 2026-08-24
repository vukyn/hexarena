// Package rng is the only source of randomness the engine uses.
//
// Nothing here reads a clock or a global generator. A Source is created from an
// explicit seed, its whole state is one integer, and every draw is integer
// arithmetic, so a battle replayed from the same seed and the same inputs
// produces the same rolls on every machine. That is what makes a battle log a
// complete record of a fight rather than a summary of one.
//
// The generator is splitmix64: small enough to keep the state recoverable, and
// well distributed enough for hit rolls and damage variance.
package rng

import "fmt"

// Source is a deterministic pseudo-random generator. It is not safe for
// concurrent use, which is deliberate: two goroutines drawing from one battle's
// source would make the battle unreproducible.
type Source struct {
	state uint64
}

// New creates a source from a seed.
func New(seed uint64) *Source { return &Source{state: seed} }

// State returns the generator's whole state, so a battle can be resumed or a
// roll can be traced back.
func (s *Source) State() uint64 { return s.state }

// Restore sets the generator back to a previously captured state.
func (s *Source) Restore(state uint64) { s.state = state }

// Clone returns an independent source at the same position, for speculative
// work that must not disturb the battle's own sequence.
func (s *Source) Clone() *Source { return &Source{state: s.state} }

// Next returns the next raw draw and advances the state.
func (s *Source) Next() uint64 {
	s.state += 0x9E3779B97F4A7C15
	z := s.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Intn returns a value in [0, n). It panics for a non-positive bound, because
// asking for a draw from an empty range is a programming error rather than a
// runtime condition.
//
// The draw is rejection sampled rather than reduced with a modulo, so every
// value in the range is equally likely. A modulo would quietly favour the low
// end of any range that does not divide the generator's period, which for a hit
// roll means a bias nobody would notice until the numbers were audited.
func (s *Source) Intn(n int) int {
	if n <= 0 {
		panic(fmt.Sprintf("rng: Intn called with a bound of %d", n))
	}
	bound := uint64(n)
	limit := ^uint64(0) - (^uint64(0) % bound) - 1
	for {
		draw := s.Next()
		if draw <= limit {
			return int(draw % bound)
		}
	}
}

// Roll returns a value in [1, sides].
func (s *Source) Roll(sides int) int { return s.Intn(sides) + 1 }

// Chance reports whether an event of the given probability happens, where the
// probability is in parts per thousand. A chance of zero never happens and a
// chance of a thousand or more always does.
func (s *Source) Chance(permille int) bool {
	if permille <= 0 {
		return false
	}
	if permille >= 1000 {
		return true
	}
	return s.Intn(1000) < permille
}

// Between returns a value in [low, high]. It panics when high is below low.
func (s *Source) Between(low, high int) int {
	if high < low {
		panic(fmt.Sprintf("rng: Between called with %d..%d", low, high))
	}
	return low + s.Intn(high-low+1)
}

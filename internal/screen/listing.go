package screen

// The two things every listing in this package is built out of: where the
// cursor is allowed to be, and which slice of the rows a window that size can
// hold. Both moved here from cmd/hexforge-tui with the screens that call them,
// and both are still called from there — a listing that has not moved yet asks
// for the same answer, and two answers to "where does the cursor stop" is the
// kind of drift this package exists to prevent.

// Clamp keeps an index or a level inside its range, and returns the low bound
// when the range is empty.
func Clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// Window is the slice of a list to draw, keeping the cursor inside it.
//
// It scrolls by the least it can: the window only moves when the cursor would
// leave it, so stepping back and forth across one boundary does not make the
// whole list jump about.
func Window(total, cursor, room int) (from, to int) {
	if room >= total {
		return 0, total
	}
	from = cursor - room/2
	if from < 0 {
		from = 0
	}
	if from+room > total {
		from = total - room
	}
	return from, from + room
}

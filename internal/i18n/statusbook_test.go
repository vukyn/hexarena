package i18n_test

import (
	"testing"

	"github.com/vukyn/hexarena/internal/core/status"
	"github.com/vukyn/hexarena/internal/seed"
)

// shippedStatuses is the status book the describers read names out of.
//
// It is loaded rather than passed as nil, even by the tests below that describe a
// hand-built skill naming statuses no book declares. A nil book is a supported
// answer — Lang.statusName falls back to the compiled table — but it is the
// answer that exercises nothing: the whole point of the parameter is that an
// authored name reaches a sentence, and a suite that never hands one a book
// would go green with the lookup deleted.
func shippedStatuses(t *testing.T) *status.Book {
	t.Helper()
	kinds, err := seed.StatusBook()
	if err != nil {
		t.Fatalf("load the shipped statuses: %v", err)
	}
	return kinds
}

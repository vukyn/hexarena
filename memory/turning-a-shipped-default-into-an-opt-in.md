---
name: turning-a-shipped-default-into-an-opt-in
description: "hexarena step 6: watching shipped ON for every room at step 4 and step 6 made it opt-in — ⚠️ the cost is every earlier test that used the feature, and the thing to establish is that they went RED rather than quietly stopping; ⚠️ a doc that narrowed a ban once will describe the leftover wrongly, so the second narrowing lands on a sentence that already forbids it"
metadata:
  type: project
---

`feat/host-watch-flag` (step 6 of TODO.md § *Spectators*): `-watch`, off by
default, `room.Config.Watchable`, a gate refusal of its own. Step 4 had shipped
watching **on for every room with no opt-in**, so this was a default being taken
away rather than a feature being added.

**The cost is the earlier tests, and the danger is that they pass.** Seventeen
tests written at step 4 join a watcher. If any of them had asserted something
weaker than "the watcher was welcomed", it would have gone on passing against a
room that now refuses every watcher — a test that silently stopped exercising the
thing it is named after, inside a green suite. They did not: **six in
`internal/room`, nine in `internal/socket`, one in `internal/wire`, plus
`cmd/hexarena-tui`'s `shown_test.go`** — seventeen in all — went red, because
each of them either asserts `Admission.Watching` or dials a watcher and expects
a welcome.

**So the deliverable of a flip like this is the loudness, not the fix.** Run the
suite *before* touching a single fixture and keep the list; "I updated the
fixtures" is not evidence, and a count of red tests measured against a count of
call sites is. Both were derived rather than remembered:
`grep -rn --include='*.go' -E "Watch: *true|Watch *= *true"` finds every watching
hello in the module (three files — two fixtures and one inline), and every call
site of the fixture is then read off `grep -n "watchingHello("`.

**The fixture shape that keeps the default visible.** A `watchable(cfg)` wrapper
per test package, not a `config()` that quietly turns it on: the wrapper composes
with `draftingConfig` and, more to the point, leaves the *default* what a test
gets when it says nothing — so the arm that expects a refusal cannot be written
by accident against a room nobody can open. Its own guard is
`if (room.Config{}).Watchable { t.Fatal(...) }`, read off the zero value rather
than assumed.

⚠️ **The trap: a ban that was narrowed once describes what is left WRONGLY.**
`internal/wire`'s `TestTheOnlyCodeAboutAWatcherIsTheCap` began as
`TestAWatcherIsRefusedByNoCodeOfItsOwn`, banning *any* refusal code named after
watching. Step 4 narrowed it to an allowlist of one for the transport's cap, and
the sentence describing the remainder read that a code named for a watcher's
**admission** was still forbidden. Step 6's code is exactly that — a watcher that
cannot be let in because the room was never opened to spectators — and it is
forbidden on precisely the *no-argument-behind-it* grounds the first narrowing
threw out. The allowlist is two now, both of them "a watcher that cannot be let
in", and what stays banned is a code about a watcher's **squad or side**. When a
flat rule is narrowed, the leftover has to be re-derived from the argument, not
copied from the old sentence.

⚠️ **Do not reuse a refusal whose advice is FALSE rather than merely unhelpful.**
`CodeTooManyWatchers` was the tempting reuse and it is the worse of the two
misdirections in this file: `CodeRoomFull` says something irrelevant to somebody
who asked for no seat, but the cap says *wait for one of the current watchers to
leave and paste the code again* — and where nobody is watching and nobody can,
that is a reader sent to wait for a queue that will never move. The whole chain
for one code is: the constant **declared last** with the "declared last" comment
moved onto it, `codeNames`, `internal/i18n/keys.go`, a line in **both** books,
`internal/i18n/protocol.go`, and an entry in `cmd/hexarena-tui/shown_test.go`
whose arithmetic is `len(gate)+len(inMatch)+len(owed) == wire.CodeCount-1`.

⚠️ **Configuration is not state, and the sentence that keeps it apart is where
it is read.** `Watchable` is one field, fixed before anybody joins, read in
exactly one place (the gate, on one hello) and never written — so the room still
holds **no count, no list and no cap** of watchers, `seatCount` is still 2 and
the roster is in the order it would have been in. That is also why
`socket.MaxWatchers` stayed in the transport: whether a room may be watched and
how many may watch it are different questions, and the cap bounds a fan-out
inside the room's `exchange` lock over connections only the transport holds.

⚠️ **A banner guard that greps for a word it is reading cannot fail.** The
earlier `-draft` guard asserted a refusal *lacked* two words the refusal never
contained. Here the banner says a spectator pastes the same N characters a
player does, so the guard is `fmt.Sprintf("SAME %d characters", len(held.code))`
— equality against the room's own code and `wire.RoomCodeLength` — and the
mutation `wire.RoomCodeLength → 8` reddens it. And setting `chosen.watch`
directly proves nothing about a **flag**: parse `[]string{"-watch"}` through
`flags(&chosen)` instead, and read the default from the same parse, or a flag
registered with a default of `true` looks exactly like the design.

Liên quan: [[hexarena-draft-and-spectator-plan]] [[hexarena-protocol-wordings]]
[[hexarena-room-state-machine]] [[fixture-hidden-branch]]
[[mutate-the-producer-not-just-the-logic]]

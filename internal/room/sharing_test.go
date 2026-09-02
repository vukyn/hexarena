package room_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/vukyn/hexarena/internal/room"
)

// mutexCalls is every selector that touches the registry's mutex. Unlock is on
// the list as well as Lock, because the mutation that hides from a Lock-only
// walk is a lookup that **returns holding the lock** and a caller that unlocks
// after its send — which touches the mutex without ever naming Lock.
var mutexCalls = map[string]bool{
	"Lock": true, "Unlock": true,
	"RLock": true, "RUnlock": true, "TryLock": true,
}

// TestNoLockingFunctionSendsOnAChannel is the registry's load-bearing rule made
// mechanical: **the mutex guards the map and nothing else.**
//
// A registry that held the lock across the send to the room it found would keep
// the letter of "a room owns its battle in one goroutine" — one goroutine still
// owns it — while losing the whole point, because every room in the process would
// then serialise through one lock and N rooms would run at the speed of one. That
// failure is invisible: the tests pass, the race detector is silent, and the only
// symptom is a number nobody is measuring. So it is held here rather than
// described in a comment.
//
// The walk is a **reachability** analysis rather than a look at one function at a
// time, because the obvious version of the mutation is the shape that hides from
// that: a locking function need not contain the send itself, it need only call
// something that does. So every function that sends directly is marked, the mark
// is propagated to its callers to a fixed point, and any function that touches
// the mutex and can reach a send is refused. `ask` (which sends) and `lookup`
// (which locks) are therefore two functions on purpose, and inlining either into
// the other reddens this.
//
// ⚠️ **A `go` statement is not a call for this purpose**, and the exclusion is
// the rule rather than a convenience: starting a goroutine does not block, so a
// locking function that spawns one has not serialised anybody. Open does exactly
// that — it enrols under the lock and then starts the room's goroutine, which is
// the function containing every send in the file.
//
// Two limits, stated because a mechanical guard that overstates itself is worse
// than none. The call graph is keyed on the **function name** alone, so two
// methods of the same name merge — which is conservative (their locks and their
// sends are unioned, so a merge can only add refusals) and is real here, since
// Room and Registry both declare Join, Deliver, TimedOut and Left. And a call
// through an interface or a function value is invisible; nothing in this package
// makes one.
func TestNoLockingFunctionSendsOnAChannel(t *testing.T) {
	type function struct {
		file  string
		line  int
		locks bool
		sends bool
		calls []string
	}
	functions := map[string]*function{}
	// declared keeps the report in source order rather than in a map's, which is
	// the engine's own rule about a map iteration reaching an output.
	declared := []string{}
	types := map[string]bool{}
	scanned := 0
	for _, name := range packageSources(t) {
		scanned++
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range parsed.Decls {
			if kinds, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range kinds.Specs {
					if named, ok := spec.(*ast.TypeSpec); ok && named.Name != nil {
						types[named.Name.Name] = true
					}
				}
				continue
			}
			body, ok := declaration.(*ast.FuncDecl)
			if !ok || body.Body == nil || body.Name == nil {
				continue
			}
			entry, known := functions[body.Name.Name]
			if !known {
				entry = &function{file: name, line: fileSet.Position(body.Pos()).Line}
				functions[body.Name.Name] = entry
				declared = append(declared, body.Name.Name)
			}
			ast.Inspect(body.Body, func(node ast.Node) bool {
				switch found := node.(type) {
				case *ast.SendStmt:
					entry.sends = true
				case *ast.GoStmt:
					// → the note above: a goroutine started under the lock
					// blocks nobody, so it is not an edge.
					return false
				case *ast.CallExpr:
					switch called := found.Fun.(type) {
					case *ast.Ident:
						entry.calls = append(entry.calls, called.Name)
					case *ast.SelectorExpr:
						if selected := called.Sel; selected != nil {
							if mutexCalls[selected.Name] {
								entry.locks = true
								return true
							}
							entry.calls = append(entry.calls, selected.Name)
						}
					}
				}
				return true
			})
		}
	}
	// Propagate "reaches a send" to callers until nothing changes. The order the
	// map is walked in cannot reach the answer: a fixed point is the same set
	// whichever order it is computed in.
	for changed := true; changed; {
		changed = false
		for _, entry := range functions {
			if entry.sends {
				continue
			}
			for _, called := range entry.calls {
				if target, known := functions[called]; known && target.sends {
					entry.sends, changed = true, true
					break
				}
			}
		}
	}
	locking, sending, both := 0, 0, 0
	for _, name := range declared {
		entry := functions[name]
		if entry.locks {
			locking++
		}
		if entry.sends {
			sending++
		}
		if entry.locks && entry.sends {
			both++
			t.Errorf("%s:%d: %s both touches the mutex and reaches a channel send. "+
				"The mutex guards the map and nothing else: look a code up, RELEASE the lock, "+
				"then send to the room it found — or every room in the process serialises through one lock",
				entry.file, entry.line, name)
		}
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	// A walk over a package with no mutex and no channel in it would pass this
	// whatever the registry did, which is the vacuity every copy of this shape
	// records. Requiring one of each is also what says the registry's own file
	// was read: nothing else in internal/room holds either.
	if locking == 0 || sending == 0 {
		t.Fatalf("%d functions touch the mutex and %d reach a send; with either at nought this walk measures nothing",
			locking, sending)
	}
	// And the same walk internal/room's clock ban uses reads this same list, so
	// requiring the registry's own type to be declared in it is what says the ban
	// covers the new file rather than that it happens to still pass.
	if !types["Registry"] {
		t.Error("no file in packageSources declares Registry, so neither this walk nor the clock ban is reading the registry")
	}
	t.Logf("scanned %d source files, %d functions; %d touch the mutex, %d reach a channel send, %d do both",
		scanned, len(functions), locking, sending, both)
}

// TestNoRoomMethodTouchesTheMutex is what replaced the `sync` import ban in
// TestTheRoomReadsNoClock, and it is a stronger claim rather than a relaxed one.
//
// ⚠️ The old ban said "a room owns its battle in one goroutine and shares it
// with nothing; the registry takes the mutex" and enforced it by refusing `sync`
// **in the whole package** — which was written when the registry was expected to
// live somewhere else. It landed here, partly so that it would inherit the clock
// ban rather than need a second copy of it, so the import ban would have refused
// the one file it was written to accommodate.
//
// What an import ban could never say is *whose* mutex it was banning. This says
// it: no method whose receiver is Room may touch a mutex, operate on a channel or
// start a goroutine — so "a battle never does" is held **by receiver** and is
// true wherever the method is written, including inside registry.go. The
// registry's own half is TestNoLockingFunctionSendsOnAChannel.
//
// The vacuity guard is two-sided, because each half alone passes on a mistake: a
// walk that found no Room methods would pass on a renamed receiver, and a walk
// that found nothing touching a mutex anywhere would pass on a registry that had
// stopped locking at all.
func TestNoRoomMethodTouchesTheMutex(t *testing.T) {
	roomMethods, registryLocks, scanned := 0, 0, 0
	for _, name := range packageSources(t) {
		scanned++
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range parsed.Decls {
			body, ok := declaration.(*ast.FuncDecl)
			if !ok || body.Body == nil || body.Name == nil {
				continue
			}
			receiver := receiverType(body)
			if receiver != "Room" {
				if receiver == "Registry" {
					ast.Inspect(body.Body, func(node ast.Node) bool {
						call, ok := node.(*ast.CallExpr)
						if !ok {
							return true
						}
						if selected, ok := call.Fun.(*ast.SelectorExpr); ok && selected.Sel != nil && mutexCalls[selected.Sel.Name] {
							registryLocks++
						}
						return true
					})
				}
				continue
			}
			roomMethods++
			ast.Inspect(body.Body, func(node ast.Node) bool {
				// ⚠️ ast.Inspect calls its walker with a nil node to close each
				// subtree, so the position may only be read inside the arms.
				if node == nil {
					return false
				}
				at := fileSet.Position(node.Pos()).Line
				switch found := node.(type) {
				case *ast.GoStmt:
					t.Errorf("%s:%d: Room.%s starts a goroutine; a room is owned by the one goroutine the registry gave it",
						name, at, body.Name.Name)
				case *ast.SendStmt:
					t.Errorf("%s:%d: Room.%s sends on a channel; a room shares its battle with nothing and speaks only through its return values",
						name, at, body.Name.Name)
				case *ast.UnaryExpr:
					if found.Op == token.ARROW {
						t.Errorf("%s:%d: Room.%s receives from a channel; a room shares its battle with nothing",
							name, at, body.Name.Name)
					}
				case *ast.CallExpr:
					if selected, ok := found.Fun.(*ast.SelectorExpr); ok && selected.Sel != nil && mutexCalls[selected.Sel.Name] {
						t.Errorf("%s:%d: Room.%s takes a mutex; the mutex is the REGISTRY's and a battle never takes one",
							name, at, body.Name.Name)
					}
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("the scan read no source files, so it measures nothing")
	}
	if roomMethods == 0 {
		t.Fatal("the scan found no method on Room, so it measures nothing about a room")
	}
	if registryLocks == 0 {
		t.Fatal("the scan found no method on Registry touching a mutex, so it cannot tell a room's abstinence from nobody locking at all")
	}
	t.Logf("scanned %d source files: %d methods on Room, none touching a mutex, a channel or a goroutine, against %d mutex calls on Registry",
		scanned, roomMethods, registryLocks)
}

// receiverType is the bare type name a method is declared on, pointer or not,
// and the empty string for a plain function.
func receiverType(declared *ast.FuncDecl) string {
	if declared.Recv == nil || len(declared.Recv.List) == 0 {
		return ""
	}
	switch named := declared.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return named.Name
	case *ast.StarExpr:
		if pointed, ok := named.X.(*ast.Ident); ok {
			return pointed.Name
		}
	}
	return ""
}

// TestTheRegistryHandsOutNoRoom is the other half of "a room shares its battle
// with nothing", and it is the half a comment cannot hold.
//
// The *Room lives as a parameter of the goroutine that serves it. Nothing else in
// the process can reach it — not because the code is careful, but because no
// exported method of Registry mentions the type in either direction: Open builds
// one from a Config and a Deps and keeps it, and everything after that is by
// code. So a caller cannot hold a room even by mistake, which is what makes the
// invariant enforced rather than asked for.
//
// It walks the type graph of every signature rather than the signature itself,
// because the way a room would escape is inside something — a field on an Answer,
// an element of a slice, a value in a map. ⚠️ It cannot see through an
// **interface**: request.body is a wire.Body, and what a walk of static types
// says about it is nothing. That is why the request's own doc comment argues the
// point (a message is not a closure) and why this test is one of two guards
// rather than the only one.
func TestTheRegistryHandsOutNoRoom(t *testing.T) {
	registryType := reflect.TypeOf(room.NewRegistry())
	roomType := reflect.TypeOf(&room.Room{})
	if registryType.NumMethod() == 0 {
		t.Fatal("the registry has no exported methods, so this walk measures nothing")
	}
	checked := 0
	for at := range registryType.NumMethod() {
		method := registryType.Method(at)
		signature := method.Type
		for in := range signature.NumIn() {
			checked++
			if path := reaches(signature.In(in), roomType, map[reflect.Type]bool{}); path != "" {
				t.Errorf("Registry.%s takes a %s, which reaches a *room.Room by %s: nothing may hand a room in or out, "+
					"because the room's goroutine is the only thing that may touch it",
					method.Name, signature.In(in), path)
			}
		}
		for out := range signature.NumOut() {
			checked++
			if path := reaches(signature.Out(out), roomType, map[reflect.Type]bool{}); path != "" {
				t.Errorf("Registry.%s returns a %s, which reaches a *room.Room by %s: nothing may hand a room out, "+
					"because the room's goroutine is the only thing that may touch it",
					method.Name, signature.Out(out), path)
			}
		}
	}
	t.Logf("walked %d parameter and result types across %d exported methods; no *room.Room is reachable from any of them",
		checked, registryType.NumMethod())
}

// reaches reports the path by which one type can reach another through pointers,
// slices, arrays, maps, channels, struct fields and function signatures, or the
// empty string when it cannot. The receiver is walked too, deliberately: the
// registry's own fields are where a room would be kept if it were kept anywhere.
func reaches(candidate, wanted reflect.Type, seen map[reflect.Type]bool) string {
	if candidate == nil || seen[candidate] {
		return ""
	}
	if candidate == wanted || (wanted.Kind() == reflect.Pointer && candidate == wanted.Elem()) {
		return candidate.String()
	}
	seen[candidate] = true
	step := func(into reflect.Type, how string) string {
		if path := reaches(into, wanted, seen); path != "" {
			return how + " → " + path
		}
		return ""
	}
	switch candidate.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Chan:
		return step(candidate.Elem(), candidate.String())
	case reflect.Map:
		if path := step(candidate.Key(), candidate.String()+" key"); path != "" {
			return path
		}
		return step(candidate.Elem(), candidate.String())
	case reflect.Struct:
		for at := range candidate.NumField() {
			field := candidate.Field(at)
			if path := step(field.Type, candidate.String()+"."+field.Name); path != "" {
				return path
			}
		}
	case reflect.Func:
		for at := range candidate.NumIn() {
			if path := step(candidate.In(at), candidate.String()+" argument"); path != "" {
				return path
			}
		}
		for at := range candidate.NumOut() {
			if path := step(candidate.Out(at), candidate.String()+" result"); path != "" {
				return path
			}
		}
	}
	return ""
}

// TestTheRegistryIsInTheDirectoryTheClockBanWalks confirms rather than assumes
// what the task took on trust: TestTheRoomReadsNoClock reads os.ReadDir("."), so
// a file added to this package inherits the ban for free — and the registry is
// in this package **for that reason** rather than for tidiness.
//
// It is not a second copy of the ban. It asserts the one fact that would make the
// existing ban vacuous about the new file: that the file is in the list the ban
// walks and that the ban's own list is not empty.
func TestTheRegistryIsInTheDirectoryTheClockBanWalks(t *testing.T) {
	sources := packageSources(t)
	if len(sources) == 0 {
		t.Fatal("the package has no source files, so the clock ban walks nothing")
	}
	found := ""
	for _, name := range sources {
		if strings.Contains(name, "registry") {
			found = name
		}
	}
	if found == "" {
		t.Fatalf("no file in %v looks like the registry; the clock ban walks this directory, "+
			"so a registry outside it would not inherit the ban", sources)
	}
	t.Logf("the clock ban walks %d files of this package, %s among them", len(sources), found)
}

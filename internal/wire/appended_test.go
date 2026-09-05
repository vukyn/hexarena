package wire

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestTheDeclaredLastCommentSitsOnTheLastConstant holds a rule this package
// states **three** times — on Kind, on Code and on Closure — and which nothing
// held until it was broken.
//
// The rule is that a value serialises by name, so appending one cannot
// reinterpret anything a peer has already written, while inserting one would
// move the count and every table built from declaration order. The rule is
// carried by a comment reading "Declared last" on the final constant of each
// enum, and the failure it protects against is not a wrong constant: it is the
// **comment being left behind**. A kind appended after KindClosed while that
// paragraph stays on KindClosed leaves the file saying something false about
// itself, which is worse than saying nothing, because the next reader appends
// beside the comment.
//
// ⚠️ **This is a comment test and that is deliberate, because no other kind of
// test can see it.** Every totality guard in this package walks a count, and a
// count is right whichever constant the paragraph is attached to; the golden
// records bytes, and a comment has none. Measured while it was written: moving
// the phrase back onto KindClosed leaves the whole module green except this.
//
// It is written **generally** rather than against the three enums by name: any
// const block one of whose entries claims to be declared last must have that
// claim on its final entry. So the fourth enum to grow the paragraph is covered
// the day it is written, and an enum that loses the paragraph altogether is
// caught by the count below rather than passing silently.
func TestTheDeclaredLastCommentSitsOnTheLastConstant(t *testing.T) {
	const claim = "Declared last"
	claimed := map[string]string{}
	scanned := 0
	for _, entry := range mustReadPackageDir(t) {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			block, isConst := declaration.(*ast.GenDecl)
			if !isConst || block.Tok != token.CONST || len(block.Specs) == 0 {
				continue
			}
			last := ""
			if final, ok := block.Specs[len(block.Specs)-1].(*ast.ValueSpec); ok &&
				len(final.Names) > 0 {
				last = final.Names[0].Name
			}
			for at, spec := range block.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || value.Doc == nil || len(value.Names) == 0 {
					continue
				}
				if !strings.Contains(value.Doc.Text(), claim) {
					continue
				}
				held := value.Names[0].Name
				claimed[held] = name
				if at != len(block.Specs)-1 {
					t.Errorf("%s: %s carries the %q paragraph and is not the last constant of "+
						"its block — %s is. A kind, code or closure was appended and the "+
						"paragraph stayed behind, so the file now says something false about "+
						"itself and the next reader will append beside the comment",
						name, held, claim, last)
				}
			}
		}
	}
	// Non-vacuity, two ways. A walk over no files passes everything, and so does
	// a walk over files in which the paragraph has been deleted rather than
	// moved — which is the other way to make this test stop meaning anything.
	if scanned == 0 {
		t.Fatal("no source file was scanned, so this test measured nothing")
	}
	if got, want := len(claimed), 3; got < want {
		t.Errorf("%d constants carry the %q paragraph and %d enums are written under that rule "+
			"(Kind, Code, Closure); a deleted paragraph is as bad as a stale one: %v",
			got, claim, want, claimed)
	}
	t.Logf("scanned %d source files, %d constants carry the %q paragraph: %v",
		scanned, len(claimed), claim, claimed)
}

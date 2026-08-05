package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every reply the bot makes is an embed. A bare Content message is
// indistinguishable from a staff member talking, so this walks the package for
// any message built with a Content field, which is how one would creep back in.
//
// Ctx.Say/Ok/Fail and modalReply are the sanctioned ways to answer; they all
// build the embed themselves, so nothing outside them should set Content.
func TestRepliesAreEmbeds(t *testing.T) {
	files, err := filepath.Glob("*.go")

	if err != nil {
		t.Fatal(err)
	}

	// closePermEditor blanks the content deliberately, so that an editor being
	// closed does not leave earlier text sitting above the note.
	const allowed = "permeditor_apply.go"

	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == allowed {
			continue
		}

		src, err := os.ReadFile(path)

		if err != nil {
			t.Fatal(err)
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)

		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)

			if !ok {
				return true
			}

			sel, ok := lit.Type.(*ast.SelectorExpr)

			if !ok || sel.Sel.Name != "MessageCreate" {
				return true
			}

			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)

				if !ok {
					continue
				}

				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Content" {
					t.Errorf("%s: MessageCreate sets Content — replies must be embeds, use Ctx.Say/Ok/Fail",
						fset.Position(kv.Pos()))
				}
			}

			return true
		})
	}
}

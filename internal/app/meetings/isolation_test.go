package meetings

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMeetingsDoesNotImportOthers(t *testing.T) {
	forbid := []string{"internal/app/mail", "internal/app/teams", "internal/app/calendar", "internal/app/files", "internal/adapters"}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		src, err := parser.ParseFile(fset, filepath.Join(".", e.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, im := range src.Imports {
			p := strings.Trim(im.Path.Value, `"`)
			for _, f := range forbid {
				if strings.Contains(p, f) {
					t.Fatalf("%s imports %s", e.Name(), f)
				}
			}
		}
	}
}

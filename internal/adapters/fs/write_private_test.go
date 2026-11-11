package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func wantOwnerOnly(t *testing.T, p string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode %o, want 600", got)
	}
}

func TestWriteFileOverwriteIsOwnerOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "shared.txt")
	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(p, []byte("new"), true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new" {
		t.Fatalf("content %q err %v", b, err)
	}
	wantOwnerOnly(t, p)
}

func TestWriteFileRefusesExistingWithoutOverwrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "kept.txt")
	if err := os.WriteFile(p, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := WriteFile(p, []byte("new"), false)
	de, ok := err.(*domain.Error)
	if !ok || de.Class != domain.ClassUsage || de.Message != "destination exists; pass --overwrite" {
		t.Fatalf("err %#v", err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "kept" {
		t.Fatalf("content changed to %q", b)
	}
}

func TestWriteFileNewFileIsOwnerOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "new.txt")
	if err := WriteFile(p, []byte("x"), false); err != nil {
		t.Fatal(err)
	}
	wantOwnerOnly(t, p)
}

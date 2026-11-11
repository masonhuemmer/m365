package fs

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func wantNotRegular(t *testing.T, err error, p string) {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok || de.Class != domain.ClassUsage || de.Message != "destination is not a regular file: "+p {
		t.Fatalf("err %#v", err)
	}
}

func TestWriteFileOverwriteRefusesASymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	wantNotRegular(t, WriteFile(link, []byte("new"), true), link)
	if b, _ := os.ReadFile(target); string(b) != "keep" {
		t.Fatalf("target changed to %q", b)
	}
}

func TestWriteFileOverwriteRefusesADevice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null on Windows")
	}
	wantNotRegular(t, WriteFile("/dev/null", []byte("new"), true), "/dev/null")
}

// fakeOwnedFile records what writeOwnerOnly does to it.
type fakeOwnedFile struct {
	chmodErr  error
	truncated bool
	written   int
	closed    bool
}

func (f *fakeOwnedFile) Chmod(os.FileMode) error     { return f.chmodErr }
func (f *fakeOwnedFile) Truncate(int64) error        { f.truncated = true; return nil }
func (f *fakeOwnedFile) Write(b []byte) (int, error) { f.written += len(b); return len(b), nil }
func (f *fakeOwnedFile) Close() error                { f.closed = true; return nil }

func TestWriteOwnerOnlyKeepsTheOldContentWhenChmodFails(t *testing.T) {
	f := &fakeOwnedFile{chmodErr: errors.New("not the owner")}
	if err := writeOwnerOnly(f, []byte("new")); err == nil {
		t.Fatal("want an error")
	}
	if f.truncated || f.written != 0 || !f.closed {
		t.Fatalf("truncated=%v written=%d closed=%v", f.truncated, f.written, f.closed)
	}
}

func TestWriteOwnerOnlyTruncatesBeforeWriting(t *testing.T) {
	f := &fakeOwnedFile{}
	if err := writeOwnerOnly(f, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if !f.truncated || f.written != 3 {
		t.Fatalf("truncated=%v written=%d", f.truncated, f.written)
	}
}

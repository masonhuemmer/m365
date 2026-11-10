package fs

import (
	"errors"
	iofs "io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

func ReadAttach(path string) (domain.OutboundFile, error) {
	if path == "" || strings.Contains(path, "://") {
		if u, err := url.Parse(path); err == nil && u.Scheme != "" && u.Scheme != "file" {
			return domain.OutboundFile{}, domain.Usage("remote URL attachments are forbidden")
		}
		if path == "" {
			return domain.OutboundFile{}, domain.Usage("missing attach path")
		}
	}
	st, err := os.Stat(path)
	if err != nil {
		return domain.OutboundFile{}, domain.Usagef("unreadable file: %s", path)
	}
	if st.IsDir() {
		return domain.OutboundFile{}, domain.Usagef("not a file: %s", path)
	}
	return domain.OutboundFile{Path: path, Name: filepath.Base(path), Size: st.Size()}, nil
}

func WriteFile(path string, data []byte, overwrite bool) error {
	if path == "" {
		return domain.Usage("destination path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil && !os.IsExist(err) {
		return domain.Usagef("unwritable path: %s", path)
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o600) // #nosec G304 -- caller --out path
	if errors.Is(err, iofs.ErrExist) {
		return domain.Usage("destination exists; pass --overwrite")
	}
	if err != nil {
		return domain.Usagef("unwritable path: %s", path)
	}
	if err := writeOwnerOnly(f, data); err != nil {
		return domain.Usagef("unwritable path: %s", path)
	}
	return nil
}

// writeOwnerOnly makes an overwritten file 0600 too: OpenFile keeps an existing file's mode.
func writeOwnerOnly(f *os.File, data []byte) error {
	err := f.Chmod(0o600)
	if err == nil {
		_, err = f.Write(data)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

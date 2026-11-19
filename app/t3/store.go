package t3

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Store keeps tern's credentials and cache in the user config directory
// (%AppData%\tern on Windows). Each value is one JSON file. On Windows the
// bytes are sealed with DPAPI, so they only open for the same Windows user.
type Store struct {
	dir string
}

func OpenStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenStoreAt(filepath.Join(base, "tern"))
}

func OpenStoreAt(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Get reads name into v. It reports false when there's nothing stored.
func (s *Store) Get(name string, v any) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	plain, err := unprotect(raw)
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(plain, v)
}

func (s *Store) Put(name string, v any) error {
	plain, err := json.Marshal(v)
	if err != nil {
		return err
	}
	sealed, err := protect(plain)
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, name+".tmp")
	if err := os.WriteFile(tmp, sealed, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, name))
}

func (s *Store) Delete(name string) error {
	err := os.Remove(filepath.Join(s.dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

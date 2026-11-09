// Package localfile persists managed local files with atomic replacement.
package localfile

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteAtomic syncs a temporary file, replaces the destination, and syncs its directory.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	return writeAtomic(path, data, perm, SyncDirectory)
}

func writeAtomic(path string, data []byte, perm os.FileMode, syncDir func(string) error) error {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()           // Best effort after an earlier write/close error.
		_ = os.Remove(file.Name()) // Best effort; rename normally removes the temporary path.
	}()
	if err = file.Chmod(perm); err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

// SyncDirectory flushes directory metadata where supported by the platform.
func SyncDirectory(path string) (resultErr error) {
	directory, err := os.Open(path) //nolint:gosec // callers provide the directory of a managed local state/config file
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := directory.Close(); resultErr == nil {
			resultErr = closeErr
		}
	}()
	info, err := directory.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("cannot sync a non-directory path")
	}
	return syncDirectory(directory)
}

package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Expand replaces a leading ~ or ~/ with the current user's home directory.
// Non-absolute paths without ~ are returned unchanged so the caller can decide.
func Expand(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if p == "~" {
		return home()
	}
	if strings.HasPrefix(p, "~/") {
		h, err := home()
		if err != nil {
			return "", err
		}
		return filepath.Join(h, p[2:]), nil
	}
	return p, nil
}

// Contract turns an absolute path under $HOME back into ~/... form.
// Used when adding a file to config so stored paths are portable across hosts.
func Contract(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	h, err := home()
	if err != nil {
		return abs, nil
	}
	if abs == h {
		return "~", nil
	}
	if strings.HasPrefix(abs, h+string(filepath.Separator)) {
		return "~/" + abs[len(h)+1:], nil
	}
	return abs, nil
}

func home() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return h, nil
}

// USBRoot locates the key-traveler USB root by looking for config.toml in:
//  1. $KTRAVELER_USB
//  2. The directory of the running binary
//  3. /media/$USER/*/config.toml and /run/media/$USER/*/config.toml
func USBRoot() (string, error) {
	if env := os.Getenv("KTRAVELER_USB"); env != "" {
		if hasConfig(env) {
			return env, nil
		}
		return "", fmt.Errorf("KTRAVELER_USB=%q does not contain config.toml", env)
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if hasConfig(dir) {
			return dir, nil
		}
	}

	user := os.Getenv("USER")
	if user != "" {
		for _, base := range []string{"/media/" + user, "/run/media/" + user} {
			entries, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, e := range entries {
				candidate := filepath.Join(base, e.Name())
				if hasConfig(candidate) {
					return candidate, nil
				}
			}
		}
	}

	return "", errors.New("cannot locate key-traveler USB (no config.toml found — set KTRAVELER_USB or run init)")
}

func hasConfig(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, "config.toml"))
	return err == nil && !fi.IsDir()
}

// WriteAtomic writes data to path via a temp file in the same directory,
// fsyncs to disk, then renames. The temp file is cleaned up on error.
func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".ktraveler-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		if tmp != "" {
			os.Remove(tmp)
		}
	}()

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	tmp = "" // rename succeeded — disarm cleanup
	return nil
}

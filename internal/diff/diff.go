package diff

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ShowUnified prints a unified diff between two files to stdout.
// Uses /usr/bin/diff if available; falls back to a minimal line-by-line diff.
func ShowUnified(w io.Writer, left, right, leftLabel, rightLabel string) error {
	cmd := exec.Command("diff", "-u",
		"--label", leftLabel,
		"--label", rightLabel,
		left, right)
	cmd.Stdout = w
	cmd.Stderr = w
	// diff exits 1 when files differ; that's not an error for us.
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return nil
	}
	return err
}

// TmpDir returns a temp directory on tmpfs (RAM) so decrypted plaintext never
// hits a real disk. Falls back to the default temp dir if /dev/shm is absent.
func TmpDir() string {
	if fi, err := os.Stat("/dev/shm"); err == nil && fi.IsDir() {
		return "/dev/shm"
	}
	return os.TempDir()
}

// WriteTemp writes data to a new file inside TmpDir and returns its path.
// Caller must os.Remove it when done.
func WriteTemp(prefix string, data []byte) (string, error) {
	f, err := os.CreateTemp(TmpDir(), prefix+"-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	// Restrict perms — this is decrypted secret material.
	if err := os.Chmod(name, 0o600); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// Prompt reads a single line of user input, trimmed and lowercased.
// `choices` like "lvsa" constrains the accepted first character; empty accepts anything.
func Prompt(question, choices string) (string, error) {
	fmt.Print(question)
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			fmt.Print(question)
			continue
		}
		if choices == "" {
			return line, nil
		}
		if strings.ContainsRune(choices, rune(line[0])) {
			return string(line[0]), nil
		}
		fmt.Printf("invalid choice; please pick one of [%s]: ", strings.Join(strings.Split(choices, ""), ","))
	}
}

// Confirm asks a yes/no question. Default is the answer returned on empty input.
func Confirm(question string, def bool) (bool, error) {
	suffix := " [y/N] "
	if def {
		suffix = " [Y/n] "
	}
	fmt.Print(question + suffix)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def, nil
	}
	return line == "y" || line == "yes" || line == "o" || line == "oui", nil
}

// ShortPath makes a path shorter for display (drops TmpDir prefix).
func ShortPath(p string) string {
	if strings.HasPrefix(p, TmpDir()) {
		return filepath.Base(p)
	}
	return p
}

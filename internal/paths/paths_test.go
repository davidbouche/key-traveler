package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home")
	}
	tests := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"/absolute/path", "/absolute/path"},
		{"relative", "relative"},
	}
	for _, tt := range tests {
		got, err := Expand(tt.in)
		if err != nil {
			t.Errorf("Expand(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Expand(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExpand_Empty(t *testing.T) {
	_, err := Expand("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestContract(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home")
	}
	tests := []struct {
		in   string
		want string
	}{
		{filepath.Join(home, "foo"), "~/foo"},
		{home, "~"},
		{"/outside/home", "/outside/home"},
	}
	for _, tt := range tests {
		got, err := Contract(tt.in)
		if err != nil {
			t.Errorf("Contract(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Contract(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.txt")
	data := []byte("atomic content")

	if err := WriteAtomic(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", got, data)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %o, want 0644", fi.Mode().Perm())
	}
}

func TestWriteAtomic_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Errorf("content after overwrite = %q, want %q", got, "new")
	}
}

func TestWriteAtomic_NoLeftoverOnError(t *testing.T) {
	dir := t.TempDir()
	// Write to a non-existent subdirectory should fail
	path := filepath.Join(dir, "nosub", "file.txt")
	err := WriteAtomic(path, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error writing to non-existent dir")
	}
	// No .tmp files should be left
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

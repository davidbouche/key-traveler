package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func setTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "key-traveler")
}

func TestGenerate_CreatesIdentity(t *testing.T) {
	setTestDir(t)
	id, pub, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if id == nil {
		t.Fatal("identity is nil")
	}
	if pub == "" {
		t.Fatal("pubkey is empty")
	}

	path, _ := IdentityPath()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("permissions = %o, want 0600", fi.Mode().Perm())
	}
}

func TestGenerate_RefusesOverwrite(t *testing.T) {
	setTestDir(t)
	if _, _, err := Generate(); err != nil {
		t.Fatal(err)
	}
	_, _, err := Generate()
	if err == nil {
		t.Error("expected error on second Generate")
	}
}

func TestLoadOrGenerate_GeneratesWhenAbsent(t *testing.T) {
	setTestDir(t)
	id, pub, err := LoadOrGenerate()
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || pub == "" {
		t.Fatal("expected valid identity")
	}
}

func TestLoadOrGenerate_LoadsWhenPresent(t *testing.T) {
	setTestDir(t)
	_, origPub, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	id, pub, err := LoadOrGenerate()
	if err != nil {
		t.Fatal(err)
	}
	if pub != origPub {
		t.Errorf("pubkey changed: got %s, want %s", pub, origPub)
	}
	if id == nil {
		t.Fatal("identity is nil")
	}
}

func TestLoad_InsecurePermissions(t *testing.T) {
	confDir := setTestDir(t)
	os.MkdirAll(confDir, 0o700)
	path := filepath.Join(confDir, "identity.txt")
	os.WriteFile(path, []byte("AGE-SECRET-KEY-1DUMMY\n"), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("expected error for insecure permissions")
	}
}

func TestSaveHostname_LoadHostname(t *testing.T) {
	setTestDir(t)
	if _, err := SaveHostname("myhost"); err != nil {
		t.Fatal(err)
	}
	name, err := LoadHostname()
	if err != nil {
		t.Fatal(err)
	}
	if name != "myhost" {
		t.Errorf("hostname = %q, want %q", name, "myhost")
	}
}

func TestLoadHostname_Fallback(t *testing.T) {
	setTestDir(t)
	name, err := LoadHostname()
	if err != nil {
		t.Fatal(err)
	}
	if name == "" {
		t.Error("expected non-empty hostname from os.Hostname fallback")
	}
}

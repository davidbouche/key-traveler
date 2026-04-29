package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := &Config{Vault: VaultMeta{Version: SchemaVersion}}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := setupRoot(t)
	cfg := &Config{
		Vault: VaultMeta{Version: SchemaVersion},
		Hosts: []Host{{Name: "laptop", Pubkey: "age1test", EnrolledAt: time.Now().UTC().Truncate(time.Second)}},
		Files: []TrackedFile{{Path: "~/.ssh/id_rsa", Vault: "ssh-id_rsa.age"}},
		Patterns: []Pattern{{Pattern: "~/.ssh/id_*", AddedAt: time.Now().UTC().Truncate(time.Second)}},
	}
	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Vault.Version != SchemaVersion {
		t.Errorf("version = %d, want %d", loaded.Vault.Version, SchemaVersion)
	}
	if len(loaded.Hosts) != 1 || loaded.Hosts[0].Name != "laptop" {
		t.Errorf("hosts = %+v", loaded.Hosts)
	}
	if len(loaded.Files) != 1 || loaded.Files[0].Path != "~/.ssh/id_rsa" {
		t.Errorf("files = %+v", loaded.Files)
	}
	if len(loaded.Patterns) != 1 || loaded.Patterns[0].Pattern != "~/.ssh/id_*" {
		t.Errorf("patterns = %+v", loaded.Patterns)
	}
}

func TestLoad_UnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	data := []byte("[vault]\nversion = 999\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for unsupported schema version")
	}
}

func TestLoad_VersionZeroAccepted(t *testing.T) {
	dir := t.TempDir()
	data := []byte("[vault]\nversion = 0\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err != nil {
		t.Errorf("version 0 should be accepted: %v", err)
	}
}

func TestAddHost_Duplicate(t *testing.T) {
	cfg := &Config{}
	h := Host{Name: "laptop", Pubkey: "age1abc"}
	if err := cfg.AddHost(h); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddHost(h); err == nil {
		t.Error("expected error on duplicate host name")
	}
	h2 := Host{Name: "desktop", Pubkey: "age1abc"}
	if err := cfg.AddHost(h2); err == nil {
		t.Error("expected error on duplicate pubkey")
	}
}

func TestAddHost_Empty(t *testing.T) {
	cfg := &Config{}
	if err := cfg.AddHost(Host{Name: "", Pubkey: "age1abc"}); err == nil {
		t.Error("expected error for empty name")
	}
	if err := cfg.AddHost(Host{Name: "laptop", Pubkey: ""}); err == nil {
		t.Error("expected error for empty pubkey")
	}
}

func TestAddFile_DuplicateAndVaultName(t *testing.T) {
	cfg := &Config{}
	tf, err := cfg.AddFile("~/.ssh/id_rsa")
	if err != nil {
		t.Fatal(err)
	}
	if tf.Vault != "ssh-id_rsa.age" {
		t.Errorf("vault name = %q, want %q", tf.Vault, "ssh-id_rsa.age")
	}
	_, err = cfg.AddFile("~/.ssh/id_rsa")
	if err == nil {
		t.Error("expected error on duplicate file")
	}
}

func TestAddFile_VaultNameCollision(t *testing.T) {
	cfg := &Config{}
	cfg.AddFile("~/.ssh/id_rsa")
	// A path that would produce the same slug
	cfg.Files = append(cfg.Files, TrackedFile{Path: "~/other", Vault: "gpg-key.age"})
	tf, err := cfg.AddFile("~/.gpg/key")
	if err != nil {
		t.Fatal(err)
	}
	if tf.Vault == "gpg-key.age" {
		t.Error("vault name should have been deduped but was not")
	}
}

func TestRecipients(t *testing.T) {
	cfg := &Config{
		Hosts: []Host{
			{Name: "a", Pubkey: "age1a"},
			{Name: "b", Pubkey: "age1b"},
		},
	}
	r := cfg.Recipients()
	if len(r) != 2 || r[0] != "age1a" || r[1] != "age1b" {
		t.Errorf("Recipients = %v", r)
	}
}

func TestAddPattern_Duplicate(t *testing.T) {
	cfg := &Config{}
	if _, err := cfg.AddPattern("~/.ssh/id_*"); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.AddPattern("~/.ssh/id_*"); err == nil {
		t.Error("expected error on duplicate pattern")
	}
}

func TestRemovePattern(t *testing.T) {
	cfg := &Config{}
	cfg.AddPattern("~/.ssh/id_*")
	if err := cfg.RemovePattern("~/.ssh/id_*"); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Patterns) != 0 {
		t.Error("pattern not removed")
	}
}

func TestRemoveFile(t *testing.T) {
	cfg := &Config{}
	cfg.AddFile("~/.ssh/id_rsa")
	removed, err := cfg.RemoveFile("~/.ssh/id_rsa")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Path != "~/.ssh/id_rsa" {
		t.Errorf("removed = %+v", removed)
	}
	if len(cfg.Files) != 0 {
		t.Error("file not removed from list")
	}
}

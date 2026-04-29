package manifest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New()
	if m.Files == nil {
		t.Fatal("Files map is nil")
	}
	if len(m.Files) != 0 {
		t.Errorf("expected empty, got %d entries", len(m.Files))
	}
}

func TestLoad_Missing(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 0 {
		t.Errorf("expected empty manifest, got %d entries", len(m.Files))
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := New()
	now := time.Now().UTC().Truncate(time.Second)
	st := m.Get("~/.ssh/id_rsa")
	st.LastPush = &PushRecord{
		Host:  "laptop",
		At:    now,
		MD5:   "abc123",
		Mtime: now,
		Mode:  "0600",
		Size:  1024,
	}
	st.Pulls["laptop"] = &PullRecord{At: now, MD5: "abc123"}

	if err := Save(dir, m); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	ls := loaded.Files["~/.ssh/id_rsa"]
	if ls == nil {
		t.Fatal("file state not found after round-trip")
	}
	if ls.LastPush.Host != "laptop" {
		t.Errorf("host = %q, want %q", ls.LastPush.Host, "laptop")
	}
	if ls.LastPush.MD5 != "abc123" {
		t.Errorf("md5 = %q, want %q", ls.LastPush.MD5, "abc123")
	}
	pr := ls.Pulls["laptop"]
	if pr == nil || pr.MD5 != "abc123" {
		t.Errorf("pull record = %+v", pr)
	}
}

func TestGet_LazyCreate(t *testing.T) {
	m := New()
	st := m.Get("~/new/file")
	if st == nil {
		t.Fatal("Get returned nil")
	}
	if st.Pulls == nil {
		t.Error("Pulls map is nil on lazy-created state")
	}
	st2 := m.Get("~/new/file")
	if st != st2 {
		t.Error("Get returned different pointers for same path")
	}
}

func TestDelete(t *testing.T) {
	m := New()
	m.Get("~/a")
	m.Get("~/b")
	m.Delete("~/a")
	if _, ok := m.Files["~/a"]; ok {
		t.Error("file not deleted")
	}
	if _, ok := m.Files["~/b"]; !ok {
		t.Error("wrong file deleted")
	}
}

func TestLoad_Corrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for corrupt manifest")
	}
}

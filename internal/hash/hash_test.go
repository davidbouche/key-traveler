package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestMD5File(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello world\n")
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := MD5File(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Errorf("MD5File = %s, want %s", got, want)
	}
}

func TestMD5File_Missing(t *testing.T) {
	_, err := MD5File("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMD5Bytes(t *testing.T) {
	data := []byte("test data")
	got := MD5Bytes(data)
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Errorf("MD5Bytes = %s, want %s", got, want)
	}
}

func TestMD5Bytes_Empty(t *testing.T) {
	got := MD5Bytes(nil)
	sum := sha256.Sum256(nil)
	want := hex.EncodeToString(sum[:])
	if got != want {
		t.Errorf("MD5Bytes(nil) = %s, want %s", got, want)
	}
}

func TestIsBinary(t *testing.T) {
	dir := t.TempDir()

	text := filepath.Join(dir, "text.txt")
	if err := os.WriteFile(text, []byte("just text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "binary.dat")
	if err := os.WriteFile(bin, []byte("has\x00null"), 0o644); err != nil {
		t.Fatal(err)
	}

	if ok, _ := IsBinary(text); ok {
		t.Error("text file reported as binary")
	}
	if ok, _ := IsBinary(bin); !ok {
		t.Error("binary file not detected")
	}
}

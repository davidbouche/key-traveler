package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/davidbouche/key-traveler/internal/paths"
)

const FileName = "manifest.json"

type Manifest struct {
	Files map[string]*FileState `json:"files"`
}

type FileState struct {
	LastPush *PushRecord            `json:"last_push,omitempty"`
	Pulls    map[string]*PullRecord `json:"pulls,omitempty"`
}

type PushRecord struct {
	Host    string    `json:"host"`
	At      time.Time `json:"at"`
	MD5     string    `json:"md5"`
	Mtime   time.Time `json:"mtime"`
	Mode    string    `json:"mode"`
	UIDName string    `json:"uid_name"`
	GIDName string    `json:"gid_name"`
	Size    int64     `json:"size"`
}

type PullRecord struct {
	At  time.Time `json:"at"`
	MD5 string    `json:"md5"`
}

// New returns an empty manifest ready to be written.
func New() *Manifest {
	return &Manifest{Files: map[string]*FileState{}}
}

// Load reads manifest.json from the USB root. Missing file returns an empty
// manifest (the first push on a fresh USB will populate it).
func Load(usbRoot string) (*Manifest, error) {
	path := filepath.Join(usbRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return New(), nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if m.Files == nil {
		m.Files = map[string]*FileState{}
	}
	return &m, nil
}

// Save writes manifest.json atomically to the USB root.
func Save(usbRoot string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	data = append(data, '\n')
	return paths.WriteAtomic(filepath.Join(usbRoot, FileName), data, 0o644)
}

// Get returns (and lazily creates) the FileState for a given stored path.
func (m *Manifest) Get(storedPath string) *FileState {
	st, ok := m.Files[storedPath]
	if !ok {
		st = &FileState{Pulls: map[string]*PullRecord{}}
		m.Files[storedPath] = st
	}
	if st.Pulls == nil {
		st.Pulls = map[string]*PullRecord{}
	}
	return st
}

// Delete drops a tracked file from the manifest.
func (m *Manifest) Delete(storedPath string) {
	delete(m.Files, storedPath)
}

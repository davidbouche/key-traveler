package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/david/key-traveler/internal/paths"
)

const (
	// SchemaVersion bumps when the TOML format changes incompatibly.
	SchemaVersion = 1

	FileName     = "config.toml"
	ManifestName = "manifest.json"
	VaultDir     = "vault"
	PendingDir   = "pending"
)

type Config struct {
	Vault    VaultMeta     `toml:"vault"`
	Hosts    []Host        `toml:"hosts"`
	Patterns []Pattern     `toml:"patterns"`
	Files    []TrackedFile `toml:"files"`
}

type VaultMeta struct {
	Version int `toml:"version"`
}

type Host struct {
	Name       string    `toml:"name"`
	Pubkey     string    `toml:"pubkey"`
	EnrolledAt time.Time `toml:"enrolled_at"`
}

type TrackedFile struct {
	// Path stored in ~/... form when under home.
	Path  string `toml:"path"`
	Vault string `toml:"vault"`
}

// Pattern is a glob expression (filepath.Glob syntax) that is re-evaluated on
// every sync-like command. Newly matching files are auto-added to [[files]].
type Pattern struct {
	Pattern string    `toml:"pattern"`
	AddedAt time.Time `toml:"added_at,omitempty"`
}

// Load reads config.toml from the USB root.
func Load(usbRoot string) (*Config, error) {
	path := filepath.Join(usbRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &c, nil
}

// Save writes config.toml atomically to the USB root.
func Save(usbRoot string, c *Config) error {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return paths.WriteAtomic(filepath.Join(usbRoot, FileName), buf.Bytes(), 0o644)
}

// Host lookups.
func (c *Config) HostByName(name string) *Host {
	for i := range c.Hosts {
		if c.Hosts[i].Name == name {
			return &c.Hosts[i]
		}
	}
	return nil
}

// Recipients returns the list of pubkey strings for all enrolled hosts.
func (c *Config) Recipients() []string {
	out := make([]string, 0, len(c.Hosts))
	for _, h := range c.Hosts {
		out = append(out, h.Pubkey)
	}
	return out
}

// AddHost appends a host, refusing on duplicate name or pubkey.
func (c *Config) AddHost(h Host) error {
	if h.Name == "" {
		return errors.New("host name is empty")
	}
	if h.Pubkey == "" {
		return errors.New("host pubkey is empty")
	}
	for _, existing := range c.Hosts {
		if existing.Name == h.Name {
			return fmt.Errorf("host %q already enrolled", h.Name)
		}
		if existing.Pubkey == h.Pubkey {
			return fmt.Errorf("pubkey already enrolled under host %q", existing.Name)
		}
	}
	c.Hosts = append(c.Hosts, h)
	return nil
}

// FindFile returns the TrackedFile matching p (comparing stored path), nil if absent.
func (c *Config) FindFile(p string) *TrackedFile {
	for i := range c.Files {
		if c.Files[i].Path == p {
			return &c.Files[i]
		}
	}
	return nil
}

// AddFile appends a file. The vault filename is derived from the path if empty.
// Returns the resulting TrackedFile.
func (c *Config) AddFile(storedPath string) (*TrackedFile, error) {
	if storedPath == "" {
		return nil, errors.New("path is empty")
	}
	if existing := c.FindFile(storedPath); existing != nil {
		return existing, fmt.Errorf("file %q already tracked", storedPath)
	}
	vault := vaultNameFor(storedPath, c.Files)
	tf := TrackedFile{Path: storedPath, Vault: vault}
	c.Files = append(c.Files, tf)
	return &c.Files[len(c.Files)-1], nil
}

// FindPattern returns the Pattern matching the expression, or nil.
func (c *Config) FindPattern(expr string) *Pattern {
	for i := range c.Patterns {
		if c.Patterns[i].Pattern == expr {
			return &c.Patterns[i]
		}
	}
	return nil
}

// AddPattern appends a glob pattern; rejects duplicates.
func (c *Config) AddPattern(expr string) (*Pattern, error) {
	if expr == "" {
		return nil, errors.New("pattern is empty")
	}
	if existing := c.FindPattern(expr); existing != nil {
		return existing, fmt.Errorf("pattern %q already tracked", expr)
	}
	c.Patterns = append(c.Patterns, Pattern{Pattern: expr, AddedAt: time.Now().UTC()})
	return &c.Patterns[len(c.Patterns)-1], nil
}

// RemovePattern drops a pattern by expression. Literal [[files]] already
// resolved from this pattern are left alone — use `remove` to drop them.
func (c *Config) RemovePattern(expr string) error {
	for i, p := range c.Patterns {
		if p.Pattern == expr {
			c.Patterns = append(c.Patterns[:i], c.Patterns[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pattern %q not tracked", expr)
}

// RemoveFile drops a tracked entry. Returns the removed entry for callers that
// need to purge the vault file.
func (c *Config) RemoveFile(storedPath string) (*TrackedFile, error) {
	for i, f := range c.Files {
		if f.Path == storedPath {
			copied := f
			c.Files = append(c.Files[:i], c.Files[i+1:]...)
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("file %q not tracked", storedPath)
}

// vaultNameFor builds a slug-like filename unique within the existing list.
func vaultNameFor(storedPath string, existing []TrackedFile) string {
	// Strip leading ~/ for slugging.
	s := strings.TrimPrefix(storedPath, "~/")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, ".")
	// Replace separators with dashes.
	s = strings.ReplaceAll(s, string(filepath.Separator), "-")
	s = strings.ReplaceAll(s, " ", "_")
	if s == "" {
		s = "file"
	}
	base := s + ".age"
	name := base
	n := 1
	for {
		taken := false
		for _, e := range existing {
			if e.Vault == name {
				taken = true
				break
			}
		}
		if !taken {
			return name
		}
		n++
		name = fmt.Sprintf("%s-%d.age", s, n)
	}
}

package identity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

// LocalDir is the per-host config directory holding the private key and the
// cached hostname used to index the manifest.
func LocalDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "key-traveler"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "key-traveler"), nil
}

// IdentityPath returns the path to the private key file.
func IdentityPath() (string, error) {
	d, err := LocalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "identity.txt"), nil
}

// HostnameFile returns the path to the cached hostname file.
func HostnameFile() (string, error) {
	d, err := LocalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "hostname"), nil
}

// Generate creates a fresh X25519 identity and writes it to disk with mode 0600.
// Returns the identity (for immediate use) and its recipient (pubkey string).
func Generate() (*age.X25519Identity, string, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, "", err
	}
	dir, err := LocalDir()
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, "", err
	}
	path, err := IdentityPath()
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(path); err == nil {
		return nil, "", fmt.Errorf("identity already exists at %s (refusing to overwrite)", path)
	}
	content := fmt.Sprintf("# created by key-traveler\n%s\n", id.String())
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return nil, "", err
	}
	return id, id.Recipient().String(), nil
}

// Load reads the local private key. Refuses to load if permissions are too lax.
func Load() (*age.X25519Identity, error) {
	path, err := IdentityPath()
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("identity not found at %s (run `ktraveler enroll` or `ktraveler enroll-request`)", path)
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("identity file %s has insecure permissions %o (must be 0600)", path, fi.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, err := age.ParseX25519Identity(line)
		if err != nil {
			return nil, fmt.Errorf("parsing identity: %w", err)
		}
		return id, nil
	}
	return nil, errors.New("no identity key found in identity file")
}

// SaveHostname persists the chosen hostname so subsequent commands can identify
// this host without re-prompting. Returns the path written.
func SaveHostname(name string) (string, error) {
	path, err := HostnameFile()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(name+"\n"), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// LoadHostname reads the cached hostname. Falls back to os.Hostname().
func LoadHostname() (string, error) {
	path, err := HostnameFile()
	if err == nil {
		if data, err := os.ReadFile(path); err == nil {
			name := strings.TrimSpace(string(data))
			if name != "" {
				return name, nil
			}
		}
	}
	h, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return h, nil
}

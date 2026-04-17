package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/manifest"
)

// Init bootstraps a new USB root at the given directory.
func Init(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler init <usb-path>")
	}
	root := args[0]

	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", root, err)
	}
	if err := os.MkdirAll(filepath.Join(root, config.VaultDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, config.PendingDir), 0o755); err != nil {
		return err
	}

	cfgPath := filepath.Join(root, config.FileName)
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing %s", cfgPath)
	}

	cfg := &config.Config{Vault: config.VaultMeta{Version: config.SchemaVersion}}
	if err := config.Save(root, cfg); err != nil {
		return err
	}
	if err := manifest.Save(root, manifest.New()); err != nil {
		return err
	}

	fmt.Printf("Initialized key-traveler vault at %s\n", root)
	fmt.Printf("Next: on this host, run `KTRAVELER_USB=%s ktraveler enroll <host-name>`\n", root)
	return nil
}

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/diff"
	"github.com/david/key-traveler/internal/manifest"
	"github.com/david/key-traveler/internal/paths"
	"github.com/david/key-traveler/internal/patterns"
)

// Add declares a file to be tracked. It does not push the file; run sync/push after.
func Add(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler add <path>")
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	for _, raw := range args {
		stored, err := paths.Contract(raw)
		if err != nil {
			return err
		}
		expanded, err := paths.Expand(stored)
		if err != nil {
			return err
		}
		if _, err := os.Stat(expanded); err != nil {
			return fmt.Errorf("%s: %w", expanded, err)
		}
		tf, err := cfg.AddFile(stored)
		if err != nil {
			return err
		}
		fmt.Printf("added %s -> vault/%s\n", stored, tf.Vault)
	}
	return config.Save(root, cfg)
}

// Remove stops tracking a file. Optionally purges the vault blob.
func Remove(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler remove <path> [--purge]")
	}
	purge := false
	var targets []string
	for _, a := range args {
		if a == "--purge" {
			purge = true
			continue
		}
		targets = append(targets, a)
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	mf, err := manifest.Load(root)
	if err != nil {
		return err
	}

	for _, raw := range targets {
		stored, err := paths.Contract(raw)
		if err != nil {
			return err
		}
		removed, err := cfg.RemoveFile(stored)
		if err != nil {
			return err
		}
		mf.Delete(stored)
		if purge {
			blob := filepath.Join(root, config.VaultDir, removed.Vault)
			if err := os.Remove(blob); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			fmt.Printf("removed %s and purged vault/%s\n", stored, removed.Vault)
		} else {
			fmt.Printf("removed %s (vault/%s kept — re-run with --purge to delete)\n", stored, removed.Vault)
		}
	}

	if err := config.Save(root, cfg); err != nil {
		return err
	}
	return manifest.Save(root, mf)
}

// List prints tracked files with their vault filename.
func List(_ []string) error {
	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	// Resolve patterns in memory so the user sees what they currently match.
	// Read-only command: don't persist auto-additions here.
	matches, _ := patterns.Resolve(cfg)

	fmt.Printf("vault: %s (%d host(s), %d file(s), %d pattern(s))\n",
		root, len(cfg.Hosts), len(cfg.Files), len(cfg.Patterns))

	if len(cfg.Hosts) == 0 {
		fmt.Println("no hosts enrolled yet.")
	} else {
		fmt.Println("hosts:")
		for _, h := range cfg.Hosts {
			fmt.Printf("  - %s (%s)\n", h.Name, diff.ShortPath(h.Pubkey))
		}
	}

	if len(cfg.Patterns) > 0 {
		fmt.Println("patterns:")
		for _, p := range cfg.Patterns {
			fmt.Printf("  - %q\n", p.Pattern)
			for _, m := range matches {
				if m.Pattern == p.Pattern {
					tag := ""
					if m.IsNew {
						tag = "  (new, pending)"
					}
					fmt.Printf("      · %s%s\n", m.Path, tag)
				}
			}
		}
	}

	if len(cfg.Files) == 0 {
		fmt.Println("no files tracked.")
		return nil
	}
	fmt.Println("files:")
	for _, f := range cfg.Files {
		fmt.Printf("  - %s -> vault/%s\n", f.Path, f.Vault)
	}
	return nil
}

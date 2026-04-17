package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidbouche/key-traveler/internal/config"
	"github.com/davidbouche/key-traveler/internal/diff"
	"github.com/davidbouche/key-traveler/internal/manifest"
	"github.com/davidbouche/key-traveler/internal/paths"
	"github.com/davidbouche/key-traveler/internal/patterns"
)

// isGlob decides whether a user-supplied argument should be treated as a
// filepath.Glob pattern rather than a literal path. We only trigger on `*`
// and `?`: `[` alone occurs in legitimate filenames (e.g. `foo[1].txt`) and
// would cause confusing false positives.
func isGlob(s string) bool {
	return strings.ContainsAny(s, "*?")
}

// Add accepts a mix of literal paths and glob patterns. Each argument is
// routed based on its shape: patterns register in [[patterns]] and are
// resolved immediately; plain paths register in [[files]].
func Add(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler add <path-or-pattern>...")
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	touchedPattern := false
	for _, raw := range args {
		if isGlob(raw) {
			stored, err := paths.Contract(raw)
			if err != nil {
				return err
			}
			if _, err := cfg.AddPattern(stored); err != nil {
				return err
			}
			fmt.Printf("added pattern %q\n", stored)
			touchedPattern = true
			continue
		}

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
		fmt.Printf("added file %s -> vault/%s\n", stored, tf.Vault)
	}

	// Resolve patterns right away so the user sees what a newly-added one
	// matches, consistent with the former `add-pattern` behaviour.
	if touchedPattern {
		matches, err := patterns.Resolve(cfg)
		if err != nil {
			return err
		}
		for _, m := range matches {
			if m.IsNew {
				fmt.Printf("  matched (new) %s\n", m.Path)
			} else {
				fmt.Printf("  matched       %s (already tracked)\n", m.Path)
			}
		}
	}
	return config.Save(root, cfg)
}

// Remove accepts a mix of tracked paths and registered patterns. Patterns
// are removed from [[patterns]]; files are removed from [[files]] and,
// optionally with --purge, their encrypted blob is also deleted.
//
// Removing a pattern does NOT drop the files it already matched. Use
// individual `remove <path>` calls if you want to also drop those.
func Remove(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler remove <path-or-pattern>... [--purge]")
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
	if len(targets) == 0 {
		return errors.New("remove: no target given (need at least one path or pattern)")
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
		if isGlob(raw) {
			if err := cfg.RemovePattern(stored); err != nil {
				return err
			}
			fmt.Printf("removed pattern %q (already-matched files kept — run `remove <path>` to drop them)\n", stored)
			continue
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
			fmt.Printf("removed file %s and purged vault/%s\n", stored, removed.Vault)
		} else {
			fmt.Printf("removed file %s (vault/%s kept — re-run with --purge to delete)\n", stored, removed.Vault)
		}
	}

	if err := config.Save(root, cfg); err != nil {
		return err
	}
	return manifest.Save(root, mf)
}

// List prints tracked files, patterns and hosts for the current vault.
func List(_ []string) error {
	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

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

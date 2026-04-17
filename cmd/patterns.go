package cmd

import (
	"errors"
	"fmt"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/paths"
	"github.com/david/key-traveler/internal/patterns"
)

// AddPattern registers a glob pattern. Does not immediately resolve it — the
// next sync/status/list/… call will pick up all current matches.
func AddPattern(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler add-pattern <glob>")
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	for _, expr := range args {
		if _, err := cfg.AddPattern(expr); err != nil {
			return err
		}
		fmt.Printf("added pattern %q\n", expr)
	}

	// Immediately resolve so the user sees what matches right now.
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
	return config.Save(root, cfg)
}

// RemovePattern drops a pattern. Already-resolved [[files]] stay; the user
// must `remove` them individually if desired.
func RemovePattern(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler remove-pattern <glob>")
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	for _, expr := range args {
		if err := cfg.RemovePattern(expr); err != nil {
			return err
		}
		fmt.Printf("removed pattern %q (already-matched files kept — use `remove` to drop them)\n", expr)
	}
	return config.Save(root, cfg)
}

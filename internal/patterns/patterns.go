package patterns

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/davidbouche/key-traveler/internal/config"
	"github.com/davidbouche/key-traveler/internal/paths"
)

// Match is one concrete file that resolved from a pattern.
type Match struct {
	Pattern string
	Path    string // stored form (~/…)
	IsNew   bool   // true if it was auto-added to cfg.Files during this call
}

// Resolve expands every pattern in cfg and auto-adds newly matching files to
// cfg.Files (in memory). The caller decides whether to persist cfg.
//
// The list of matches returned is sorted by pattern then by path. Files
// already in cfg.Files are reported with IsNew=false so callers can show
// "file X is matched by pattern Y" context if they want.
func Resolve(cfg *config.Config) ([]Match, error) {
	var out []Match

	for _, p := range cfg.Patterns {
		expanded, err := paths.Expand(p.Pattern)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", p.Pattern, err)
		}
		// filepath.Glob does not expand ~, so we expand first then glob.
		hits, err := filepath.Glob(expanded)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", p.Pattern, err)
		}
		sort.Strings(hits)

		for _, hit := range hits {
			stored, err := paths.Contract(hit)
			if err != nil {
				return nil, err
			}
			if existing := cfg.FindFile(stored); existing != nil {
				out = append(out, Match{Pattern: p.Pattern, Path: stored, IsNew: false})
				continue
			}
			tf, err := cfg.AddFile(stored)
			if err != nil {
				return nil, err
			}
			_ = tf
			out = append(out, Match{Pattern: p.Pattern, Path: stored, IsNew: true})
		}
	}

	return out, nil
}

// NewlyAdded is a convenience filter returning only the matches that caused a
// file to be added to cfg.Files during this Resolve pass.
func NewlyAdded(ms []Match) []Match {
	var out []Match
	for _, m := range ms {
		if m.IsNew {
			out = append(out, m)
		}
	}
	return out
}

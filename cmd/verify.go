package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/hash"
	"github.com/david/key-traveler/internal/identity"
	"github.com/david/key-traveler/internal/manifest"
	"github.com/david/key-traveler/internal/paths"
	"github.com/david/key-traveler/internal/vault"
)

// Verify checks every vault blob: can we decrypt it, and does its md5 match
// the manifest's last_push.md5?
func Verify(_ []string) error {
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
	id, err := identity.Load()
	if err != nil {
		return err
	}

	problems := 0
	for _, f := range cfg.Files {
		src := filepath.Join(root, config.VaultDir, f.Vault)
		plaintext, err := vault.DecryptFile(src, id)
		if err != nil {
			fmt.Printf("  FAIL %s: %v\n", f.Vault, err)
			problems++
			continue
		}
		got := hash.MD5Bytes(plaintext)
		st := mf.Get(f.Path)
		if st.LastPush == nil {
			fmt.Printf("  WARN %s: no last_push recorded\n", f.Vault)
			continue
		}
		if got != st.LastPush.MD5 {
			fmt.Printf("  FAIL %s: md5 mismatch (vault=%s, manifest=%s)\n",
				f.Vault, shortHash(got), shortHash(st.LastPush.MD5))
			problems++
			continue
		}
		fmt.Printf("  ok   %s (%d bytes)\n", f.Vault, len(plaintext))
	}
	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	fmt.Printf("all %d file(s) verified.\n", len(cfg.Files))
	return nil
}

package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidbouche/key-traveler/internal/config"
	"github.com/davidbouche/key-traveler/internal/hash"
	"github.com/davidbouche/key-traveler/internal/manifest"
)

func writeVaultBlob(t *testing.T, root, vaultName string, data []byte) {
	t.Helper()
	dir := filepath.Join(root, config.VaultDir)
	os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(filepath.Join(dir, vaultName), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildDecision(t *testing.T) {
	now := time.Now().UTC()
	me := "laptop"

	tests := []struct {
		name        string
		localData   []byte // nil = file does not exist
		vaultData   []byte // nil = blob does not exist
		lastPush    *manifest.PushRecord
		pullsMe     *manifest.PullRecord
		wantAction  action
	}{
		{
			name:       "both missing",
			wantAction: actMissing,
		},
		{
			name:       "local only, no vault",
			localData:  []byte("secret"),
			wantAction: actPush,
		},
		{
			name:       "vault only, no local",
			vaultData:  []byte("encrypted-blob"),
			wantAction: actPull,
		},
		{
			name:      "in sync",
			localData: []byte("secret"),
			vaultData: []byte("encrypted-blob"),
			lastPush:  &manifest.PushRecord{Host: me, At: now, MD5: hash.MD5Bytes([]byte("secret"))},
			pullsMe:   &manifest.PullRecord{At: now, MD5: hash.MD5Bytes([]byte("secret"))},
			wantAction: actNone,
		},
		{
			name:      "vault advanced, local unchanged",
			localData: []byte("old"),
			vaultData: []byte("encrypted-blob"),
			lastPush:  &manifest.PushRecord{Host: "desktop", At: now, MD5: hash.MD5Bytes([]byte("new"))},
			pullsMe:   &manifest.PullRecord{At: now.Add(-time.Hour), MD5: hash.MD5Bytes([]byte("old"))},
			wantAction: actPull,
		},
		{
			name:      "local advanced, vault unchanged",
			localData: []byte("modified"),
			vaultData: []byte("encrypted-blob"),
			lastPush:  &manifest.PushRecord{Host: me, At: now, MD5: hash.MD5Bytes([]byte("original"))},
			pullsMe:   &manifest.PullRecord{At: now, MD5: hash.MD5Bytes([]byte("original"))},
			wantAction: actPush,
		},
		{
			name:      "both changed — conflict",
			localData: []byte("local-change"),
			vaultData: []byte("encrypted-blob"),
			lastPush:  &manifest.PushRecord{Host: "desktop", At: now, MD5: hash.MD5Bytes([]byte("vault-change"))},
			pullsMe:   &manifest.PullRecord{At: now.Add(-time.Hour), MD5: hash.MD5Bytes([]byte("common-ancestor"))},
			wantAction: actConflict,
		},
		{
			name:      "first time host sees file — conflict",
			localData: []byte("local-version"),
			vaultData: []byte("encrypted-blob"),
			lastPush:  &manifest.PushRecord{Host: "desktop", At: now, MD5: hash.MD5Bytes([]byte("different"))},
			pullsMe:   nil,
			wantAction: actConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()

			localPath := filepath.Join(home, ".ssh", "id_rsa")
			storedPath := localPath // using absolute path for test simplicity
			vaultName := "ssh-id_rsa.age"
			f := config.TrackedFile{Path: storedPath, Vault: vaultName}

			if tt.localData != nil {
				os.MkdirAll(filepath.Dir(localPath), 0o700)
				os.WriteFile(localPath, tt.localData, 0o600)
			}

			if tt.vaultData != nil {
				writeVaultBlob(t, root, vaultName, tt.vaultData)
			}

			mf := manifest.New()
			if tt.lastPush != nil || tt.pullsMe != nil {
				st := mf.Get(storedPath)
				st.LastPush = tt.lastPush
				if tt.pullsMe != nil {
					st.Pulls[me] = tt.pullsMe
				}
			}

			d, err := buildDecision(root, me, f, mf)
			if err != nil {
				t.Fatal(err)
			}
			if d.act != tt.wantAction {
				t.Errorf("action = %s, want %s", d.act.label(), tt.wantAction.label())
			}
		})
	}
}

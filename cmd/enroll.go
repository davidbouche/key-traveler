package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/identity"
	"github.com/david/key-traveler/internal/paths"
	"github.com/david/key-traveler/internal/vault"
)

// Enroll is used on the FIRST host only. It generates a local identity and adds
// its pubkey to config.toml. If other hosts exist already, use enroll-request
// from the new host + enroll-approve from an existing host.
func Enroll(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler enroll <host-name>")
	}
	name := args[0]

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if len(cfg.Hosts) > 0 {
		return fmt.Errorf("vault already has %d enrolled host(s); use `ktraveler enroll-request` on this host then `ktraveler enroll-approve` on an existing one", len(cfg.Hosts))
	}

	id, pub, err := identity.Generate()
	if err != nil {
		return err
	}
	_ = id // identity is on disk now; we only need the pubkey here

	if err := cfg.AddHost(config.Host{
		Name:       name,
		Pubkey:     pub,
		EnrolledAt: time.Now().UTC(),
	}); err != nil {
		return err
	}
	if err := config.Save(root, cfg); err != nil {
		return err
	}
	if _, err := identity.SaveHostname(name); err != nil {
		return err
	}

	idPath, _ := identity.IdentityPath()
	fmt.Printf("Generated local identity at %s (mode 0600)\n", idPath)
	fmt.Printf("Enrolled host %q on vault at %s\n", name, root)
	fmt.Println("You can now `ktraveler add <path>` and `ktraveler push`.")
	return nil
}

// EnrollRequest runs on a NEW host. It generates a local identity and drops a
// pending enrollment file on the USB stick so an existing host can approve.
func EnrollRequest(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler enroll-request <host-name>")
	}
	name := args[0]

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if cfg.HostByName(name) != nil {
		return fmt.Errorf("host %q already enrolled", name)
	}

	_, pub, err := identity.Generate()
	if err != nil {
		return err
	}
	if _, err := identity.SaveHostname(name); err != nil {
		return err
	}

	pendingDir := filepath.Join(root, config.PendingDir)
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		return err
	}
	req := pendingRequest{
		Name:        name,
		Pubkey:      pub,
		RequestedAt: time.Now().UTC(),
	}
	data, _ := json.MarshalIndent(req, "", "  ")
	data = append(data, '\n')
	reqPath := filepath.Join(pendingDir, sanitize(name)+".json")
	if err := paths.WriteAtomic(reqPath, data, 0o644); err != nil {
		return err
	}

	idPath, _ := identity.IdentityPath()
	fmt.Printf("Generated local identity at %s (mode 0600)\n", idPath)
	fmt.Printf("Enrollment request written to %s\n", reqPath)
	fmt.Println("Now go to an already-enrolled host and run `ktraveler enroll-approve`.")
	return nil
}

// EnrollApprove runs on an EXISTING host. It reads pending requests, adds
// their pubkeys to config.toml, and re-encrypts every vault/*.age so the new
// hosts can decrypt.
func EnrollApprove(_ []string) error {
	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if len(cfg.Hosts) == 0 {
		return errors.New("no existing hosts; run `ktraveler enroll <name>` instead")
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}

	pendingDir := filepath.Join(root, config.PendingDir)
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("no pending enrollments.")
			return nil
		}
		return err
	}

	var approved []pendingRequest
	var processed []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(pendingDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var req pendingRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		if cfg.HostByName(req.Name) != nil {
			fmt.Printf("skipping %s: already enrolled\n", req.Name)
			processed = append(processed, path)
			continue
		}
		if err := cfg.AddHost(config.Host{
			Name:       req.Name,
			Pubkey:     req.Pubkey,
			EnrolledAt: time.Now().UTC(),
		}); err != nil {
			return err
		}
		approved = append(approved, req)
		processed = append(processed, path)
	}

	if len(approved) == 0 {
		fmt.Println("no new enrollments to approve.")
		return nil
	}

	// Re-encrypt all vault files with the updated recipient list.
	recipients := cfg.Recipients()
	vaultDir := filepath.Join(root, config.VaultDir)
	for _, f := range cfg.Files {
		src := filepath.Join(vaultDir, f.Vault)
		if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := vault.ReEncrypt(src, id, recipients); err != nil {
			return fmt.Errorf("re-encrypting %s: %w", f.Vault, err)
		}
		fmt.Printf("  re-encrypted %s\n", f.Vault)
	}

	if err := config.Save(root, cfg); err != nil {
		return err
	}

	// Clean up consumed pending files.
	for _, p := range processed {
		_ = os.Remove(p)
	}

	for _, r := range approved {
		fmt.Printf("approved host %q (pubkey %s)\n", r.Name, r.Pubkey)
	}
	return nil
}

type pendingRequest struct {
	Name        string    `json:"name"`
	Pubkey      string    `json:"pubkey"`
	RequestedAt time.Time `json:"requested_at"`
}

func sanitize(name string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "..", "_")
	return r.Replace(name)
}

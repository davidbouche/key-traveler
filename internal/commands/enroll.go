package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davidbouche/key-traveler/internal/config"
	"github.com/davidbouche/key-traveler/internal/identity"
	"github.com/davidbouche/key-traveler/internal/paths"
	"github.com/davidbouche/key-traveler/internal/vault"
)

// Enroll is the master dispatcher for the enroll subcommand tree.
// Subcommands: list, request, approve.
func Enroll(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler enroll <list|request|approve> [args...]")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list", "ls":
		return enrollList(rest)
	case "request", "req":
		return enrollRequest(rest)
	case "approve":
		return enrollApprove(rest)
	default:
		return fmt.Errorf("unknown enroll subcommand %q (expected: list, request, approve)", sub)
	}
}

// enrollList prints enrolled hosts and pending requests.
func enrollList(_ []string) error {
	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}

	fmt.Printf("Vault: %s\n\n", root)

	if len(cfg.Hosts) == 0 {
		fmt.Println("No enrolled hosts yet.")
	} else {
		fmt.Printf("Enrolled hosts (%d):\n", len(cfg.Hosts))
		for _, h := range cfg.Hosts {
			fmt.Printf("  - %-15s  %s  enrolled %s\n",
				h.Name, h.Pubkey, h.EnrolledAt.Local().Format("2006-01-02 15:04"))
		}
	}

	pending, err := readPendingRequests(root)
	if err != nil {
		return err
	}
	fmt.Println()
	if len(pending) == 0 {
		fmt.Println("No pending requests.")
		return nil
	}
	fmt.Printf("Pending requests (%d):\n", len(pending))
	for _, p := range pending {
		fmt.Printf("  - %-15s  %s  requested %s\n",
			p.Name, p.Pubkey, p.RequestedAt.Local().Format("2006-01-02 15:04"))
	}
	fmt.Println()
	fmt.Println("Approve them on an already-enrolled host with:")
	fmt.Println("  ktraveler enroll approve <name>")
	fmt.Println("  ktraveler enroll approve --all")
	return nil
}

// enrollRequest generates the local identity and registers this host.
// If no host is yet enrolled on the vault, the request is auto-approved
// (written straight into config.toml). Otherwise a pending/<name>.json file
// is created for approval from an already-enrolled host.
func enrollRequest(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler enroll request <host-name>")
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
	if existing, _ := readPendingRequests(root); len(existing) > 0 {
		for _, p := range existing {
			if p.Name == name {
				return fmt.Errorf("a pending request for %q already exists", name)
			}
		}
	}

	_, pub, err := identity.Generate()
	if err != nil {
		return err
	}
	if _, err := identity.SaveHostname(name); err != nil {
		return err
	}
	idPath, _ := identity.IdentityPath()
	fmt.Printf("Generated local identity at %s (mode 0600)\n", idPath)

	// Empty vault: auto-approve. The first host cannot be approved by anyone
	// else, so the "request" model collapses into immediate enrollment.
	if len(cfg.Hosts) == 0 {
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
		fmt.Printf("Enrolled host %q on vault at %s (first host — auto-approved).\n", name, root)
		fmt.Println("You can now `ktraveler add <path>` and `ktraveler sync`.")
		return nil
	}

	// Otherwise: drop a pending request for approval on another host.
	pendingDir := filepath.Join(root, config.PendingDir)
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		return err
	}
	req := pendingRequest{Name: name, Pubkey: pub, RequestedAt: time.Now().UTC()}
	data, _ := json.MarshalIndent(req, "", "  ")
	data = append(data, '\n')
	reqPath := filepath.Join(pendingDir, sanitize(name)+".json")
	if err := paths.WriteAtomic(reqPath, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Request written to %s\n", reqPath)
	fmt.Println("Now go to an already-enrolled host and run:")
	fmt.Printf("  ktraveler enroll approve %s\n", name)
	return nil
}

// enrollApprove reads pending requests and integrates the chosen one(s).
//
//	enroll approve <name>    approve a single named request
//	enroll approve --all     approve every pending request
func enrollApprove(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler enroll approve <name>|--all")
	}

	root, err := paths.USBRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	if len(cfg.Hosts) == 0 {
		return errors.New("vault has no enrolled hosts; run `ktraveler enroll request <name>` first")
	}
	id, err := identity.Load()
	if err != nil {
		return err
	}

	pending, err := readPendingRequests(root)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		fmt.Println("no pending requests.")
		return nil
	}

	all := args[0] == "--all"
	wanted := ""
	if !all {
		wanted = args[0]
	}

	var toApprove []pendingRequest
	var consumedPaths []string
	for _, p := range pending {
		if !all && p.Name != wanted {
			continue
		}
		if cfg.HostByName(p.Name) != nil {
			fmt.Printf("skipping %s: already enrolled\n", p.Name)
			consumedPaths = append(consumedPaths, p.path)
			continue
		}
		if err := cfg.AddHost(config.Host{
			Name:       p.Name,
			Pubkey:     p.Pubkey,
			EnrolledAt: time.Now().UTC(),
		}); err != nil {
			return err
		}
		toApprove = append(toApprove, p)
		consumedPaths = append(consumedPaths, p.path)
	}

	if !all && len(toApprove) == 0 {
		return fmt.Errorf("no pending request named %q (run `ktraveler enroll list` to see available ones)", wanted)
	}
	if len(toApprove) == 0 {
		fmt.Println("nothing to approve.")
		return nil
	}

	// Re-encrypt every vault blob for the new recipient list.
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
	for _, p := range consumedPaths {
		_ = os.Remove(p)
	}
	for _, r := range toApprove {
		fmt.Printf("approved host %q (pubkey %s)\n", r.Name, r.Pubkey)
	}
	return nil
}

type pendingRequest struct {
	Name        string    `json:"name"`
	Pubkey      string    `json:"pubkey"`
	RequestedAt time.Time `json:"requested_at"`

	path string // filesystem path of the request file (not serialised)
}

func readPendingRequests(root string) ([]pendingRequest, error) {
	pendingDir := filepath.Join(root, config.PendingDir)
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var out []pendingRequest
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		p := filepath.Join(pendingDir, e.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var req pendingRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		req.path = p
		out = append(out, req)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func sanitize(name string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "..", "_")
	return r.Replace(name)
}

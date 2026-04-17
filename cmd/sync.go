package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"filippo.io/age"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/diff"
	"github.com/david/key-traveler/internal/hash"
	"github.com/david/key-traveler/internal/identity"
	"github.com/david/key-traveler/internal/manifest"
	"github.com/david/key-traveler/internal/paths"
	"github.com/david/key-traveler/internal/vault"
)

// action describes what sync decided for a file.
type action int

const (
	actNone action = iota
	actPush
	actPull
	actConflict
	actMissing
)

func (a action) label() string {
	switch a {
	case actNone:
		return "in-sync"
	case actPush:
		return "push"
	case actPull:
		return "pull"
	case actConflict:
		return "conflict"
	case actMissing:
		return "missing"
	}
	return "?"
}

type decision struct {
	file    config.TrackedFile
	act     action
	localPath string
	vaultPath string

	localExists bool
	vaultExists bool
	localMD5    string
	lastPush    *manifest.PushRecord
	pullsMe     *manifest.PullRecord
}

// buildDecision computes what sync would do for one tracked file.
func buildDecision(root, me string, f config.TrackedFile, mf *manifest.Manifest) (*decision, error) {
	local, err := paths.Expand(f.Path)
	if err != nil {
		return nil, err
	}
	vaultPath := filepath.Join(root, config.VaultDir, f.Vault)

	d := &decision{
		file:      f,
		localPath: local,
		vaultPath: vaultPath,
	}

	if st := mf.Get(f.Path); st != nil {
		d.lastPush = st.LastPush
		if pull, ok := st.Pulls[me]; ok {
			d.pullsMe = pull
		}
	}

	if fi, err := os.Stat(local); err == nil && !fi.IsDir() {
		d.localExists = true
		md5, err := hash.MD5File(local)
		if err != nil {
			return nil, err
		}
		d.localMD5 = md5
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if fi, err := os.Stat(vaultPath); err == nil && !fi.IsDir() {
		d.vaultExists = true
	}

	switch {
	case !d.localExists && !d.vaultExists:
		d.act = actMissing
	case !d.vaultExists:
		d.act = actPush
	case !d.localExists:
		d.act = actPull
	case d.lastPush != nil && d.localMD5 == d.lastPush.MD5:
		// local already matches the latest vault version.
		d.act = actNone
	case d.pullsMe == nil:
		// first time this host sees the file, and local + vault differ.
		d.act = actConflict
	case d.localMD5 == d.pullsMe.MD5 && d.lastPush != nil && d.pullsMe.MD5 != d.lastPush.MD5:
		// vault advanced, local did not.
		d.act = actPull
	case d.localMD5 != d.pullsMe.MD5 && d.lastPush != nil && d.pullsMe.MD5 == d.lastPush.MD5:
		// local advanced, vault did not.
		d.act = actPush
	default:
		d.act = actConflict
	}

	return d, nil
}

// Status prints what sync would do, without making changes.
func Status(_ []string) error {
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
	me, err := identity.LoadHostname()
	if err != nil {
		return err
	}

	fmt.Printf("host: %s   vault: %s\n", me, root)
	if len(cfg.Files) == 0 {
		fmt.Println("no files tracked.")
		return nil
	}

	for _, f := range cfg.Files {
		d, err := buildDecision(root, me, f, mf)
		if err != nil {
			return err
		}
		fmt.Printf("  %-10s %s\n", d.act.label(), f.Path)
		if d.act == actConflict {
			summarizeConflict(d)
		}
	}
	return nil
}

// Sync is the interactive full flow: auto fast-forwards, prompts on conflicts.
func Sync(_ []string) error {
	return runSync(syncModeInteractive)
}

// Push forces local → vault for any file that differs (with confirmation).
func Push(_ []string) error {
	return runSync(syncModePush)
}

// Pull forces vault → local for any file that differs (with confirmation).
func Pull(_ []string) error {
	return runSync(syncModePull)
}

type syncMode int

const (
	syncModeInteractive syncMode = iota
	syncModePush
	syncModePull
)

func runSync(mode syncMode) error {
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
	me, err := identity.LoadHostname()
	if err != nil {
		return err
	}
	if cfg.HostByName(me) == nil {
		return fmt.Errorf("this host (%q) is not enrolled on the vault; run `ktraveler enroll` or `enroll-request`", me)
	}

	id, err := identity.Load()
	if err != nil {
		return err
	}
	recipients := cfg.Recipients()
	changed := false

	for _, f := range cfg.Files {
		d, err := buildDecision(root, me, f, mf)
		if err != nil {
			return err
		}
		switch mode {
		case syncModePush:
			if d.act == actPush || d.act == actConflict || d.act == actPull {
				if d.act != actPush {
					ok, err := diff.Confirm(fmt.Sprintf("force push %s (overwrites vault)?", f.Path), false)
					if err != nil || !ok {
						fmt.Printf("  skipped %s\n", f.Path)
						continue
					}
				}
				if err := doPush(root, me, d, mf, recipients); err != nil {
					return err
				}
				changed = true
			}
		case syncModePull:
			if d.act == actPull || d.act == actConflict || d.act == actPush {
				if d.act != actPull {
					ok, err := diff.Confirm(fmt.Sprintf("force pull %s (overwrites local)?", f.Path), false)
					if err != nil || !ok {
						fmt.Printf("  skipped %s\n", f.Path)
						continue
					}
				}
				if err := doPull(me, d, id, mf); err != nil {
					return err
				}
				changed = true
			}
		case syncModeInteractive:
			switch d.act {
			case actNone, actMissing:
				// nothing to do; missing files are reported but skipped
				if d.act == actMissing {
					fmt.Printf("  missing  %s (not present anywhere — add it or remove the entry)\n", f.Path)
				}
			case actPush:
				fmt.Printf("  push     %s\n", f.Path)
				if err := doPush(root, me, d, mf, recipients); err != nil {
					return err
				}
				changed = true
			case actPull:
				fmt.Printf("  pull     %s\n", f.Path)
				if err := doPull(me, d, id, mf); err != nil {
					return err
				}
				changed = true
			case actConflict:
				c, err := resolveConflict(d, id)
				if err != nil {
					return err
				}
				switch c {
				case "l":
					if err := doPush(root, me, d, mf, recipients); err != nil {
						return err
					}
					changed = true
				case "v":
					if err := doPull(me, d, id, mf); err != nil {
						return err
					}
					changed = true
				case "s":
					fmt.Printf("  skipped %s\n", f.Path)
				case "a":
					fmt.Println("aborted by user.")
					if changed {
						return manifest.Save(root, mf)
					}
					return nil
				}
			}
		}
	}

	if changed {
		return manifest.Save(root, mf)
	}
	fmt.Println("nothing to do.")
	return nil
}

func summarizeConflict(d *decision) {
	if d.lastPush != nil {
		fmt.Printf("      local md5=%s  vault md5=%s (last push by %s at %s)\n",
			shortHash(d.localMD5), shortHash(d.lastPush.MD5),
			d.lastPush.Host, d.lastPush.At.Local().Format(time.RFC3339))
	} else {
		fmt.Printf("      local md5=%s  vault md5=(unknown)\n", shortHash(d.localMD5))
	}
}

func resolveConflict(d *decision, id age.Identity) (string, error) {
	fmt.Printf("\n⚠ CONFLICT on %s\n", d.file.Path)

	localFi, _ := os.Stat(d.localPath)
	localMtime := ""
	if localFi != nil {
		localMtime = localFi.ModTime().Local().Format(time.RFC3339)
	}
	fmt.Printf("  local : md5=%s  mtime=%s  size=%s\n",
		shortHash(d.localMD5), localMtime, fmtSize(localFi))

	if d.lastPush != nil {
		fmt.Printf("  vault : md5=%s  mtime=%s  size=%d  (pushed by %s at %s)\n",
			shortHash(d.lastPush.MD5),
			d.lastPush.Mtime.Local().Format(time.RFC3339),
			d.lastPush.Size,
			d.lastPush.Host,
			d.lastPush.At.Local().Format(time.RFC3339))
	} else {
		fmt.Println("  vault : (no push record)")
	}

	// Decrypt vault into tmpfs for the diff.
	plaintext, err := vault.DecryptFile(d.vaultPath, id)
	if err != nil {
		return "", err
	}
	vaultTmp, err := diff.WriteTemp("ktraveler-vault", plaintext)
	if err != nil {
		return "", err
	}
	defer os.Remove(vaultTmp)

	localIsBin, _ := hash.IsBinary(d.localPath)
	vaultIsBin, _ := hash.IsBinary(vaultTmp)
	if localIsBin || vaultIsBin {
		fmt.Println("  (binary file — diff suppressed)")
	} else {
		fmt.Println()
		_ = diff.ShowUnified(os.Stdout, d.localPath, vaultTmp,
			"local:"+d.file.Path, "vault:"+d.file.Vault)
	}

	return diff.Prompt(
		"\n  action? [l]ocal→push / [v]ault→pull / [s]kip / [a]bort: ",
		"lvsa",
	)
}

func doPush(root, me string, d *decision, mf *manifest.Manifest, recipients []string) error {
	data, err := os.ReadFile(d.localPath)
	if err != nil {
		return err
	}
	if err := vault.EncryptToFile(d.vaultPath, data, recipients); err != nil {
		return err
	}

	fi, err := os.Stat(d.localPath)
	if err != nil {
		return err
	}
	md5 := hash.MD5Bytes(data)
	uidName, gidName := ownerNames(fi)
	now := time.Now().UTC()

	st := mf.Get(d.file.Path)
	st.LastPush = &manifest.PushRecord{
		Host:    me,
		At:      now,
		MD5:     md5,
		Mtime:   fi.ModTime().UTC(),
		Mode:    fmt.Sprintf("%04o", fi.Mode().Perm()),
		UIDName: uidName,
		GIDName: gidName,
		Size:    fi.Size(),
	}
	st.Pulls[me] = &manifest.PullRecord{At: now, MD5: md5}
	fmt.Printf("    -> pushed (md5=%s)\n", shortHash(md5))
	_ = root
	return nil
}

func doPull(me string, d *decision, id age.Identity, mf *manifest.Manifest) error {
	plaintext, err := vault.DecryptFile(d.vaultPath, id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(d.localPath), 0o755); err != nil {
		return err
	}

	mode := os.FileMode(0o600)
	if d.lastPush != nil && d.lastPush.Mode != "" {
		if m, err := strconv.ParseUint(d.lastPush.Mode, 8, 32); err == nil {
			mode = os.FileMode(m) & os.ModePerm
		}
	}
	if err := paths.WriteAtomic(d.localPath, plaintext, mode); err != nil {
		return err
	}

	// Restore ownership if running as root and the user/group still exist.
	if d.lastPush != nil && os.Geteuid() == 0 {
		if uid, gid, ok := lookupIDs(d.lastPush.UIDName, d.lastPush.GIDName); ok {
			_ = os.Chown(d.localPath, uid, gid)
		}
	}

	// Restore mtime as a bonus so diffs show meaningful dates.
	if d.lastPush != nil && !d.lastPush.Mtime.IsZero() {
		_ = os.Chtimes(d.localPath, time.Now(), d.lastPush.Mtime)
	}

	md5 := hash.MD5Bytes(plaintext)
	st := mf.Get(d.file.Path)
	st.Pulls[me] = &manifest.PullRecord{At: time.Now().UTC(), MD5: md5}
	fmt.Printf("    -> pulled (md5=%s, mode=%04o)\n", shortHash(md5), mode.Perm())
	return nil
}

func ownerNames(fi os.FileInfo) (string, string) {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ""
	}
	u, _ := user.LookupId(strconv.Itoa(int(stat.Uid)))
	g, _ := user.LookupGroupId(strconv.Itoa(int(stat.Gid)))
	uname := ""
	gname := ""
	if u != nil {
		uname = u.Username
	}
	if g != nil {
		gname = g.Name
	}
	return uname, gname
}

func lookupIDs(uname, gname string) (int, int, bool) {
	u, err := user.Lookup(uname)
	if err != nil {
		return 0, 0, false
	}
	g, err := user.LookupGroup(gname)
	if err != nil {
		return 0, 0, false
	}
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(g.Gid)
	return uid, gid, true
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func fmtSize(fi os.FileInfo) string {
	if fi == nil {
		return "-"
	}
	return fmt.Sprintf("%d", fi.Size())
}

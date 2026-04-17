# key-traveler

Transport secrets (SSH keys, API keys, sensitive configuration files) across
several Linux machines using a USB stick.

Design principle: the USB stick never holds any private key — only `age`
blobs encrypted for *every* enrolled host. Each host keeps its own decryption
key in `~/.config/key-traveler/identity.txt`. Losing the USB stick therefore
exposes no secret.

An unencrypted `manifest.json`, also on the stick, acts as the source of
truth for detecting divergences between the local copy of a file and the
copy on the stick (md5 + mtime + permissions). When a conflict is detected,
an interactive resolution shows a diff and asks which version to keep.

> Looking for something similar? See [COMPARISON.md](COMPARISON.md) for how
> key-traveler relates to chezmoi, passage, git-annex, SOPS, VeraCrypt,
> KeePassXC and others — and when one of them is probably a better fit.

## Threat model

- **Encryption**: [age](https://age-encryption.org) (X25519 +
  ChaCha20-Poly1305), multi-recipient.
- **One key per host**: every machine owns its own X25519 key pair stored in
  `~/.config/key-traveler/identity.txt` with `0600` permissions. The tool
  refuses to run if these permissions have been loosened.
- **Stolen USB stick**: no secret is exposed, because the private key
  required to decrypt is not on the stick.
- **Revoking a host**: remove its pubkey from `config.toml` and re-encrypt
  the whole vault. *(No dedicated command in v1 — achievable by hand plus a
  second `enroll-approve` run.)*
- **Temporary files**: during a conflict the vault copy is decrypted into
  `/dev/shm` (RAM-backed tmpfs, mode `0600`) and removed right after. Plain
  text never lands on persistent disk.

## Installation

Requirements: Go 1.25+ to build, `diff` (package `diffutils`) available on
the target machines for the conflict diff view.

```sh
git clone <repo> key-traveler
cd key-traveler
make build-strip        # produces ./ktraveler (~3.5 MiB)
make install-usb USB=/media/<user>/<label>
```

The binary and the vault data live side by side at the root of the USB
stick. The tool locates the stick automatically by looking for `config.toml`
in:

1. the `--usb <path>` (or `-u <path>`) command-line flag if given;
2. `$KTRAVELER_USB` if set;
3. the directory of the running binary (the normal case — `ktraveler`
   launched directly from the stick);
4. `/media/$USER/*` and `/run/media/$USER/*`.

The `--usb` flag can appear anywhere on the command line and accepts
`--usb=<path>` / `-u=<path>` forms too. Useful when the stick is at an
unusual mount point or when you want to override `$KTRAVELER_USB` for a
single invocation:

```sh
ktraveler --usb /mnt/backup-vault sync
ktraveler list -u=/mnt/backup-vault
```

## Shell completion

The binary can emit its own completion script for bash, zsh and fish:

```sh
# bash — persist (preferred if your ~/.bashrc.d/ is already sourced)
ktraveler completion bash > ~/.bashrc.d/ktraveler-completion.sh

# bash — one-shot for the current shell
source <(ktraveler completion bash)

# zsh (any directory in $fpath works; example with a user-local one)
mkdir -p ~/.zsh/completions
ktraveler completion zsh > ~/.zsh/completions/_ktraveler
# then add to ~/.zshrc, before `compinit`:
#   fpath=(~/.zsh/completions $fpath)

# fish
ktraveler completion fish > ~/.config/fish/completions/ktraveler.fish
```

Completion covers the command list, the `--purge` flag on `remove`, file
and directory completion for arguments that take paths (`init`, `add`,
`remove`), and the three valid shells on `completion`.

## Commands

| Command                               | Action |
|---------------------------------------|--------|
| `ktraveler init <dir>`                | Initialise a fresh USB vault (`config.toml`, `manifest.json`, `vault/`, `pending/`). |
| `ktraveler enroll <name>`             | **First host only.** Generate the local key and register its pubkey. |
| `ktraveler enroll-request <name>`     | On a *new* host: generate the local key and drop an enrollment request into `pending/`. |
| `ktraveler enroll-approve`            | On an *already enrolled* host: accept pending requests and **re-encrypt the whole vault** for the new hosts. |
| `ktraveler add <path>…`               | Track one or more files (paths are stored as `~/…` when under `$HOME`). |
| `ktraveler remove <path>… [--purge]`  | Stop tracking; `--purge` also deletes the encrypted blob. |
| `ktraveler add-pattern <glob>…`       | Track a glob pattern — resolved on every sync, new matches auto-added. |
| `ktraveler remove-pattern <glob>…`    | Stop tracking a pattern; already-matched files stay (use `remove` to drop them). |
| `ktraveler list`                      | List enrolled hosts, patterns and tracked files. |
| `ktraveler status`                    | Show what `sync` would do (dry run, no changes). |
| `ktraveler sync`                      | Interactive sync: automatic fast-forward, prompt on conflicts. |
| `ktraveler push`                      | Force local → vault for every differing file (asks for confirmation on conflicts). |
| `ktraveler pull`                      | Force vault → local for every differing file (asks for confirmation on conflicts). |
| `ktraveler verify`                    | Integrity check: every `.age` blob decrypts and its md5 matches the manifest. |
| `ktraveler completion <shell>`        | Emit a completion script for `bash`, `zsh` or `fish`. |

## Typical setup

### First host (laptop)

```sh
ktraveler init /media/david/SECRETS
KTRAVELER_USB=/media/david/SECRETS ktraveler enroll laptop
ktraveler add ~/.ssh/id_ed25519
ktraveler add ~/.ssh/config
ktraveler add ~/.config/rclone/rclone.conf
ktraveler sync                  # first push
```

### Second host (desktop)

```sh
# 1) on the desktop, with the USB stick plugged in
ktraveler enroll-request desktop

# 2) back on the laptop (same stick plugged in)
ktraveler enroll-approve        # re-encrypts the vault to include desktop

# 3) back on the desktop
ktraveler pull                  # retrieves every secret, permissions restored
```

### Tracking files by pattern

To stop thinking about `add` every time you create a new key, register a glob
pattern — every sync-like command re-evaluates it and auto-adds any new
matches to the config.

```sh
ktraveler add-pattern '~/.ssh/id_*'
# -> matches id_ed25519, id_ed25519.pub, id_rsa, id_rsa.pub …

# Later, after generating a new pair:
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519_work
ktraveler sync
# picked up pattern "~/.ssh/id_*" → ~/.ssh/id_ed25519_work
# picked up pattern "~/.ssh/id_*" → ~/.ssh/id_ed25519_work.pub
#   push     ~/.ssh/id_ed25519_work
#   push     ~/.ssh/id_ed25519_work.pub
```

Quote the pattern so your shell doesn't pre-expand it:

```sh
ktraveler add-pattern '~/.ssh/id_*'      # good — pattern stored literally
ktraveler add-pattern ~/.ssh/id_*        # shell expands NOW; you only snapshot today's matches
```

Syntax follows Go's `filepath.Glob` (`*`, `?`, `[abc]`). Recursive `**` is
not supported — add one pattern per directory if needed.

`remove-pattern` drops the pattern but leaves the files it already matched
in place, so you don't lose history by mistake. Use `remove <path>` to drop
an individual file.

### Day-to-day usage

Before starting work on a machine:

```sh
ktraveler sync
```

Before leaving with the USB stick:

```sh
ktraveler sync
```

## Divergence detection

For every tracked file, `status` and `sync` compare three anchors:

- `md5_local` — digest of the file on the current host;
- `pulls[me].md5` — digest recorded on the last push/pull *from this host*;
- `last_push.md5` — digest of the most recent push *from any host*.

| `md5_local`  | vs `pulls[me]`  | vs `last_push` | Decision                                |
|--------------|-----------------|----------------|------------------------------------------|
| missing      | —               | —              | pull (first time on this host)           |
| = `pulls`    | = `last_push`   | —              | `in-sync`, nothing to do                 |
| = `pulls`    | ≠ `last_push`   | —              | fast-forward pull                        |
| ≠ `pulls`    | = `last_push`   | —              | fast-forward push                        |
| ≠ `pulls`    | ≠ `last_push`   | —              | **conflict** → interactive prompt        |

On a conflict, the tool prints a unified diff between the local copy and the
decrypted vault copy (for text files), or a size/md5/date summary (for
binaries), then prompts for:

- `l` — **local** wins: the local version is encrypted and pushed;
- `v` — **vault** wins: the vault version is decrypted and written locally;
- `s` — **skip**: move on to the next file, nothing is touched;
- `a` — **abort**: stop immediately; actions already committed during the
  session are preserved.

## Permissions and ownership

When pushing, `key-traveler` records in the manifest:

- the Unix mode (e.g. `0600` for an SSH key);
- the owning user and group names;
- the size, mtime and md5.

When pulling, the mode is restored. The user/group are restored only when
the tool runs as root (not needed for user-owned secrets).

Missing parent directories are created automatically with mode `0700`, so
`~/.ssh/` on a fresh machine comes out restrictive enough for SSH to
accept keys placed inside. Already-existing directories are never
modified — ktraveler only creates what is missing.

## USB layout

```
/media/david/SECRETS/
├── ktraveler              # statically linked Go binary
├── config.toml            # tracked files + host pubkeys
├── manifest.json          # md5 + mtime + mode + owner + per-host history
├── pending/               # outstanding enroll-request files
└── vault/
    ├── ssh-id_ed25519.age
    ├── ssh-config.age
    └── …
```

## Per-host layout

```
~/.config/key-traveler/
├── identity.txt           # age private key (mode 0600, never on the USB)
└── hostname               # host name as stored in the manifest
```

Override the location with `XDG_CONFIG_HOME` (useful for tests or sandboxes).

## `config.toml` format

```toml
[vault]
version = 1

[[hosts]]
name = "laptop"
pubkey = "age1…"
enrolled_at = 2026-04-17T12:56:23Z

[[patterns]]
pattern = "~/.ssh/id_*"
added_at = 2026-04-17T12:56:28Z

[[files]]
path = "~/.ssh/id_ed25519"
vault = "ssh-id_ed25519.age"
```

## `manifest.json` format

```json
{
  "files": {
    "~/.ssh/id_ed25519": {
      "last_push": {
        "host": "laptop",
        "at": "2026-04-17T12:56:33Z",
        "md5": "…",
        "mtime": "2026-04-17T12:55:00Z",
        "mode": "0600",
        "uid_name": "david",
        "gid_name": "david",
        "size": 411
      },
      "pulls": {
        "laptop":  { "at": "2026-04-17T12:56:33Z", "md5": "…" },
        "desktop": { "at": "2026-04-17T13:02:14Z", "md5": "…" }
      }
    }
  }
}
```

## Development

```sh
make build       # local binary (not stripped, handy for debugging)
make build-strip # release binary (-s -w -trimpath)
make vet         # go vet ./...
make test        # go test ./...
```

Source tree:

```
main.go                       argument parsing + dispatch
cmd/
  init.go                     ktraveler init
  enroll.go                   enroll / enroll-request / enroll-approve
  add.go                      add / remove / list
  sync.go                     status / sync / push / pull + detection
  verify.go                   vault integrity check
internal/
  config/                     TOML (atomic read/write)
  manifest/                   JSON (atomic read/write)
  vault/                      age encryption, re-encrypt-all
  identity/                   local private key, hostname
  patterns/                   glob expansion, auto-add of new matches
  paths/                      ~ expansion, $HOME contraction, USB detection, WriteAtomic
  hash/                       streaming md5 + binary detection
  diff/                       unified diff via /usr/bin/diff, prompts, tmpfs helpers
```

Go dependencies:

- `filippo.io/age` — encryption
- `github.com/BurntSushi/toml` — TOML parsing
- standard library for everything else

## Known limitations (v1)

- **No version history**: only one state per file lives in the vault.
  Picking "local wins" on a conflict drops the previous vault version.
- **No dedicated `revoke` command** for removing a host (doable by editing
  `config.toml` then forcing an `enroll-approve`-style re-encryption).
- **Linux only** (macOS/Windows untested).
- **No locking on the USB stick**: if two hosts share the same physical
  stick over the network and sync at the same time, atomic writes protect
  each individual file but do not serialise the whole operation. Intended
  usage: one person, one machine at a time.

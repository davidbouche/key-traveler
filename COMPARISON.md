# How key-traveler compares to other tools

The goal of this page is to **situate** key-traveler in a busy landscape of
secret-handling and file-syncing tools, not to argue that it is better than
any of them. Most of the projects mentioned below are mature, well
maintained, and excellent at what they set out to do — often covering a much
broader scope than key-traveler. If one of them fits your workflow, use it;
this page tries to give you enough information to decide.

## The niche key-traveler targets

Key-traveler was built around a specific combination of constraints:

- **Single-user, multiple machines they own themselves** — not teams, not
  sharing across an organisation.
- **Offline transport via a physical USB stick.** No cloud service, no
  server, no peer-to-peer networking.
- **Per-file tracking with original paths and Unix permissions.** Secrets
  are restored in place (e.g. `~/.ssh/id_ed25519` with mode `0600`), not
  inside a vault tree.
- **Per-host encryption keys.** Each machine holds its own age X25519
  private key; the USB stick itself never carries any private key. Losing
  the stick exposes nothing.
- **Interactive conflict resolution per file** when the same tracked file
  has been edited on more than one host since the last sync — diff on
  screen, choose local or vault or skip.

Many of the tools below solve an overlapping problem but make different
trade-offs for very good reasons.

## Summary table

| Tool                          | USB-first offline | Per-host recipients | Tracks files in place with modes | Per-file conflict UI | Main audience                    |
|-------------------------------|:-----------------:|:-------------------:|:--------------------------------:|:--------------------:|----------------------------------|
| **key-traveler**              | ✅                | ✅                  | ✅                               | ✅                   | One user, several Linux boxes    |
| [chezmoi]                     | via git-on-USB    | ✅                  | ✅                               | via git merge tools  | Dotfiles at scale, all platforms |
| [passage]                     | via git-on-USB    | ✅                  | pass-tree only                   | via git merge tools  | Password store heirs of `pass`   |
| [gopass]                      | via git-on-USB    | partial             | pass-tree only                   | via git merge tools  | Teams using pass, richer CLI     |
| [git-annex] + `directory` + gcrypt | ✅           | via gcrypt          | ✅                               | git merge            | Large-file sneakernet            |
| [yadm] + [git-crypt]          | via git-on-USB    | ❌ (shared key)     | ✅                               | git merge            | Dotfiles with encryption         |
| [SOPS]                        | n/a (storage tool)| ✅ (age/KMS)        | values inside files              | n/a                  | Infra-as-code, mixed files       |
| [gocryptfs] / [VeraCrypt]     | ✅                | ❌ (passphrase)     | filesystem transparent           | ❌                   | Encrypted container on removable |
| [KeePassXC] portable          | ✅                | ❌ (master key)     | inside KDBX database             | ❌                   | Personal password vault          |
| [Bitwarden] / [1Password] CLI | ❌ (cloud)        | team model          | inside vault                     | server-side          | Cross-device, mobile-friendly    |

Anything marked "via git-on-USB" means the tool assumes a git remote, and
you can host that remote as a bare repository on the USB stick. The
transport works, but day-to-day it is a git workflow rather than a
stick-centric one.

## Close neighbours

### chezmoi

[chezmoi] is a very actively developed cross-platform dotfiles manager with
thousands of contributors and frequent releases. Some of its strengths that
key-traveler does **not** replicate:

- Runs on Linux, macOS, Windows and BSD.
- Rich **templating** so the same source file can render differently on each
  host (hostname, OS, arch, arbitrary user data). Key-traveler assumes
  identical paths between hosts.
- Built-in `chezmoi init` onboarding for a new machine: one command pulls
  everything from a git remote and applies it.
- Pluggable encryption: **age, gpg, or rage** — with age, per-file recipient
  lists can be configured.
- `chezmoi diff`, `chezmoi merge`, integrations with `vimdiff`, `meld`,
  `kdiff3` and others.
- Mature documentation and a large user base, including in enterprise
  contexts.

Where key-traveler positions differently: it is single-binary, USB-first,
and offers a dedicated interactive `local / vault / skip / abort` prompt on
secret conflicts rather than delegating to a generic merge tool. If you
already use git fluently and want dotfile management with encryption as a
first-class feature, chezmoi is likely the better fit.

### passage (and the `pass` family)

[passage] is a fork of the venerable `pass` password manager that replaces
GPG with age, and inherits the rich `pass` ecosystem — editor integrations,
clipboard helpers, browser extensions, mobile apps, git synchronisation.
Things it does that key-traveler does not:

- Hierarchical namespace (`personal/github.com`, `work/aws/prod`, etc.) with
  excellent ergonomics for passwords specifically.
- Very broad client support via the `pass` protocol — mobile apps, browser
  plugins, OS keyrings.
- Dedicated commands for generation, clipboard handling, OTP.

Key-traveler is not trying to replace a password manager. It transports
arbitrary files with their original paths and permissions (SSH private
keys, config files, tokens embedded in dotfiles), which `pass`-style trees
do not represent natively. If your secrets fit a key/value model, passage
is excellent.

### gopass

[gopass] is a more recent, feature-rich reimplementation of `pass` in Go,
with multi-user support, audit logging, templates and richer CLI UX. Same
structural relationship to key-traveler as passage: complementary rather
than competing. If you need team features inside a `pass`-compatible tree,
gopass is mature and well supported.

### git-annex with a `directory` remote and gcrypt

[git-annex] has explicit, **first-class support** for encrypted sneakernet
workflows via its `directory` special remote plus [gcrypt]. Strengths
key-traveler does not match:

- Tracks arbitrary-size binary content, with **content-addressed storage**
  and deduplication — useful if you need to ship gigabytes of data.
- Multiple remotes side by side: you can sync via USB **and** an S3 bucket
  **and** a self-hosted server, all describing the same repository.
- Very rich metadata, copy policies ("keep at least N copies"), trust levels.
- Years of production use, including in academic and archival settings.

Key-traveler focuses on a much smaller problem — a handful of small
text-and-binary secret files — and tries to stay at "one static binary on
the stick" complexity. If you are comfortable with git and want a robust,
multi-remote syncing layer with encryption, git-annex is the serious
long-term answer.

### yadm + git-crypt (or transcrypt)

[yadm] is a dotfiles manager based directly on git, with a small encrypted
files list processed via [git-crypt] or [transcrypt]. It is simpler than
chezmoi and closer to a plain git workflow. Compared to key-traveler it
uses a **single shared symmetric key** across all machines rather than
per-host recipients, so the trade-off on lost-stick exposure is different.
If you already host your dotfiles in git and just want a small subset
encrypted, yadm is a light, mature option.

### SOPS

[SOPS] (Secrets OPerationS) is a very widely used tool for encrypting
**values inside structured files** (YAML, JSON, ENV, INI, binary). It
supports age, GPG and several cloud KMSs (AWS, GCP, Azure, HashiCorp
Vault). Excellent for infrastructure repositories where a config file is
partly public and partly secret. It is not a transport tool and does not
manage conflict resolution between machines — it is the encryption layer
other workflows (CI pipelines, Ansible, Helm) build on top of.

### Encrypted filesystem containers — gocryptfs, VeraCrypt

[gocryptfs] and [VeraCrypt] produce an encrypted volume you mount, copy
files into, and unmount. On a USB stick this is a well-understood,
user-friendly model with noteworthy properties key-traveler lacks:

- Transparent use — once mounted, any program reads and writes normally.
- Mature, audited implementations (VeraCrypt in particular).
- VeraCrypt supports **hidden volumes** with plausible deniability.

The trade-off is a **single passphrase** securing everything, no per-host
keys, and no divergence detection: the filesystem is authoritative, so if
two hosts write different versions between two mounts the latest mount
wins. For many use cases this is exactly what users want; for others, the
per-file conflict UI of key-traveler is the point.

### KeePassXC portable

[KeePassXC] can run fully portable from a USB stick, offers SSH-agent
integration, TOTP, attachments and a polished GUI. Things it covers that
key-traveler does not:

- GUI and mobile apps (KeePass2Android, Strongbox, etc.).
- Auto-type, browser integration, password generation and audit.
- Attachments stored inside the KDBX database.

Scope difference: KeePassXC is a personal password vault with file
attachments; key-traveler is a transport for files that must be restored
**in place** on disk with their original permissions. The two can coexist
peacefully.

### Bitwarden and 1Password CLI

[Bitwarden] (self-hostable) and [1Password] are full-featured commercial
password managers with excellent cross-device support, mobile apps, team
features and browser integration. They operate **over the network** by
design — which is their strength (seamless sync across phones and
browsers) and the reason they are outside key-traveler's niche, which is
explicitly offline.

## Decision guide

A rough map, not a ranking:

- You want **mobile apps and browser integration**, sync everywhere, team
  sharing → Bitwarden or 1Password.
- You live in git and want **dotfile management with optional
  encryption** across Linux, macOS and Windows → chezmoi.
- You already use **pass** and want its ecosystem plus age → passage or
  gopass.
- You need to sneakernet **gigabytes of content** with multiple remotes and
  are willing to invest in git-annex's model → git-annex + gcrypt.
- You want a **transparent encrypted folder** on a USB stick, one
  passphrase, no conflict logic → VeraCrypt or gocryptfs.
- You want a **personal password vault** with a GUI → KeePassXC portable.
- You want **per-host age keys, files restored in place with their Unix
  permissions, an offline USB stick you can lose without exposing secrets,
  and a per-file interactive conflict prompt** → key-traveler.

If you are undecided between key-traveler and chezmoi — the two most
similar options — try chezmoi first with its age integration. It covers the
larger surface area and is maintained by a much bigger community. Pick
key-traveler if the combination of *offline-only*, *per-host keys*, and
*dedicated conflict UI* is what you actually care about.

[chezmoi]: https://www.chezmoi.io/
[passage]: https://github.com/FiloSottile/passage
[gopass]: https://github.com/gopasspw/gopass
[git-annex]: https://git-annex.branchable.com/
[gcrypt]: https://git-annex.branchable.com/tips/fully_encrypted_git_repositories_with_gcrypt/
[yadm]: https://yadm.io/
[git-crypt]: https://github.com/AGWA/git-crypt
[transcrypt]: https://github.com/elasticdog/transcrypt
[SOPS]: https://github.com/getsops/sops
[gocryptfs]: https://nuetzlich.net/gocryptfs/
[VeraCrypt]: https://veracrypt.io/
[KeePassXC]: https://keepassxc.org/
[Bitwarden]: https://bitwarden.com/
[1Password]: https://1password.com/

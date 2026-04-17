package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/david/key-traveler/cmd"
)

type command struct {
	name     string
	aliases  []string
	argsHint string
	summary  string
	details  string   // multi-line body, no leading/trailing newline
	examples []string // each rendered as "  $ <example>"
	run      func(args []string) error
}

// registry is the single source of truth for the CLI. The top-level usage,
// per-command help, and the dispatcher all walk this slice.
var registry = []command{
	{
		name:     "init",
		argsHint: "<usb-path>",
		summary:  "initialize a new USB vault",
		details: `Create the layout (config.toml, manifest.json, vault/, pending/) at the
given path. The directory is created if it does not exist. Refuses to
overwrite an existing config.toml.

After init, export KTRAVELER_USB so subsequent commands find the vault
without a full path; the init output suggests a shell-specific snippet.`,
		examples: []string{
			"ktraveler init /media/david/KEY-TRAVELER",
			"ktraveler init /run/media/david/ktraveler",
		},
		run: cmd.Init,
	},
	{
		name:     "enroll",
		argsHint: "<host-name>",
		summary:  "enroll THIS host (first host only)",
		details: `Generate a fresh age X25519 identity at ~/.config/key-traveler/identity.txt
(mode 0600) and register its public key in config.toml under <host-name>.
Refuses to run if another host is already enrolled — use enroll-request
and enroll-approve for additional hosts.`,
		examples: []string{"ktraveler enroll laptop"},
		run:      cmd.Enroll,
	},
	{
		name:     "enroll-request",
		argsHint: "<host-name>",
		summary:  "request enrollment of a new host",
		details: `Run this on the NEW host. It generates a local identity and drops a
JSON request file in pending/ on the USB stick. Then take the stick to an
already-enrolled host and run enroll-approve.`,
		examples: []string{"ktraveler enroll-request desktop"},
		run:      cmd.EnrollRequest,
	},
	{
		name:    "enroll-approve",
		summary: "approve pending enrollment requests",
		details: `Run this on an ALREADY-ENROLLED host. It reads every pending/*.json,
adds the new hosts' pubkeys to config.toml, and re-encrypts every vault
blob so the new hosts can decrypt. Approved request files are removed
from pending/ at the end.`,
		examples: []string{"ktraveler enroll-approve"},
		run:      cmd.EnrollApprove,
	},
	{
		name:     "add",
		argsHint: "<path>...",
		summary:  "track one or more files",
		details: `Record each file as tracked in config.toml. Files must already exist.
Paths under $HOME are stored in ~/… form so they stay portable across
hosts with the same home layout. Tracking does not push the file — run
sync or push after.`,
		examples: []string{
			"ktraveler add ~/.ssh/id_ed25519",
			"ktraveler add ~/.ssh/config ~/.config/rclone/rclone.conf",
		},
		run: cmd.Add,
	},
	{
		name:     "remove",
		aliases:  []string{"rm"},
		argsHint: "<path>... [--purge]",
		summary:  "stop tracking one or more files",
		details: `Drop the file(s) from config.toml and from manifest.json.

Flags:
  --purge   also delete the encrypted blob from vault/ (default: keep it).

Without --purge the blob is kept so other hosts can still pull a last
copy if needed. Use --purge when you want the secret fully gone.`,
		examples: []string{
			"ktraveler remove ~/.ssh/old_key",
			"ktraveler remove --purge ~/.ssh/old_key",
		},
		run: cmd.Remove,
	},
	{
		name:     "add-pattern",
		argsHint: "<glob>...",
		summary:  "track a glob pattern (re-evaluated on every sync)",
		details: `Register a filepath.Glob pattern. Every sync / push / pull / list /
status call re-resolves it; newly matching files are auto-added to
config.toml and pushed on the same run.

Quote the pattern so your shell does NOT pre-expand it — otherwise you
only snapshot today's matches, which defeats the purpose.

Syntax: *, ?, [abc] are supported. Recursive ** is not — register one
pattern per directory if you need several.`,
		examples: []string{
			"ktraveler add-pattern '~/.ssh/id_*'",
			"ktraveler add-pattern '~/.config/myapp/*.token'",
		},
		run: cmd.AddPattern,
	},
	{
		name:     "remove-pattern",
		argsHint: "<glob>...",
		summary:  "stop tracking a pattern",
		details: `Drop the pattern from config.toml. Files that were already added from
that pattern stay tracked — use `+"`remove <path>`"+` to drop them
individually. This is intentional, to avoid silently deleting vault
content.`,
		examples: []string{"ktraveler remove-pattern '~/.ssh/id_*'"},
		run:      cmd.RemovePattern,
	},
	{
		name:    "list",
		aliases: []string{"ls"},
		summary: "list hosts, patterns and tracked files",
		details: `Print the vault summary: enrolled hosts (with their pubkey fingerprint),
registered glob patterns (with the concrete files each currently matches),
and every individually-tracked file with its vault blob name.

Read-only. Patterns are resolved in memory only; run sync to persist new
matches.`,
		examples: []string{"ktraveler list"},
		run:      cmd.List,
	},
	{
		name:    "status",
		summary: "show what sync would do (dry run)",
		details: `For each tracked file, compare the local copy's md5 against the
manifest's last-push record and this host's last-pull record. Prints a
per-file action label: in-sync, push, pull, conflict, or missing.
Displays pending pattern matches as "(pending)". Makes no changes.`,
		examples: []string{"ktraveler status"},
		run:      cmd.Status,
	},
	{
		name:    "sync",
		summary: "interactive sync with conflict resolution",
		details: `Resolve patterns, then for each tracked file:
  - in-sync            skipped.
  - fast-forward push  file pushed automatically.
  - fast-forward pull  file pulled automatically.
  - conflict           print a unified diff between local and the
                       decrypted vault copy, prompt for:
                         [l] local wins   (push)
                         [v] vault wins   (pull)
                         [s] skip         (no change)
                         [a] abort        (stop; already-committed
                                           actions in this session are kept)

Binary files show size + md5 + dates instead of a diff. The decrypted
vault copy is written to /dev/shm (RAM) during the conflict display and
removed right after.`,
		examples: []string{"ktraveler sync"},
		run:      cmd.Sync,
	},
	{
		name:    "push",
		summary: "force local -> vault for every differing file",
		details: `Like sync, but always prefers the local version. Asks y/N on files
where the vault would be overwritten (conflicts or pull-pending states);
plain fast-forward pushes run without asking.`,
		examples: []string{"ktraveler push"},
		run:      cmd.Push,
	},
	{
		name:    "pull",
		summary: "force vault -> local for every differing file",
		details: `Like sync, but always prefers the vault version. Asks y/N on files
where local would be overwritten (conflicts or push-pending states);
plain fast-forward pulls run without asking.

Restores the recorded Unix mode on every pulled file. Owner/group are
restored only when the tool runs as root.`,
		examples: []string{"ktraveler pull"},
		run:      cmd.Pull,
	},
	{
		name:    "verify",
		summary: "check vault integrity",
		details: `Walk every tracked file and ensure its vault blob (1) decrypts with the
local identity and (2) has an md5 that matches manifest.last_push.md5.
Reports ok / FAIL per file and exits non-zero on any failure.`,
		examples: []string{"ktraveler verify"},
		run:      cmd.Verify,
	},
	{
		name:     "completion",
		argsHint: "<bash|zsh|fish>",
		summary:  "emit a shell completion script",
		details: `Prints a completion script for the requested shell to stdout. See the
README for installation paths. Covers top-level commands, the --purge
flag on remove, path/directory completion on init/add/remove, and shell
names on completion itself.`,
		examples: []string{
			"ktraveler completion bash > ~/.bashrc.d/ktraveler-completion.sh",
			"source <(ktraveler completion bash)",
		},
		run: cmd.Completion,
	},
}

const usagePreamble = `ktraveler — transport encrypted secrets on a USB stick.

Usage:
  ktraveler <command> [args...]
  ktraveler <command> --help           show detailed help for a command
  ktraveler help [<command>]           same, alternate form

Commands:
`

const usageSuffix = `
Environment:
  KTRAVELER_USB    path to the vault root; overrides auto-detection.
                   Auto-detection looks at the binary's directory, then
                   /media/$USER/* and /run/media/$USER/*.
  XDG_CONFIG_HOME  base directory for the local identity; defaults to
                   ~/.config. The private key lives in
                   <base>/key-traveler/identity.txt (mode 0600 required).
`

func main() {
	if len(os.Args) < 2 {
		printTopUsage(os.Stdout)
		os.Exit(2)
	}

	first := os.Args[1]
	rest := os.Args[2:]

	// `ktraveler help` or `ktraveler -h/--help`.
	if first == "-h" || first == "--help" || first == "help" {
		if len(rest) == 0 {
			printTopUsage(os.Stdout)
			return
		}
		if c := lookup(rest[0]); c != nil {
			printCommandHelp(os.Stdout, c)
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command %q\n", rest[0])
		printTopUsage(os.Stderr)
		os.Exit(2)
	}

	c := lookup(first)
	if c == nil {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", first)
		printTopUsage(os.Stderr)
		os.Exit(2)
	}

	// `ktraveler <cmd> --help|-h` at any position after the command.
	for _, a := range rest {
		if a == "-h" || a == "--help" {
			printCommandHelp(os.Stdout, c)
			return
		}
	}

	if err := c.run(rest); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func lookup(name string) *command {
	for i := range registry {
		if registry[i].name == name {
			return &registry[i]
		}
		for _, a := range registry[i].aliases {
			if a == name {
				return &registry[i]
			}
		}
	}
	return nil
}

func printTopUsage(w *os.File) {
	fmt.Fprint(w, usagePreamble)
	// Two-column layout: "name args  — summary".
	nameWidth := 0
	for _, c := range registry {
		left := c.name
		if c.argsHint != "" {
			left += " " + c.argsHint
		}
		if len(left) > nameWidth {
			nameWidth = len(left)
		}
	}
	for _, c := range registry {
		left := c.name
		if c.argsHint != "" {
			left += " " + c.argsHint
		}
		fmt.Fprintf(w, "  %-*s  %s\n", nameWidth, left, c.summary)
	}
	fmt.Fprint(w, usageSuffix)
}

func printCommandHelp(w *os.File, c *command) {
	fmt.Fprintf(w, "ktraveler %s", c.name)
	if c.argsHint != "" {
		fmt.Fprintf(w, " %s", c.argsHint)
	}
	fmt.Fprintf(w, "\n\n  %s\n", c.summary)
	if len(c.aliases) > 0 {
		fmt.Fprintf(w, "\nAliases: %s\n", strings.Join(c.aliases, ", "))
	}
	if c.details != "" {
		fmt.Fprintf(w, "\n%s\n", c.details)
	}
	if len(c.examples) > 0 {
		fmt.Fprintln(w, "\nExamples:")
		for _, e := range c.examples {
			fmt.Fprintf(w, "  $ %s\n", e)
		}
	}
}

package main

import (
	"errors"
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
		argsHint: "<list|request|approve> [args...]",
		summary:  "manage host enrollments (list / request / approve)",
		details: `Subcommands:

  enroll list
      Print enrolled hosts and pending requests.

  enroll request <host-name>
      Generate this host's local age X25519 identity
      (~/.config/key-traveler/identity.txt, mode 0600) and register it.
      If the vault has no hosts yet, the request is auto-approved (first
      host). Otherwise, a JSON file is dropped in pending/ for approval.

  enroll approve <host-name>
  enroll approve --all
      Run on an already-enrolled host. Adds the named (or every) pending
      request to config.toml and re-encrypts every vault blob so the new
      host(s) can decrypt. Approved request files are removed from
      pending/.`,
		examples: []string{
			"ktraveler enroll request laptop",
			"ktraveler enroll list",
			"ktraveler enroll approve desktop",
			"ktraveler enroll approve --all",
		},
		run: cmd.Enroll,
	},
	{
		name:     "add",
		argsHint: "<path-or-pattern>...",
		summary:  "track files and / or glob patterns",
		details: `Each argument is inspected: if it contains a glob metachar (* or ?) it
is registered as a pattern ([[patterns]] in config.toml); otherwise it
is registered as a literal file ([[files]]). The two can be mixed in
the same call.

Literal files must already exist locally. Paths under $HOME are stored
in ~/… form so they stay portable across hosts with the same home
layout.

Quote patterns so your shell does NOT pre-expand them — otherwise you
only snapshot today's matches, which defeats the purpose. Supported
glob syntax is filepath.Glob: *, ?, [abc]. Recursive ** is not
supported — register one pattern per directory if needed.

Newly-added items are not pushed immediately; run ktraveler sync after.`,
		examples: []string{
			"ktraveler add ~/.ssh/config",
			"ktraveler add '~/.ssh/id_*'",
			"ktraveler add ~/.barry.toml '~/.config/myapp/*.token'",
		},
		run: cmd.Add,
	},
	{
		name:     "remove",
		aliases:  []string{"rm"},
		argsHint: "<path-or-pattern>... [--purge]",
		summary:  "stop tracking files and / or patterns",
		details: `Mirrors add: arguments containing * or ? are treated as registered
patterns and are removed from [[patterns]]; literal paths are removed
from [[files]] and from manifest.json.

Flags:
  --purge   for file removals, also delete the encrypted blob from
            vault/. Default: keep the blob so other hosts can still
            pull one last copy if they need to.

Removing a pattern does NOT drop the files it already matched — that
is deliberate, to avoid silently losing vault content. Use individual
file removals if you also want to drop those.`,
		examples: []string{
			"ktraveler remove ~/.ssh/old_key",
			"ktraveler remove --purge ~/.ssh/old_key",
			"ktraveler remove '~/.ssh/id_*'",
		},
		run: cmd.Remove,
	},
	{
		name:    "list",
		aliases: []string{"ls"},
		summary: "list hosts, patterns and tracked files",
		details: `Print the vault summary: enrolled hosts (with their pubkey), registered
glob patterns (with the concrete files each currently matches, and
"(new, pending)" for fresh matches not yet persisted), and every
individually-tracked file with its vault blob name.

Read-only. Patterns are resolved in memory; run ktraveler sync to
persist new matches and push them.`,
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
		name:     "sync",
		argsHint: "[--push-only | --pull-only]",
		summary:  "synchronise local and vault (push + pull, interactive on conflict)",
		details: `Resolves registered patterns (auto-adding newly-matching files), then
for each tracked file:
  - in-sync            skipped.
  - fast-forward push  local newer than vault -> push automatically.
  - fast-forward pull  vault newer than local -> pull automatically.
  - conflict           both sides advanced. Print a unified diff between
                       local and the decrypted vault copy, then prompt:
                         [l] local wins   (push)
                         [v] vault wins   (pull)
                         [s] skip         (no change)
                         [a] abort        (stop; actions already committed
                                           during this session are kept)

Flags:
  --push-only   only propagate local -> vault. Pull-pending files are
                skipped; push-pending files proceed automatically;
                conflicts ask for confirmation before overwriting the
                vault version.
  --pull-only   symmetric: only propagate vault -> local. Pull-pending
                files proceed automatically; push-pending and conflicts
                ask before overwriting the local file.

Binary files show size + md5 + dates instead of a textual diff. The
decrypted vault copy is written to /dev/shm (RAM-backed tmpfs) during
the conflict display and removed immediately after.

Pulls create missing parent directories with mode 0700 (safe for
secrets); the file itself is restored with its originally-pushed mode.`,
		examples: []string{
			"ktraveler sync",
			"ktraveler sync --push-only",
			"ktraveler sync --pull-only",
		},
		run: cmd.Sync,
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
  ktraveler [global flags] <command> [args...]
  ktraveler <command> --help           show detailed help for a command
  ktraveler help [<command>]           same, alternate form

Global flags (accepted anywhere on the command line):
  -u, --usb <path>   point at the vault root for this invocation;
                     equivalent to KTRAVELER_USB=<path>. Accepts
                     --usb=<path> and -u=<path> too.

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
	args, err := extractGlobalFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	if len(args) == 0 {
		printTopUsage(os.Stdout)
		os.Exit(2)
	}

	first := args[0]
	rest := args[1:]

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

// extractGlobalFlags pulls any -u/--usb occurrences out of args (in any
// position) and sets KTRAVELER_USB accordingly, returning the remaining
// arguments ready for normal dispatch. Accepts: "-u PATH", "--usb PATH",
// "-u=PATH", "--usb=PATH".
func extractGlobalFlags(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-u" || a == "--usb":
			if i+1 >= len(args) {
				return nil, errors.New(a + " requires a path")
			}
			if err := os.Setenv("KTRAVELER_USB", args[i+1]); err != nil {
				return nil, err
			}
			i++
		case strings.HasPrefix(a, "--usb="):
			if err := os.Setenv("KTRAVELER_USB", strings.TrimPrefix(a, "--usb=")); err != nil {
				return nil, err
			}
		case strings.HasPrefix(a, "-u="):
			if err := os.Setenv("KTRAVELER_USB", strings.TrimPrefix(a, "-u=")); err != nil {
				return nil, err
			}
		default:
			out = append(out, a)
		}
	}
	return out, nil
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

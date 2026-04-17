package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/david/key-traveler/internal/config"
	"github.com/david/key-traveler/internal/manifest"
)

// Init bootstraps a new USB root at the given directory.
func Init(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler init <usb-path>")
	}
	root, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving %s: %w", args[0], err)
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", root, err)
	}
	if err := os.MkdirAll(filepath.Join(root, config.VaultDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, config.PendingDir), 0o755); err != nil {
		return err
	}

	cfgPath := filepath.Join(root, config.FileName)
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing %s", cfgPath)
	}

	cfg := &config.Config{Vault: config.VaultMeta{Version: config.SchemaVersion}}
	if err := config.Save(root, cfg); err != nil {
		return err
	}
	if err := manifest.Save(root, manifest.New()); err != nil {
		return err
	}

	printPostInit(root)
	return nil
}

func printPostInit(root string) {
	fmt.Printf("✓ Vault initialized at %s\n\n", root)

	fmt.Println("Contents:")
	fmt.Println("  config.toml    — tracked files, glob patterns and enrolled hosts")
	fmt.Println("  manifest.json  — sync ledger (md5, mtime, mode, owner per file)")
	fmt.Println("  vault/         — age-encrypted blobs")
	fmt.Println("  pending/       — enrollment requests awaiting approval")
	fmt.Println()

	fmt.Println("ktraveler locates the vault automatically when launched from the USB")
	fmt.Println("stick. If you prefer running the binary from $PATH, export KTRAVELER_USB")
	fmt.Println("in your shell so you don't have to repeat it on every command:")
	fmt.Println()
	for _, line := range shellSuggestions(root) {
		fmt.Println("    " + line)
	}
	fmt.Println()
	fmt.Println("Then restart your shell (or re-source the rc file) and enroll this host:")
	fmt.Println()
	fmt.Println("    ktraveler enroll <host-name>")
	fmt.Println()
	fmt.Printf("For one-shot use without touching your rc, prefix the command instead:\n\n")
	fmt.Printf("    KTRAVELER_USB=%s ktraveler enroll <host-name>\n", shellQuote(root))
}

// shellSuggestions returns one or two suggested commands to persist
// KTRAVELER_USB, adapted to the detected shell and to whether a ~/.bashrc.d/
// drop-in directory already exists on this host.
func shellSuggestions(root string) []string {
	shell := filepath.Base(os.Getenv("SHELL"))
	home, _ := os.UserHomeDir()

	exportLine := fmt.Sprintf(`export KTRAVELER_USB=%s`, shellQuote(root))

	switch shell {
	case "zsh":
		return []string{echoAppend(exportLine, "~/.zshrc")}
	case "fish":
		return []string{fmt.Sprintf(`set -Ux KTRAVELER_USB %s`, shellQuote(root))}
	}

	// Default to bash-style suggestions.
	bashrcD := filepath.Join(home, ".bashrc.d")
	if fi, err := os.Stat(bashrcD); err == nil && fi.IsDir() {
		return []string{echoWrite(exportLine, "~/.bashrc.d/ktraveler.sh")}
	}
	return []string{
		echoAppend(exportLine, "~/.bashrc"),
		`# or, if you keep shell fragments in ~/.bashrc.d/ sourced from .bashrc:`,
		`mkdir -p ~/.bashrc.d && ` + echoWrite(exportLine, "~/.bashrc.d/ktraveler.sh"),
	}
}

// echoWrite builds an `echo "..." > dest` command that emits line verbatim.
func echoWrite(line, dest string) string {
	return fmt.Sprintf(`echo "%s" > %s`, doubleQuoteEscape(line), dest)
}

// echoAppend builds an `echo "..." >> dest` command.
func echoAppend(line, dest string) string {
	return fmt.Sprintf(`echo "%s" >> %s`, doubleQuoteEscape(line), dest)
}

// shellQuote returns a form of s that is safe to inline inside a shell word.
// Plain strings are returned unchanged; anything containing whitespace or
// shell metacharacters gets single-quoted (with embedded single quotes
// rewritten as the standard '\'' sequence).
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, ` "'$\`+"`"+`&;<>|#*?[](){}`+"\t\n") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// doubleQuoteEscape escapes the four characters that keep special meaning
// inside bash double quotes, so the string can safely be wrapped in "...".
func doubleQuoteEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
		`$`, `\$`,
		`"`, `\"`,
	)
	return r.Replace(s)
}

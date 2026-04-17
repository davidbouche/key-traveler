package commands

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
)

//go:embed completions
var completionFS embed.FS

// Completion prints the completion script for the requested shell to stdout.
// Supported values: bash, zsh, fish.
func Completion(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: ktraveler completion <bash|zsh|fish>")
	}

	var path string
	switch args[0] {
	case "bash":
		path = "completions/ktraveler.bash"
	case "zsh":
		path = "completions/ktraveler.zsh"
	case "fish":
		path = "completions/ktraveler.fish"
	default:
		return fmt.Errorf("unsupported shell %q (choose bash, zsh or fish)", args[0])
	}

	f, err := completionFS.Open(path)
	if err != nil {
		return fmt.Errorf("loading %s completion: %w", args[0], err)
	}
	defer f.Close()
	_, err = io.Copy(os.Stdout, f)
	return err
}

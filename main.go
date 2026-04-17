package main

import (
	"fmt"
	"os"

	"github.com/david/key-traveler/cmd"
)

const usage = `ktraveler — transport encrypted secrets on a USB stick.

Usage:
  ktraveler init <usb-path>              initialize a new USB vault
  ktraveler enroll <host-name>           enroll this host (first host only)
  ktraveler enroll-request <host-name>   request enrollment of a new host
  ktraveler enroll-approve               approve pending requests (from an existing host)
  ktraveler add <path>...                track file(s) — use ~ for home-relative paths
  ktraveler remove <path>... [--purge]   stop tracking file(s)
  ktraveler add-pattern <glob>...        track a glob pattern — resolved on every sync
  ktraveler remove-pattern <glob>...     stop tracking a pattern (already-matched files kept)
  ktraveler list                         list hosts, patterns and tracked files
  ktraveler status                       show what sync would do (dry run)
  ktraveler sync                         interactive sync with conflict resolution
  ktraveler push                         force local → vault (with confirmation on conflicts)
  ktraveler pull                         force vault → local (with confirmation on conflicts)
  ktraveler verify                       check vault integrity (decrypt + md5)
  ktraveler completion <shell>           emit a completion script (bash, zsh or fish)

Set KTRAVELER_USB to point at the vault root if auto-detection fails.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	args := os.Args[2:]
	var err error
	switch os.Args[1] {
	case "init":
		err = cmd.Init(args)
	case "enroll":
		err = cmd.Enroll(args)
	case "enroll-request":
		err = cmd.EnrollRequest(args)
	case "enroll-approve":
		err = cmd.EnrollApprove(args)
	case "add":
		err = cmd.Add(args)
	case "remove", "rm":
		err = cmd.Remove(args)
	case "add-pattern":
		err = cmd.AddPattern(args)
	case "remove-pattern":
		err = cmd.RemovePattern(args)
	case "list", "ls":
		err = cmd.List(args)
	case "status":
		err = cmd.Status(args)
	case "sync":
		err = cmd.Sync(args)
	case "push":
		err = cmd.Push(args)
	case "pull":
		err = cmd.Pull(args)
	case "verify":
		err = cmd.Verify(args)
	case "completion":
		err = cmd.Completion(args)
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

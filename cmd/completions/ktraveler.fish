# fish completion for ktraveler
#
# Install:
#   ktraveler completion fish > ~/.config/fish/completions/ktraveler.fish

# Global flag: --usb / -u takes a directory
complete -c ktraveler -s u -l usb -x -a '(__fish_complete_directories)' \
    -d 'point at the vault root (like KTRAVELER_USB)'

# Top-level commands
complete -c ktraveler -n '__fish_use_subcommand' -a init       -d 'initialize a new USB vault'
complete -c ktraveler -n '__fish_use_subcommand' -a enroll     -d 'manage host enrollments (list / request / approve)'
complete -c ktraveler -n '__fish_use_subcommand' -a add        -d 'track files and / or glob patterns'
complete -c ktraveler -n '__fish_use_subcommand' -a remove     -d 'stop tracking files and / or patterns'
complete -c ktraveler -n '__fish_use_subcommand' -a rm         -d 'alias for remove'
complete -c ktraveler -n '__fish_use_subcommand' -a list       -d 'list hosts, patterns and tracked files'
complete -c ktraveler -n '__fish_use_subcommand' -a ls         -d 'alias for list'
complete -c ktraveler -n '__fish_use_subcommand' -a status     -d 'show what sync would do (dry run)'
complete -c ktraveler -n '__fish_use_subcommand' -a sync       -d 'synchronise local and vault'
complete -c ktraveler -n '__fish_use_subcommand' -a verify     -d 'check vault integrity'
complete -c ktraveler -n '__fish_use_subcommand' -a completion -d 'emit a shell completion script'
complete -c ktraveler -n '__fish_use_subcommand' -a help       -d 'print usage'

# init takes a directory argument
complete -c ktraveler -n '__fish_seen_subcommand_from init' -x -a '(__fish_complete_directories)'

# enroll subcommands
complete -c ktraveler -n '__fish_seen_subcommand_from enroll; and not __fish_seen_subcommand_from list request approve' \
    -a 'list'    -d 'print enrolled hosts and pending requests'
complete -c ktraveler -n '__fish_seen_subcommand_from enroll; and not __fish_seen_subcommand_from list request approve' \
    -a 'request' -d 'register this host (auto-approved if first)'
complete -c ktraveler -n '__fish_seen_subcommand_from enroll; and not __fish_seen_subcommand_from list request approve' \
    -a 'approve' -d 'approve a pending request'
complete -c ktraveler -n '__fish_seen_subcommand_from approve' -l all -d 'approve every pending request'

# add / remove take files
complete -c ktraveler -n '__fish_seen_subcommand_from add'       -F
complete -c ktraveler -n '__fish_seen_subcommand_from remove rm' -F
complete -c ktraveler -n '__fish_seen_subcommand_from remove rm' -l purge -d 'also delete the encrypted blob'

# sync flags
complete -c ktraveler -n '__fish_seen_subcommand_from sync' -l push-only -d 'only propagate local -> vault'
complete -c ktraveler -n '__fish_seen_subcommand_from sync' -l pull-only -d 'only propagate vault -> local'

# completion <shell>
complete -c ktraveler -n '__fish_seen_subcommand_from completion' -x -a 'bash zsh fish'

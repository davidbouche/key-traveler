# fish completion for ktraveler
#
# Install:
#   ktraveler completion fish > ~/.config/fish/completions/ktraveler.fish

# Top-level commands
complete -c ktraveler -n '__fish_use_subcommand' -a init            -d 'initialize a new USB vault'
complete -c ktraveler -n '__fish_use_subcommand' -a enroll          -d 'enroll this host (first host only)'
complete -c ktraveler -n '__fish_use_subcommand' -a enroll-request  -d 'request enrollment of a new host'
complete -c ktraveler -n '__fish_use_subcommand' -a enroll-approve  -d 'approve pending enrollment requests'
complete -c ktraveler -n '__fish_use_subcommand' -a add             -d 'track one or more files'
complete -c ktraveler -n '__fish_use_subcommand' -a remove          -d 'stop tracking one or more files'
complete -c ktraveler -n '__fish_use_subcommand' -a rm              -d 'alias for remove'
complete -c ktraveler -n '__fish_use_subcommand' -a add-pattern     -d 'track a glob pattern'
complete -c ktraveler -n '__fish_use_subcommand' -a remove-pattern  -d 'stop tracking a glob pattern'
complete -c ktraveler -n '__fish_use_subcommand' -a list            -d 'list hosts, patterns and tracked files'
complete -c ktraveler -n '__fish_use_subcommand' -a ls              -d 'alias for list'
complete -c ktraveler -n '__fish_use_subcommand' -a status          -d 'show what sync would do (dry run)'
complete -c ktraveler -n '__fish_use_subcommand' -a sync            -d 'interactive sync with conflict resolution'
complete -c ktraveler -n '__fish_use_subcommand' -a push            -d 'force local -> vault'
complete -c ktraveler -n '__fish_use_subcommand' -a pull            -d 'force vault -> local'
complete -c ktraveler -n '__fish_use_subcommand' -a verify          -d 'check vault integrity'
complete -c ktraveler -n '__fish_use_subcommand' -a completion      -d 'emit a shell completion script'
complete -c ktraveler -n '__fish_use_subcommand' -a help            -d 'print usage'

# init takes a directory argument
complete -c ktraveler -n '__fish_seen_subcommand_from init'            -x -a '(__fish_complete_directories)'

# add takes files
complete -c ktraveler -n '__fish_seen_subcommand_from add'             -F

# remove/rm: files + --purge flag
complete -c ktraveler -n '__fish_seen_subcommand_from remove rm'       -F
complete -c ktraveler -n '__fish_seen_subcommand_from remove rm' -l purge -d 'also delete the encrypted blob'

# completion takes a shell name
complete -c ktraveler -n '__fish_seen_subcommand_from completion'      -x -a 'bash zsh fish'

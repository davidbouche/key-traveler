#compdef ktraveler
#
# zsh completion for ktraveler
#
# Install (example with user-local fpath):
#   mkdir -p ~/.zsh/completions
#   ktraveler completion zsh > ~/.zsh/completions/_ktraveler
#   # then in ~/.zshrc, before `autoload -U compinit && compinit`:
#   fpath=(~/.zsh/completions $fpath)

_ktraveler() {
    local -a commands
    commands=(
        'init:initialize a new USB vault'
        'enroll:enroll this host (first host only)'
        'enroll-request:request enrollment of a new host'
        'enroll-approve:approve pending enrollment requests'
        'add:track one or more files'
        'remove:stop tracking one or more files'
        'rm:alias for remove'
        'add-pattern:track a glob pattern'
        'remove-pattern:stop tracking a glob pattern'
        'list:list hosts, patterns and tracked files'
        'ls:alias for list'
        'status:show what sync would do (dry run)'
        'sync:interactive sync with conflict resolution'
        'push:force local -> vault for differing files'
        'pull:force vault -> local for differing files'
        'verify:check vault integrity'
        'completion:emit a shell completion script'
        'help:print usage'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'ktraveler command' commands
        return
    fi

    case "${words[2]}" in
        init)
            _directories
            ;;
        add)
            _files
            ;;
        remove|rm)
            _arguments \
                '--purge[also delete the encrypted blob from the vault]' \
                '*:file:_files'
            ;;
        completion)
            if (( CURRENT == 3 )); then
                _values 'shell' 'bash' 'zsh' 'fish'
            fi
            ;;
    esac
}

_ktraveler "$@"

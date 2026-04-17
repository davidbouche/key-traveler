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
    _arguments -C \
        '(-u --usb)'{-u,--usb}'[point at the vault root (like KTRAVELER_USB)]:vault path:_directories' \
        '1:command:->cmd' \
        '*::arg:->args'

    case "$state" in
    cmd)
        local -a commands
        commands=(
            'init:initialize a new USB vault'
            'enroll:manage host enrollments (list / request / approve)'
            'add:track files and / or glob patterns'
            'remove:stop tracking files and / or patterns'
            'rm:alias for remove'
            'list:list hosts, patterns and tracked files'
            'ls:alias for list'
            'status:show what sync would do (dry run)'
            'sync:synchronise local and vault'
            'verify:check vault integrity'
            'completion:emit a shell completion script'
            'help:print usage'
        )
        _describe -t commands 'ktraveler command' commands
        ;;
    args)
        case "${words[1]}" in
        init)
            _directories
            ;;
        enroll)
            if (( CURRENT == 2 )); then
                local -a subs
                subs=(
                    'list:print enrolled hosts and pending requests'
                    'request:register this host (auto-approved if first host)'
                    'approve:approve a pending request'
                )
                _describe -t enroll-sub 'enroll subcommand' subs
            elif [[ "${words[2]}" == "approve" ]]; then
                _values 'approve target' '--all[approve every pending request]'
            fi
            ;;
        add)
            _files
            ;;
        remove|rm)
            _arguments \
                '--purge[also delete the encrypted blob]' \
                '*:file:_files'
            ;;
        sync)
            _values 'sync mode' '--push-only[only propagate local -> vault]' '--pull-only[only propagate vault -> local]'
            ;;
        completion)
            if (( CURRENT == 2 )); then
                _values 'shell' 'bash' 'zsh' 'fish'
            fi
            ;;
        help)
            _values 'command' init enroll add remove list status sync verify completion
            ;;
        esac
        ;;
    esac
}

_ktraveler "$@"

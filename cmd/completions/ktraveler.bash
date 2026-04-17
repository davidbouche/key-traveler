# bash completion for ktraveler
#
# Install:
#   ktraveler completion bash > ~/.bashrc.d/ktraveler-completion.sh
#   (or)
#   source <(ktraveler completion bash)

_ktraveler_complete() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    local commands="init enroll enroll-request enroll-approve add remove rm add-pattern remove-pattern list ls status sync push pull verify completion help"

    if [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
        return
    fi

    local cmd="${COMP_WORDS[1]}"
    case "$cmd" in
        init)
            COMPREPLY=($(compgen -d -- "$cur"))
            ;;
        add)
            COMPREPLY=($(compgen -f -- "$cur"))
            ;;
        remove|rm)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=($(compgen -W "--purge" -- "$cur"))
            else
                COMPREPLY=($(compgen -f -- "$cur"))
            fi
            ;;
        completion)
            if [[ $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            fi
            ;;
    esac
}

complete -F _ktraveler_complete ktraveler

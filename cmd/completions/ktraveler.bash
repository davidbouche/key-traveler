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

    local commands="init enroll add remove rm list ls status sync verify completion help"
    local global_flags="--usb -u"

    # After --usb or -u: complete directories.
    if [[ "$prev" == "--usb" || "$prev" == "-u" ]]; then
        COMPREPLY=($(compgen -d -- "$cur"))
        return
    fi

    # Locate the effective top-level command, skipping global flag pairs.
    local cmd="" cmd_idx=0 i=1
    while (( i < COMP_CWORD )); do
        local w="${COMP_WORDS[i]}"
        case "$w" in
            -u|--usb)     ((i+=2)); continue ;;
            --usb=*|-u=*) ((i++));  continue ;;
        esac
        cmd="$w"
        cmd_idx=$i
        break
    done

    if [[ -z "$cmd" ]]; then
        COMPREPLY=($(compgen -W "$commands $global_flags" -- "$cur"))
        return
    fi

    # Relative position of the current word within the command's own args.
    local pos=$(( COMP_CWORD - cmd_idx ))

    case "$cmd" in
        init)
            COMPREPLY=($(compgen -d -- "$cur"))
            ;;
        enroll)
            if (( pos == 1 )); then
                COMPREPLY=($(compgen -W "list request approve" -- "$cur"))
            elif [[ "${COMP_WORDS[cmd_idx+1]}" == "approve" ]]; then
                COMPREPLY=($(compgen -W "--all" -- "$cur"))
            fi
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
        sync)
            COMPREPLY=($(compgen -W "--push-only --pull-only" -- "$cur"))
            ;;
        completion)
            if (( pos == 1 )); then
                COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur"))
            fi
            ;;
        help)
            if (( pos == 1 )); then
                COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            fi
            ;;
    esac
}

complete -F _ktraveler_complete ktraveler

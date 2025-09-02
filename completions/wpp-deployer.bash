#!/bin/bash

_wpp_deployer_completions() {
    local cur prev words cword
    _init_completion || return

    # Main commands
    local commands="install deploy delete list exec exec-all help version"
    
    # Docker compose subcommands
    local docker_compose_commands="up down restart start stop pause unpause ps logs exec run build pull push config"
    
    # Get list of sites
    _get_sites() {
        if command -v wpp-deployer >/dev/null 2>&1; then
            wpp-deployer list 2>/dev/null
        fi
    }

    case $cword in
        1)
            # Complete main commands
            COMPREPLY=($(compgen -W "$commands" -- "$cur"))
            ;;
        2)
            case $prev in
                deploy|delete|exec)
                    if [[ $prev == "exec" && $cur == "-r" ]]; then
                        COMPREPLY=("-r")
                    else
                        # Complete with site names
                        COMPREPLY=($(compgen -W "$(_get_sites)" -- "$cur"))
                    fi
                    ;;
                exec-all)
                    if [[ $cur == "-r" ]]; then
                        COMPREPLY=("-r")
                    else
                        # Complete with docker-compose commands
                        COMPREPLY=($(compgen -W "$docker_compose_commands" -- "$cur"))
                    fi
                    ;;
                list|install|help|version)
                    # These commands take no arguments
                    ;;
                *)
                    ;;
            esac
            ;;
        3)
            case ${words[1]} in
                exec)
                    if [[ ${words[2]} == "-r" ]]; then
                        # After -r flag, complete with site names
                        COMPREPLY=($(compgen -W "$(_get_sites)" -- "$cur"))
                    else
                        # After sitename, complete with docker-compose commands
                        COMPREPLY=($(compgen -W "$docker_compose_commands" -- "$cur"))
                    fi
                    ;;
                exec-all)
                    if [[ ${words[2]} == "-r" ]]; then
                        # After -r flag, complete with docker-compose commands
                        COMPREPLY=($(compgen -W "$docker_compose_commands" -- "$cur"))
                    else
                        # Additional docker-compose arguments/flags
                        case $cur in
                            -*)
                                COMPREPLY=($(compgen -W "-d --detach -f --file --profile --project-name --project-directory" -- "$cur"))
                                ;;
                            *)
                                COMPREPLY=()
                                ;;
                        esac
                    fi
                    ;;
                *)
                    ;;
            esac
            ;;
        4)
            case ${words[1]} in
                exec)
                    if [[ ${words[2]} == "-r" ]]; then
                        # After "exec -r sitename", complete with docker-compose commands
                        COMPREPLY=($(compgen -W "$docker_compose_commands" -- "$cur"))
                    else
                        # Additional docker-compose arguments/flags
                        case $cur in
                            -*)
                                COMPREPLY=($(compgen -W "-d --detach -f --file --profile --project-name --project-directory" -- "$cur"))
                                ;;
                            *)
                                COMPREPLY=()
                                ;;
                        esac
                    fi
                    ;;
                *)
                    ;;
            esac
            ;;
        *)
            # For additional arguments, provide common docker-compose flags
            case $cur in
                -*)
                    COMPREPLY=($(compgen -W "-d --detach -f --file --profile --project-name --project-directory --volumes -v" -- "$cur"))
                    ;;
                *)
                    COMPREPLY=()
                    ;;
            esac
            ;;
    esac
}

complete -F _wpp_deployer_completions wpp-deployer 
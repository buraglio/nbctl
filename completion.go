package main

import (
	"fmt"
	"os"
)

func runCompletion(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nbctl completion <bash|zsh|fish>")
		os.Exit(1)
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletion)
	case "zsh":
		fmt.Print(zshCompletion)
	case "fish":
		fmt.Print(fishCompletion)
	default:
		fmt.Fprintf(os.Stderr, "unknown shell: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Usage: nbctl completion <bash|zsh|fish>")
		os.Exit(1)
	}
}

const bashCompletion = `# nbctl bash completion
# Add to ~/.bashrc: eval "$(nbctl completion bash)"

_nbctl_completion() {
    local cur prev words cword
    if declare -f _init_completion > /dev/null 2>&1; then
        _init_completion || return
    else
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    local top_cmds="peers groups users setup-key routes policy sync zonefile watch events posture-check completion version help"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${top_cmds}" -- "${cur}") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        peers)
            COMPREPLY=( $(compgen -W "list show update delete accessible" -- "${cur}") )
            ;;
        groups)
            COMPREPLY=( $(compgen -W "list show create update delete" -- "${cur}") )
            ;;
        users)
            COMPREPLY=( $(compgen -W "list create delete invite approve" -- "${cur}") )
            ;;
        setup-key)
            COMPREPLY=( $(compgen -W "list show create revoke delete" -- "${cur}") )
            ;;
        routes)
            COMPREPLY=( $(compgen -W "list show create update delete" -- "${cur}") )
            ;;
        policy)
            COMPREPLY=( $(compgen -W "list show create delete" -- "${cur}") )
            ;;
        sync|zonefile|watch)
            COMPREPLY=()
            ;;
        events)
            COMPREPLY=( $(compgen -W "audit traffic" -- "${cur}") )
            ;;
        posture-check)
            COMPREPLY=( $(compgen -W "list show delete" -- "${cur}") )
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
            ;;
        *)
            COMPREPLY=( $(compgen -W "${top_cmds}" -- "${cur}") )
            ;;
    esac

    return 0
}

complete -F _nbctl_completion nbctl
`

const zshCompletion = `# nbctl zsh completion
# Add to ~/.zshrc: eval "$(nbctl completion zsh)"

_nbctl() {
    local state

    _arguments \
        '1: :->command' \
        '*: :->args'

    case $state in
        command)
            local commands=(
                'peers:Manage peers'
                'groups:Manage groups'
                'users:Manage users'
                'setup-key:Manage setup keys'
                'routes:Manage network routes'
                'policy:Manage access policies'
                'sync:Sync peer IPs to Cloudflare DNS'
                'zonefile:Generate a BIND-format zone file'
                'watch:Sync continuously on a repeating interval'
                'events:View audit and traffic events'
                'posture-check:Manage posture checks'
                'completion:Generate shell completion scripts'
                'version:Print version information'
                'help:Show help'
            )
            _describe 'command' commands
            ;;
        args)
            case ${words[2]} in
                peers)
                    local subcmds=('list:List peers' 'show:Show peer details' 'update:Update a peer' 'delete:Delete a peer' 'accessible:List accessible peers')
                    _describe 'subcommand' subcmds
                    ;;
                groups)
                    local subcmds=('list:List groups' 'show:Show group details' 'create:Create a group' 'update:Update a group' 'delete:Delete a group')
                    _describe 'subcommand' subcmds
                    ;;
                users)
                    local subcmds=('list:List users' 'create:Create a user' 'delete:Delete a user' 'invite:Invite a user' 'approve:Approve a user')
                    _describe 'subcommand' subcmds
                    ;;
                setup-key)
                    local subcmds=('list:List setup keys' 'show:Show a setup key' 'create:Create a setup key' 'revoke:Revoke a setup key' 'delete:Delete a setup key')
                    _describe 'subcommand' subcmds
                    ;;
                routes)
                    local subcmds=('list:List routes' 'show:Show route details' 'create:Create a route' 'update:Update a route' 'delete:Delete a route')
                    _describe 'subcommand' subcmds
                    ;;
                policy)
                    local subcmds=('list:List policies' 'show:Show policy details' 'create:Create a policy' 'delete:Delete a policy')
                    _describe 'subcommand' subcmds
                    ;;
                sync|zonefile|watch)
                    ;;
                events)
                    local subcmds=('audit:List audit events' 'traffic:List traffic events')
                    _describe 'subcommand' subcmds
                    ;;
                posture-check)
                    local subcmds=('list:List posture checks' 'show:Show posture check details' 'delete:Delete a posture check')
                    _describe 'subcommand' subcmds
                    ;;
                completion)
                    local subcmds=('bash:Generate bash completion script' 'zsh:Generate zsh completion script' 'fish:Generate fish completion script')
                    _describe 'subcommand' subcmds
                    ;;
            esac
            ;;
    esac
}

compdef _nbctl nbctl
`

const fishCompletion = `# nbctl fish completion
# Save to ~/.config/fish/completions/nbctl.fish: nbctl completion fish > ~/.config/fish/completions/nbctl.fish

set -l top_cmds peers groups users setup-key routes policy sync zonefile watch events posture-check completion version help

# Disable file completions for nbctl
complete -c nbctl -f

# Top-level commands (only when no subcommand has been given yet)
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a peers          -d "Manage peers"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a groups         -d "Manage groups"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a users          -d "Manage users"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a setup-key      -d "Manage setup keys"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a routes         -d "Manage network routes"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a policy         -d "Manage access policies"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a sync           -d "Sync peer IPs to Cloudflare DNS"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a zonefile       -d "Generate a BIND-format zone file"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a watch          -d "Sync continuously on a repeating interval"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a events         -d "View audit and traffic events"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a posture-check  -d "Manage posture checks"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a completion     -d "Generate shell completion scripts"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a version        -d "Print version information"
complete -c nbctl -f -n "not __fish_seen_subcommand_from $top_cmds" -a help           -d "Show help"

# Sub-commands
complete -c nbctl -f -n "__fish_seen_subcommand_from peers"         -a list       -d "List peers"
complete -c nbctl -f -n "__fish_seen_subcommand_from peers"         -a show       -d "Show peer details"
complete -c nbctl -f -n "__fish_seen_subcommand_from peers"         -a update     -d "Update a peer"
complete -c nbctl -f -n "__fish_seen_subcommand_from peers"         -a delete     -d "Delete a peer"
complete -c nbctl -f -n "__fish_seen_subcommand_from peers"         -a accessible -d "List accessible peers"

complete -c nbctl -f -n "__fish_seen_subcommand_from groups"        -a list   -d "List groups"
complete -c nbctl -f -n "__fish_seen_subcommand_from groups"        -a show   -d "Show group details"
complete -c nbctl -f -n "__fish_seen_subcommand_from groups"        -a create -d "Create a group"
complete -c nbctl -f -n "__fish_seen_subcommand_from groups"        -a update -d "Update a group"
complete -c nbctl -f -n "__fish_seen_subcommand_from groups"        -a delete -d "Delete a group"

complete -c nbctl -f -n "__fish_seen_subcommand_from users"         -a list    -d "List users"
complete -c nbctl -f -n "__fish_seen_subcommand_from users"         -a create  -d "Create a user"
complete -c nbctl -f -n "__fish_seen_subcommand_from users"         -a delete  -d "Delete a user"
complete -c nbctl -f -n "__fish_seen_subcommand_from users"         -a invite  -d "Invite a user"
complete -c nbctl -f -n "__fish_seen_subcommand_from users"         -a approve -d "Approve a user"

complete -c nbctl -f -n "__fish_seen_subcommand_from setup-key"     -a list   -d "List setup keys"
complete -c nbctl -f -n "__fish_seen_subcommand_from setup-key"     -a show   -d "Show a setup key"
complete -c nbctl -f -n "__fish_seen_subcommand_from setup-key"     -a create -d "Create a setup key"
complete -c nbctl -f -n "__fish_seen_subcommand_from setup-key"     -a revoke -d "Revoke a setup key"
complete -c nbctl -f -n "__fish_seen_subcommand_from setup-key"     -a delete -d "Delete a setup key"

complete -c nbctl -f -n "__fish_seen_subcommand_from routes"        -a list   -d "List routes"
complete -c nbctl -f -n "__fish_seen_subcommand_from routes"        -a show   -d "Show route details"
complete -c nbctl -f -n "__fish_seen_subcommand_from routes"        -a create -d "Create a route"
complete -c nbctl -f -n "__fish_seen_subcommand_from routes"        -a update -d "Update a route"
complete -c nbctl -f -n "__fish_seen_subcommand_from routes"        -a delete -d "Delete a route"

complete -c nbctl -f -n "__fish_seen_subcommand_from policy"        -a list   -d "List policies"
complete -c nbctl -f -n "__fish_seen_subcommand_from policy"        -a show   -d "Show policy details"
complete -c nbctl -f -n "__fish_seen_subcommand_from policy"        -a create -d "Create a policy"
complete -c nbctl -f -n "__fish_seen_subcommand_from policy"        -a delete -d "Delete a policy"

complete -c nbctl -f -n "__fish_seen_subcommand_from events"        -a audit   -d "List audit events"
complete -c nbctl -f -n "__fish_seen_subcommand_from events"        -a traffic -d "List traffic events"

complete -c nbctl -f -n "__fish_seen_subcommand_from posture-check" -a list   -d "List posture checks"
complete -c nbctl -f -n "__fish_seen_subcommand_from posture-check" -a show   -d "Show posture check details"
complete -c nbctl -f -n "__fish_seen_subcommand_from posture-check" -a delete -d "Delete a posture check"

complete -c nbctl -f -n "__fish_seen_subcommand_from completion"    -a bash -d "Generate bash completion script"
complete -c nbctl -f -n "__fish_seen_subcommand_from completion"    -a zsh  -d "Generate zsh completion script"
complete -c nbctl -f -n "__fish_seen_subcommand_from completion"    -a fish -d "Generate fish completion script"
`

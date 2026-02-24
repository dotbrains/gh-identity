# gh-identity shell hook for Zsh
# Source this file or install via: gh identity init

__gh_identity_hook() {
    local hook_bin="$HOME/.config/gh-identity/bin/gh-identity-hook"
    if [[ -x "$hook_bin" ]]; then
        eval "$("$hook_bin" --shell zsh)"
    fi
}

# Use chpwd hook for directory change detection.
autoload -Uz add-zsh-hook
add-zsh-hook chpwd __gh_identity_hook

# Run on initial load.
__gh_identity_hook

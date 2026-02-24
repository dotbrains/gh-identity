# gh-identity shell hook for Bash
# Source this file or install via: gh identity init

__gh_identity_hook() {
    local hook_bin="$HOME/.config/gh-identity/bin/gh-identity-hook"
    if [[ -x "$hook_bin" ]]; then
        eval "$("$hook_bin" --shell bash)"
    fi
}

# Hook into cd by wrapping PROMPT_COMMAND.
if [[ -z "$__GH_IDENTITY_PROMPT_INSTALLED" ]]; then
    export __GH_IDENTITY_PROMPT_INSTALLED=1
    PROMPT_COMMAND="__gh_identity_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi

# Run on initial load.
__gh_identity_hook

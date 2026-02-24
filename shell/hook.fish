# gh-identity shell hook for Fish
# Source this file or install via: gh identity init

function __gh_identity_hook --on-variable PWD
    set -l hook_bin "$HOME/.config/gh-identity/bin/gh-identity-hook"
    if test -x "$hook_bin"
        eval ($hook_bin --shell fish)
    end
end

# Run on initial load to set identity for the current directory.
__gh_identity_hook

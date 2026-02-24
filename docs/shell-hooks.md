# Shell Hooks

## Overview

`gh-identity` uses shell hooks to automatically switch identities when you change directories. The hook invokes the lightweight `gh-identity-hook` binary, which resolves the active profile and emits shell-specific export statements.

## Supported Shells

### Fish

Installed to `~/.config/fish/conf.d/gh-identity.fish`. Uses `--on-variable PWD` to detect directory changes.

```fish
function __gh_identity_hook --on-variable PWD
    eval ($HOME/.config/gh-identity/bin/gh-identity-hook --shell fish)
end
```

### Bash

Appended to `~/.bashrc`. Uses `PROMPT_COMMAND` to run before each prompt.

```bash
eval "$($HOME/.config/gh-identity/bin/gh-identity-hook --shell bash)"
```

### Zsh

Appended to `~/.zshrc`. Uses the `chpwd` hook.

```zsh
autoload -Uz add-zsh-hook
add-zsh-hook chpwd __gh_identity_hook
```

## Manual Installation

If `gh identity init` didn't install the hook, you can source the hook scripts directly:

```sh
# Fish
source /path/to/gh-identity/shell/hook.fish

# Bash
source /path/to/gh-identity/shell/hook.bash

# Zsh
source /path/to/gh-identity/shell/hook.zsh
```

## Troubleshooting

1. **Hook not firing:** Ensure the hook binary exists at `~/.config/gh-identity/bin/gh-identity-hook` and is executable.
2. **Wrong identity:** Run `gh identity status` to see which binding matched. Check `bindings.yml` for conflicting entries.
3. **Slow shell startup:** The hook binary is designed to resolve in <5ms. If it's slow, check that `gh auth token` responds quickly.

Run `gh identity doctor` to validate the full setup.

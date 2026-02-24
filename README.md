# gh-identity

![gh-identity](https://raw.githubusercontent.com/dotbrains/gh-identity/main/assets/og-image.svg)

[![CI](https://github.com/dotbrains/gh-identity/actions/workflows/ci.yml/badge.svg)](https://github.com/dotbrains/gh-identity/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dotbrains/gh-identity)](https://goreportcard.com/report/github.com/dotbrains/gh-identity)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![Go](https://img.shields.io/badge/-Go-00ADD8?style=flat-square&logo=go&logoColor=white)
![GitHub CLI](https://img.shields.io/badge/-GitHub_CLI-181717?style=flat-square&logo=github&logoColor=white)
![Cobra](https://img.shields.io/badge/-Cobra-00ADD8?style=flat-square&logo=go&logoColor=white)
![YAML](https://img.shields.io/badge/-YAML-CB171E?style=flat-square&logo=yaml&logoColor=white)
![macOS](https://img.shields.io/badge/-macOS-000000?style=flat-square&logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/-Linux-FCC624?style=flat-square&logo=linux&logoColor=black)

> *"One shell, many identities."*

A `gh` CLI extension that provides seamless multi-account management, automatic context-based account switching, and per-directory identity binding.

## Problem

`gh` supports multiple authenticated accounts, but switching is manual and global. `gh-identity` solves this by:

- Automatically switching accounts based on the current directory
- Binding directory trees to specific GitHub identities
- Switching git author config (`user.name`, `user.email`) in tandem with the `gh` active account
- Using git's native `includeIf` for durable identity across all tools
- Providing per-shell token isolation via `GH_TOKEN` (no global state mutation)

## Installation

### Via `gh` extension

```sh
gh extension install dotbrains/gh-identity
```

### Via Homebrew

```sh
brew install dotbrains/tap/gh-identity
```

### From source

```sh
git clone https://github.com/dotbrains/gh-identity.git
cd gh-identity
make build
make install
```

## Quickstart

```sh
# 1. Authenticate your GitHub accounts (if not already done)
gh auth login  # repeat for each account

# 2. Run interactive setup
gh identity init

# 3. Bind directories to profiles
gh identity bind ~/code/personal personal
gh identity bind ~/code/work work

# 4. Verify
cd ~/code/personal/some-repo
gh identity status
```

## Commands

### `gh identity init`

Interactive first-time setup. Discovers authenticated accounts, creates profiles, and installs the shell hook.

### `gh identity profile add <name>`

Create a new identity profile interactively.

### `gh identity profile list`

List all configured profiles. The active profile is marked with `*`, the default with `→`.

### `gh identity profile remove <name>`

Remove a profile and its associated bindings.

### `gh identity bind [<path>] <profile>`

Bind a directory (defaults to `$PWD`) to a profile.

### `gh identity unbind [<path>]`

Remove the binding for a directory.

### `gh identity switch <profile>`

Manually activate a profile for the current shell session.

### `gh identity status`

Display the active identity, bound directory, and source.

```
  Profile:  personal
  Account:  nicholasadamou
  Name:     Nicholas Adamou
  Email:    nicholasadamou@users.noreply.github.com
  SSH Key:  ~/.ssh/id_ed25519_personal
  Bound by: ~/code/github.com/dotbrains
```

### `gh identity clone <repo> [--profile <profile>]`

Clone a repo and automatically bind it to the specified profile.

### `gh identity doctor`

Validate the full setup: profiles, auth, SSH keys, shell hook, and bindings.

## How It Works

### Token Strategy

`gh-identity` **never** calls `gh auth switch`. The shell hook runs `gh auth token -u <user>` and exports the result as `GH_TOKEN` per-shell. Two terminals in different directories use different accounts simultaneously with zero conflict.

### Git Identity

`gh-identity` uses git's native `includeIf "gitdir:..."` mechanism. When you bind a directory, it writes a profile-specific gitconfig fragment and adds an `includeIf` entry to `~/.gitconfig`. This works in all tools — not just shells with the hook installed.

### Shell Hook

On every directory change, a lightweight binary (`gh-identity-hook`) resolves the active profile and exports environment variables. Supported shells: Fish, Bash, Zsh.

## Configuration

Config lives in `~/.config/gh-identity/`:

- `profiles.yml` — identity profiles
- `bindings.yml` — directory-to-profile mappings
- `git/` — per-profile gitconfig fragments
- `bin/` — hook binary

## License

MIT

# Architecture

## Overview

`gh-identity` consists of two binaries and a set of shell hook scripts:

1. **`gh-identity`** — the main `gh` extension binary (cobra CLI)
2. **`gh-identity-hook`** — a lightweight binary invoked on every directory change
3. **Shell hook scripts** — per-shell wrappers that invoke `gh-identity-hook`

## Data Flow

```
gh identity CLI ──reads/writes──▶ profiles.yml
                ──reads/writes──▶ bindings.yml
                ──writes────────▶ ~/.gitconfig includeIf

gh-identity-hook ──reads────────▶ profiles.yml
                 ──reads────────▶ bindings.yml
                 ──calls────────▶ gh auth token
                 ──exports──────▶ GH_TOKEN / GIT_* env vars
```

## Package Structure

- `cmd/gh-identity/` — extension entry point
- `cmd/gh-identity-hook/` — hook binary entry point
- `internal/config/` — YAML config I/O (profiles, bindings, paths)
- `internal/resolve/` — binding resolution (deepest-match directory walk)
- `internal/gitconfig/` — `includeIf` directive management
- `internal/ghauth/` — `gh auth` interface (token retrieval, user listing)
- `internal/hook/` — hook resolution logic (shared by hook binary)
- `internal/cmd/` — cobra command tree

## Binding Resolution

1. Load all bindings from `bindings.yml`
2. For each binding, check if the current directory is equal to or a child of the binding path
3. Among all matching bindings, select the **deepest** (most specific) one
4. If no binding matches, fall back to the default profile

## Token Strategy

- `GH_TOKEN` is exported per-shell, never written globally
- `~/.config/gh/hosts.yml` is never modified by `gh-identity`
- The token is refreshed only when the active profile changes (on `cd`)

## Git Identity Strategy

- Per-profile gitconfig fragments are written to `~/.config/gh-identity/git/<profile>.gitconfig`
- `includeIf "gitdir:..."` entries are added to `~/.gitconfig`
- Environment variables (`GIT_AUTHOR_NAME`, etc.) are also exported as belt-and-suspenders

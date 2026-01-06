# git-subclone

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yejune/git-subclone?include_prereleases)](https://github.com/yejune/git-subclone/releases)
[![CI](https://github.com/yejune/git-subclone/actions/workflows/ci.yml/badge.svg)](https://github.com/yejune/git-subclone/actions/workflows/ci.yml)

Manage nested git repositories with independent push capability.

## Why Subclone?

Git submodules are powerful but come with friction:

- **Complex workflow**: `git clone --recursive`, `git submodule update --init`
- **Detached HEAD**: Easy to lose commits when switching branches
- **Push confusion**: Changes need to be pushed from inside the submodule first

Git subtrees solve some problems but create others:

- **No clear boundary**: Subtree history mixes with parent history
- **Special commands**: `git subtree push` with arcane syntax
- **No independent repo**: Can't easily work on the subtree as a separate project

**Subclone takes a different approach:**

| Feature | Submodule | Subtree | Subclone |
|---------|-----------|---------|----------|
| Simple clone | `--recursive` required | Yes | Yes (with hook) |
| Intuitive push | Yes | Special command | Yes |
| Files in parent repo | Pointer only | Yes | Yes |
| Clear manifest | `.gitmodules` | No | `.subclones.yaml` |
| Independent repository | Yes | No | Yes |
| Easy to understand | No | No | Yes |

**Subclone = Best of both worlds**

- Source files tracked by parent (like subtree)
- Independent `.git` for direct push (like submodule)
- Simple manifest file for clear management
- No special commands to remember

## Features

- **Clone as subclone**: `git subclone <url>` - just like `git clone`
- **Sync all**: Pull/clone all subclones with one command
- **Direct push**: Push changes directly to subclone's remote
- **Auto-sync hook**: Optionally sync after checkout
- **Self-update**: Update the binary with `git subclone selfupdate`
- **Recursive sync**: Sync subclones within subclones

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yejune/tap/git-subclone
```

### Using curl

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yejune/git-subclone/releases/latest/download/git-subclone-darwin-arm64 -o /usr/local/bin/git-subclone
chmod +x /usr/local/bin/git-subclone

# macOS (Intel)
curl -L https://github.com/yejune/git-subclone/releases/latest/download/git-subclone-darwin-amd64 -o /usr/local/bin/git-subclone
chmod +x /usr/local/bin/git-subclone

# Linux (x86_64)
curl -L https://github.com/yejune/git-subclone/releases/latest/download/git-subclone-linux-amd64 -o /usr/local/bin/git-subclone
chmod +x /usr/local/bin/git-subclone
```

### Using Go

```bash
go install github.com/yejune/git-subclone@latest
```

### From Source

```bash
git clone https://github.com/yejune/git-subclone.git
cd git-subclone
go build -o git-subclone
sudo mv git-subclone /usr/local/bin/
```

## Quick Start

```bash
# Clone a repository as subclone
git subclone https://github.com/user/repo.git

# With custom path
git subclone https://github.com/user/repo.git packages/repo

# With specific branch
git subclone -b develop https://github.com/user/repo.git

# SSH format
git subclone git@github.com:user/repo.git
```

## Commands

### `git subclone [url] [path]`

Clone a repository as a subclone (default command).

```bash
git subclone https://github.com/user/lib.git              # -> ./lib/
git subclone https://github.com/user/lib.git packages/lib # -> ./packages/lib/
git subclone -b develop git@github.com:user/lib.git       # specific branch
```

### `git subclone add [url] [path]`

Add a new subclone (same as default command).

```bash
git subclone add https://github.com/user/lib.git packages/lib
git subclone add git@github.com:user/lib.git packages/lib -b develop
```

### `git subclone sync`

Clone or pull all registered subclones.

```bash
git subclone sync             # sync all subclones
git subclone sync --recursive # recursively sync nested subclones
```

### `git subclone list`

List all registered subclones.

```bash
git subclone list    # list subclones
git subclone ls      # alias
```

### `git subclone status`

Show detailed status of all subclones.

```bash
git subclone status  # shows branch, commits ahead/behind, modified files
```

### `git subclone push [path]`

Push changes in subclones.

```bash
git subclone push packages/lib  # push specific subclone
git subclone push --all         # push all modified subclones
```

### `git subclone remove [path]`

Remove a subclone.

```bash
git subclone remove packages/lib              # remove and delete files
git subclone rm packages/lib --keep-files     # remove from manifest, keep files
```

### `git subclone init`

Install git hooks for auto-sync.

```bash
git subclone init  # installs post-checkout hook to auto-sync
```

### `git subclone selfupdate`

Update git-subclone to the latest version.

```bash
git subclone selfupdate  # downloads and installs latest release
```

## How It Works

### Directory Structure

```
my-project/
├── .git/                    <- Parent project git
├── .subclones.yaml          <- Subclone manifest (tracked by parent)
├── .gitignore               <- Contains "packages/lib/.git/"
├── src/
│   └── main.go
└── packages/
    └── lib/
        ├── .git/            <- Subclone's independent git
        └── lib.go           <- Tracked by BOTH repos
```

### Key Points

1. **Independent Git**: Each subclone has its own `.git` directory
2. **Source Tracking**: Parent tracks subclone's source files (not `.git`)
3. **Direct Push**: `cd packages/lib && git push` works as expected
4. **Manifest File**: `.subclones.yaml` records all subclones

### Manifest Format

```yaml
subclones:
  - path: packages/lib
    repo: https://github.com/user/lib.git
    branch: main
  - path: packages/utils
    repo: git@github.com:user/utils.git
```

## License

MIT

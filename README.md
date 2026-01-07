# git-sub

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yejune/git-sub?include_prereleases)](https://github.com/yejune/git-sub/releases)
[![CI](https://github.com/yejune/git-sub/actions/workflows/ci.yml/badge.svg)](https://github.com/yejune/git-sub/actions/workflows/ci.yml)

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
| Clear manifest | `.gitmodules` | No | `.gitsubs` |
| Independent repository | Yes | No | Yes |
| Easy to understand | No | No | Yes |

**Subclone = Best of both worlds**

- Source files tracked by parent (like subtree)
- Independent `.git` for direct push (like submodule)
- Simple manifest file for clear management
- No special commands to remember

## Features

- **Clone as subclone**: `git-sub <url>` - just like `git clone`
- **Sync all**: Pull/clone all subclones with one command
- **Direct push**: Push changes directly to subclone's remote
- **Auto-sync hook**: Optionally sync after checkout
- **Self-update**: Update the binary with `git-sub selfupdate`
- **Recursive sync**: Sync subclones within subclones

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yejune/tap/git-sub
```

### Using curl

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yejune/git-sub/releases/latest/download/git-sub-darwin-arm64 -o /usr/local/bin/git-sub
chmod +x /usr/local/bin/git-sub

# macOS (Intel)
curl -L https://github.com/yejune/git-sub/releases/latest/download/git-sub-darwin-amd64 -o /usr/local/bin/git-sub
chmod +x /usr/local/bin/git-sub

# Linux (x86_64)
curl -L https://github.com/yejune/git-sub/releases/latest/download/git-sub-linux-amd64 -o /usr/local/bin/git-sub
chmod +x /usr/local/bin/git-sub
```

### Using Go

```bash
go install github.com/yejune/git-sub@latest
```

### From Source

```bash
git clone https://github.com/yejune/git-sub.git
cd git-sub
go build -o git-sub
sudo mv git-sub /usr/local/bin/
```

## Quick Start

```bash
# Clone a repository as subclone
git-sub https://github.com/user/repo.git

# With custom path
git-sub https://github.com/user/repo.git packages/repo

# With specific branch
git-sub -b develop https://github.com/user/repo.git

# SSH format
git-sub git@github.com:user/repo.git
```

## Commands

### `git-sub [url] [path]`

Clone a repository as a subclone (default command).

```bash
git-sub https://github.com/user/lib.git              # -> ./lib/
git-sub https://github.com/user/lib.git packages/lib # -> ./packages/lib/
git-sub -b develop git@github.com:user/lib.git       # specific branch
```

### `git-sub add [url] [path]`

Add a new subclone (same as default command).

```bash
git-sub add https://github.com/user/lib.git packages/lib
git-sub add git@github.com:user/lib.git packages/lib -b develop
```

### `git-sub sync`

Clone or pull all registered subclones.

```bash
git-sub sync             # sync all subclones
git-sub sync --recursive # recursively sync nested subclones
```

### `git-sub list`

List all registered subclones.

```bash
git-sub list    # list subclones
git-sub ls      # alias
```

### `git-sub status`

Show detailed status of all subclones.

```bash
git-sub status  # shows branch, commits ahead/behind, modified files
```

### `git-sub push [path]`

Push changes in subclones.

```bash
git-sub push packages/lib  # push specific subclone
git-sub push --all         # push all modified subclones
```

### `git-sub remove [path]`

Remove a subclone.

```bash
git-sub remove packages/lib              # remove and delete files
git-sub rm packages/lib --keep-files     # remove from manifest, keep files
```

### `git-sub init`

Install git hooks for auto-sync.

```bash
git-sub init  # installs post-checkout hook to auto-sync
```

### `git-sub selfupdate`

Update git-sub to the latest version.

```bash
git-sub selfupdate  # downloads and installs latest release
```

## How It Works

### Directory Structure

```
my-project/
├── .git/                    <- Parent project git
├── .gitsubs          <- Subclone manifest (tracked by parent)
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
4. **Manifest File**: `.gitsubs` records all subclones

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

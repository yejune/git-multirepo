# git-sub

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yejune/git-sub?include_prereleases)](https://github.com/yejune/git-sub/releases)
[![CI](https://github.com/yejune/git-sub/actions/workflows/ci.yml/badge.svg)](https://github.com/yejune/git-sub/actions/workflows/ci.yml)

Manage nested git repositories with independent push capability.

## Why git-sub?

Git submodules are powerful but come with friction:

- **Complex workflow**: `git clone --recursive`, `git submodule update --init`
- **Detached HEAD**: Easy to lose commits when switching branches
- **Push confusion**: Changes need to be pushed from inside the submodule first

Git subtrees solve some problems but create others:

- **No clear boundary**: Subtree history mixes with parent history
- **Special commands**: `git subtree push` with arcane syntax
- **No independent repo**: Can't easily work on the subtree as a separate project

**git-sub takes a different approach:**

| Feature | Submodule | Subtree | git-sub |
|---------|-----------|---------|---------|
| Simple clone | `--recursive` required | Yes | Yes (with hook) |
| Intuitive push | Yes | Special command | Yes |
| Files in parent repo | Pointer only | Yes | Yes |
| Clear manifest | `.gitmodules` | No | `.gitsubs` |
| Independent repository | Yes | No | Yes |
| Easy to understand | No | No | Yes |

**git-sub = Best of both worlds**

- Source files tracked by parent (like subtree)
- Independent `.git` for direct push (like submodule)
- Simple manifest file for clear management
- No special commands to remember

## Features

- **Clone as sub**: `git sub clone <url>` - just like `git clone`
- **Sync all**: Pull/clone all subs with one command
- **Direct push**: Each sub has independent `.git` - just `cd` and `git push`
- **Auto-sync hook**: Optionally sync after checkout
- **Self-update**: Update the binary with `git sub selfupdate`
- **Recursive sync**: Sync subs within subs

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
# Clone a repository as sub
git sub clone https://github.com/user/repo.git

# With custom path
git sub clone https://github.com/user/repo.git packages/repo

# With specific branch
git sub clone -b develop https://github.com/user/repo.git

# SSH format
git sub clone git@github.com:user/repo.git
```

## Commands

### `git sub clone [url] [path]`

Clone a repository as a sub (default command).

```bash
git sub clone https://github.com/user/lib.git              # -> ./lib/
git sub clone https://github.com/user/lib.git packages/lib # -> ./packages/lib/
git sub clone -b develop git@github.com:user/lib.git       # specific branch
```

### `git sub add [url] [path]`

Add a new sub (same as clone command).

```bash
git sub add https://github.com/user/lib.git packages/lib
git sub add git@github.com:user/lib.git packages/lib -b develop
```

### `git sub sync`

Clone or pull all registered subs.

```bash
git sub sync             # sync all subs
git sub sync --recursive # recursively sync nested subs
```

### `git sub list`

List all registered subs.

```bash
git sub list    # list subs
git sub ls      # alias
```

### `git sub status`

Show detailed status of all subs.

```bash
git sub status  # shows branch, commits ahead/behind, modified files
```

### `git sub remove [path]`

Remove a sub.

```bash
git sub remove packages/lib              # remove and delete files
git sub rm packages/lib --keep-files     # remove from manifest, keep files
```

### `git sub init`

Install git hooks for auto-sync.

```bash
git sub init  # installs post-checkout hook to auto-sync
```

### `git sub selfupdate`

Update git-sub to the latest version.

```bash
git sub selfupdate  # downloads and installs latest release
```

## How It Works

### Directory Structure

```
my-project/
├── .git/                    <- Parent project git
├── .gitsubs                 <- Sub manifest (tracked by parent)
├── .gitignore               <- Contains "packages/lib/.git/"
├── src/
│   └── main.go
└── packages/
    └── lib/
        ├── .git/            <- Sub's independent git
        └── lib.go           <- Tracked by BOTH repos
```

### Key Points

1. **Independent Git**: Each sub has its own `.git` directory (local only)
2. **Source Tracking**: Parent tracks sub's source files (not `.git`)
3. **Direct Push**: `cd packages/lib && git push` works as expected
4. **Manifest File**: `.gitsubs` records all subs for recreation

### Workflow

**Developer A adds a sub:**
```bash
git sub clone https://github.com/user/lib.git packages/lib
# Creates: packages/lib/.git/ (local)
# Ignores: packages/lib/.git/ → .gitignore
# Tracks: packages/lib/*.go → parent repo
# Records: path, repo, commit hash → .gitsubs

git add .
git commit -m "Add lib sub"
git push  # Pushes: source files + .gitsubs (NOT .git)
```

**Developer A updates sub:**
```bash
cd packages/lib
git commit && git push  # ← Must push to remote!

cd ../..
git add packages/lib/    # Stage updated source
git sub sync             # ← Auto-updates .gitsubs with new commit!
git commit -m "Update lib"
git push
```

**Developer B clones:**
```bash
git clone <parent-repo>
# Gets: .gitsubs + source files
# Missing: packages/lib/.git/

git sub sync  # or use post-checkout hook
# Reads: .gitsubs commit hash
# Restores: .git at exact commit
# Now: cd packages/lib && git push works!
```

**Key Points:**
- `.git` directories are never pushed
- Commit hashes ensure version consistency
- `git sub sync` handles everything automatically
- Unpushed commits trigger warnings

### Manifest Format

```yaml
subclones:
  - path: packages/lib
    repo: https://github.com/user/lib.git
    branch: main
    commit: abc123def456789...  # Exact commit hash
  - path: packages/utils
    repo: git@github.com:user/utils.git
    commit: 789def456abc123...
```

## License

MIT

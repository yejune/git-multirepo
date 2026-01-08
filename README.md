# git-workspace

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yejune/git-workspace?include_prereleases)](https://github.com/yejune/git-workspace/releases)
[![CI](https://github.com/yejune/git-workspace/actions/workflows/ci.yml/badge.svg)](https://github.com/yejune/git-workspace/actions/workflows/ci.yml)

Manage nested git repositories with independent push capability.

## Why git-workspace?

Git submodules are powerful but come with friction:

- **Complex workflow**: `git clone --recursive`, `git submodule update --init`
- **Detached HEAD**: Easy to lose commits when switching branches
- **Push confusion**: Changes need to be pushed from inside the submodule first

Git subtrees solve some problems but create others:

- **No clear boundary**: Subtree history mixes with parent history
- **Special commands**: `git subtree push` with arcane syntax
- **No independent repo**: Can't easily work on the subtree as a separate project

**git-workspace takes a different approach:**

| Feature | Submodule | Subtree | git-workspace |
|---------|-----------|---------|---------------|
| Simple clone | `--recursive` required | Yes | Yes (with hook) |
| Intuitive push | Yes | Special command | Yes |
| Files in parent repo | Pointer only | Yes | Yes |
| Clear manifest | `.gitmodules` | No | `.workspaces` |
| Independent repository | Yes | No | Yes |
| Easy to understand | No | No | Yes |

**git-workspace = Best of both worlds**

- Source files tracked by parent (like subtree)
- Independent `.git` for direct push (like submodule)
- Simple manifest file for clear management
- No special commands to remember

## Features

- **Clone as workspace**: `git workspace clone <url>` - just like `git clone`
- **Sync all**: Pull/clone all workspaces with one command
- **Direct push**: Each workspace has independent `.git` - just `cd` and `git push`
- **Auto-sync hook**: Optionally sync after checkout
- **Self-update**: Update the binary with `git workspace selfupdate`
- **Recursive sync**: Sync workspaces within workspaces

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yejune/tap/git-workspace
```

### Using curl

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yejune/git-workspace/releases/latest/download/git-workspace-darwin-arm64 -o /usr/local/bin/git-workspace
chmod +x /usr/local/bin/git-workspace

# macOS (Intel)
curl -L https://github.com/yejune/git-workspace/releases/latest/download/git-workspace-darwin-amd64 -o /usr/local/bin/git-workspace
chmod +x /usr/local/bin/git-workspace

# Linux (x86_64)
curl -L https://github.com/yejune/git-workspace/releases/latest/download/git-workspace-linux-amd64 -o /usr/local/bin/git-workspace
chmod +x /usr/local/bin/git-workspace
```

### Using Go

```bash
go install github.com/yejune/git-workspace@latest
```

### From Source

```bash
git clone https://github.com/yejune/git-workspace.git
cd git-workspace
go build -o git-workspace
sudo mv git-workspace /usr/local/bin/
```

## Quick Start

```bash
# Clone a repository as workspace
git workspace clone https://github.com/user/repo.git

# With custom path
git workspace clone https://github.com/user/repo.git packages/repo

# With specific branch
git workspace clone -b develop https://github.com/user/repo.git

# SSH format
git workspace clone git@github.com:user/repo.git
```

## Commands

### `git workspace clone [url] [path]`

Clone a repository as a sub (default command).

```bash
git workspace clone https://github.com/user/lib.git              # -> ./lib/
git workspace clone https://github.com/user/lib.git packages/lib # -> ./packages/lib/
git workspace clone -b develop git@github.com:user/lib.git       # specific branch
```

### `git workspace add [url] [path]`

Add a new sub (same as clone command).

```bash
git workspace add https://github.com/user/lib.git packages/lib
git workspace add git@github.com:user/lib.git packages/lib -b develop
```

### `git workspace sync`

Auto-discover subs or sync from .workspaces. Has two modes:

**Mode 1: Discovery Mode** (no .workspaces)
```bash
# Situation: .workspaces doesn't exist
packages/lib/.git/      # existing sub
packages/utils/.git/    # existing sub

git workspace sync
# → Recursively scans directories
# → Auto-detects .git folders
# → Extracts remote, branch, commit
# → Creates .workspaces automatically
```

**Mode 2: Sync Mode** (has .workspaces)
```bash
# Situation: .workspaces exists
git workspace sync
# → Reads .workspaces
# → Restores missing .git directories
# → Installs/updates hooks
# → Updates commit hashes if pushed
```

**Use Cases:**
- Migrating existing project to git-sub
- Recovering from deleted .workspaces
- First-time setup: just clone and run sync

### `git workspace list`

List all registered subs.

```bash
git workspace list    # list subs
git workspace ls      # alias
```

### `git workspace status`

Show detailed status of all subs.

```bash
git workspace status  # shows branch, commits ahead/behind, modified files
```

### `git workspace remove [path]`

Remove a sub.

```bash
git workspace remove packages/lib              # remove and delete files
git workspace rm packages/lib --keep-files     # remove from manifest, keep files
```

### `git workspace init`

Install git hooks for auto-sync.

```bash
git workspace init  # installs post-checkout hook to auto-sync
```

### `git workspace selfupdate`

Update git-workspace to the latest version.

```bash
git workspace selfupdate  # downloads and installs latest release
```

## How It Works

### Directory Structure

```
my-project/
├── .git/                    <- Parent project git
├── .workspaces              <- Sub manifest (tracked by parent)
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
4. **Manifest File**: `.workspaces` records all subs for recreation

### Workflow

**Developer A adds a sub:**
```bash
git workspace clone https://github.com/user/lib.git packages/lib
# Creates: packages/lib/.git/ (local)
# Ignores: packages/lib/.git/ → .gitignore
# Tracks: packages/lib/*.go → parent repo
# Records: path, repo, commit hash → .workspaces

git add .
git commit -m "Add lib sub"
git push  # Pushes: source files + .workspaces (NOT .git)
```

**Developer A updates sub:**
```bash
cd packages/lib
git commit && git push  # ← Must push to remote!

cd ../..
git add packages/lib/    # Stage updated source
git workspace sync             # ← Auto-updates .workspaces with new commit!
git commit -m "Update lib"
git push
```

**Developer B clones:**
```bash
git clone <parent-repo>
# Gets: .workspaces + source files
# Missing: packages/lib/.git/

git workspace sync  # or use post-checkout hook
# Reads: .workspaces commit hash
# Restores: .git at exact commit
# Now: cd packages/lib && git push works!
```

**Key Points:**
- `.git` directories are never pushed
- Commit hashes ensure version consistency
- `git workspace sync` handles everything automatically
- Unpushed commits trigger warnings

### Manifest Format

```yaml
# .workspaces (new filename)
workspaces:
  - path: packages/lib
    repo: https://github.com/user/lib.git
    commit: abc123def456789...  # Exact commit hash
  - path: packages/utils
    repo: git@github.com:user/utils.git
    commit: 789def456abc123...
```

## License

MIT

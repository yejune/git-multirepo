# git-multirepo

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yejune/git-multirepo?include_prereleases)](https://github.com/yejune/git-multirepo/releases)
[![CI](https://github.com/yejune/git-multirepo/actions/workflows/ci.yml/badge.svg)](https://github.com/yejune/git-multirepo/actions/workflows/ci.yml)

Manage nested git repositories with independent push capability.

## Why git-multirepo?

Git submodules are powerful but come with friction:

- **Complex workflow**: `git clone --recursive`, `git submodule update --init`
- **Detached HEAD**: Easy to lose commits when switching branches
- **Push confusion**: Changes need to be pushed from inside the submodule first

Git subtrees solve some problems but create others:

- **History pollution**: Subtree history merges with parent history
- **Complex commands**: `git subtree split`, `git subtree push --prefix=...`
- **Implicit tracking**: No clear manifest of what's a subtree vs regular code

**git-multirepo takes a different approach:**

| Feature | Submodule | Subtree | git-multirepo |
|---------|-----------|---------|---------------|
| Simple clone | `--recursive` required | Yes | Yes |
| Independent push | Yes | `subtree push` | Yes (just `cd` and `git push`) |
| History separation | Yes | No (merges) | Yes |
| Clear manifest | `.gitmodules` | No | `.git.multirepos` |
| Independent repository | Yes | Yes | Yes |
| Intuitive commands | No | No | Yes |

**git-multirepo = Submodule simplicity + Manifest clarity**

- Source files tracked by parent (like subtree)
- Independent `.git` for direct push (like submodule)
- Simple manifest file for clear management
- No complex commands - just `clone`, `sync`, `pull`

## Features

- **Clone as workspace**: `git multirepo clone <url>` - just like `git clone`
- **Sync all**: Pull/clone all workspaces with one command
- **Direct push**: Each workspace has independent `.git` - just `cd` and `git push`
- **Auto-sync hook**: Optionally sync after checkout
- **Self-update**: Update the binary with `git multirepo selfupdate`
- **Recursive sync**: Sync workspaces within workspaces
- **Adaptive concurrency**: Automatically optimizes worker count based on CPU cores

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yejune/tap/git-multirepo
```

### Using curl

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yejune/git-multirepo/releases/latest/download/git-multirepo-darwin-arm64 -o /usr/local/bin/git-multirepo
chmod +x /usr/local/bin/git-multirepo

# macOS (Intel)
curl -L https://github.com/yejune/git-multirepo/releases/latest/download/git-multirepo-darwin-amd64 -o /usr/local/bin/git-multirepo
chmod +x /usr/local/bin/git-multirepo

# Linux (x86_64)
curl -L https://github.com/yejune/git-multirepo/releases/latest/download/git-multirepo-linux-amd64 -o /usr/local/bin/git-multirepo
chmod +x /usr/local/bin/git-multirepo
```

### Using Go

```bash
go install github.com/yejune/git-multirepo@latest
```

### From Source

```bash
git clone https://github.com/yejune/git-multirepo.git
cd git-multirepo
go build -o git-multirepo
sudo mv git-multirepo /usr/local/bin/
```

## Quick Start

```bash
# Clone a repository as workspace
git multirepo clone https://github.com/user/repo.git

# With custom path
git multirepo clone https://github.com/user/repo.git packages/repo

# With specific branch
git multirepo clone -b develop https://github.com/user/repo.git

# SSH format
git multirepo clone git@github.com:user/repo.git
```

## Configuration

### Environment Variables

**GIT_MULTIREPO_WORKERS** - Control parallel processing concurrency

```bash
# Default: CPU cores × 2 (automatically detected)
# Maximum: 32 (prevents excessive context switching)
# Minimum: 1

# Examples:
export GIT_MULTIREPO_WORKERS=8    # Use 8 workers
export GIT_MULTIREPO_WORKERS=16   # Use 16 workers (high concurrency)
export GIT_MULTIREPO_WORKERS=4    # Use 4 workers (resource-constrained)

# Unset to use default (CPU × 2)
unset GIT_MULTIREPO_WORKERS
```

**When to customize:**
- **High-performance systems**: Increase for faster sync (e.g., 24 workers on 16-core CPU)
- **Resource-constrained environments**: Decrease to reduce memory/CPU usage (e.g., 4 workers)
- **Network-limited systems**: Lower value reduces concurrent network requests
- **Default works for most cases**: CPU × 2 is optimized for I/O-bound git operations

## Commands

### `git multirepo clone [url] [path]`

Clone a repository as a workspace (default command).

```bash
git multirepo clone https://github.com/user/lib.git              # -> ./lib/
git multirepo clone https://github.com/user/lib.git packages/lib # -> ./packages/lib/
git multirepo clone -b develop git@github.com:user/lib.git       # specific branch
```

### `git multirepo sync`

Auto-discover workspaces or sync from .git.multirepos. Has two modes:

**Mode 1: Discovery Mode** (no .git.multirepos)
```bash
# Situation: .git.multirepos doesn't exist
packages/lib/.git/      # existing workspace
packages/utils/.git/    # existing workspace

git multirepo sync
# → Recursively scans directories
# → Auto-detects .git folders
# → Extracts remote, branch, commit
# → Creates .git.multirepos automatically
```

**Mode 2: Sync Mode** (has .git.multirepos)
```bash
# Situation: .git.multirepos exists
git multirepo sync
# → Reads .git.multirepos
# → Restores missing .git directories
# → Installs/updates hooks
# → Updates commit hashes if pushed
```

**Use Cases:**
- Migrating existing project to git-multirepo
- Recovering from deleted .git.multirepos
- First-time setup: just clone and run sync

### `git multirepo list`

List all registered workspaces.

```bash
git multirepo list    # list workspaces
git multirepo ls      # alias
```

### `git multirepo status`

Show detailed status of all workspaces.

```bash
git multirepo status  # shows branch, commits ahead/behind, modified files
```

### `git multirepo branch [workspace-path]`

Show current branch for workspaces.

```bash
git multirepo branch                    # show all workspace branches
git multirepo branch packages/lib       # show branch of specific workspace
```

Displays:
- Workspace path
- Repository URL
- Current branch
- Remote tracking branch (if exists)

### `git multirepo pull [workspace-path]`

Pull latest changes from remote for workspaces.

```bash
git multirepo pull                      # pull all workspaces
git multirepo pull packages/lib         # pull specific workspace
# Automatically handles keep files with patch application
# See "How It Works: Sync & Pull Workflow" for details
```

### `git multirepo reset`

Reset skip-worktree flags and restore files to HEAD state.

```bash
git multirepo reset                     # reset all (skip + ignore)
git multirepo reset skip                # reset skip-worktree only
git multirepo reset ignore              # reset ignore patterns only
# Creates backup before resetting
# See "Command Workflows: Complete Reference" for details
```

### `git multirepo remove [path]`

Remove a workspace.

```bash
git multirepo remove packages/lib              # remove and delete files
git multirepo rm packages/lib --keep-files     # remove from manifest, keep files

# IMPORTANT: Run sync before remove to preserve local modifications
git multirepo sync          # saves modified keep files to .multirepos/backup/
git multirepo remove <path> # then remove workspace
```

### `git multirepo selfupdate`

Update git-multirepo to the latest version.

```bash
git multirepo selfupdate  # downloads and installs latest release
```

## How It Works: Sync & Pull Workflow

Understanding the workflow is crucial for using git-multirepo effectively. Here's what happens under the hood.

### Sync Workflow

When you run `git multirepo sync`, the following steps occur:

```
1. Unskip → 2. Detect Changes → 3. Backup → 4. Create Patches → 5. Re-skip
```

**Why this order?**

1. **Unskip keep files** (`git update-index --no-skip-worktree`)
   - Makes modified files visible to git
   - Without this, `git diff HEAD` returns empty results

2. **Detect all modified files** (`git diff HEAD`)
   - Now git can see the actual changes
   - Identifies which files need backup/patch

3. **Backup modified files**
   - Saves original file content to `.multirepos/backup/modified/YYYY/MM/DD/`
   - Daily timestamped backups for safety

4. **Create patches** (`git diff HEAD file`)
   - Generates unified diff patches
   - Saved to `.multirepos/patches/workspace/file.patch`
   - Backup patches also saved to `.multirepos/backup/patched/YYYY/MM/DD/`

5. **Re-apply skip-worktree** (`git update-index --skip-worktree`)
   - Protects local modifications from git operations
   - Files become invisible to `git pull`, `git reset`, etc.

**Result:** Your local changes are safely backed up and protected from accidental overwrites.

---

### Pull Workflow

When you run `git multirepo pull`, the following steps occur:

```
1. Unskip → 2. Git Pull → 3. Apply Patches → 4. Re-skip
```

**Why this order?**

1. **Unskip keep files** (`git update-index --no-skip-worktree`)
   - Allows remote changes to be pulled into these files
   - Without this, `git pull` would skip these files entirely

2. **Pull from remote** (`git pull`)
   - Updates workspace to latest remote version
   - Keep files now contain the **new upstream version**

3. **Apply patches** (`patch -p1 < file.patch`)
   - Attempts to merge your local changes with new upstream
   - **If conflicts occur:** Patch fails with error → User must manually resolve
   - This is **intentional behavior** - you want to review conflicts

4. **Re-apply skip-worktree**
   - Protects the merged result from future git operations

**When Conflicts Happen (Expected):**

```bash
$ git multirepo pull

apps/api.config:
  ✓ Pulled latest changes
  ✗ Patch failed: Your local changes conflict with upstream

  Manual steps required:
  1. Check the conflict in apps/api.config/config.json
  2. Manually merge your changes with new upstream
  3. Run: git multirepo sync
```

**Why we want manual conflict resolution:**
- Automatic merging of config files is dangerous
- You need to see what changed upstream vs your local edits
- Critical files (DB credentials, API keys) require human review
- Better to fail safely than silently corrupt configuration

---

### Skip-worktree Protection

The `skip-worktree` flag tells git to **ignore local modifications** to tracked files.

**What it protects against:**
- ❌ `git pull` overwriting your config files
- ❌ `git reset --hard` destroying local changes
- ❌ `git checkout` switching and losing edits
- ❌ Accidental `git add` staging protected files

**What it doesn't protect:**
- ✅ `git multirepo pull` temporarily unskips (intentional)
- ✅ Manual `git update-index --no-skip-worktree` (you asked for it)
- ✅ Direct file deletion with `rm` (filesystem operation)

**Best Practice:**
Always use `git multirepo` commands for workspaces with keep files.

---

## Command Workflows: Complete Reference

This section provides detailed internal operation flows for all commands.

### 1. sync - Workspace Synchronization

**Purpose**: Backup keep files and apply skip-worktree

**Workflow**:
```
1. Unskip keep files
   └─ git update-index --no-skip-worktree <files>
   └─ Purpose: Allow git diff to see changes

2. Detect modified files
   └─ git diff --name-only HEAD
   └─ git ls-files -v (skip-worktree files)

3. Create backups
   ├─ Original files → .multirepos/backup/modified/YYYY/MM/DD/
   ├─ Patch files → .multirepos/patches/
   └─ Patch backups → .multirepos/backup/patched/YYYY/MM/DD/

4. Re-skip keep files
   └─ git update-index --skip-worktree <files>
   └─ Purpose: Protect from git pull

✅ Data protection: All changes are backed up
✅ Recoverable: Restore from .multirepos/backup/
```

**Safety**:
- ✅ Triple backup (original, patch, patch backup)
- ✅ Version preservation with timestamps
- ✅ Protection via re-applied skip-worktree

---

### 2. pull - Fetch Remote Changes

**Purpose**: Update from remote while protecting keep files

**Workflow**:
```
1. Detect remote changes
   └─ git fetch origin
   └─ git diff HEAD origin/<branch> <file>

2. Keep file handling (when changes exist)
   ├─ Backup current state (NEW!)
   ├─ Create patch (local changes)
   ├─ Backup patch (NEW!)
   ├─ Reset file (remote version)
   ├─ Check patch conflicts (NEW!)
   └─ Apply patch (on success) or guide recovery (on failure)

3. Execute git pull
   └─ git pull

⚠️ On conflicts: Manual resolution required (intentional behavior)
✅ Data protection: Original backed up even on failure
```

**Safety**:
- ✅ Backup original before patch application
- ✅ Conflict detection prevents data loss
- ✅ Backup location shown on failure

---

### 3. reset - Restore to Initial State

**Purpose**: Reset skip-worktree and ignore patterns

**Workflow**:
```
git multirepo reset          # Full reset
git multirepo reset ignore   # Ignore only
git multirepo reset skip     # Skip-worktree only

Reset Skip Operation:
1. Backup keep files (NEW!)
   └─ .multirepos/backup/modified/YYYY/MM/DD/

2. Unskip
   └─ git update-index --no-skip-worktree

3. Restore from HEAD
   └─ git checkout HEAD -- <files>

4. Re-skip
   └─ git update-index --skip-worktree

5. Execute git pull

⚠️ Warning: Local modifications overwritten by HEAD
✅ Data protection: Recoverable from backup
```

**Safety**:
- ✅ Automatic backup before HEAD restoration
- ✅ Backup location displayed

---

### 4. remove - Remove Workspace

**Purpose**: Remove from manifest and delete files (optional)

**Workflow**:
```
git multirepo remove <path>              # Delete files too
git multirepo remove --keep-files <path> # Remove from manifest only

Operation:
1. Show modified file warning (NEW!)
   └─ Display list of files to be deleted

2. User confirmation
   └─ Prompt if --force not used

3. Remove from manifest
   └─ Update .git.multirepos

4. Delete files (when --keep-files not used)
   └─ rm -rf <workspace-path>

⚠️ Important: Remove deletes workspace directory immediately
💡 Best practice: Run `git multirepo sync` before remove
   → Saves modified keep files to .multirepos/backup/
💡 Alternative: Use --keep-files to preserve files
```

**Safety**:
- ✅ Modified file warning
- ✅ Confirmation prompt
- ⚠️ No git-level protection (direct directory deletion)

---

### 5. status & list - Query Commands

**status**: Show git status of each workspace
**list**: Show workspace tree structure

**Safety**:
- ✅ Read-only (no data modification)
- ✅ Zero data loss risk

---

## Backup Directory Structure

```
.multirepos/
  backup/
    modified/           # Original file backups
      2026/01/09/
        apps/api.log/
          config.json.20260109_143022
          config.json.20260109_150130  # Multiple versions preserved
    patched/            # Patch file backups
      2026/01/09/
        apps/api.log/
          config.json.patch.20260109_143022
    archived/           # Monthly archives
      2025-12-modified.tar.gz
      2025-12-patched.tar.gz
      2025-11-modified.tar.gz
  patches/              # Active patches (latest)
    apps/api.log/
      config.json.patch
```

**Timestamp format**: `YYYYMMDD_HHMMSS`
**Retention policy**: Manual cleanup (no automatic deletion)

### Archiving Policy

| Item | Policy |
|------|--------|
| **Original preservation** | Current month only (YYYY/MM/) |
| **Archiving target** | All months before current month |
| **Archiving frequency** | Monthly compression |
| **Original handling** | Deleted after archiving |
| **Archive files** | **Permanent preservation (never auto-deleted)** |
| **Compression format** | `.tar.gz` |
| **Filename format** | `YYYY-MM-{modified\|patched}.tar.gz` |
| **Without archiving** | Originals keep accumulating (safe but disk grows) |

**Archiving Process**:
```
Trigger: Auto-check during sync (once per 24 hours)

1. Check current month
   └─ e.g., January 2026

2. Find previous month directories
   └─ modified/2025/12/, 2025/11/, 2025/10/, ... all

3. Compress each month
   ├─ tar -czf archived/2025-12-modified.tar.gz modified/2025/12/
   ├─ tar -czf archived/2025-12-patched.tar.gz patched/2025/12/
   └─ ...

4. Verify archives
   └─ tar -tzf for integrity check

5. Delete originals
   ├─ rm -rf modified/2025/12/
   └─ ...

6. Keep current month as-is
   └─ modified/2026/01/ preserved (originals kept)

7. Permanent archive preservation
   └─ archived/*.tar.gz never auto-deleted
```

**Recovery from archives**:
```bash
# List archived backups
ls -lh .multirepos/backup/archived/

# Extract specific month
tar -xzf .multirepos/backup/archived/2025-12-modified.tar.gz \
    -C .multirepos/backup/modified/

# Extract specific file only
tar -xzf .multirepos/backup/archived/2025-12-modified.tar.gz \
    2025/12/09/apps/api/config.json.20251209_143022
```

---

## Data Recovery Guide

### When files are accidentally reset

```bash
# 1. Find backup
ls .multirepos/backup/modified/2026/01/09/

# 2. Check latest backup
ls -lt .multirepos/backup/modified/2026/01/09/apps/api.log/

# 3. Recover
cp .multirepos/backup/modified/2026/01/09/apps/api.log/config.json.20260109_143022 \
   apps/api.log/config.json
```

### When patch application fails after pull

```bash
# Patch is saved in .git/git-multirepo/patches/ or backup
cd apps/api.log
patch -p1 < ../../.multirepos/patches/apps/api.log/config.json.patch
```

### When workspace is accidentally deleted

```bash
# ⚠️ remove deletes the workspace directory immediately
# Best practice: Run `git multirepo sync` before remove
# → Saves modified keep files to .multirepos/backup/

# If you forgot to sync before remove:
# 1. Modified keep files are lost (no backup)
# 2. Unmodified files can be recovered by re-cloning
git multirepo clone <url> <path>
```

### When recovering from archived backups

```bash
# Check archived backup size
du -sh .multirepos/backup/archived/*.tar.gz

# Extract specific month's backup
tar -xzf .multirepos/backup/archived/2025-12-modified.tar.gz \
    -C .multirepos/backup/

# Now files are in modified/2025/12/ - follow normal recovery steps
```

---

## How It Works

### Directory Structure

```
my-project/
├── .git/                    <- Parent project git
├── .git.multirepos          <- Workspace manifest (tracked by parent)
├── .gitignore               <- Contains "packages/lib/.git/"
├── .multirepos/             <- Backups and patches (gitignored)
│   ├── backup/              <- Modified file backups
│   └── patches/             <- Diff patches
├── src/
│   └── main.go
└── packages/
    └── lib/
        ├── .git/            <- Workspace's independent git (gitignored)
        └── lib.go           <- Tracked by parent repo
```

### Key Points

1. **Independent Git**: Each workspace has its own `.git` directory (local only)
2. **Source Tracking**: Parent tracks workspace's source files (not `.git`)
3. **Direct Push**: `cd packages/lib && git push` works as expected
4. **Manifest File**: `.git.multirepos` records all workspaces for recreation

### Workflow

**Developer A adds a workspace:**
```bash
git multirepo clone https://github.com/user/lib.git packages/lib
# Creates: packages/lib/.git/ (local)
# Ignores: packages/lib/.git/ → .gitignore
# Tracks: packages/lib/*.go → parent repo
# Records: path, repo, commit hash → .git.multirepos

git add .
git commit -m "Add lib workspace"
git push  # Pushes: source files + .git.multirepos (NOT .git)
```

**Developer A updates workspace:**
```bash
cd packages/lib
git commit && git push  # ← Must push to remote!

cd ../..
git add packages/lib/    # Stage updated source
git multirepo sync             # ← Auto-updates .git.multirepos with new commit!
git commit -m "Update lib"
git push
```

**Developer B clones:**
```bash
git clone <parent-repo>
# Gets: .git.multirepos + source files
# Missing: packages/lib/.git/

git multirepo sync  # or use post-checkout hook
# Reads: .git.multirepos commit hash
# Restores: .git at exact commit
# Now: cd packages/lib && git push works!
```

**Key Points:**
- `.git` directories are never pushed
- Commit hashes ensure version consistency
- `git multirepo sync` handles everything automatically
- Unpushed commits trigger warnings

### Manifest Format

```yaml
# .git.multirepos
workspaces:
  - path: packages/lib
    repo: https://github.com/user/lib.git
    commit: abc123def456789...
    keep:                          # Optional: local config files
      - config.json                # These files are backed up and restored
      - .env.local                 # Applied with skip-worktree
```

### Keep Files & Local Configuration

Preserve local configuration files across syncs and pulls:

**First sync with modifications:**
```bash
# You have local changes: config.json, .env, settings.yml
git multirepo sync

# Output:
# ✓ Found 3 modified files and added to keep list:
#   - config.json
#   - .env
#   - settings.yml
#
# Edit .git.multirepos to keep only the files you need
```

**.git.multirepos auto-updated:**
```yaml
workspaces:
  - path: apps/api
    repo: https://github.com/user/api.git
    keep:
      - config.json    # Auto-added
      - .env           # Auto-added
      - settings.yml   # Auto-added
```

**Edit to keep only what you need:**
```yaml
workspaces:
  - path: apps/api
    repo: https://github.com/user/api.git
    keep:
      - config.json    # Keep this
      # Removed .env and settings.yml
```

**How it works:**
- All modified files → patches created in `.multirepos/patches/`
- Keep files → restored with skip-worktree on pull/sync
- Non-keep files → patches saved but not restored (git updates them)
- Daily snapshots → `.multirepos/backup/` for history

**Directory structure:**
```
.git.multirepos                    # Configuration
.multirepos/
├── patches/{workspace}/           # Latest patches (for restore)
│   └── config.json.patch
└── backup/                        # Historical backups
    ├── modified/YYYY/MM/DD/      # Original files
    └── patched/YYYY/MM/DD/       # Patch history
```

## License

MIT

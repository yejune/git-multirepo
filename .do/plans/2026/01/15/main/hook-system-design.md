# 플랜: Git Multirepo 훅 시스템 설계 명확화

## 작업 요약

**목적:** 훅 시스템의 설계를 명확히 하고, 마커 정책을 통일하며, status 명령이 올바른 훅을 확인하도록 수정

**문제점:**
1. Workspace의 post-commit 훅이 status에서 감지되지 않음
2. post-checkout 훅은 마커 있지만, post-commit 훅은 마커 없음
3. IsWorkspaceHookInstalled()가 정확한 문자열 매칭 사용 (마커 추가 시 실패)
4. status.go가 모든 repo에서 post-checkout만 확인 (workspace의 post-commit 무시)

**해결 방안:**
1. post-commit 훅에도 마커 추가
2. IsWorkspaceHookInstalled()를 마커 기반 감지로 변경
3. status.go에서 Root는 post-checkout, Workspace는 post-commit 확인
4. 훅 상태 표시 시 훅 타입 명시 (post-checkout/post-commit)

**브랜치:** main

---

## 훅 시스템 설계

### 1. Root Repository: post-checkout 훅

**목적:** 브랜치 전환 시 워크스페이스 자동 동기화

**시나리오:**
```bash
cd /Users/max/Front          # Root
git checkout feature-branch  # 브랜치 전환
# → post-checkout 훅 실행
# → git-multirepo sync
# → 모든 워크스페이스가 feature-branch의 manifest에 맞춰 동기화
```

**훅 내용:**
```bash
#!/bin/sh
# === git-multirepo hook START ===
# git-multirepo post-checkout hook
# Automatically syncs subs after checkout
if command -v git-multirepo >/dev/null 2>&1; then
    git-multirepo sync
fi
# === git-multirepo hook END ===
```

**설치 시점:** `git multirepo install-hook`

---

### 2. Workspace: post-commit 훅

**목적:** 커밋 후 부모 repo의 manifest 자동 업데이트

**시나리오:**
```bash
cd /Users/max/Front/apps/api.log  # Workspace
git commit -m "fix bug"            # 커밋
# → post-commit 훅 실행
# → 부모 찾기 (.git.multirepos)
# → cd /Users/max/Front && git-multirepo sync
# → 부모의 .git.multirepos에 새 커밋 해시 기록
```

**훅 내용 (마커 추가):**
```bash
#!/bin/sh
# === git-multirepo hook START ===
# git-multirepo post-commit hook for sub repositories
# Automatically updates parent's .git.multirepos after commit

# Find parent repository (look for .git.multirepos)
find_parent() {
    local dir="$1"
    while [ "$dir" != "/" ] && [ "$dir" != "." ]; do
        dir=$(dirname "$dir")
        if [ -f "$dir/.git.multirepos" ]; then
            echo "$dir"
            return 0
        fi
    done
    return 1
}

# Get current repository root
SUB_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ -z "$SUB_ROOT" ]; then
    exit 0
fi

# Find parent repository
PARENT_ROOT=$(find_parent "$SUB_ROOT")
if [ -z "$PARENT_ROOT" ]; then
    # Not a sub repository, exit silently
    exit 0
fi

# Check if git-multirepo is available
if ! command -v git-multirepo >/dev/null 2>&1; then
    exit 0
fi

# Update parent's .git.multirepos
cd "$PARENT_ROOT" && git-multirepo sync 2>/dev/null || true
# === git-multirepo hook END ===
```

**설치 시점:** `git multirepo sync` 또는 `git multirepo clone` (자동)

---

## 구현 상세

### 파일 1: internal/hooks/hooks.go

#### 수정 1: postCommitHook 상수에 마커 추가 (lines 30-67)

**현재:**
```go
const postCommitHook = `#!/bin/sh
# git-multirepo post-commit hook for sub repositories
...
cd "$PARENT_ROOT" && git-multirepo sync 2>/dev/null || true
`
```

**변경 후:**
```go
const postCommitHook = `#!/bin/sh
` + hookMarkerStart + `
# git-multirepo post-commit hook for sub repositories
...
cd "$PARENT_ROOT" && git-multirepo sync 2>/dev/null || true
` + hookMarkerEnd
```

#### 수정 2: IsWorkspaceHookInstalled를 마커 기반으로 변경 (line 189)

**현재:**
```go
func IsWorkspaceHookInstalled(workspacePath string) bool {
    hookPath := filepath.Join(workspacePath, ".git", "hooks", "post-commit")
    content, err := os.ReadFile(hookPath)
    if err != nil {
        return false
    }
    return string(content) == postCommitHook  // ❌ 정확한 매칭
}
```

**변경 후:**
```go
func IsWorkspaceHookInstalled(workspacePath string) bool {
    hookPath := filepath.Join(workspacePath, ".git", "hooks", "post-commit")
    content, err := os.ReadFile(hookPath)
    if err != nil {
        return false
    }
    return strings.Contains(string(content), hookMarkerStart)  // ✓ 마커 기반
}
```

#### 수정 3: InstallWorkspaceHook를 merge 지원으로 변경 (lines 168-180)

**현재:** 덮어쓰기만 함

**변경 후:** post-checkout과 동일하게 merge 지원
```go
func InstallWorkspaceHook(workspacePath string) error {
    hooksDir := filepath.Join(workspacePath, ".git", "hooks")
    if err := os.MkdirAll(hooksDir, 0755); err != nil {
        return err
    }

    hookPath := filepath.Join(hooksDir, "post-commit")

    // Read existing hook if exists
    existingContent := ""
    if content, err := os.ReadFile(hookPath); err == nil {
        existingContent = string(content)

        // Check if our hook is already installed
        if strings.Contains(existingContent, hookMarkerStart) {
            return nil // Already installed
        }
    }

    // Merge: existing + our hook
    var newContent string
    if existingContent == "" {
        newContent = postCommitHook
    } else {
        // Append our hook to existing
        newContent = existingContent
        if !strings.HasSuffix(newContent, "\n") {
            newContent += "\n"
        }
        newContent += "\n" + postCommitHook
    }

    return os.WriteFile(hookPath, []byte(newContent), 0755)
}
```

---

### 파일 2: cmd/status.go

#### 수정 1: getHookStatusForRepo에 isRoot 파라미터 추가

**현재 (line 314):**
```go
func getHookStatusForRepo(repoPath string) (symbol string, description string) {
    status := getHookStatus(repoPath)
    // ...
}
```

**변경 후:**
```go
func getHookStatusForRepo(repoPath string, isRoot bool) (symbol string, description string) {
    var status HookStatus

    if isRoot {
        // Check post-checkout
        status = getHookStatus(repoPath, "post-checkout")
    } else {
        // Check post-commit
        status = getHookStatus(repoPath, "post-commit")
    }

    switch status {
    case HookOurs:
        hookType := "post-checkout"
        if !isRoot {
            hookType = "post-commit"
        }
        return "✓", fmt.Sprintf("Installed (%s)", hookType)
    case HookMixed:
        return "⚠️ ", "Merged with other hook"
    case HookOtherOnly:
        return "⚠️ ", "Other hook only (git-multirepo not installed)"
    case HookNone:
        return "✗", "Not installed"
    }

    return "?", "Unknown"
}
```

#### 수정 2: getHookStatus를 훅 타입별로 확인하도록 변경

**현재 (lines 275-312):**
```go
func getHookStatus(repoRoot string) HookStatus {
    hasOurs := hooks.IsInstalled(repoRoot)  // post-checkout만 확인
    hasAny := hooks.HasHook(repoRoot)       // post-checkout만 확인
    // ...
}
```

**변경 후:**
```go
func getHookStatus(repoRoot string, hookType string) HookStatus {
    hookPath := filepath.Join(repoRoot, ".git", "hooks", hookType)
    content, err := os.ReadFile(hookPath)
    if err != nil {
        return HookNone
    }

    hasOurs := strings.Contains(string(content), hooks.MarkerStart)

    if !hasOurs {
        return HookOtherOnly
    }

    // Check if mixed by removing our section
    startIdx := strings.Index(string(content), hooks.MarkerStart)
    endIdx := strings.Index(string(content), hooks.MarkerEnd)

    if startIdx == -1 || endIdx == -1 {
        return HookNone
    }

    before := string(content[:startIdx])
    after := string(content[endIdx+len(hooks.MarkerEnd):])
    remaining := strings.TrimSpace(before + after)

    if remaining == "" || remaining == "#!/bin/sh" {
        return HookOurs
    }

    return HookMixed
}
```

#### 수정 3: runStatus에서 호출 시 isRoot 전달

**Root 호출 (line 489):**
```go
symbol, desc := getHookStatusForRepo(ctx.RepoRoot, true)  // isRoot=true
```

**Workspace 호출 (line 546):**
```go
symbol, desc := getHookStatusForRepo(fullPath, false)  // isRoot=false
```

#### 수정 4: getHookSummary 수정하여 post-commit도 확인

**현재 (lines 653-677):**
```go
func getHookSummary(ctx *common.WorkspaceContext) string {
    // Root만 확인
    switch getHookStatus(ctx.RepoRoot) {
    // ...
    }

    // Workspaces도 post-checkout만 확인
    for _, ws := range ctx.Manifest.Workspaces {
        switch getHookStatus(fullPath) {
        // ...
        }
    }
}
```

**변경 후:**
```go
func getHookSummary(ctx *common.WorkspaceContext) string {
    ours, mixed, otherOnly, total := 0, 0, 0, 0

    // Root repo (post-checkout)
    total++
    switch getHookStatus(ctx.RepoRoot, "post-checkout") {
    case HookOurs:
        ours++
    case HookMixed:
        mixed++
    case HookOtherOnly:
        otherOnly++
    }

    // Workspaces (post-commit)
    for _, ws := range ctx.Manifest.Workspaces {
        fullPath := filepath.Join(ctx.RepoRoot, ws.Path)
        total++

        switch getHookStatus(fullPath, "post-commit") {
        case HookOurs:
            ours++
        case HookMixed:
            mixed++
        case HookOtherOnly:
            otherOnly++
        }
    }

    installed := ours + mixed

    if installed == total {
        return fmt.Sprintf("All %d hooks installed (1 root, %d workspaces).", total, total-1)
    }

    msg := fmt.Sprintf("Summary: %d/%d hooks installed", installed, total)
    if mixed > 0 {
        msg += fmt.Sprintf(" (%d merged)", mixed)
    }
    if otherOnly > 0 {
        msg += fmt.Sprintf(", %d need installation", otherOnly)
    }

    if installed < total {
        msg += ". Run 'git multirepo install-hook' to install."
    }

    return msg
}
```

---

### 파일 3: cmd/hook.go (선택적)

**현재:** install-hook이 모든 repo에 post-checkout 설치

**향후 개선 (선택):**
- Multirepo root 감지 시 workspace에는 post-commit 설치
- 현재는 sync/clone이 자동으로 post-commit 설치하므로 필수 아님

---

## 테스트 수정

### 파일: cmd/status_integrity_test.go

#### 수정할 테스트:

1. **TestStatusHookInfo** (lines 465-532)
   - getHookStatusForRepo 호출 시 isRoot 파라미터 추가

2. **TestStatusHookInfo_WithWorkspaces** (lines 534-631)
   - Workspace 섹션에서 post-commit 훅 확인하도록 수정

3. **TestStatusHookDifferentiation** (lines 633-738)
   - 4가지 상태가 post-checkout(root), post-commit(workspace)별로 구분되는지 검증

---

## 검증 계획

### 1. /Users/max/Front에서 확인

**실행:**
```bash
cd /Users/max/Front
git-multirepo status
```

**기대 결과:**
```
Repository: . (root)
  Hook: ✓ Installed (post-checkout)  ← 마커 있음
  Branch: main

Repository: apps/api.log
  Hook: ✓ Installed (post-commit)    ← 마커 있음, 이제 감지됨
  Branch: master

...

Summary: 23/23 hooks installed (1 root, 22 workspaces).
```

### 2. 새로운 workspace에서 확인

**실행:**
```bash
cd /Users/max/Front
git multirepo clone https://github.com/user/new-repo.git apps/new-repo
git multirepo status
```

**기대 결과:**
- apps/new-repo에 post-commit 훅 자동 설치 (마커 포함)
- status에서 "✓ Installed (post-commit)" 표시

### 3. 테스트 실행

**실행:**
```bash
cd /Users/max/Work/git-multirepo.workspace/git-multirepo
go test ./cmd -run "TestStatus.*Hook" -v
go test ./internal/hooks -v
go test ./cmd -v
```

**기대:** 모든 테스트 통과

---

## 변경 파일 요약

| 파일 | 변경 내용 | 예상 라인 |
|------|----------|-----------|
| internal/hooks/hooks.go | postCommitHook에 마커 추가 | 30-67 |
| internal/hooks/hooks.go | IsWorkspaceHookInstalled 마커 기반으로 변경 | 189 |
| internal/hooks/hooks.go | InstallWorkspaceHook merge 지원 추가 | 168-180 |
| cmd/status.go | getHookStatusForRepo isRoot 파라미터 추가 | 314-342 |
| cmd/status.go | getHookStatus hookType 파라미터 추가 | 275-312 |
| cmd/status.go | runStatus에서 isRoot 전달 | 489, 546 |
| cmd/status.go | getHookSummary post-commit 확인 추가 | 653-677 |
| cmd/status_integrity_test.go | 테스트 수정 | 465-738 |

**예상 변경량:**
- 제거: ~20줄
- 추가: ~100줄
- 수정: ~50줄

---

## 예상 출력 비교

### Before (현재):
```
Repository: . (root)
  Hook: ✓ Installed (git-multirepo only)

Repository: apps/api.log
  Hook: ✗ Not installed              ← 실제로는 post-commit 훅 있음

Summary: 1/23 hooks installed.
```

### After (개선):
```
Repository: . (root)
  Hook: ✓ Installed (post-checkout)  ← 훅 타입 명시

Repository: apps/api.log
  Hook: ✓ Installed (post-commit)    ← post-commit 감지됨

Summary: 23/23 hooks installed (1 root, 22 workspaces).
```

---

## 플랜 백업 경로 규칙 (추가 요구사항)

**현재 경로:** `.claude/plans/{random-name}.md`

**새 경로 규칙:** `.do/plans/yyyy/mm/dd/{branch}/{제목}.md`

**예시:**
- `.do/plans/2026/01/15/main/hook-system-design.md`
- `.do/plans/2026/01/15/feature/user-auth.md`

**CLAUDE.md 지침 업데이트:**
```markdown
### 플랜 관리 규칙 [HARD]

#### 플랜 생성
- [HARD] 플랜모드(EnterPlanMode) 종료 시 반드시 플랜을 파일로 저장
- [HARD] 저장 경로: `.do/plans/{YYYY}/{MM}/{DD}/{branch}/{제목}.md`
  - {YYYY}: 4자리 연도 (예: 2026)
  - {MM}: 2자리 월 (예: 01)
  - {DD}: 2자리 일 (예: 15)
  - {branch}: git 브랜치명 (예: main, feature/auth)
  - {제목}: 작업 내용을 설명하는 짧은 제목 (kebab-case 권장)
- [HARD] 디렉토리가 없으면 자동으로 생성
- 예시: `.do/plans/2026/01/15/main/hook-system-design.md`
```

---

## 최종 목표

✅ post-commit 훅에 마커 추가하여 일관성 확보
✅ IsWorkspaceHookInstalled가 마커 기반으로 감지
✅ status.go가 Root는 post-checkout, Workspace는 post-commit 확인
✅ 훅 상태 표시 시 훅 타입 명시
✅ /Users/max/Front에서 22개 workspace "✓ Installed" 표시
✅ 모든 테스트 통과
✅ 플랜 파일 백업 경로에 브랜치명 포함

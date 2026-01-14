# 병렬 처리 롤백 솔루션

**목표:** 36초 성능 복구
**방법:** 커밋 `8295be5^` (병렬 처리 이전) 버전으로 복원
**소요 시간:** 5분

---

## 🎯 롤백 대상

### 제거할 함수

1. **`discoverWorkspaces()`** (라인 257-305)
   - Channel 기반 discovery
   - 불필요한 복잡성

2. **`processWorkspacesParallel()`** (라인 308-380)
   - Worker pool 구현
   - Mutex contention 주범

3. **`workspaceDiscovery` 타입** (라인 251-254)
   - 헬퍼 구조체
   - 더 이상 필요 없음

### 복원할 함수

**`scanForWorkspaces()`** - 순차 처리 버전
- 간단하고 검증됨
- filepath.Walk 내에서 직접 처리
- 36초 성능 보장

---

## 📝 구현 계획

### 1단계: 백업 (선택사항)

```bash
# 현재 병렬 처리 버전 백업
cp cmd/sync.go cmd/sync.go.parallel.backup
```

### 2단계: 이전 버전 확인

```bash
# 병렬 처리 이전 커밋 확인
git show 8295be5^:cmd/sync.go > /tmp/sync_sequential.go

# 차이 확인
diff cmd/sync.go /tmp/sync_sequential.go | head -100
```

### 3단계: 선택적 복원

**복원할 부분:** `scanForWorkspaces()` 함수만

**제거할 부분:**
- `type workspaceDiscovery struct` (라인 251-254)
- `func discoverWorkspaces()` (라인 257-305)
- `func processWorkspacesParallel()` (라인 308-380)

**복원할 코드:** (라인 248-327을 아래로 교체)

```go
// scanForWorkspaces recursively scans directories for git repositories
func scanForWorkspaces(repoRoot string) ([]manifest.WorkspaceEntry, error) {
	var workspaces []manifest.WorkspaceEntry

	// Walk the directory tree
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip parent's .git directory
		if path == filepath.Join(repoRoot, ".git") {
			return filepath.SkipDir
		}

		// Check if this is a .git directory
		if !info.IsDir() || info.Name() != ".git" {
			return nil
		}

		// Get the repository path (parent of .git)
		workspacePath := filepath.Dir(path)

		// Skip if it's the parent repo itself
		if workspacePath == repoRoot {
			return filepath.SkipDir
		}

		// Get relative path from parent
		relPath, err := filepath.Rel(repoRoot, workspacePath)
		if err != nil {
			return nil
		}

		// Extract git info
		repo, err := git.GetRemoteURL(workspacePath)
		if err != nil {
			// Warning only - continue processing workspace with empty remote
			fmt.Printf("⚠ %s\n", i18n.T("warn_no_remote", relPath))
			repo = "" // Empty remote is valid for local-only repos
		}

		// Detect modified files for auto-keep
		var keepFiles []string
		// Get skip-worktree files (these are the keep files)
		skipFiles, err := git.ListSkipWorktree(workspacePath)
		if err == nil && len(skipFiles) > 0 {
			keepFiles = skipFiles
		} else {
			// Fallback: detect modified files for first-time setup
			var modifiedFiles []string
			git.WithSkipWorktreeTransaction(workspacePath, []string{}, func() error {
				var err error
				modifiedFiles, err = git.GetModifiedFiles(workspacePath)
				return err
			})
			if len(modifiedFiles) > 0 {
				// Clean up file list
				for _, file := range modifiedFiles {
					if strings.TrimSpace(file) != "" {
						keepFiles = append(keepFiles, file)
					}
				}
			}
		}

		fmt.Printf("  %s\n", i18n.T("found_sub", relPath))

		workspaces = append(workspaces, manifest.WorkspaceEntry{
			Path: relPath,
			Repo: repo,
			Keep: keepFiles,
		})

		// Skip descending into this workspace's subdirectories
		return filepath.SkipDir
	})

	return workspaces, err
}
```

**주의사항:**
- `failed_get_remote` → `warn_no_remote` (메시지 변경됨)
- 에러 처리 로직이 약간 다름 (경고만 출력하고 계속 진행)

### 4단계: Import 정리

**확인 필요:**
```go
import (
    // ... 기존 import ...
    "context"  // 다른 곳에서 사용하는지 확인
    "sync"     // 다른 곳에서 사용하는지 확인
    "golang.org/x/sync/errgroup"  // 제거 가능
)
```

**명령어:**
```bash
# 사용되지 않는 import 확인
grep -n "context\." cmd/sync.go
grep -n "sync\." cmd/sync.go
grep -n "errgroup" cmd/sync.go
```

**결과에 따라:**
- `processKeepFiles`에서 사용 안 함 → 제거 가능
- 다른 함수에서 사용 → 유지

### 5단계: 테스트

```bash
# 빌드 확인
go build ./cmd/...

# 단위 테스트
go test -v ./cmd -run "^TestScanForWorkspaces_"

# 성능 테스트
go test -v ./cmd -run "^TestSync_Performance_100Workspaces$"
# Expected: ~36-40초 (워크스페이스 생성 포함)

# 전체 테스트
go test -v ./cmd -timeout 5m
```

### 6단계: 커밋

```bash
git add cmd/sync.go
git commit -m "fix: revert parallel workspace scanning (40% performance regression)

Rollback worker pool implementation due to performance degradation:
- Before (sequential): 36s for 100 workspaces
- After (parallel): 50.42s for 100 workspaces (40% slower)

Root cause: Mutex contention and goroutine overhead exceeded
parallelization benefits in test environment (tmpfs).

Restored simple sequential filepath.Walk implementation.

Related: #perf"
```

---

## 🧪 검증 체크리스트

- [ ] 빌드 성공 (`go build ./cmd/...`)
- [ ] Import 오류 없음
- [ ] 기능 테스트 통과 (57개)
- [ ] 성능 36초대 복구
- [ ] 에러 처리 동작 확인
- [ ] 경고 메시지 출력 확인

---

## 🔧 대안: Git 명령 사용 (더 안전)

수동 복원이 어려우면 git을 사용:

```bash
# scanForWorkspaces 함수만 복원
git show 8295be5^:cmd/sync.go | \
    sed -n '/^func scanForWorkspaces/,/^func processKeepFiles/p' | \
    head -n -1 > /tmp/scan_func.go

# 수동으로 복사/붙여넣기
```

또는 전체 파일 복원 후 필요한 부분만 유지:

```bash
# 전체 복원
git checkout 8295be5^ -- cmd/sync.go

# processKeepFiles 이하는 현재 버전 유지 (필요시)
git diff HEAD cmd/sync.go
```

---

## ⚠️ 주의사항

### 메시지 변경 확인

**병렬 이전:**
```go
fmt.Println(i18n.T("failed_get_remote", relPath, err))
```

**병렬 이후:**
```go
fmt.Printf("⚠ %s\n", i18n.T("warn_no_remote", relPath))
```

**선택:**
- `warn_no_remote` 사용 (권장) - 더 명확한 경고
- 또는 `failed_get_remote` 복원 - 완전한 롤백

### 에러 처리 차이

**병렬 이전:**
```go
repo, err := git.GetRemoteURL(workspacePath)
if err != nil {
    fmt.Println(i18n.T("failed_get_remote", relPath, err))
    return filepath.SkipDir  // 워크스페이스 스킵
}
```

**권장 (병렬 이후 로직):**
```go
repo, err := git.GetRemoteURL(workspacePath)
if err != nil {
    fmt.Printf("⚠ %s\n", i18n.T("warn_no_remote", relPath))
    repo = ""  // 빈 remote로 계속 진행
}
```

**이유:** Local-only 저장소 지원 (더 유연함)

---

## 📊 예상 결과

### 롤백 전 (현재)

```
TestSync_Performance_100Workspaces
  ⚡ Performance requirement: 100 workspaces × 50 files < 3 seconds
--- PASS: TestSync_Performance_100Workspaces (50.42s)
```

### 롤백 후 (예상)

```
TestSync_Performance_100Workspaces
  ⚡ Performance requirement: 100 workspaces × 50 files < 3 seconds
--- PASS: TestSync_Performance_100Workspaces (36-40s)
```

**개선:** 40% 성능 회복 (50.42s → 36s)

---

## 🚀 다음 단계

롤백 후 실제 최적화 방향:

1. **Git 명령어 배치 최적화**
   - `GetRemoteURL` + `ListSkipWorktree` + `GetModifiedFiles`
   - 3번 → 1번 호출로 통합
   - 예상: 36s → 25s

2. **실제 I/O 환경 테스트**
   - tmpfs 아닌 실제 디스크에서 측정
   - 병렬화 이득이 실제로 있는지 확인

3. **프로파일링 기반 최적화**
   - `go test -cpuprofile=cpu.out`
   - 실제 병목 지점 파악
   - 데이터 기반 최적화

---

**작성:** Do Performance Expert
**검증:** 롤백 후 성능 테스트로 확인
**목표:** 36초 성능 복구

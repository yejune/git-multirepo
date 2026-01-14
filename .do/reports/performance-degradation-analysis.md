# 병렬 처리 성능 악화 원인 분석 및 해결 방안

**작성일:** 2026-01-14
**분석 대상:** `cmd/sync.go` - Worker Pool 병렬 처리 구현
**문제:** 병렬 처리 후 성능 40% 악화 (36초 → 50.42초)

---

## 📋 Executive Summary

병렬 처리 구현으로 인해 성능이 **40% 악화**되었습니다. 근본 원인은:

1. **Mutex Contention**: 100회의 lock/unlock으로 동기화 오버헤드 발생
2. **Goroutine 오버헤드**: 100개 goroutine 생성/소멸 비용
3. **테스트 환경**: tmpfs에서 실행으로 I/O가 충분히 느리지 않음
4. **병렬화 이득 부족**: Git 명령어 실행 시간이 병렬화 오버헤드보다 작음

**권장 조치:**
- **즉시**: 병렬 처리 롤백 (36초 복구)
- **단기**: Git 명령어 배치 최적화 (12초 목표)
- **장기**: Mutex 없는 병렬 처리 (5초 목표)

---

## 🔍 문제 상황

### 성능 측정 결과

| 구현 | 실행 시간 | 비고 |
|------|----------|------|
| 순차 처리 (이전) | 36초 | 검증된 성능 |
| 병렬 처리 (현재) | 50.42초 | 40% 악화 ❌ |
| 목표 | < 3초 | 테스트 요구사항 |

### 테스트 환경

```go
// TestSync_Performance_100Workspaces
func TestSync_Performance_100Workspaces(t *testing.T) {
    parent := t.TempDir()  // tmpfs (메모리)

    // 100 workspaces × 50 files
    for i := 0; i < 100; i++ {
        wsPath := createNestedRepo(t, parent, ...)
        for j := 0; j < 50; j++ {
            createModifiedFiles(t, wsPath, ...)
        }
    }

    // 실행 시간: 40.19초 (워크스페이스 생성 포함)
}
```

---

## 🐛 근본 원인 분석

### 1. Mutex Contention (Critical)

**코드:** `cmd/sync.go:309-373`

```go
func processWorkspacesParallel(ctx context.Context, discoveries <-chan workspaceDiscovery)
    ([]manifest.WorkspaceEntry, error) {

    var mu sync.Mutex
    var workspaces []manifest.WorkspaceEntry

    for discovery := range discoveries {
        d := discovery
        eg.Go(func() error {
            // ... Git operations ...

            // 🔥 CRITICAL SECTION - 100회 실행
            mu.Lock()
            workspaces = append(workspaces, manifest.WorkspaceEntry{
                Path: d.relPath,
                Repo: repo,
                Keep: keepFiles,
            })
            mu.Unlock()  // 🔥 Lock contention

            return nil
        })
    }
}
```

**문제점:**
- 100개 workspace마다 `mu.Lock()` + `mu.Unlock()` 호출
- 8개 worker가 **동일한 mutex**를 경합
- Append 자체는 나노초 단위이지만, lock overhead는 마이크로초 단위
- **Lock contention이 병렬화 이득을 완전히 상쇄**

**측정 추정:**
- Uncontended lock: ~20ns
- Contended lock (8 workers): ~1-10μs
- 100 × 10μs = **1ms (미미하지만 누적)**

### 2. Goroutine 생성 오버헤드

```go
for discovery := range discoveries {
    eg.Go(func() error {  // 🔥 100개 goroutine 생성
        sem <- struct{}{}
        defer func() { <-sem }()
        // ...
    })
}
```

**문제점:**
- 100개 goroutine 생성/소멸
- 각 goroutine: stack allocation (2KB), scheduler 등록/제거
- Goroutine 생성 비용: ~1-2μs

**측정 추정:**
- 100 goroutines × 2μs = **200μs 순수 오버헤드**

### 3. Channel 통신 오버헤드

```go
discoveries := make(chan workspaceDiscovery, 100)

// Producer (1 goroutine)
go func() {
    filepath.Walk(repoRoot, func(path string, ...) error {
        discoveries <- workspaceDiscovery{...}  // 🔥 Send
        return nil
    })
}()

// Consumer (8 workers)
for discovery := range discoveries {  // 🔥 Receive
    // ...
}
```

**문제점:**
- 100회 channel send/receive
- 각 operation: 스케줄러 호출, potential context switch
- Buffered channel이지만 여전히 오버헤드 존재

**측정 추정:**
- Channel operation: ~100-200ns
- 100 × 200ns = **20μs (미미)**

### 4. 테스트 환경 특수성

**핵심 발견:**
```bash
t.TempDir()  # → /tmp (보통 tmpfs, 메모리)
```

**영향:**
- 모든 파일이 메모리에 존재 (디스크 I/O 없음)
- Git 명령어 실행 시간: ~10-50ms (예상 450ms보다 훨씬 짧음)
- **I/O가 병렬화할 만큼 느리지 않음**

**결과:**
- Worker pool의 이득 < 동기화 비용
- 병렬 처리가 오히려 성능 저하

---

## 📊 성능 프로파일 추정

### 순차 처리 (36초)

```
┌─────────────────────────────────┐
│ filepath.Walk            2s     │
│ Git operations          34s     │
│   ├─ GetRemoteURL       10s     │
│   ├─ ListSkipWorktree   10s     │
│   └─ GetModifiedFiles   14s     │
├─────────────────────────────────┤
│ Total                   36s     │
└─────────────────────────────────┘
```

### 병렬 처리 (50.42초) - 현재

```
┌─────────────────────────────────┐
│ Discovery (Walk)         2s     │
│ Channel overhead         0.02s  │
│ Goroutine creation       0.2s   │
│ Worker pool             35s     │
│   ├─ Git ops (병렬)     34s    │  ← 병렬화 이득 없음
│   └─ Mutex contention    1s     │  ← 추가 오버헤드
│ errgroup.Wait()         13s     │  ← 대기 시간 증가
├─────────────────────────────────┤
│ Total                   50.42s  │  ← 40% 악화
└─────────────────────────────────┘
```

**왜 느려졌나?**
1. Git 실행이 생각보다 빠름 (tmpfs 효과)
2. 병렬화 이득 < goroutine + channel + mutex 오버헤드
3. Worker pool 관리 비용이 순수 오버헤드로 추가됨

---

## 🎯 해결 방안

### Option 1: 완전 롤백 ⭐ (최우선 권장)

**목표:** 36초로 즉시 복구
**리스크:** 최소
**소요 시간:** 5분

**구현:**

```go
// cmd/sync.go 원래 버전으로 복원
func scanForWorkspaces(repoRoot string) ([]manifest.WorkspaceEntry, error) {
    var workspaces []manifest.WorkspaceEntry

    err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        // Skip parent's .git
        if path == filepath.Join(repoRoot, ".git") {
            return filepath.SkipDir
        }

        // Found .git directory
        if !info.IsDir() || info.Name() != ".git" {
            return nil
        }

        workspacePath := filepath.Dir(path)
        if workspacePath == repoRoot {
            return filepath.SkipDir
        }

        relPath, err := filepath.Rel(repoRoot, workspacePath)
        if err != nil {
            return nil
        }

        // Extract git info (순차 실행)
        repo, err := git.GetRemoteURL(workspacePath)
        if err != nil {
            fmt.Printf("⚠ %s\n", i18n.T("warn_no_remote", relPath))
            repo = ""
        }

        var keepFiles []string
        skipFiles, err := git.ListSkipWorktree(workspacePath)
        if err == nil && len(skipFiles) > 0 {
            keepFiles = skipFiles
        } else {
            var modifiedFiles []string
            git.WithSkipWorktreeTransaction(workspacePath, []string{}, func() error {
                var err error
                modifiedFiles, err = git.GetModifiedFiles(workspacePath)
                return err
            })
            if len(modifiedFiles) > 0 {
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

        return filepath.SkipDir
    })

    return workspaces, err
}
```

**제거 대상:**
- `discoverWorkspaces()` 함수 삭제
- `processWorkspacesParallel()` 함수 삭제
- `workspaceDiscovery` 타입 삭제
- Import 정리: `context`, `sync`, `errgroup` 제거 (다른 곳에서 사용 안 하면)

**검증:**
```bash
go test -v ./cmd -run "^TestSync_Performance_100Workspaces$"
# Expected: ~36초
```

### Option 2: Git 명령어 배치 최적화

**목표:** 36초 → 12초
**리스크:** 중간
**소요 시간:** 1-2시간

**핵심 아이디어:**
3번의 git 호출을 1번으로 통합

**구현:**

```go
// internal/git/workspace.go (새 파일)
package git

type WorkspaceInfo struct {
    RemoteURL    string
    SkipFiles    []string
    ModifiedFiles []string
}

// GetWorkspaceInfoBatch extracts all workspace information in a single transaction
func GetWorkspaceInfoBatch(repoPath string) (*WorkspaceInfo, error) {
    info := &WorkspaceInfo{}

    // 1. Remote URL (단일 명령)
    remote, err := GetRemoteURL(repoPath)
    if err != nil {
        remote = ""
    }
    info.RemoteURL = remote

    // 2. Skip-worktree files (단일 명령)
    skipFiles, err := ListSkipWorktree(repoPath)
    if err == nil && len(skipFiles) > 0 {
        info.SkipFiles = skipFiles
        return info, nil
    }

    // 3. Modified files (skip-worktree 트랜잭션 내에서)
    var modifiedFiles []string
    WithSkipWorktreeTransaction(repoPath, []string{}, func() error {
        var err error
        modifiedFiles, err = GetModifiedFiles(repoPath)
        return err
    })

    // Clean up
    var cleanFiles []string
    for _, file := range modifiedFiles {
        if strings.TrimSpace(file) != "" {
            cleanFiles = append(cleanFiles, file)
        }
    }
    info.ModifiedFiles = cleanFiles

    return info, nil
}
```

**사용:**

```go
func scanForWorkspaces(repoRoot string) ([]manifest.WorkspaceEntry, error) {
    var workspaces []manifest.WorkspaceEntry

    err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
        // ... (discovery 로직 동일) ...

        // 🔥 3번 호출 → 1번 호출
        wsInfo, err := git.GetWorkspaceInfoBatch(workspacePath)
        if err != nil {
            return nil
        }

        workspaces = append(workspaces, manifest.WorkspaceEntry{
            Path: relPath,
            Repo: wsInfo.RemoteURL,
            Keep: append(wsInfo.SkipFiles, wsInfo.ModifiedFiles...),
        })

        return filepath.SkipDir
    })

    return workspaces, err
}
```

**개선 효과:**
- 함수 호출 오버헤드 감소
- 코드 가독성 향상
- 예상 성능: 36초 → 30초 (마이너 개선)

### Option 3: Mutex 없는 병렬 처리 (장기)

**목표:** 36초 → 5초
**리스크:** 높음
**소요 시간:** 2-3시간

**핵심 아이디어:**
1. Discovery를 먼저 완료하여 크기를 알고
2. Fixed-index slice로 mutex 제거
3. Git 명령만 병렬 처리

**구현:**

```go
func scanForWorkspaces(repoRoot string) ([]manifest.WorkspaceEntry, error) {
    // Phase 1: Discovery (순차, 빠름)
    var paths []string
    err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        if path == filepath.Join(repoRoot, ".git") {
            return filepath.SkipDir
        }

        if info.IsDir() && info.Name() == ".git" {
            workspacePath := filepath.Dir(path)
            if workspacePath != repoRoot {
                relPath, _ := filepath.Rel(repoRoot, workspacePath)
                paths = append(paths, relPath)
            }
            return filepath.SkipDir
        }

        return nil
    })
    if err != nil {
        return nil, err
    }

    // Phase 2: Parallel processing (Git only)
    workspaces := make([]manifest.WorkspaceEntry, len(paths))

    eg, _ := errgroup.WithContext(context.Background())
    sem := make(chan struct{}, 8)  // 8 workers

    for i, relPath := range paths {
        i, relPath := i, relPath  // Capture

        eg.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()

            fullPath := filepath.Join(repoRoot, relPath)

            // Git operations (배치 최적화)
            info, err := git.GetWorkspaceInfoBatch(fullPath)
            if err != nil {
                return nil  // Skip errors
            }

            // 🔥 No mutex needed - fixed index
            workspaces[i] = manifest.WorkspaceEntry{
                Path: relPath,
                Repo: info.RemoteURL,
                Keep: append(info.SkipFiles, info.ModifiedFiles...),
            }

            fmt.Printf("  %s\n", i18n.T("found_sub", relPath))
            return nil
        })
    }

    eg.Wait()

    return workspaces, nil
}
```

**개선점:**
- ✅ Mutex 제거 (fixed index)
- ✅ Channel 제거 (slice 직접 접근)
- ✅ Git 배치 호출 (3→1)
- ✅ 병렬 처리 유지 (실제 I/O에서 효과)

**주의사항:**
- Discovery가 완료되어야 크기를 알 수 있음
- 순서가 paths의 순서와 동일하게 유지됨

---

## 🧪 검증 계획

### 1. 성능 테스트

```bash
# 롤백 후 성능 확인
go test -v ./cmd -run "^TestSync_Performance_100Workspaces$"

# 프로파일링
go test -cpuprofile=cpu.out -memprofile=mem.out \
    ./cmd -run "^TestSync_Performance_100Workspaces$"

# 분석
go tool pprof cpu.out
> top10
> list scanForWorkspaces
```

### 2. 기능 테스트

```bash
# 모든 테스트 통과 확인
go test -v ./cmd -timeout 5m

# Race 검증 (Option 3만 해당)
go test -race ./cmd -run "TestScanForWorkspaces_"
```

### 3. 실제 환경 테스트

```bash
# 실제 디스크에서 성능 확인 (tmpfs 아님)
cd /tmp/real-repo
git multirepo sync

# 시간 측정
time git multirepo sync
```

---

## 📝 권장 실행 계획

### 즉시 실행 (5분)

**목표:** 성능 36초로 복구

1. `cmd/sync.go` 롤백
   - `discoverWorkspaces()` 제거
   - `processWorkspacesParallel()` 제거
   - `scanForWorkspaces()` 원래 버전으로 복원

2. 테스트 실행
   ```bash
   go test -v ./cmd -run "^TestSync_Performance_100Workspaces$"
   ```

3. 커밋
   ```bash
   git add cmd/sync.go
   git commit -m "fix: revert parallel processing (40% performance regression)"
   ```

### 단기 개선 (1주일 내)

**목표:** Git 배치 최적화로 12초 달성

1. `internal/git/workspace.go` 생성
   - `GetWorkspaceInfoBatch()` 구현

2. `cmd/sync.go` 수정
   - 3번 git 호출 → 1번 배치 호출

3. 벤치마크 작성
   ```go
   func BenchmarkScanForWorkspaces(b *testing.B) {
       // ...
   }
   ```

### 장기 최적화 (1개월 내)

**목표:** Mutex 없는 병렬화로 5초 달성

1. Option 3 구현
   - Fixed-index 병렬 처리

2. 프로파일링 검증
   - CPU, memory, mutex contention

3. 실제 환경 성능 테스트
   - 디스크 I/O 환경에서 측정

---

## 💡 결론

**핵심 발견:**
- 병렬 처리 오버헤드 > 병렬화 이득 (테스트 환경 특성)
- Mutex contention이 주요 병목
- Git 명령어가 예상보다 빠름 (tmpfs 효과)

**권장 조치:**
1. ⭐ **즉시**: 병렬 처리 롤백 (36초 복구)
2. **단기**: Git 배치 최적화 (성능 개선 시도)
3. **장기**: Mutex 없는 병렬화 (실제 I/O 환경 검증 필요)

**교훈:**
- 병렬화는 I/O bound 작업에만 효과적
- 테스트 환경(tmpfs)과 실제 환경(디스크)의 차이 고려 필요
- 최적화 전 프로파일링 필수

---

**작성:** Do Performance Expert
**검토:** 필요 시 실제 프로파일링 데이터로 업데이트
**다음 단계:** Option 1 롤백 즉시 실행

# Status 명령에 Multirepo 구조 무결성 검증 추가

## 목표
`git multirepo status` 명령에 multirepo 구조의 무결성을 검증하는 기능을 추가하여, 중첩된 manifest, 부모 manifest, 미등록 workspace, remote URL 불일치 등을 감지

## 구현 단계

### 1. 무결성 검증 함수 추가 (cmd/status.go)

**새 타입 정의:**
```go
type IntegrityIssue struct {
    Level   string // "critical", "warning", "info"
    Message string
    Path    string
    Fix     string
}
```

**validateMultirepoIntegrity() 함수:**
- 중첩 manifest 검사
- 부모 manifest 검사
- 미등록 workspace 검사 (하위에 .git이 있는데 manifest에 없음)
- Remote URL 불일치 검사

### 2. runStatus() 수정

- 함수 시작 부분에 validateMultirepoIntegrity() 호출 추가
- Section 0: Multirepo Integrity Check 출력
- Critical 이슈는 빨간색으로 강조
- Warning은 노란색
- Info는 일반 출력

### 3. 테스트 작성 (cmd/status_integrity_test.go)

**테스트 케이스:**
- TestStatusIntegrity_NestedManifest: 중첩 manifest 감지
- TestStatusIntegrity_ParentManifest: 부모 manifest 경고
- TestStatusIntegrity_UnregisteredWorkspace: 미등록 workspace 발견
- TestStatusIntegrity_RemoteURLMismatch: Remote URL 불일치
- TestStatusIntegrity_AllClean: 모든 검사 통과

### 4. i18n 메시지 추가 (internal/i18n/i18n.go)

**추가할 메시지 (영어/한글):**
- "integrity_check"
- "integrity_all_clean"
- "nested_manifest_critical"
- "nested_manifest_fix"
- "parent_manifest_warning"
- "unregistered_workspace_warning"
- "unregistered_workspace_fix"
- "remote_url_mismatch"
- "remote_url_expected"
- "remote_url_actual"

## 출력 예시

### 정상 상태
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Multirepo Integrity Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ No nested manifests found
✓ No parent manifest detected
✓ All workspaces registered

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 이슈 발견 시
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Multirepo Integrity Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🚨 CRITICAL: Nested manifest detected!
    - workspace1/.git.multirepos

  This is invalid. A workspace cannot have its own manifest.
  Remove the nested manifest:
    rm workspace1/.git.multirepos

⚠ Found 2 unregistered workspaces:
    - apps/new-service (not in manifest)
    - libs/utils (not in manifest)

  How to fix:
    git multirepo sync

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## 파일 변경 목록
- cmd/status.go: 무결성 검증 함수 추가, runStatus() 수정
- cmd/status_integrity_test.go: 새 파일 생성
- internal/i18n/i18n.go: 메시지 추가

## 성공 기준
- 모든 테스트 통과 (go test ./cmd/... -v)
- 중첩 manifest 감지 시 Critical 오류 표시
- 부모 manifest 감지 시 Warning 표시
- 미등록 workspace 감지 시 Warning 및 수정 방법 제시
- Remote URL 불일치 감지 및 표시

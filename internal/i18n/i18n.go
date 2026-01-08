// Package i18n provides internationalization support for git-sub
package i18n

import "fmt"

var currentLang = "en"

var messages = map[string]map[string]string{
	"en": {
		// Pull command
		"uncommitted_files":   "%d uncommitted file(s)",
		"clean_directory":     "Clean",
		"pull_confirm":        "Pull? (Y/n): ",
		"pull_updated":        "✓ Updated (%d file(s) changed)",
		"pull_already_uptodate": "✓ Already up to date",
		"pull_failed":         "✗ Failed",
		"pull_skipped":        "→ Skipped",
		"run_status":          "→ Run: git sub status %s",
		"not_git_repo":        "→ Not a git repository, skipping",
		"failed_get_branch":   "✗ Failed to get branch: %v",
		"failed_read_input":   "✗ Failed to read input: %v",
		"no_subs_registered":  "No subs registered",
		"sub_not_found":       "sub not found: %s",

		// Status command
		"local_status":        "Local Status:",
		"files_modified":      "✗ %d file(s) modified:",
		"files_untracked":     "⚠ %d file(s) untracked:",
		"files_staged":        "● %d file(s) staged:",
		"clean_working_tree":  "✓ Clean working tree",
		"remote_status":       "Remote Status:",
		"commits_behind":      "→ %d commit(s) behind origin/%s",
		"commits_ahead":       "→ %d commit(s) ahead (unpushed)",
		"up_to_date":          "✓ Up to date with origin",
		"cannot_fetch":        "⚠ Cannot fetch from remote",
		"skip_files":          "Skip Files:",
		"skip_file_changed":   "⚠ %s changed in remote",
		"skip_remote_added":   "Remote added new lines",
		"skip_remote_removed": "Remote removed lines",
		"skip_remote_modified": "Remote modified content",
		"skip_file_protected": "(Your local file is protected by skip-worktree)",
		"no_remote_changes":   "✓ No remote changes in skip files",
		"how_to_resolve":      "How to resolve:",
		"no_action_needed":    "✓ No action needed",
		"not_cloned":          "(not cloned)",
		"resolve_commit":      "1. Commit or stash changes:",
		"resolve_or_gitignore": "# Or add untracked files to .gitignore",
		"resolve_pull":        "2. Pull updates:",
		"resolve_push":        "3. Push commits:",
		"resolve_skip":        "4. (Optional) Update skip files:",
		"resolve_review":      "# Review and merge changes",

		// Sync command
		"syncing":              "Syncing configuration...",
		"installing_hooks":     "→ Installing git hooks",
		"hooks_installed":      "✓ Installed",
		"hooks_failed":         "✗ Failed: %v",
		"no_gitsubs_found":     "\n→ No .workspaces found. Scanning for existing sub repositories...",
		"no_subs_found":        "✓ No sub repositories found",
		"to_add_sub":           "\nTo add a sub, use:",
		"cmd_git_sub_clone":    "  git sub clone <url> <path>",
		"created_gitsubs":      "\n✓ Created .workspaces with %d sub(s)",
		"applying_ignore":      "\n→ Applying ignore patterns",
		"applied_patterns":     "✓ Applied %d patterns",
		"applying_skip_mother": "→ Applying skip-worktree to mother repo",
		"applied_files":        "✓ Applied to %d files",
		"processing_mother_keep": "→ Processing mother repo keep files",
		"no_subclones":         "\nNo subclones registered.",
		"processing_subclones": "\n→ Processing subclones:",
		"initializing_git":     "→ Initializing .git (source files already present)",
		"failed_initialize":    "✗ Failed to initialize: %v",
		"failed_update_gitignore": "⚠ Failed to update .gitignore: %v",
		"initialized_git":      "✓ Initialized .git directory",
		"cloning_from":         "→ Cloning from %s",
		"failed_create_dir":    "✗ Failed to create directory: %v",
		"clone_failed":         "✗ Clone failed: %v",
		"cloned_successfully":  "✓ Cloned successfully",
		"has_unpushed":         "⚠ Has unpushed commits (%s)",
		"push_first":           "Push first: cd %s && git push",
		"updated_commit":       "✓ Updated commit: %s → %s",
		"adding_to_gitignore":  "→ Adding to .gitignore",
		"added_to_gitignore":   "✓ Added",
		"applying_skip_sub":    "→ Applying skip-worktree (%d files)",
		"skip_applied":         "✓ Applied",
		"no_skip_config":       "✓ No skip-worktree config",
		"processing_keep_files": "→ Processing keep files (%d files)",
		"installing_hook":      "→ Installing post-commit hook",
		"hook_installed":       "✓ Hook installed",
		"hook_failed":          "⚠ Failed to install hook: %v",
		"completed_issues":     "⚠ Completed with %d issue(s)",
		"all_success":          "✓ All configurations applied successfully",
		"found_sub":            "Found: %s",
		"failed_get_remote":    "⚠ %s: failed to get remote URL: %v",
		"failed_get_commit":    "⚠ %s: failed to get commit: %v",
		"failed_scan":          "failed to scan directories: %w",
	},
	"ko": {
		// Pull command
		"uncommitted_files":   "작업 디렉토리 %d개 파일 수정됨",
		"clean_directory":     "작업 디렉토리 깨끗함",
		"pull_confirm":        "Pull 하시겠습니까? (Y/n): ",
		"pull_updated":        "✓ 업데이트됨 (%d개 파일 변경됨)",
		"pull_already_uptodate": "✓ 이미 최신 상태",
		"pull_failed":         "✗ 실패",
		"pull_skipped":        "→ 건너뜀",
		"run_status":          "→ 실행: git sub status %s",
		"not_git_repo":        "→ git 저장소가 아님, 건너뜀",
		"failed_get_branch":   "✗ 브랜치 확인 실패: %v",
		"failed_read_input":   "✗ 입력 읽기 실패: %v",
		"no_subs_registered":  "등록된 sub이 없습니다",
		"sub_not_found":       "sub을 찾을 수 없음: %s",

		// Status command
		"local_status":        "로컬 상태:",
		"files_modified":      "✗ %d개 파일 수정됨:",
		"files_untracked":     "⚠ %d개 파일 추적 안 됨:",
		"files_staged":        "● %d개 파일 스테이징됨:",
		"clean_working_tree":  "✓ 작업 트리 깨끗함",
		"remote_status":       "원격 상태:",
		"commits_behind":      "→ origin/%s보다 %d개 커밋 뒤처짐",
		"commits_ahead":       "→ %d개 커밋 앞섬 (푸시 안 됨)",
		"up_to_date":          "✓ origin과 최신 상태",
		"cannot_fetch":        "⚠ 원격에서 가져올 수 없음",
		"skip_files":          "Skip 파일:",
		"skip_file_changed":   "⚠ %s 원격에서 변경됨",
		"skip_remote_added":   "원격에서 새 줄 추가됨",
		"skip_remote_removed": "원격에서 줄 삭제됨",
		"skip_remote_modified": "원격에서 내용 수정됨",
		"skip_file_protected": "(로컬 파일은 skip-worktree로 보호됨)",
		"no_remote_changes":   "✓ skip 파일에 원격 변경사항 없음",
		"how_to_resolve":      "해결 방법:",
		"no_action_needed":    "✓ 조치 필요 없음",
		"not_cloned":          "(복제되지 않음)",
		"resolve_commit":      "1. 변경사항 커밋 또는 stash:",
		"resolve_or_gitignore": "# 또는 추적 안 된 파일을 .gitignore에 추가",
		"resolve_pull":        "2. 업데이트 받기:",
		"resolve_push":        "3. 커밋 푸시:",
		"resolve_skip":        "4. (선택) skip 파일 업데이트:",
		"resolve_review":      "# 변경사항 검토 및 병합",

		// Sync command
		"syncing":              "동기화 중...",
		"installing_hooks":     "→ git 훅 설치 중",
		"hooks_installed":      "✓ 설치됨",
		"hooks_failed":         "✗ 실패: %v",
		"no_gitsubs_found":     "\n→ .workspaces를 찾을 수 없음. 기존 sub 저장소 검색 중...",
		"no_subs_found":        "✓ sub 저장소를 찾지 못했습니다",
		"to_add_sub":           "\nsub을 추가하려면:",
		"cmd_git_sub_clone":    "  git sub clone <url> <path>",
		"created_gitsubs":      "\n✓ %d개 sub로 .workspaces 생성됨",
		"applying_ignore":      "\n→ ignore 패턴 적용 중",
		"applied_patterns":     "✓ %d개 패턴 적용됨",
		"applying_skip_mother": "→ 메인 저장소에 skip-worktree 적용 중",
		"applied_files":        "✓ %d개 파일에 적용됨",
		"processing_mother_keep": "→ 메인 저장소 keep 파일 처리 중",
		"no_subclones":         "\n등록된 subclone이 없습니다.",
		"processing_subclones": "\n→ subclone 처리 중:",
		"initializing_git":     "→ .git 초기화 중 (소스 파일 이미 존재)",
		"failed_initialize":    "✗ 초기화 실패: %v",
		"failed_update_gitignore": "⚠ .gitignore 업데이트 실패: %v",
		"initialized_git":      "✓ .git 디렉토리 초기화됨",
		"cloning_from":         "→ %s에서 복제 중",
		"failed_create_dir":    "✗ 디렉토리 생성 실패: %v",
		"clone_failed":         "✗ 복제 실패: %v",
		"cloned_successfully":  "✓ 복제 성공",
		"has_unpushed":         "⚠ 푸시 안 된 커밋 있음 (%s)",
		"push_first":           "먼저 푸시: cd %s && git push",
		"updated_commit":       "✓ 커밋 업데이트됨: %s → %s",
		"adding_to_gitignore":  "→ .gitignore에 추가 중",
		"added_to_gitignore":   "✓ 추가됨",
		"applying_skip_sub":    "→ skip-worktree 적용 중 (%d개 파일)",
		"skip_applied":         "✓ 적용됨",
		"no_skip_config":       "✓ skip-worktree 설정 없음",
		"processing_keep_files": "→ keep 파일 처리 중 (%d개 파일)",
		"installing_hook":      "→ post-commit 훅 설치 중",
		"hook_installed":       "✓ 훅 설치됨",
		"hook_failed":          "⚠ 훅 설치 실패: %v",
		"completed_issues":     "⚠ %d개 문제와 함께 완료됨",
		"all_success":          "✓ 모든 설정이 성공적으로 적용됨",
		"found_sub":            "발견: %s",
		"failed_get_remote":    "⚠ %s: 원격 URL 가져오기 실패: %v",
		"failed_get_commit":    "⚠ %s: 커밋 가져오기 실패: %v",
		"failed_scan":          "디렉토리 스캔 실패: %w",
	},
}

// SetLanguage sets the current language for messages
func SetLanguage(lang string) {
	if lang == "ko" || lang == "en" {
		currentLang = lang
	}
}

// T translates a message key to the current language
func T(key string, args ...interface{}) string {
	msg, ok := messages[currentLang][key]
	if !ok {
		// Fallback to key if not found
		return key
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

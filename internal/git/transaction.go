package git

import (
	"fmt"
	"log"
)

// WithSkipWorktreeTransaction executes workFunc within a skip-worktree transaction.
// Pattern: UnapplySkipWorktree → work → ApplySkipWorktree (defer)
// This ensures skip-worktree is always re-applied, even on error.
func WithSkipWorktreeTransaction(
	workspacePath string,
	keepFiles []string,
	workFunc func() error,
) error {
	if len(keepFiles) == 0 {
		return workFunc() // No keep files, no transaction needed
	}

	// BEGIN TRANSACTION: Unskip
	if err := UnapplySkipWorktree(workspacePath, keepFiles); err != nil {
		return fmt.Errorf("failed to unapply skip-worktree: %w", err)
	}

	// ENSURE COMMIT: Re-skip even on error (defer)
	defer func() {
		if err := ApplySkipWorktree(workspacePath, keepFiles); err != nil {
			log.Printf("⚠️  Failed to re-apply skip-worktree: %v", err)
		}
	}()

	// EXECUTE WORK
	return workFunc()
}

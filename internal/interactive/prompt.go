package interactive

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// ResolveConflict shows a menu to resolve patch conflicts
// Returns selected index (0-based)
func ResolveConflict(filename string, options []string) (int, error) {
	var selected string
	prompt := &survey.Select{
		Message: fmt.Sprintf("📝 Conflict in %s - Choose action:", filename),
		Options: options,
		Description: func(value string, index int) string {
			descriptions := map[string]string{
				"Skip this patch": "Continue without applying this change",
				"Apply anyway":    "Force apply despite conflicts",
				"Edit manually":   "Open in editor to resolve manually",
				"Show diff":       "View the conflicting changes",
				"Abort all":       "Stop the entire operation",
			}
			if desc, ok := descriptions[value]; ok {
				return desc
			}
			return ""
		},
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return -1, err
	}

	for i, opt := range options {
		if opt == selected {
			return i, nil
		}
	}

	return -1, fmt.Errorf("selected option not found")
}

// ShowDiff displays a diff with pager
func ShowDiff(diff string) error {
	// Try to use git's pager or less/more
	pager := os.Getenv("PAGER")
	if pager == "" {
		// Try common pagers
		for _, p := range []string{"less", "more", "cat"} {
			if _, err := exec.LookPath(p); err == nil {
				pager = p
				break
			}
		}
	}

	if pager == "" {
		// Fallback: just print
		fmt.Println(diff)
		return nil
	}

	cmd := exec.Command(pager)
	if pager == "less" {
		cmd.Args = append(cmd.Args, "-R") // Enable color
	}
	cmd.Stdin = strings.NewReader(diff)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Confirm asks yes/no question
func Confirm(message string) (bool, error) {
	result := false
	prompt := &survey.Confirm{
		Message: message,
		Default: false,
	}

	err := survey.AskOne(prompt, &result)
	return result, err
}

// SelectFiles allows multi-selection of files
func SelectFiles(files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to select from")
	}

	var selected []string
	prompt := &survey.MultiSelect{
		Message: "📂 Select files:",
		Options: files,
		Description: func(value string, index int) string {
			return fmt.Sprintf("File %d of %d", index+1, len(files))
		},
		PageSize: 15,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, err
	}

	return selected, nil
}

// ConfirmYesNo prompts user with Y/n (default: yes)
// Returns true if user presses Enter or types y/yes
func ConfirmYesNo(message string) (bool, error) {
	result := true // Default to yes
	prompt := &survey.Confirm{
		Message: message,
		Default: true,
	}

	err := survey.AskOne(prompt, &result)
	return result, err
}

// ConfirmYN prompts user with y/N (default: no)
// Returns true only if user explicitly types y/yes
func ConfirmYN(message string) (bool, error) {
	result := false // Default to no
	prompt := &survey.Confirm{
		Message: message,
		Default: false,
	}

	err := survey.AskOne(prompt, &result)
	return result, err
}

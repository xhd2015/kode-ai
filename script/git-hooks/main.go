package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/xgo/support/cmd"
	"github.com/xhd2015/xgo/support/fileutil"
	"github.com/xhd2015/xgo/support/git"
)

// usage:
//
//	go run ./script/git-hooks install
//	go run ./script/git-hooks pre-commit
//	go run ./script/git-hooks pre-commit --no-commit --no-update-version
//	go run ./script/git-hooks post-commit
func main() {
	args := os.Args[1:]
	var cmd string
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	var noCommit bool
	var noUpdateVersion bool
	var amend bool
	for _, arg := range args {
		if arg == "--no-commit" {
			noCommit = true
			continue
		}
		if arg == "--amend" {
			amend = true
			continue
		}
		if arg == "--no-update-version" {
			noUpdateVersion = true
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "unexpected arg: %s\n", arg)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "unrecognized flag: %s\n", arg)
		os.Exit(1)
	}
	if cmd == "" {
		fmt.Fprintf(os.Stderr, "requires command\n")
		os.Exit(1)
	}
	var err error
	switch cmd {
	case "install":
		err = install()
	case "pre-commit":
		err = preCommitCheck(noCommit, amend, noUpdateVersion)
	case "post-commit":
		err = postCommitCheck(noCommit)
	default:
		fmt.Fprintf(os.Stderr, "unrecognized command: %s\n", cmd)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

const preCommitCmdHead = "# go-script git-hooks"

// NOTE: no empty lines in between
const preCommitCmd = `# see: https://stackoverflow.com/questions/19387073/how-to-detect-commit-amend-by-pre-commit-hook
is_amend=$(ps -ocommand= -p $PPID | grep -e '--amend')
# echo "is amend: $is_amend"
# args is always empty
# echo "args: ${args[@]}"
flags=()
if [[ -n $is_amend ]];then
    flags=("${flags[@]}" --amend)
fi
go run ./script/git-hooks pre-commit "${flags[@]}"
`

const postCommitCmdHead = "# go-script git-hooks"
const postCommitCmd = "go run ./script/git-hooks post-commit"

func preCommitCheck(noCommit bool, amend bool, noUpdateVersion bool) error {
	gitDir, err := git.ShowTopLevel("")
	if err != nil {
		return err
	}
	rootDir, err := filepath.Abs(gitDir)
	if err != nil {
		return err
	}

	var affectedFiles []string

	// update revision
	revision, err := cmd.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	revision = strings.TrimSpace(revision)

	if revision == "" {
		return fmt.Errorf("cannot get revision")
	}

	revisionFile := filepath.Join(rootDir, "run", "REVISION.txt")
	err = os.WriteFile(revisionFile, []byte(revision+"+1"), 0644)
	if err != nil {
		return err
	}
	affectedFiles = append(affectedFiles, revisionFile)

	if !noCommit {
		err = cmd.Dir(rootDir).Run("git", append([]string{"add"}, affectedFiles...)...)
		if err != nil {
			return nil
		}
	}

	return nil
}

func postCommitCheck(noCommit bool) error {
	// do nothing
	return nil
}

func install() error {
	// NOTE: is git dir, not toplevel dir when in worktree mode
	gitDir, err := git.GetGitDir("")
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	err = os.MkdirAll(hooksDir, 0755)
	if err != nil {
		return err
	}

	err = installHook(filepath.Join(hooksDir, "pre-commit"), preCommitCmdHead, preCommitCmd)
	if err != nil {
		return fmt.Errorf("pre-commit: %w", err)
	}

	err = installHook(filepath.Join(hooksDir, "post-commit"), postCommitCmdHead, postCommitCmd)
	if err != nil {
		return fmt.Errorf("post-commit: %w", err)
	}
	return nil
}

func installHook(hookFile string, head string, cmd string) error {
	var needChmod bool
	err := fileutil.Patch(hookFile, func(data []byte) ([]byte, error) {
		if len(data) == 0 {
			needChmod = true
		}
		content := string(data)
		lines := strings.Split(content, "\n")
		idx := -1
		n := len(lines)
		for i := 0; i < n; i++ {
			if strings.Contains(lines[i], head) {
				idx = i
				break
			}
		}
		if idx < 0 {
			// insert
			lines = append(lines, head, cmd, "")
		} else {
			// replace
			endIdx := idx + 1
			for ; endIdx < n; endIdx++ {
				if strings.TrimSpace(lines[endIdx]) == "" {
					break
				}
			}
			oldLines := lines
			lines = lines[:idx]
			lines = append(lines, head, cmd, "")
			if endIdx < n {
				lines = append(lines, oldLines[endIdx:]...)
			}
		}

		return []byte(strings.Join(lines, "\n")), nil
	})

	if err != nil {
		return err
	}

	// chmod to what? it is 0755 already
	if needChmod {
		err := os.Chmod(hookFile, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

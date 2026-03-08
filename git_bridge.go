package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
)

type GitResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	ExitCode int    `json:"exitCode"`
}

type GitChange struct {
	Path           string `json:"path"`
	OriginalPath   string `json:"originalPath"`
	IndexStatus    string `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
}

type ChangeListResult struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Changes []GitChange `json:"changes"`
}

type SmartSyncResult struct {
	GitResult
	Action string `json:"action"`
}

func (a *App) Clone(repoURL string, destination string) GitResult {
	args := []string{"clone", repoURL}
	if strings.TrimSpace(destination) != "" {
		args = append(args, destination)
	}

	return a.runGit("", args...)
}

func (a *App) Pull(repoPath string) GitResult {
	return a.runGit(repoPath, "pull")
}

func (a *App) Push(repoPath string) GitResult {
	return a.runGit(repoPath, "push")
}

func (a *App) Fetch(repoPath string) GitResult {
	return a.runGit(repoPath, "fetch")
}

func (a *App) Add(repoPath string, paths []string) GitResult {
	args := []string{"add"}
	if len(paths) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, paths...)
	}

	return a.runGit(repoPath, args...)
}

func (a *App) Status(repoPath string) GitResult {
	return a.runGit(repoPath, "status", "--short", "--branch")
}

func (a *App) Branch(repoPath string, name string) GitResult {
	if strings.TrimSpace(name) == "" {
		return a.runGit(repoPath, "branch")
	}

	return a.runGit(repoPath, "branch", name)
}

func (a *App) Checkout(repoPath string, target string, create bool) GitResult {
	if strings.TrimSpace(target) == "" {
		return GitResult{Success: false, Error: "checkout target is required", ExitCode: 1}
	}

	if create {
		return a.runGit(repoPath, "checkout", "-b", target)
	}

	return a.runGit(repoPath, "checkout", target)
}

func (a *App) Commit(repoPath string, message string) GitResult {
	if strings.TrimSpace(message) == "" {
		return GitResult{Success: false, Error: "commit message is required", ExitCode: 1}
	}

	return a.runGit(repoPath, "commit", "-m", message)
}

func (a *App) SmartSync(repoPath string) SmartSyncResult {
	statusResult := a.runGit(repoPath, "status", "--porcelain", "--branch")
	if !statusResult.Success {
		return SmartSyncResult{GitResult: statusResult, Action: "status"}
	}

	ahead, behind := parseAheadBehind(statusResult.Output)
	if behind > 0 {
		return SmartSyncResult{GitResult: a.runGit(repoPath, "pull"), Action: "pull"}
	}
	if ahead > 0 {
		return SmartSyncResult{GitResult: a.runGit(repoPath, "push"), Action: "push"}
	}

	return SmartSyncResult{GitResult: a.runGit(repoPath, "fetch"), Action: "fetch"}
}

func (a *App) ListChanges(repoPath string) ChangeListResult {
	result := a.runGit(repoPath, "status", "--porcelain")
	if !result.Success {
		return ChangeListResult{Success: false, Error: result.Error}
	}

	return ChangeListResult{Success: true, Changes: parsePorcelainChanges(result.Output)}
}

func (a *App) CommitSelected(repoPath string, paths []string, message string, description string) GitResult {
	if strings.TrimSpace(message) == "" {
		return GitResult{Success: false, Error: "commit message is required", ExitCode: 1}
	}

	if len(paths) > 0 {
		addArgs := append([]string{"add", "--"}, paths...)
		addResult := a.runGit(repoPath, addArgs...)
		if !addResult.Success {
			return addResult
		}
	}

	commitArgs := []string{"commit", "-m", message}
	if strings.TrimSpace(description) != "" {
		commitArgs = append(commitArgs, "-m", description)
	}
	if len(paths) > 0 {
		commitArgs = append(commitArgs, "--")
		commitArgs = append(commitArgs, paths...)
	}

	return a.runGit(repoPath, commitArgs...)
}

func (a *App) runGit(repoPath string, args ...string) GitResult {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return GitResult{Success: false, Error: "git executable not found in PATH", ExitCode: 1}
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, gitPath, args...)
	if strings.TrimSpace(repoPath) != "" {
		cmd.Dir = repoPath
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errText := strings.TrimSpace(stderr.String())

	if err == nil {
		return GitResult{Success: true, Output: output, Error: errText, ExitCode: 0}
	}

	exitCode := 1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	if errText == "" {
		errText = err.Error()
	}
	if output == "" {
		output = readTrimmedOutput(exitErr)
	}

	return GitResult{Success: false, Output: output, Error: errText, ExitCode: exitCode}
}

func readTrimmedOutput(exitErr *exec.ExitError) string {
	if exitErr == nil || len(exitErr.Stderr) == 0 {
		return ""
	}

	return strings.TrimSpace(string(exitErr.Stderr))
}

func parseAheadBehind(statusOutput string) (int, int) {
	lines := strings.Split(statusOutput, "\n")
	if len(lines) == 0 {
		return 0, 0
	}

	branchLine := strings.TrimSpace(lines[0])
	return parseCounter(branchLine, "ahead "), parseCounter(branchLine, "behind ")
}

func parseCounter(text string, label string) int {
	index := strings.Index(text, label)
	if index < 0 {
		return 0
	}

	start := index + len(label)
	end := start
	for end < len(text) {
		char := text[end]
		if char < '0' || char > '9' {
			break
		}
		end++
	}
	if end == start {
		return 0
	}

	count := 0
	for i := start; i < end; i++ {
		count = count*10 + int(text[i]-'0')
	}
	return count
}

func parsePorcelainChanges(output string) []GitChange {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return []GitChange{}
	}

	lines := strings.Split(trimmed, "\n")
	changes := make([]GitChange, 0, len(lines))
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		change := GitChange{
			IndexStatus:    string(line[0]),
			WorktreeStatus: string(line[1]),
			Path:           strings.TrimSpace(line[3:]),
		}
		if change.Path == "" {
			continue
		}
		if strings.Contains(change.Path, " -> ") {
			parts := strings.SplitN(change.Path, " -> ", 2)
			if len(parts) == 2 {
				change.OriginalPath = parts[0]
				change.Path = parts[1]
			}
		}
		changes = append(changes, change)
	}

	return changes
}

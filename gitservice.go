package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type GitService struct{}

type GitResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	ExitCode int    `json:"exitCode"`
}

type RepositoryOperationResult struct {
	Success      bool     `json:"success"`
	Error        string   `json:"error"`
	Repository   string   `json:"repository"`
	Repositories []string `json:"repositories"`
	Current      string   `json:"current"`
	Cancelled    bool     `json:"cancelled"`
}

type HarborData struct {
	Repositories      []string `json:"repositories"`
	CurrentRepository string   `json:"currentRepository"`
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

func (s *GitService) Clone(repoURL string, destination string) GitResult {
	args := []string{"clone", repoURL}
	if strings.TrimSpace(destination) != "" {
		args = append(args, destination)
	}
	return s.runGit("", args...)
}

func (s *GitService) Pull(repoPath string) GitResult {
	return s.runGit(repoPath, "pull")
}

func (s *GitService) Push(repoPath string) GitResult {
	return s.runGit(repoPath, "push")
}

func (s *GitService) Fetch(repoPath string) GitResult {
	return s.runGit(repoPath, "fetch")
}

func (s *GitService) Add(repoPath string, paths []string) GitResult {
	args := []string{"add"}
	if len(paths) == 0 {
		args = append(args, ".")
	} else {
		args = append(args, paths...)
	}
	return s.runGit(repoPath, args...)
}

func (s *GitService) Status(repoPath string) GitResult {
	return s.runGit(repoPath, "status", "--short", "--branch")
}

func (s *GitService) Branch(repoPath string, name string) GitResult {
	if strings.TrimSpace(name) == "" {
		return s.runGit(repoPath, "branch")
	}
	return s.runGit(repoPath, "branch", name)
}

func (s *GitService) Checkout(repoPath string, target string, create bool) GitResult {
	if strings.TrimSpace(target) == "" {
		return GitResult{Success: false, Error: "checkout target is required", ExitCode: 1}
	}
	if create {
		return s.runGit(repoPath, "checkout", "-b", target)
	}
	return s.runGit(repoPath, "checkout", target)
}

func (s *GitService) Commit(repoPath string, message string) GitResult {
	if strings.TrimSpace(message) == "" {
		return GitResult{Success: false, Error: "commit message is required", ExitCode: 1}
	}
	return s.runGit(repoPath, "commit", "-m", message)
}

func (s *GitService) SmartSync(repoPath string) SmartSyncResult {
	statusResult := s.runGit(repoPath, "status", "--porcelain", "--branch")
	if !statusResult.Success {
		return SmartSyncResult{GitResult: statusResult, Action: "status"}
	}
	ahead, behind := parseAheadBehind(statusResult.Output)
	if behind > 0 {
		return SmartSyncResult{GitResult: s.runGit(repoPath, "pull"), Action: "pull"}
	}
	if ahead > 0 {
		return SmartSyncResult{GitResult: s.runGit(repoPath, "push"), Action: "push"}
	}
	return SmartSyncResult{GitResult: s.runGit(repoPath, "fetch"), Action: "fetch"}
}

func (s *GitService) ListChanges(repoPath string) ChangeListResult {
	unstagedResult := s.runGit(repoPath, "diff", "--name-status")
	if !unstagedResult.Success {
		return ChangeListResult{Success: false, Error: unstagedResult.Error}
	}

	stagedResult := s.runGit(repoPath, "diff", "--cached", "--name-status")
	if !stagedResult.Success {
		return ChangeListResult{Success: false, Error: stagedResult.Error}
	}

	untrackedResult := s.runGit(repoPath, "ls-files", "--others", "--exclude-standard")
	if !untrackedResult.Success {
		return ChangeListResult{Success: false, Error: untrackedResult.Error}
	}

	changes := mergeDiffChanges(unstagedResult.Output, stagedResult.Output, untrackedResult.Output)
	return ChangeListResult{Success: true, Changes: changes}
}

func (s *GitService) CommitSelected(repoPath string, paths []string, message string, description string) GitResult {
	if strings.TrimSpace(message) == "" {
		return GitResult{Success: false, Error: "commit message is required", ExitCode: 1}
	}

	if len(paths) > 0 {
		addArgs := append([]string{"add", "--"}, paths...)
		addResult := s.runGit(repoPath, addArgs...)
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
	return s.runGit(repoPath, commitArgs...)
}

func (s *GitService) AddLocalRepository(repoPath string) RepositoryOperationResult {
	normalizedPath, err := normalizePath(repoPath)
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}
	if err := validateGitRepository(normalizedPath); err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}
	if !containsPath(harborData.Repositories, normalizedPath) {
		harborData.Repositories = append(harborData.Repositories, normalizedPath)
		sort.Strings(harborData.Repositories)
	}
	harborData.CurrentRepository = normalizedPath
	if err := saveHarborData(harborData); err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	s.emitRepositoriesUpdated(normalizedPath)
	return RepositoryOperationResult{Success: true, Repository: normalizedPath, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (s *GitService) SelectAndAddLocalRepository() RepositoryOperationResult {
	app := application.Get()
	if app == nil {
		return RepositoryOperationResult{Success: false, Error: "application not available"}
	}

	selectedPath, err := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle("Select Local Git Repository").
		PromptForSingleSelection()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	if strings.TrimSpace(selectedPath) == "" {
		harborData, listErr := loadHarborData()
		if listErr != nil {
			return RepositoryOperationResult{Success: false, Error: listErr.Error()}
		}
		return RepositoryOperationResult{Success: true, Cancelled: true, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
	}

	return s.AddLocalRepository(selectedPath)
}

func (s *GitService) ListRepositories() RepositoryOperationResult {
	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}
	return RepositoryOperationResult{Success: true, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (s *GitService) SetCurrentRepository(repoPath string) RepositoryOperationResult {
	normalizedPath, err := normalizePath(repoPath)
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}
	if !containsPath(harborData.Repositories, normalizedPath) {
		return RepositoryOperationResult{Success: false, Error: "repository is not in the saved list"}
	}

	harborData.CurrentRepository = normalizedPath
	if err := saveHarborData(harborData); err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	s.emitRepositoriesUpdated(normalizedPath)
	return RepositoryOperationResult{Success: true, Repository: normalizedPath, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (s *GitService) GetCurrentRepository() RepositoryOperationResult {
	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}
	return RepositoryOperationResult{Success: true, Repository: harborData.CurrentRepository, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (s *GitService) emitRepositoriesUpdated(path string) {
	app := application.Get()
	if app != nil {
		app.Event.Emit("harbor:repositories-updated", path)
	}
}

func (s *GitService) runGit(repoPath string, args ...string) GitResult {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return GitResult{Success: false, Error: "git executable not found in PATH", ExitCode: 1}
	}

	ctx := context.Background()
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

func normalizePath(path string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", errors.New("repository path is required")
	}

	absPath, err := filepath.Abs(trimmedPath)
	if err != nil {
		return "", fmt.Errorf("invalid repository path: %w", err)
	}
	return filepath.Clean(absPath), nil
}

func validateGitRepository(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %w", err)
	}
	if !info.IsDir() {
		return errors.New("repository path must be a directory")
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return errors.New("git executable not found in PATH")
	}

	cmd := exec.Command(gitPath, "-C", path, "rev-parse", "--is-inside-work-tree")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		return fmt.Errorf("not a git repository: %s", message)
	}
	if strings.TrimSpace(stdout.String()) != "true" {
		return errors.New("not a git repository")
	}

	return nil
}

func harborFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".harbor"), nil
}

func loadHarborData() (HarborData, error) {
	path, err := harborFilePath()
	if err != nil {
		return HarborData{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return HarborData{Repositories: []string{}}, nil
		}
		return HarborData{}, fmt.Errorf("failed reading %s: %w", path, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return HarborData{Repositories: []string{}}, nil
	}

	var harborData HarborData
	if unmarshalErr := json.Unmarshal(data, &harborData); unmarshalErr == nil {
		harborData.Repositories = normalizeRepositoryList(harborData.Repositories)
		if harborData.CurrentRepository != "" && !containsPath(harborData.Repositories, harborData.CurrentRepository) {
			harborData.CurrentRepository = ""
		}
		return harborData, nil
	}

	var repositoryList []string
	if unmarshalErr := json.Unmarshal(data, &repositoryList); unmarshalErr == nil {
		harborData = HarborData{Repositories: normalizeRepositoryList(repositoryList)}
		if saveErr := saveHarborData(harborData); saveErr != nil {
			return HarborData{}, saveErr
		}
		return harborData, nil
	}

	return HarborData{}, fmt.Errorf("invalid .harbor format in %s", path)
}

func saveHarborData(harborData HarborData) error {
	path, err := harborFilePath()
	if err != nil {
		return err
	}

	harborData.Repositories = normalizeRepositoryList(harborData.Repositories)
	if harborData.CurrentRepository != "" {
		normalizedCurrent, normalizeErr := normalizePath(harborData.CurrentRepository)
		if normalizeErr != nil || !containsPath(harborData.Repositories, normalizedCurrent) {
			harborData.CurrentRepository = ""
		} else {
			harborData.CurrentRepository = normalizedCurrent
		}
	}

	payload := HarborData{Repositories: harborData.Repositories, CurrentRepository: harborData.CurrentRepository}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed encoding repositories: %w", err)
	}
	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		return fmt.Errorf("failed writing %s: %w", path, writeErr)
	}

	return nil
}

func normalizeRepositoryList(repositories []string) []string {
	normalized := make([]string, 0, len(repositories))
	for _, path := range repositories {
		cleanPath, err := normalizePath(path)
		if err != nil || containsPath(normalized, cleanPath) {
			continue
		}
		normalized = append(normalized, cleanPath)
	}

	sort.Strings(normalized)
	return normalized
}

func containsPath(paths []string, candidate string) bool {
	for _, path := range paths {
		if samePath(path, candidate) {
			return true
		}
	}
	return false
}

func samePath(pathA string, pathB string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(pathA), filepath.Clean(pathB))
	}
	return filepath.Clean(pathA) == filepath.Clean(pathB)
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

func mergeDiffChanges(unstagedOutput string, stagedOutput string, untrackedOutput string) []GitChange {
	changesByPath := map[string]GitChange{}

	applyDiffNameStatus(changesByPath, unstagedOutput, false)
	applyDiffNameStatus(changesByPath, stagedOutput, true)
	applyUntracked(changesByPath, untrackedOutput)

	changes := make([]GitChange, 0, len(changesByPath))
	for _, change := range changesByPath {
		changes = append(changes, change)
	}

	sort.Slice(changes, func(i int, j int) bool {
		return strings.ToLower(changes[i].Path) < strings.ToLower(changes[j].Path)
	})

	return changes
}

func applyDiffNameStatus(changesByPath map[string]GitChange, output string, staged bool) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		statusCode := strings.TrimSpace(parts[0])
		status := " "
		if statusCode != "" {
			status = string(statusCode[0])
		}

		path := strings.TrimSpace(parts[1])
		originalPath := ""
		if strings.HasPrefix(statusCode, "R") || strings.HasPrefix(statusCode, "C") {
			if len(parts) >= 3 {
				originalPath = path
				path = strings.TrimSpace(parts[2])
			}
		}

		if path == "" {
			continue
		}

		change, exists := changesByPath[path]
		if !exists {
			change = GitChange{Path: path}
		}
		if originalPath != "" {
			change.OriginalPath = originalPath
		}

		if staged {
			change.IndexStatus = status
		} else {
			change.WorktreeStatus = status
		}

		changesByPath[path] = change
	}
}

func applyUntracked(changesByPath map[string]GitChange, output string) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return
	}

	lines := strings.Split(trimmed, "\n")
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}

		change, exists := changesByPath[path]
		if !exists {
			change = GitChange{Path: path}
		}
		change.WorktreeStatus = "?"
		changesByPath[path] = change
	}
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

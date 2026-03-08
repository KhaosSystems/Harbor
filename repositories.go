package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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

func (a *App) AddLocalRepository(repoPath string) RepositoryOperationResult {
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
	if harborData.CurrentRepository == "" {
		harborData.CurrentRepository = normalizedPath
	}

	if err := saveHarborData(harborData); err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	return RepositoryOperationResult{
		Success:      true,
		Repository:   normalizedPath,
		Repositories: harborData.Repositories,
		Current:      harborData.CurrentRepository,
	}
}

func (a *App) SelectAndAddLocalRepository() RepositoryOperationResult {
	if a.ctx == nil {
		return RepositoryOperationResult{Success: false, Error: "application context not ready"}
	}

	selectedPath, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: "Select Local Git Repository"})
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

	return a.AddLocalRepository(selectedPath)
}

func (a *App) ListRepositories() RepositoryOperationResult {
	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	return RepositoryOperationResult{Success: true, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (a *App) SetCurrentRepository(repoPath string) RepositoryOperationResult {
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

	return RepositoryOperationResult{Success: true, Repository: normalizedPath, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
}

func (a *App) GetCurrentRepository() RepositoryOperationResult {
	harborData, err := loadHarborData()
	if err != nil {
		return RepositoryOperationResult{Success: false, Error: err.Error()}
	}

	return RepositoryOperationResult{Success: true, Repository: harborData.CurrentRepository, Repositories: harborData.Repositories, Current: harborData.CurrentRepository}
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

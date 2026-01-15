package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stateDirName  = ".planq"
	stateFileName = "state.json"
)

// GlobalState tracks planq state across repositories.
type GlobalState struct {
	MainWorkspaces map[string]MainWorkspaceEntry `json:"main_workspaces"`
}

// MainWorkspaceEntry tracks a main workspace for a repository.
type MainWorkspaceEntry struct {
	Name     string `json:"name"`
	RepoPath string `json:"repo_path"`
}

// StateDir returns the path to the global planq state directory.
func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, stateDirName), nil
}

// StateFile returns the path to the global state file.
func StateFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

// Load reads the global state from disk.
func Load() (*GlobalState, error) {
	stateFile, err := StateFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalState{MainWorkspaces: make(map[string]MainWorkspaceEntry)}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state GlobalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	if state.MainWorkspaces == nil {
		state.MainWorkspaces = make(map[string]MainWorkspaceEntry)
	}
	return &state, nil
}

// Save writes the global state to disk.
func (s *GlobalState) Save() error {
	stateDir, err := StateDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	stateFile, err := StateFile()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

// HasMainWorkspace checks if a main workspace exists for the given repo.
func (s *GlobalState) HasMainWorkspace(repoPath string) bool {
	_, exists := s.MainWorkspaces[repoPath]
	return exists
}

// GetMainWorkspace returns the main workspace for a repo if it exists.
func (s *GlobalState) GetMainWorkspace(repoPath string) (MainWorkspaceEntry, bool) {
	entry, exists := s.MainWorkspaces[repoPath]
	return entry, exists
}

// SetMainWorkspace records a main workspace for a repo.
func (s *GlobalState) SetMainWorkspace(repoPath, name string) {
	s.MainWorkspaces[repoPath] = MainWorkspaceEntry{
		Name:     name,
		RepoPath: repoPath,
	}
}

// RemoveMainWorkspace removes the main workspace entry for a repo.
func (s *GlobalState) RemoveMainWorkspace(repoPath string) {
	delete(s.MainWorkspaces, repoPath)
}

// FindMainWorkspaceByName finds a main workspace entry by its name.
func (s *GlobalState) FindMainWorkspaceByName(name string) (string, bool) {
	for repoPath, entry := range s.MainWorkspaces {
		if entry.Name == name {
			return repoPath, true
		}
	}
	return "", false
}

// GetMainWorkspaceNames returns a set of all main workspace names.
func (s *GlobalState) GetMainWorkspaceNames() map[string]bool {
	names := make(map[string]bool)
	for _, entry := range s.MainWorkspaces {
		names[entry.Name] = true
	}
	return names
}

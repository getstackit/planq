package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReviewState tracks whether a workspace needs review.
type ReviewState struct {
	NeedsReview bool       `json:"needs_review"`
	FlaggedAt   *time.Time `json:"flagged_at,omitempty"`
}

// ReviewFile returns the path to the review state file.
func (w *Workspace) ReviewFile() string {
	return filepath.Join(w.PlanqDir(), "review.json")
}

// GetReviewState returns the current review state.
func (w *Workspace) GetReviewState() (*ReviewState, error) {
	data, err := os.ReadFile(w.ReviewFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &ReviewState{NeedsReview: false}, nil
		}
		return nil, fmt.Errorf("failed to read review file: %w", err)
	}

	var state ReviewState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse review file: %w", err)
	}

	return &state, nil
}

// SetNeedsReview marks the workspace as needing review.
func (w *Workspace) SetNeedsReview() error {
	// Ensure .planq directory exists
	if err := os.MkdirAll(w.PlanqDir(), 0755); err != nil {
		return fmt.Errorf("failed to create planq directory: %w", err)
	}

	now := time.Now()
	state := ReviewState{
		NeedsReview: true,
		FlaggedAt:   &now,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal review state: %w", err)
	}

	if err := os.WriteFile(w.ReviewFile(), data, 0644); err != nil {
		return fmt.Errorf("failed to write review file: %w", err)
	}

	return nil
}

// ClearReview clears the needs review flag.
func (w *Workspace) ClearReview() error {
	state := ReviewState{
		NeedsReview: false,
		FlaggedAt:   nil,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal review state: %w", err)
	}

	if err := os.WriteFile(w.ReviewFile(), data, 0644); err != nil {
		return fmt.Errorf("failed to write review file: %w", err)
	}

	return nil
}

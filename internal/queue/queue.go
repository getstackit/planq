// Package queue provides functionality for queuing work items for later.
package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Dir returns the path to the queue directory for a project.
func Dir(projectRoot string) string {
	return filepath.Join(projectRoot, ".planq", "queue")
}

// Add saves a text item to the queue and returns the created file path.
func Add(projectRoot, text string) (string, error) {
	queueDir := Dir(projectRoot)

	// Create the queue directory if it doesn't exist
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create queue directory: %w", err)
	}

	// Generate timestamp-based filename
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := timestamp + ".md"
	filePath := filepath.Join(queueDir, filename)

	// Handle potential collision by adding suffix
	for i := 1; fileExists(filePath); i++ {
		filename = fmt.Sprintf("%s-%d.md", timestamp, i)
		filePath = filepath.Join(queueDir, filename)
	}

	// Write the content
	content := strings.TrimSpace(text) + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write queue item: %w", err)
	}

	return filePath, nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Item represents a queued work item.
type Item struct {
	Filename string
	Content  string
}

// List returns all queued items, sorted by filename (oldest first).
func List(projectRoot string) ([]Item, error) {
	queueDir := Dir(projectRoot)

	entries, err := os.ReadDir(queueDir)
	if os.IsNotExist(err) {
		return nil, nil // No queue directory = no items
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read queue directory: %w", err)
	}

	var items []Item
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(queueDir, entry.Name()))
		if err != nil {
			continue // Skip unreadable files
		}

		items = append(items, Item{
			Filename: entry.Name(),
			Content:  strings.TrimSpace(string(content)),
		})
	}

	return items, nil
}

package cli

import (
	"os"
	"strings"
	"testing"
)

func TestRunInitExisting(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "chronicle-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Change to temp directory
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Create an existing chronicle.yaml
	if err := os.WriteFile("chronicle.yaml", []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test that init fails when file exists
	err = runInit(nil, nil)
	if err == nil {
		t.Error("Expected error when chronicle.yaml already exists")
	}
}

func TestRunInitSuccess(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "chronicle-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Change to temp directory
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Test that init succeeds
	err = runInit(nil, nil)
	if err != nil {
		t.Errorf("runInit() unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat("chronicle.yaml"); os.IsNotExist(err) {
		t.Error("chronicle.yaml was not created")
	}

	// Verify content
	content, err := os.ReadFile("chronicle.yaml")
	if err != nil {
		t.Fatalf("Failed to read chronicle.yaml: %v", err)
	}

	if len(content) == 0 {
		t.Error("chronicle.yaml is empty")
	}

	// Verify key fields are present
	contentStr := string(content)
	requiredFields := []string{"name:", "version:", "discovery:", "paths:", "scenarios:"}
	for _, field := range requiredFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("chronicle.yaml missing required field: %s", field)
		}
	}
}

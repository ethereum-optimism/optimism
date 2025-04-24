package welcome

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWelcomeMessage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "welcome-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: No welcome file, should return a default message
	msg, err := GetWelcomeMessage(tempDir)
	if err != nil {
		t.Errorf("GetWelcomeMessage returned error: %v", err)
	}
	if msg == "" {
		t.Error("Expected a non-empty default message")
	}

	// Test case 2: Create a custom welcome file
	testMessage := "This is a test welcome message"
	welcomeFile := filepath.Join(tempDir, DefaultWelcomeFile)
	err = os.WriteFile(welcomeFile, []byte(testMessage), 0644)
	if err != nil {
		t.Fatalf("Failed to write test welcome file: %v", err)
	}

	msg, err = GetWelcomeMessage(tempDir)
	if err != nil {
		t.Errorf("GetWelcomeMessage returned error: %v", err)
	}
	if !strings.Contains(msg, testMessage) {
		t.Errorf("Message does not contain the expected content. Got: %s", msg)
	}

	// Test case 3: Disable welcome with environment variable
	os.Setenv(DisableEnvVar, "1")
	defer os.Unsetenv(DisableEnvVar)

	msg, err = GetWelcomeMessage(tempDir)
	if err != nil {
		t.Errorf("GetWelcomeMessage returned error: %v", err)
	}
	if msg != "" {
		t.Errorf("Expected empty message when disabled by env var, got: %s", msg)
	}

	// Test case 4: Disable welcome with flag file
	os.Unsetenv(DisableEnvVar)
	disableFile := filepath.Join(".", DisableFlagFile)
	_, err = os.Create(disableFile)
	if err != nil {
		t.Fatalf("Failed to create disable flag file: %v", err)
	}
	defer os.Remove(disableFile)

	msg, err = GetWelcomeMessage(tempDir)
	if err != nil {
		t.Errorf("GetWelcomeMessage returned error: %v", err)
	}
	if msg != "" {
		t.Errorf("Expected empty message when disabled by flag file, got: %s", msg)
	}
}

func TestLoadMessages(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "welcome-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test case 1: No welcome file, should use default messages
	messages, err := loadMessages(tempDir)
	if err != nil {
		t.Errorf("loadMessages returned error: %v", err)
	}
	if len(messages) != len(defaultMessages) {
		t.Errorf("Expected %d default messages, got %d", len(defaultMessages), len(messages))
	}

	// Test case 2: Empty welcome file, should use default messages
	welcomeFile := filepath.Join(tempDir, DefaultWelcomeFile)
	err = os.WriteFile(welcomeFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write test welcome file: %v", err)
	}

	messages, err = loadMessages(tempDir)
	if err != nil {
		t.Errorf("loadMessages returned error: %v", err)
	}
	if len(messages) != len(defaultMessages) {
		t.Errorf("Expected %d default messages for empty file, got %d", len(defaultMessages), len(messages))
	}

	// Test case 3: Custom welcome file with multiple messages
	customMessages := []string{
		"Message 1",
		"Message 2",
		"Message 3",
	}
	err = os.WriteFile(welcomeFile, []byte(strings.Join(customMessages, "\n")), 0644)
	if err != nil {
		t.Fatalf("Failed to write test welcome file: %v", err)
	}

	messages, err = loadMessages(tempDir)
	if err != nil {
		t.Errorf("loadMessages returned error: %v", err)
	}
	if len(messages) != len(customMessages) {
		t.Errorf("Expected %d custom messages, got %d", len(customMessages), len(messages))
	}
	for i, msg := range customMessages {
		if messages[i] != msg {
			t.Errorf("Message %d doesn't match expected value. Expected: %s, Got: %s", i, msg, messages[i])
		}
	}
}

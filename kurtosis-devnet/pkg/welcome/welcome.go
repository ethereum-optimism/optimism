package welcome

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// DisableEnvVar is the environment variable to disable welcome messages
	DisableEnvVar = "KURTOSIS_DEVNET_DISABLE_WELCOME"

	// DisableFlagFile is a file that can be created to disable welcome messages
	DisableFlagFile = ".disable-welcome"

	// DefaultWelcomeFile is the default file containing welcome messages
	DefaultWelcomeFile = "welcome_messages.txt"
)

var defaultMessages = []string{
	"Welcome to Kurtosis Devnet! Run 'just devnet-test DEVNET_NAME test-name.sh' to run tests against your devnet.",
	"New feature: You can customize network configs using template files. Check the README for more details.",
	"Tip: Use 'kurtosis enclave inspect DEVNET_NAME' to see details about your running devnet.",
	"Did you know? You can run multiple devnets simultaneously with different configurations.",
	"Need help? Check out the README.md file in the kurtosis-devnet directory for troubleshooting.",
}

// GetWelcomeMessage returns a random welcome message
func GetWelcomeMessage(basePath string) (string, error) {
	// Check if welcome messages are disabled
	if isDisabled() {
		return "", nil
	}

	messages, err := loadMessages(basePath)
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "", nil
	}

	// Seed random with current time
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Pick a random message
	message := messages[r.Intn(len(messages))]

	return formatMessage(message), nil
}

// isDisabled checks if welcome messages are disabled
func isDisabled() bool {
	// Check environment variable
	if os.Getenv(DisableEnvVar) != "" {
		return true
	}

	// Check for disable flag file in current directory
	if _, err := os.Stat(DisableFlagFile); err == nil {
		return true
	}

	return false
}

// loadMessages loads welcome messages from file or uses defaults
func loadMessages(basePath string) ([]string, error) {
	welcomeFilePath := filepath.Join(basePath, DefaultWelcomeFile)

	data, err := os.ReadFile(welcomeFilePath)
	if err != nil {
		// If file doesn't exist, use default messages
		if os.IsNotExist(err) {
			return defaultMessages, nil
		}
		return nil, fmt.Errorf("error reading welcome messages file: %w", err)
	}

	messages := strings.Split(string(data), "\n")

	// Filter out empty lines
	var filteredMessages []string
	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg != "" {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	if len(filteredMessages) == 0 {
		return defaultMessages, nil
	}

	return filteredMessages, nil
}

// formatMessage formats a welcome message with fancy borders
func formatMessage(message string) string {
	// Create a box around the message
	width := len(message) + 4

	var sb strings.Builder
	sb.WriteString("\n\033[1;36m") // Cyan, bold

	// Top border
	sb.WriteString("+" + strings.Repeat("-", width-2) + "+\n")

	// Message with side borders
	sb.WriteString("| " + message + " |\n")

	// Bottom border
	sb.WriteString("+" + strings.Repeat("-", width-2) + "+")

	sb.WriteString("\033[0m\n") // Reset color

	return sb.String()
}

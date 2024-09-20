//go:build !cannon32 && !cannon64
// +build !cannon32,!cannon64

package exec

import (
	_ "embed"
	"fmt"
	"os"
	"syscall"
)

//go:embed embeds/cannon32
var cannon32 []byte

//go:embed embeds/cannon64
var cannon64 []byte

func ExecuteCannon(args []string, isCannon32 bool) error {
	cannonProgramName, cannonProgramBin := "cannon32", cannon32
	if !isCannon32 {
		cannonProgramName, cannonProgramBin = "cannon64", cannon64
	}

	cannonProgramPath, err := extractTempFile(cannonProgramName, cannonProgramBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting %s: %v\n", cannonProgramName, err)
		os.Exit(1)
	}
	defer os.Remove(cannonProgramPath)

	if err := os.Chmod(cannonProgramPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting execute permission for %s: %v\n", cannonProgramName, err)
		os.Exit(1)
	}

	execArgs := append([]string{cannonProgramName}, args...)

	if err := syscall.Exec(cannonProgramPath, execArgs, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %s: %v\n", cannonProgramName, err)
		os.Exit(1)
	}

	panic("unreachable")
}

func extractTempFile(name string, data []byte) (string, error) {
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, name+"-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

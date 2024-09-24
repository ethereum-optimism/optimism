package main

import (
	"embed"
	"errors"
	"path/filepath"
	"fmt"
	"os"
	"syscall"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

//go:embed embeds
var vmFS embed.FS

const baseDir = "embeds"

func ExecuteCannon(args []string, ver versions.StateVersion) error {
	switch ver {
	case versions.VersionSingleThreaded, versions.VersionMultiThreaded:
	default:
		return errors.New("unsupported verrsion")
	}

	cannonProgramName := vmFilename(ver)
	cannonProgramBin, err := vmFS.ReadFile(cannonProgramName)
	if err != nil {
		return err
	}
	cannonProgramPath, err := extractTempFile(filepath.Base(cannonProgramName), cannonProgramBin)
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

func vmFilename(ver versions.StateVersion) string {
	return fmt.Sprintf("%s/cannon-%d", baseDir, ver)
}

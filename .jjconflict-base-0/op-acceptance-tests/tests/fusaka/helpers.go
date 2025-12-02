package fusaka

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// ConfigureDevstackEnvVars sets the appropriate env vars to use a mise-installed geth binary for
// the L1 EL. This is useful in Osaka acceptance tests since op-geth does not include full Osaka
// support. This is meant to run before presets.DoMain in a TestMain function. It will log to
// stdout. ResetDevstackEnvVars should be used to reset the environment variables when TestMain
// exits.
//
// Note that this is a no-op if either [sysgo.DevstackL1ELKindVar] or [sysgo.GethExecPathEnvVar]
// are set.
//
// The returned callback resets any modified environment variables.
func ConfigureDevstackEnvVars() func() {
	if _, ok := os.LookupEnv(sysgo.DevstackL1ELKindEnvVar); ok {
		return func() {}
	}
	if _, ok := os.LookupEnv(sysgo.GethExecPathEnvVar); ok {
		return func() {}
	}

	cmd := exec.Command("mise", "which", "geth")
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to find mise-installed geth: %v\n", err)
		return func() {}
	}
	execPath := strings.TrimSpace(buf.String())
	fmt.Println("Found mise-installed geth:", execPath)
	_ = os.Setenv(sysgo.GethExecPathEnvVar, execPath)
	_ = os.Setenv(sysgo.DevstackL1ELKindEnvVar, "geth")
	return func() {
		_ = os.Unsetenv(sysgo.GethExecPathEnvVar)
		_ = os.Unsetenv(sysgo.DevstackL1ELKindEnvVar)
	}
}

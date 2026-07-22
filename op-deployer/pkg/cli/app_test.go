package cli

import "testing"

func TestNewAppDoesNotRegisterUpgradeCommand(t *testing.T) {
	app := NewApp("v0.0.0-test")
	for _, command := range app.Commands {
		if command.Name == "upgrade" {
			t.Fatal("upgrade command is still registered")
		}
	}
}

package cli

import "testing"

func TestNewAppDoesNotRegisterRemovedCommands(t *testing.T) {
	app := NewApp("v0.0.0-test")
	for _, command := range app.Commands {
		if command.Name == "upgrade" || command.Name == "manage" {
			t.Fatalf("%s command is still registered", command.Name)
		}
	}
}

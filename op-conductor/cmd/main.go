package main

import (
	"github.com/ethereum-optimism/optimism/op-conductor/app"
)

var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	app.Run(app.Config{
		Version:   Version,
		GitCommit: GitCommit,
		GitDate:   GitDate,
	})
}

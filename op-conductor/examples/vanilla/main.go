// Command vanilla runs op-conductor as a drop-in, importing it as a library
// rather than building from cmd.
//
// It is behaviourally identical to the op-conductor binary: same name, same
// flags, same config, same lifecycle. This file is the entire amount of code
// required, which is the point of the example.
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

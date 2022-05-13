package version

import (
	"fmt"
	"runtime"
)

var (
	Version   string
	GitCommit string
	BuildDate string
	GoVersion = runtime.Version()
)

func Info() string {
	return fmt.Sprintf("(version=%s, gitcommit=%s)", Version, GitCommit)
}

func BuildContext() string {
	return fmt.Sprintf("(go=%s, date=%s)", GoVersion, BuildDate)
}

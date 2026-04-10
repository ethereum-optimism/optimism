package flags

import "github.com/urfave/cli/v2"

const EnvVarPrefix = "OPRM"

var (
	ConfigFlag = &cli.PathFlag{
		Name:    "config",
		Usage:   "Path to the oprm config file.",
		EnvVars: []string{EnvVarPrefix + "_CONFIG"},
	}
	RunsDirFlag = &cli.PathFlag{
		Name:    "runs-dir",
		Usage:   "Directory where release run journals are stored.",
		EnvVars: []string{EnvVarPrefix + "_RUNS_DIR"},
	}
	BaseBranchFlag = &cli.StringFlag{
		Name:    "base-branch",
		Usage:   "Base branch used for release verification tasks.",
		EnvVars: []string{EnvVarPrefix + "_BASE_BRANCH"},
	}
	GitHubOwnerFlag = &cli.StringFlag{
		Name:    "github-owner",
		Usage:   "GitHub owner for the main monorepo release target.",
		EnvVars: []string{EnvVarPrefix + "_GITHUB_OWNER"},
	}
	GitHubRepoFlag = &cli.StringFlag{
		Name:    "github-repo",
		Usage:   "GitHub repo for the main monorepo release target.",
		EnvVars: []string{EnvVarPrefix + "_GITHUB_REPO"},
	}
	OpGethOwnerFlag = &cli.StringFlag{
		Name:    "op-geth-owner",
		Usage:   "GitHub owner for the op-geth release target.",
		EnvVars: []string{EnvVarPrefix + "_OP_GETH_OWNER"},
	}
	OpGethRepoFlag = &cli.StringFlag{
		Name:    "op-geth-repo",
		Usage:   "GitHub repo for the op-geth release target.",
		EnvVars: []string{EnvVarPrefix + "_OP_GETH_REPO"},
	}
	OpGethCheckoutFlag = &cli.PathFlag{
		Name:    "op-geth-checkout",
		Usage:   "Path to the local op-geth checkout used for tag and branch verification tasks.",
		EnvVars: []string{EnvVarPrefix + "_OP_GETH_CHECKOUT"},
	}
	PrintFlag = &cli.BoolFlag{
		Name:  "print",
		Usage: "Print the journal contents to stdout instead of only printing its path.",
	}
	BumpFlag = &cli.StringSliceFlag{
		Name:  "bump",
		Usage: "Component-specific bump assignment, e.g. --bump op-node=patch",
	}
	ManualVersionFlag = &cli.StringSliceFlag{
		Name:  "manual-version",
		Usage: "Component-specific manual target release version, e.g. --manual-version op-node=v1.2.4",
	}
)

func GlobalFlags() []cli.Flag {
	return []cli.Flag{ConfigFlag, RunsDirFlag, BaseBranchFlag, GitHubOwnerFlag, GitHubRepoFlag, OpGethOwnerFlag, OpGethRepoFlag, OpGethCheckoutFlag}
}

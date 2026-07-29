package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type PrestateInfo struct {
	Hash    common.Hash `json:"hash"`
	Version string      `json:"version"`
	Type    string      `json:"type"`

	FppProgram         CommitInfo `json:"fpp-program"`
	ExecutionClient    CommitInfo `json:"execution-client"`
	SuperchainRegistry CommitInfo `json:"superchain-registry"`

	UpToDateChains []string        `json:"up-to-date-chains"`
	OutdatedChains []OutdatedChain `json:"outdated-chains"`
	MissingChains  []string        `json:"missing-chains"`
}

type OutdatedChain struct {
	Name string `json:"name"`
	Diff *Diff  `json:"diff,omitempty"`
}

type CommitInfo struct {
	Commit  string `json:"commit"`
	DiffUrl string `json:"diff-url"`
	DiffCmd string `json:"diff-cmd"`
}

func NewCommitInfo(org string, repository string, commit string, mainBranch string, dir string) CommitInfo {
	return CommitInfo{
		Commit:  commit,
		DiffUrl: fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", org, repository, commit, mainBranch),
		DiffCmd: fmt.Sprintf("git fetch && git diff %s...origin/%s %s", commit, mainBranch, dir),
	}
}

type Diff struct {
	Msg      string `json:"message"`
	Prestate any    `json:"prestate"`
	Latest   any    `json:"latest"`
}

package examples

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-challenger/fault"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func FullGame() {
	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))),
	)

	canonical := "abcdefgh"
	disputed := "abcdexyz"
	maxDepth := uint64(3)
	canonicalProvider := fault.NewAlphabetProvider(canonical, maxDepth)
	disputedProvider := fault.NewAlphabetProvider(disputed, maxDepth)

	root := fault.Claim{
		ClaimData: fault.ClaimData{
			Value:    common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000077a"),
			Position: fault.NewPosition(0, 0),
		},
	}

	o := fault.NewOrchestrator(maxDepth, []fault.TraceProvider{canonicalProvider, disputedProvider}, []string{"charlie", "mallory"}, []bool{false, true}, root)
	o.Start()
}

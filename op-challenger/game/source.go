package game

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type LoaderCreator func(caller bind.ContractCaller, gameAddr common.Address, dir string) (types.GamePlayer, error)

type Source struct {
	logger         log.Logger
	contractCaller bind.ContractCaller
	gameLoader     *gameLoader
	types          map[uint8]LoaderCreator
}

func NewGameSource(logger log.Logger, addr common.Address, caller bind.ContractCaller) (*Source, error) {
	factoryCaller, err := bindings.NewDisputeGameFactoryCaller(addr, caller)
	if err != nil {
		return nil, fmt.Errorf("bind dispute game factory caller: %w", err)
	}
	loader := NewGameLoader(factoryCaller)
	return &Source{
		logger:         logger,
		contractCaller: caller,
		gameLoader:     loader,
		types:          make(map[uint8]LoaderCreator),
	}, nil
}

func (s *Source) RegisterGameType(code uint8, createLoader LoaderCreator) {
	s.types[code] = createLoader
}

func (s *Source) FetchAllGamesAtBlock(ctx context.Context, earliestTimestamp uint64, blockNumber *big.Int) ([]types.PlayerCreator, error) {
	games, err := s.gameLoader.FetchAllGamesAtBlock(ctx, earliestTimestamp, blockNumber)
	if err != nil {
		return nil, err
	}
	var loaders []types.PlayerCreator
	for _, game := range games {
		creator, ok := s.types[game.GameType]
		if !ok {
			s.logger.Warn("Skipping unsupported game type", "game", game.Proxy, "type", game.GameType)
			continue
		}
		loaders = append(loaders, &Creator{
			contractCaller: s.contractCaller,
			addr:           game.Proxy,
			creator:        creator,
		})
	}
	return loaders, nil
}

type Creator struct {
	contractCaller bind.ContractCaller
	addr           common.Address
	creator        LoaderCreator
}

func (c *Creator) Addr() common.Address {
	return c.addr
}

func (c *Creator) Create(dir string) (types.GamePlayer, error) {
	return c.creator(c.contractCaller, c.addr, dir)
}

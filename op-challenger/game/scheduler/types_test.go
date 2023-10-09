package scheduler

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
)

func asPlayerCreators(addrs ...common.Address) []types.PlayerCreator {
	var creators []types.PlayerCreator
	for _, addr := range addrs {
		creators = append(creators, &stubPlayerCreator{
			addr:   addr,
			status: types.GameStatusInProgress,
		})
	}
	return creators
}

func asStubPlayerCreators(addrs ...common.Address) []*stubPlayerCreator {
	var creators []*stubPlayerCreator
	for _, addr := range addrs {
		creators = append(creators, &stubPlayerCreator{
			addr:   addr,
			status: types.GameStatusInProgress,
		})
	}
	return creators
}

func toPlayerCreators(stubs []*stubPlayerCreator) []types.PlayerCreator {
	var out []types.PlayerCreator
	for _, stub := range stubs {
		out = append(out, stub)
	}
	return out
}

type stubPlayerCreator struct {
	addr          common.Address
	status        types.GameStatus
	created       bool
	creationError error
	dir           string
}

func (c *stubPlayerCreator) ProgressGame(ctx context.Context) types.GameStatus {
	return c.status
}

func (c *stubPlayerCreator) Status() types.GameStatus {
	return c.status
}

func (c *stubPlayerCreator) Addr() common.Address {
	return c.addr
}

func (c *stubPlayerCreator) Create(dir string) (types.GamePlayer, error) {
	if c.created {
		return nil, errors.New("already created game")
	}
	if c.creationError != nil {
		return nil, c.creationError
	}
	c.created = true
	c.dir = dir
	return c, nil
}

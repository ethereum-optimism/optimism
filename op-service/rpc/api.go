package rpc

import (
	"context"
	"fmt"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

type CommonAdminAPI struct {
	log log.Logger
}

func NewCommonAdminAPI(log log.Logger) *CommonAdminAPI {
	return &CommonAdminAPI{
		log: log,
	}
}

func (n *CommonAdminAPI) SetLogLevel(ctx context.Context, lvlStr string) error {
	lvl, err := oplog.LevelFromString(lvlStr)
	if err != nil {
		return err
	}

	h := n.log.Handler()
	// We set the log level, and do not wrap the handler with an additional filter handler,
	// as the underlying handler would otherwise also still filter with the previous log level.
	lvlSetter, ok := h.(oplog.LvlSetter)
	if !ok {
		return fmt.Errorf("log handler type %T cannot change log level", h)
	}
	lvlSetter.SetLogLevel(lvl)
	return nil
}

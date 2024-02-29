package utils

import (
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/ethereum-optimism/optimism/op-plasma/avail/config"
)

var localNonce uint32 = 0

const cfgPath = "../"

func GetAccountNonce(accountNonce uint32) uint32 {
	if accountNonce > localNonce {
		localNonce = accountNonce
		return accountNonce
	}
	localNonce++
	return localNonce
}

func getSubstrateApi(ApiURL string) (*gsrpc.SubstrateAPI, error) {
	api, err := gsrpc.NewSubstrateAPI(ApiURL)

	if err != nil {
		return &gsrpc.SubstrateAPI{}, err
	}
	return api, nil
}

func getConfig() config.Config {
	//Load variables
	var config config.Config
	err := config.GetConfig(cfgPath)
	if err != nil {
		panic(fmt.Sprintf("cannot get config:%v", err))
	}
	return config
}

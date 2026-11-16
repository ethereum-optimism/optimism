package h

import "example.com/geth/types"

func Apply() error {
	_, err := types.Apply()
	return err
}

func Used(start uint64) uint64 {
	left := types.Call(start)
	return start - left
}

func SetAccessList(p *types.Payload) {
	empty := []byte{}
	p.AccessList = &empty
}

func Withdrawals(p *types.Payload) []byte { return *p.Withdrawals }

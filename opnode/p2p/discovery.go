package p2p

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"net"
)

func (conf *Config) Discovery(log log.Logger) (*enode.LocalNode, *discover.UDPv5, error) {
	localNode := enode.NewLocalNode(conf.DiscoveryDB, conf.Priv)
	if conf.AdvertiseIP != nil {
		localNode.SetStaticIP(conf.AdvertiseIP)
	}
	if conf.AdvertiseUDPPort != 0 {
		localNode.SetFallbackUDP(int(conf.AdvertiseUDPPort))
	}

	udpAddr := &net.UDPAddr{
		IP:   conf.ListenIP,
		Port: int(conf.ListenUDPPort),
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, nil, err
	}

	cfg := discover.Config{
		PrivateKey:   conf.Priv,
		NetRestrict:  nil,
		Bootnodes:    conf.Bootnodes,
		Unhandled:    nil, // Not used in dv5
		Log:          log,
		ValidSchemes: enode.ValidSchemes,
	}
	udpV5, err := discover.ListenV5(conn, localNode, cfg)
	if err != nil {
		return nil, nil, err
	}
	return localNode, udpV5, nil
}

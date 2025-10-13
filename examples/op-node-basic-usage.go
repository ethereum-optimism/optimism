package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// Ejemplo básico de uso de op-node
// Contribución de vaiosx.base.eth

func main() {
	// Configuración básica
	config := &opnode.Config{
		L1: &opnode.L1EndpointConfig{
			L1NodeAddr: "https://mainnet.infura.io/v3/YOUR_KEY",
			L1TrustRPC: true,
		},
		L2: &opnode.L2EndpointConfig{
			L2EngineAddr: "http://localhost:8545",
			L2EngineJWT:  "jwt-secret",
		},
		Rollup: &rollup.Config{
			Genesis: rollup.Genesis{
				L1: rollup.BlockID{
					Hash:   "0x...",
					Number: 12345678,
				},
				L2: rollup.BlockID{
					Hash:   "0x...",
					Number: 0,
				},
				L2Time: uint64(time.Now().Unix()),
			},
			BlockTime:             2,
			MaxSequencerDrift:     600,
			SeqWindowSize:         3600,
			L1ChainID:             1,
			L2ChainID:             10,
		},
		RPC: &opnode.RPCConfig{
			ListenAddr: "0.0.0.0",
			ListenPort: 8547,
		},
	}

	// Crear instancia de op-node
	node, err := opnode.New(context.Background(), config)
	if err != nil {
		log.Fatal("Error creando op-node:", err)
	}

	// Iniciar op-node
	if err := node.Start(context.Background()); err != nil {
		log.Fatal("Error iniciando op-node:", err)
	}

	fmt.Println("✅ OP Node iniciado correctamente")
	fmt.Println("🔗 RPC disponible en: http://localhost:8547")
	
	// Mantener el proceso activo
	select {}
}

package utils

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/devnet-sdk/shell/env"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

type TestReorgManager struct {
	t            devtest.CommonT
	env          *env.DevnetEnv
	blockBuilder *TestBlockBuilder
	pos          *TestPOS
}

func NewTestReorgManager(t devtest.CommonT) *TestReorgManager {
	url := os.Getenv(env.EnvURLVar)
	if url == "" {
		t.Errorf("environment variable %s is not set", env.EnvURLVar)
		return nil
	}

	env, err := env.LoadDevnetFromURL(url)
	if err != nil {
		t.Errorf("failed to load devnet environment from URL %s: %v", url, err)
		return nil
	}

	var engineURL, rpcURL string
	for _, node := range env.Env.L1.Nodes {
		el, ok := node.Services["el"]
		if !ok {
			continue
		}

		engine, ok := el.Endpoints["engine-rpc"]
		if !ok {
			continue
		}

		rpc, ok := el.Endpoints["rpc"]
		if !ok {
			continue
		}

		engineURL = fmt.Sprintf("http://%s:%d", engine.Host, engine.Port)
		rpcURL = fmt.Sprintf("http://%s:%d", rpc.Host, rpc.Port)
		break
	}

	if engineURL == "" || rpcURL == "" {
		t.Errorf("could not find engine or RPC endpoints in the devnet environment")
		return nil
	}

	blockBuilder := NewTestBlockBuilder(t, TestBlockBuilderConfig{
		GethRPC:                rpcURL,
		EngineRPC:              engineURL,
		JWTSecret:              env.Env.L1.JWT,
		safeBlockDistance:      10,
		finalizedBlockDistance: 20,
	})

	pos := NewTestPOS(t, rpcURL, blockBuilder)
	return &TestReorgManager{t, env, blockBuilder, pos}
}

func (m *TestReorgManager) StopL1CL() {
	m.t.Log("Stopping L1 CL services")

	panic("not implemented. TODO(@theochap): implement this `https://github.com/op-rs/kona/issues/3174`")

	// kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	// if err != nil {
	// 	m.t.Errorf("failed to create kurtosis context: %v", err)
	// 	return
	// }

	// // Use a bounded context to avoid hanging tests if Kurtosis call stalls.
	// ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer cancel()
	// enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, m.env.Env.Name)
	// if err != nil {
	// 	m.t.Errorf("failed to get enclave context: %v", err)
	// 	return
	// }

	// for _, node := range m.env.Env.L1.Nodes {
	// 	cl, ok := node.Services["cl"]
	// 	if !ok {
	// 		continue
	// 	}

	// 	svcCtx, err := enclaveCtx.GetServiceContext(cl.Name)
	// 	if err != nil {
	// 		m.t.Errorf("failed to get service context for %s: %v", cl.Name, err)
	// 		return
	// 	}

	// 	_, _, err = svcCtx.ExecCommand([]string{"sh", "-c", "kill 1"})
	// 	if err != nil {
	// 		m.t.Errorf("failed to stop service %s: %v", cl.Name, err)
	// 		return
	// 	}
	// }
}

func (m *TestReorgManager) GetBlockBuilder() *TestBlockBuilder {
	return m.blockBuilder
}

func (m *TestReorgManager) GetPOS() *TestPOS {
	return m.pos
}

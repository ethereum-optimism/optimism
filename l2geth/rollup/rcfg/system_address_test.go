package rcfg

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/l2geth/common"
)

func TestSystemAddressFor(t *testing.T) {
	tests := []struct {
		deployer0 common.Address
		deployer1 common.Address
		chainId   int64
	}{
		{
			common.HexToAddress("0xcDE47C1a5e2d60b9ff262b0a3b6d486048575Ad9"),
			common.HexToAddress("0x53A6eecC2dD4795Fcc68940ddc6B4d53Bd88Bd9E"),
			10,
		},
		{
			common.HexToAddress("0xd23eb5c2dd7035e6eb4a7e129249d9843123079f"),
			common.HexToAddress("0xa81224490b9fa4930a2e920550cd1c9106bb6d9e"),
			69,
		},
		{
			common.HexToAddress("0xc30276833798867c1dbc5c468bf51ca900b44e4c"),
			common.HexToAddress("0x5c679a57e018f5f146838138d3e032ef4913d551"),
			420,
		},
		{
			common.HexToAddress("0xc30276833798867c1dbc5c468bf51ca900b44e4c"),
			common.HexToAddress("0x5c679a57e018f5f146838138d3e032ef4913d551"),
			421,
		},
	}
	for _, tt := range tests {
		chainID := big.NewInt(tt.chainId)
		sad0 := SystemAddressFor(chainID, tt.deployer0)
		if sad0 != SystemAddress0 {
			t.Fatalf("expected %s, got %s", SystemAddress0.String(), sad0.String())
		}
		sad1 := SystemAddressFor(chainID, tt.deployer1)
		if sad1 != SystemAddress1 {
			t.Fatalf("expected %s, got %s", SystemAddress1.String(), sad1.String())
		}
		if SystemAddressFor(chainID, randAddr()) != ZeroSystemAddress {
			t.Fatalf("expected zero address, but got a non-zero one instead")
		}
	}

	// test env fallback
	addr0 := randAddr()
	addr1 := randAddr()
	chainID := big.NewInt(999)
	if SystemAddressFor(chainID, addr0) != ZeroSystemAddress {
		t.Fatalf("expected zero address, but got a non-zero one instead")
	}
	if SystemAddressFor(chainID, addr1) != ZeroSystemAddress {
		t.Fatalf("expected zero address, but got a non-zero one instead")
	}
	if err := os.Setenv("SYSTEM_ADDRESS_0_DEPLOYER", addr0.String()); err != nil {
		t.Fatalf("error setting env for deployer 0: %v", err)
	}
	if err := os.Setenv("SYSTEM_ADDRESS_1_DEPLOYER", addr1.String()); err != nil {
		t.Fatalf("error setting env for deployer 1: %v", err)
	}
	initEnvSystemAddressDeployer()
	sad0 := SystemAddressFor(chainID, addr0)
	if sad0 != SystemAddress0 {
		t.Fatalf("expected %s, got %s", SystemAddress0.String(), sad0.String())
	}
	sad1 := SystemAddressFor(chainID, addr1)
	if sad1 != SystemAddress1 {
		t.Fatalf("expected %s, got %s", SystemAddress1.String(), sad1.String())
	}

	// reset
	if err := os.Setenv("SYSTEM_ADDRESS_0_DEPLOYER", ""); err != nil {
		t.Fatalf("error setting env for deployer 0: %v", err)
	}
	if err := os.Setenv("SYSTEM_ADDRESS_1_DEPLOYER", ""); err != nil {
		t.Fatalf("error setting env for deployer 1: %v", err)
	}
	initEnvSystemAddressDeployer()
}

func TestSystemAddressDeployer(t *testing.T) {
	addr0 := randAddr()
	addr1 := randAddr()
	deployer := SystemAddressDeployer{addr0, addr1}

	assertAddress(t, deployer, addr0, SystemAddress0)
	assertAddress(t, deployer, addr1, SystemAddress1)
	assertAddress(t, deployer, randAddr(), ZeroSystemAddress)

	var zeroDeployer SystemAddressDeployer
	assertAddress(t, zeroDeployer, randAddr(), ZeroSystemAddress)
}

func TestEnvSystemAddressDeployer(t *testing.T) {
	addr0 := randAddr()
	addr1 := randAddr()

	assertAddress(t, envSystemAddressDeployer, addr0, ZeroSystemAddress)
	assertAddress(t, envSystemAddressDeployer, addr1, ZeroSystemAddress)
	assertAddress(t, envSystemAddressDeployer, randAddr(), ZeroSystemAddress)

	if err := os.Setenv("SYSTEM_ADDRESS_0_DEPLOYER", addr0.String()); err != nil {
		t.Fatalf("error setting env for deployer 0: %v", err)
	}
	if err := os.Setenv("SYSTEM_ADDRESS_1_DEPLOYER", addr1.String()); err != nil {
		t.Fatalf("error setting env for deployer 1: %v", err)
	}

	initEnvSystemAddressDeployer()
	assertAddress(t, envSystemAddressDeployer, addr0, SystemAddress0)
	assertAddress(t, envSystemAddressDeployer, addr1, SystemAddress1)
	assertAddress(t, envSystemAddressDeployer, randAddr(), ZeroSystemAddress)

	tests := []struct {
		deployer0 string
		deployer1 string
		msg       string
	}{
		{
			"not an address",
			addr0.String(),
			"SYSTEM_ADDRESS_0_DEPLOYER specified but invalid",
		},
		{
			"not an address",
			"not an address",
			"SYSTEM_ADDRESS_0_DEPLOYER specified but invalid",
		},
		{
			addr0.String(),
			"not an address",
			"SYSTEM_ADDRESS_1_DEPLOYER specified but invalid",
		},
	}
	for _, tt := range tests {
		if err := os.Setenv("SYSTEM_ADDRESS_0_DEPLOYER", tt.deployer0); err != nil {
			t.Fatalf("error setting env for deployer 0: %v", err)
		}
		if err := os.Setenv("SYSTEM_ADDRESS_1_DEPLOYER", tt.deployer1); err != nil {
			t.Fatalf("error setting env for deployer 1: %v", err)
		}
		assertPanic(t, tt.msg, func() {
			initEnvSystemAddressDeployer()
		})
	}
}

func randAddr() common.Address {
	buf := make([]byte, 20)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return common.BytesToAddress(buf)
}

func assertAddress(t *testing.T, deployer SystemAddressDeployer, in common.Address, expected common.Address) {
	actual := deployer.SystemAddressFor(in)
	if actual != expected {
		t.Fatalf("bad system address. expected %s, got %s", expected.String(), actual.String())
	}
}

func assertPanic(t *testing.T, msg string, cb func()) {
	defer func() {
		if err := recover(); err != nil {
			errMsg := fmt.Sprintf("%v", err)
			if errMsg != msg {
				t.Fatalf("expected error message %s, got %v", msg, errMsg)
			}
		}
	}()

	cb()
}

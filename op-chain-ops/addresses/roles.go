package addresses

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

type L1Roles struct {
	SuperchainRoles
	OpChainRoles
}

type SuperchainRoles struct {
	SuperchainProxyAdminOwner common.Address
	SuperchainGuardian        common.Address
	ProtocolVersionsOwner     common.Address
}

type OpChainRoles struct {
	OpChainCoreRoles
	OpChainFaultProofsRoles
}

type OpChainCoreRoles struct {
	SystemConfigOwner      common.Address
	OpChainProxyAdminOwner common.Address
	OpChainGuardian        common.Address
	UnsafeBlockSigner      common.Address
	BatchSubmitter         common.Address
}

type OpChainFaultProofsRoles struct {
	Proposer   common.Address
	Challenger common.Address
}

var ErrSuperchainRoleZeroAddress = errors.New("SuperchainRole is set to zero address")

func (s *SuperchainRoles) CheckNoZeroAddresses() error {
	val := reflect.ValueOf(*s)
	typ := reflect.TypeOf(*s)

	// Iterate through all the fields
	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i)
		fieldName := typ.Field(i).Name

		if fieldValue.Interface() == (common.Address{}) {
			return fmt.Errorf("%w: %s", ErrSuperchainRoleZeroAddress, fieldName)
		}
	}
	return nil
}

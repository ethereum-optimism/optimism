// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ERC20VotesCheckpoint is an auto generated low-level Go binding around an user-defined struct.
type ERC20VotesCheckpoint struct {
	FromBlock uint32
	Votes     *big.Int
}

// GovernanceTokenMetaData contains all meta data concerning the GovernanceToken contract.
var GovernanceTokenMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegator\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"fromDelegate\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"toDelegate\",\"type\":\"address\"}],\"name\":\"DelegateChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"delegate\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"previousBalance\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newBalance\",\"type\":\"uint256\"}],\"name\":\"DelegateVotesChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burnFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"pos\",\"type\":\"uint32\"}],\"name\":\"checkpoints\",\"outputs\":[{\"components\":[{\"internalType\":\"uint32\",\"name\":\"fromBlock\",\"type\":\"uint32\"},{\"internalType\":\"uint224\",\"name\":\"votes\",\"type\":\"uint224\"}],\"internalType\":\"structERC20Votes.Checkpoint\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegatee\",\"type\":\"address\"}],\"name\":\"delegate\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegatee\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expiry\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"delegateBySig\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"delegates\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getPastTotalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getPastVotes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"getVotes\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"numCheckpoints\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6101406040523480156200001257600080fd5b50604051806040016040528060088152602001674f7074696d69736d60c01b81525080604051806040016040528060018152602001603160f81b815250604051806040016040528060088152602001674f7074696d69736d60c01b8152506040518060400160405280600281526020016104f560f41b81525081600390816200009c919062000247565b506004620000ab828262000247565b5050825160209384012082519284019290922060e08390526101008190524660a0818152604080517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f818901819052818301979097526060810194909452608080850193909352308483018190528151808603909301835260c09485019091528151919096012090529290925261012052506200014a90503362000150565b62000313565b600a80546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b634e487b7160e01b600052604160045260246000fd5b600181811c90821680620001cd57607f821691505b602082108103620001ee57634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200024257600081815260208120601f850160051c810160208610156200021d5750805b601f850160051c820191505b818110156200023e5782815560010162000229565b5050505b505050565b81516001600160401b03811115620002635762000263620001a2565b6200027b81620002748454620001b8565b84620001f4565b602080601f831160018114620002b357600084156200029a5750858301515b600019600386901b1c1916600185901b1785556200023e565b600085815260208120601f198616915b82811015620002e457888601518255948401946001909101908401620002c3565b5085821015620003035787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b60805160a05160c05160e051610100516101205161286a6200036360003960006113f2015260006114410152600061141c015260006113750152600061139f015260006113c9015261286a6000f3fe608060405234801561001057600080fd5b50600436106101c45760003560e01c8063715018a6116100f9578063a457c2d711610097578063d505accf11610071578063d505accf14610416578063dd62ed3e14610429578063f1127ed81461046f578063f2fde38b146104c157600080fd5b8063a457c2d7146103dd578063a9059cbb146103f0578063c3cda5201461040357600080fd5b80638da5cb5b116100d35780638da5cb5b146103915780638e539e8c146103af57806395d89b41146103c25780639ab24eb0146103ca57600080fd5b8063715018a61461036357806379cc67901461036b5780637ecebe001461037e57600080fd5b80633a46b1a811610166578063587cde1e11610140578063587cde1e146102945780635c19a95c146102f25780636fcfff451461030557806370a082311461032d57600080fd5b80633a46b1a81461025957806340c10f191461026c57806342966c681461028157600080fd5b806323b872dd116101a257806323b872dd1461021c578063313ce5671461022f5780633644e5151461023e578063395093511461024657600080fd5b806306fdde03146101c9578063095ea7b3146101e757806318160ddd1461020a575b600080fd5b6101d16104d4565b6040516101de919061249d565b60405180910390f35b6101fa6101f5366004612539565b610566565b60405190151581526020016101de565b6002545b6040519081526020016101de565b6101fa61022a366004612563565b61057e565b604051601281526020016101de565b61020e6105a2565b6101fa610254366004612539565b6105b1565b61020e610267366004612539565b6105fd565b61027f61027a366004612539565b6106a3565b005b61027f61028f36600461259f565b6106b9565b6102cd6102a23660046125b8565b73ffffffffffffffffffffffffffffffffffffffff9081166000908152600760205260409020541690565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101de565b61027f6103003660046125b8565b6106c6565b6103186103133660046125b8565b6106d0565b60405163ffffffff90911681526020016101de565b61020e61033b3660046125b8565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b61027f610705565b61027f610379366004612539565b610719565b61020e61038c3660046125b8565b61072e565b600a5473ffffffffffffffffffffffffffffffffffffffff166102cd565b61020e6103bd36600461259f565b610759565b6101d16107cf565b61020e6103d83660046125b8565b6107de565b6101fa6103eb366004612539565b6108a9565b6101fa6103fe366004612539565b61097a565b61027f6104113660046125e4565b610988565b61027f61042436600461263c565b610aff565b61020e6104373660046126a6565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260016020908152604080832093909416825291909152205490565b61048261047d3660046126d9565b610cbe565b60408051825163ffffffff1681526020928301517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1692810192909252016101de565b61027f6104cf3660046125b8565b610d64565b6060600380546104e390612719565b80601f016020809104026020016040519081016040528092919081815260200182805461050f90612719565b801561055c5780601f106105315761010080835404028352916020019161055c565b820191906000526020600020905b81548152906001019060200180831161053f57829003601f168201915b5050505050905090565b600033610574818585610e18565b5060019392505050565b60003361058c858285610fcb565b6105978585856110a2565b506001949350505050565b60006105ac61135b565b905090565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919061057490829086906105f8908790612795565b610e18565b600043821061066d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f4552433230566f7465733a20626c6f636b206e6f7420796574206d696e65640060448201526064015b60405180910390fd5b73ffffffffffffffffffffffffffffffffffffffff8316600090815260086020526040902061069c908361148f565b9392505050565b6106ab611576565b6106b582826115f7565b5050565b6106c33382611601565b50565b6106c3338261160b565b73ffffffffffffffffffffffffffffffffffffffff81166000908152600860205260408120546106ff906116a9565b92915050565b61070d611576565b6107176000611743565b565b610724823383610fcb565b6106b58282611601565b73ffffffffffffffffffffffffffffffffffffffff81166000908152600560205260408120546106ff565b60004382106107c4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f4552433230566f7465733a20626c6f636b206e6f7420796574206d696e6564006044820152606401610664565b6106ff60098361148f565b6060600480546104e390612719565b73ffffffffffffffffffffffffffffffffffffffff811660009081526008602052604081205480156108815773ffffffffffffffffffffffffffffffffffffffff8316600090815260086020526040902061083a6001836127ad565b8154811061084a5761084a6127c4565b60009182526020909120015464010000000090047bffffffffffffffffffffffffffffffffffffffffffffffffffffffff16610884565b60005b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff169392505050565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff871684529091528120549091908381101561096d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f0000000000000000000000000000000000000000000000000000006064820152608401610664565b6105978286868403610e18565b6000336105748185856110a2565b834211156109f2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f4552433230566f7465733a207369676e617475726520657870697265640000006044820152606401610664565b604080517fe48329057bfd03d55e49b547132e39cffd9c1820ad7b9d4c5307691425d15adf602082015273ffffffffffffffffffffffffffffffffffffffff8816918101919091526060810186905260808101859052600090610a7990610a719060a001604051602081830303815290604052805190602001206117ba565b858585611823565b9050610a848161184b565b8614610aec576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f4552433230566f7465733a20696e76616c6964206e6f6e6365000000000000006044820152606401610664565b610af6818861160b565b50505050505050565b83421115610b69576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332305065726d69743a206578706972656420646561646c696e650000006044820152606401610664565b60007f6e71edae12b1b97f4d1f60370fef10105fa2faae0126114a169c64845d6126c9888888610b988c61184b565b60408051602081019690965273ffffffffffffffffffffffffffffffffffffffff94851690860152929091166060840152608083015260a082015260c0810186905260e0016040516020818303038152906040528051906020012090506000610c00826117ba565b90506000610c1082878787611823565b90508973ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614610ca7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601e60248201527f45524332305065726d69743a20696e76616c6964207369676e617475726500006044820152606401610664565b610cb28a8a8a610e18565b50505050505050505050565b604080518082019091526000808252602082015273ffffffffffffffffffffffffffffffffffffffff83166000908152600860205260409020805463ffffffff8416908110610d0f57610d0f6127c4565b60009182526020918290206040805180820190915291015463ffffffff8116825264010000000090047bffffffffffffffffffffffffffffffffffffffffffffffffffffffff16918101919091529392505050565b610d6c611576565b73ffffffffffffffffffffffffffffffffffffffff8116610e0f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f64647265737300000000000000000000000000000000000000000000000000006064820152608401610664565b6106c381611743565b73ffffffffffffffffffffffffffffffffffffffff8316610eba576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f72657373000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff8216610f5d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f73730000000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925910160405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff811461109c578181101561108f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e63650000006044820152606401610664565b61109c8484848403610e18565b50505050565b73ffffffffffffffffffffffffffffffffffffffff8316611145576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f64726573730000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff82166111e8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f65737300000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff83166000908152602081905260409020548181101561129e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e636500000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff8085166000908152602081905260408082208585039055918516815290812080548492906112e2908490612795565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8460405161134891815260200190565b60405180910390a361109c848484611885565b60003073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156113c157507f000000000000000000000000000000000000000000000000000000000000000046145b156113eb57507f000000000000000000000000000000000000000000000000000000000000000090565b50604080517f00000000000000000000000000000000000000000000000000000000000000006020808301919091527f0000000000000000000000000000000000000000000000000000000000000000828401527f000000000000000000000000000000000000000000000000000000000000000060608301524660808301523060a0808401919091528351808403909101815260c0909201909252805191012090565b8154600090815b818110156114f35760006114aa8284611890565b9050848682815481106114bf576114bf6127c4565b60009182526020909120015463ffffffff1611156114df578092506114ed565b6114ea816001612795565b91505b50611496565b811561154c57846115056001846127ad565b81548110611515576115156127c4565b60009182526020909120015464010000000090047bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1661154f565b60005b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1695945050505050565b600a5473ffffffffffffffffffffffffffffffffffffffff163314610717576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610664565b6106b582826118ab565b6106b58282611971565b73ffffffffffffffffffffffffffffffffffffffff8281166000818152600760208181526040808420805485845282862054949093528787167fffffffffffffffffffffffff00000000000000000000000000000000000000008416811790915590519190951694919391928592917f3134e8a2e6d97e929a7e54011ea5485d7d196dd5f0ba4d4ef95803e8e3fc257f9190a461109c828483611989565b600063ffffffff82111561173f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f53616665436173743a2076616c756520646f65736e27742066697420696e203360448201527f32206269747300000000000000000000000000000000000000000000000000006064820152608401610664565b5090565b600a805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60006106ff6117c761135b565b836040517f19010000000000000000000000000000000000000000000000000000000000006020820152602281018390526042810182905260009060620160405160208183030381529060405280519060200120905092915050565b600080600061183487878787611b2e565b9150915061184181611c46565b5095945050505050565b73ffffffffffffffffffffffffffffffffffffffff811660009081526005602052604090208054600181018255905b50919050565b505050565b611880838383611e9a565b600061189f60028484186127f3565b61069c90848416612795565b6118b58282611ed9565b6002547bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1015611963576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4552433230566f7465733a20746f74616c20737570706c79207269736b73206f60448201527f766572666c6f77696e6720766f746573000000000000000000000000000000006064820152608401610664565b61109c60096120018361200d565b61197b82826121ef565b61109c60096123e38361200d565b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16141580156119c55750600081115b156118805773ffffffffffffffffffffffffffffffffffffffff831615611a7a5773ffffffffffffffffffffffffffffffffffffffff831660009081526008602052604081208190611a1a906123e38561200d565b915091508473ffffffffffffffffffffffffffffffffffffffff167fdec2bacdd2f05b59de34da9b523dff8be42e5e38e818c82fdb0bae774387a7248383604051611a6f929190918252602082015260400190565b60405180910390a250505b73ffffffffffffffffffffffffffffffffffffffff8216156118805773ffffffffffffffffffffffffffffffffffffffff821660009081526008602052604081208190611aca906120018561200d565b915091508373ffffffffffffffffffffffffffffffffffffffff167fdec2bacdd2f05b59de34da9b523dff8be42e5e38e818c82fdb0bae774387a7248383604051611b1f929190918252602082015260400190565b60405180910390a25050505050565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0831115611b655750600090506003611c3d565b8460ff16601b14158015611b7d57508460ff16601c14155b15611b8e5750600090506004611c3d565b6040805160008082526020820180845289905260ff881692820192909252606081018690526080810185905260019060a0016020604051602081039080840390855afa158015611be2573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff8116611c3657600060019250925050611c3d565b9150600090505b94509492505050565b6000816004811115611c5a57611c5a61282e565b03611c625750565b6001816004811115611c7657611c7661282e565b03611cdd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601860248201527f45434453413a20696e76616c6964207369676e617475726500000000000000006044820152606401610664565b6002816004811115611cf157611cf161282e565b03611d58576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45434453413a20696e76616c6964207369676e6174757265206c656e677468006044820152606401610664565b6003816004811115611d6c57611d6c61282e565b03611df9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202773272076616c60448201527f75650000000000000000000000000000000000000000000000000000000000006064820152608401610664565b6004816004811115611e0d57611e0d61282e565b036106c3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45434453413a20696e76616c6964207369676e6174757265202776272076616c60448201527f75650000000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff83811660009081526007602052604080822054858416835291205461188092918216911683611989565b73ffffffffffffffffffffffffffffffffffffffff8216611f56576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f2061646472657373006044820152606401610664565b8060026000828254611f689190612795565b909155505073ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604081208054839290611fa2908490612795565b909155505060405181815273ffffffffffffffffffffffffffffffffffffffff8316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a36106b560008383611885565b600061069c8284612795565b82546000908190801561206d57856120266001836127ad565b81548110612036576120366127c4565b60009182526020909120015464010000000090047bffffffffffffffffffffffffffffffffffffffffffffffffffffffff16612070565b60005b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff16925061209e83858763ffffffff16565b91506000811180156120dc575043866120b86001846127ad565b815481106120c8576120c86127c4565b60009182526020909120015463ffffffff16145b15612166576120ea826123ef565b866120f66001846127ad565b81548110612106576121066127c4565b9060005260206000200160000160046101000a8154817bffffffffffffffffffffffffffffffffffffffffffffffffffffffff02191690837bffffffffffffffffffffffffffffffffffffffffffffffffffffffff1602179055506121e6565b85604051806040016040528061217b436116a9565b63ffffffff16815260200161218f856123ef565b7bffffffffffffffffffffffffffffffffffffffffffffffffffffffff90811690915282546001810184556000938452602093849020835194909301519091166401000000000263ffffffff909316929092179101555b50935093915050565b73ffffffffffffffffffffffffffffffffffffffff8216612292576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f73000000000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604090205481811015612348576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f63650000000000000000000000000000000000000000000000000000000000006064820152608401610664565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604081208383039055600280548492906123849084906127ad565b909155505060405182815260009073ffffffffffffffffffffffffffffffffffffffff8516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a361188083600084611885565b600061069c82846127ad565b60007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff82111561173f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602760248201527f53616665436173743a2076616c756520646f65736e27742066697420696e203260448201527f32342062697473000000000000000000000000000000000000000000000000006064820152608401610664565b600060208083528351808285015260005b818110156124ca578581018301518582016040015282016124ae565b818111156124dc576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461253457600080fd5b919050565b6000806040838503121561254c57600080fd5b61255583612510565b946020939093013593505050565b60008060006060848603121561257857600080fd5b61258184612510565b925061258f60208501612510565b9150604084013590509250925092565b6000602082840312156125b157600080fd5b5035919050565b6000602082840312156125ca57600080fd5b61069c82612510565b803560ff8116811461253457600080fd5b60008060008060008060c087890312156125fd57600080fd5b61260687612510565b95506020870135945060408701359350612622606088016125d3565b92506080870135915060a087013590509295509295509295565b600080600080600080600060e0888a03121561265757600080fd5b61266088612510565b965061266e60208901612510565b9550604088013594506060880135935061268a608089016125d3565b925060a0880135915060c0880135905092959891949750929550565b600080604083850312156126b957600080fd5b6126c283612510565b91506126d060208401612510565b90509250929050565b600080604083850312156126ec57600080fd5b6126f583612510565b9150602083013563ffffffff8116811461270e57600080fd5b809150509250929050565b600181811c9082168061272d57607f821691505b60208210810361187a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156127a8576127a8612766565b500190565b6000828210156127bf576127bf612766565b500390565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082612829577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fdfea164736f6c634300080f000a",
}

// GovernanceTokenABI is the input ABI used to generate the binding from.
// Deprecated: Use GovernanceTokenMetaData.ABI instead.
var GovernanceTokenABI = GovernanceTokenMetaData.ABI

// GovernanceTokenBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GovernanceTokenMetaData.Bin instead.
var GovernanceTokenBin = GovernanceTokenMetaData.Bin

// DeployGovernanceToken deploys a new Ethereum contract, binding an instance of GovernanceToken to it.
func DeployGovernanceToken(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *GovernanceToken, error) {
	parsed, err := GovernanceTokenMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GovernanceTokenBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &GovernanceToken{GovernanceTokenCaller: GovernanceTokenCaller{contract: contract}, GovernanceTokenTransactor: GovernanceTokenTransactor{contract: contract}, GovernanceTokenFilterer: GovernanceTokenFilterer{contract: contract}}, nil
}

// GovernanceToken is an auto generated Go binding around an Ethereum contract.
type GovernanceToken struct {
	GovernanceTokenCaller     // Read-only binding to the contract
	GovernanceTokenTransactor // Write-only binding to the contract
	GovernanceTokenFilterer   // Log filterer for contract events
}

// GovernanceTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type GovernanceTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GovernanceTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GovernanceTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GovernanceTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GovernanceTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GovernanceTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GovernanceTokenSession struct {
	Contract     *GovernanceToken  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GovernanceTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GovernanceTokenCallerSession struct {
	Contract *GovernanceTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// GovernanceTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GovernanceTokenTransactorSession struct {
	Contract     *GovernanceTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// GovernanceTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type GovernanceTokenRaw struct {
	Contract *GovernanceToken // Generic contract binding to access the raw methods on
}

// GovernanceTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GovernanceTokenCallerRaw struct {
	Contract *GovernanceTokenCaller // Generic read-only contract binding to access the raw methods on
}

// GovernanceTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GovernanceTokenTransactorRaw struct {
	Contract *GovernanceTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGovernanceToken creates a new instance of GovernanceToken, bound to a specific deployed contract.
func NewGovernanceToken(address common.Address, backend bind.ContractBackend) (*GovernanceToken, error) {
	contract, err := bindGovernanceToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GovernanceToken{GovernanceTokenCaller: GovernanceTokenCaller{contract: contract}, GovernanceTokenTransactor: GovernanceTokenTransactor{contract: contract}, GovernanceTokenFilterer: GovernanceTokenFilterer{contract: contract}}, nil
}

// NewGovernanceTokenCaller creates a new read-only instance of GovernanceToken, bound to a specific deployed contract.
func NewGovernanceTokenCaller(address common.Address, caller bind.ContractCaller) (*GovernanceTokenCaller, error) {
	contract, err := bindGovernanceToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenCaller{contract: contract}, nil
}

// NewGovernanceTokenTransactor creates a new write-only instance of GovernanceToken, bound to a specific deployed contract.
func NewGovernanceTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*GovernanceTokenTransactor, error) {
	contract, err := bindGovernanceToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenTransactor{contract: contract}, nil
}

// NewGovernanceTokenFilterer creates a new log filterer instance of GovernanceToken, bound to a specific deployed contract.
func NewGovernanceTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*GovernanceTokenFilterer, error) {
	contract, err := bindGovernanceToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenFilterer{contract: contract}, nil
}

// bindGovernanceToken binds a generic wrapper to an already deployed contract.
func bindGovernanceToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(GovernanceTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GovernanceToken *GovernanceTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GovernanceToken.Contract.GovernanceTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GovernanceToken *GovernanceTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GovernanceToken.Contract.GovernanceTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GovernanceToken *GovernanceTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GovernanceToken.Contract.GovernanceTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GovernanceToken *GovernanceTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GovernanceToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GovernanceToken *GovernanceTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GovernanceToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GovernanceToken *GovernanceTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GovernanceToken.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_GovernanceToken *GovernanceTokenCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_GovernanceToken *GovernanceTokenSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _GovernanceToken.Contract.DOMAINSEPARATOR(&_GovernanceToken.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_GovernanceToken *GovernanceTokenCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _GovernanceToken.Contract.DOMAINSEPARATOR(&_GovernanceToken.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.Allowance(&_GovernanceToken.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.Allowance(&_GovernanceToken.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.BalanceOf(&_GovernanceToken.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.BalanceOf(&_GovernanceToken.CallOpts, account)
}

// Checkpoints is a free data retrieval call binding the contract method 0xf1127ed8.
//
// Solidity: function checkpoints(address account, uint32 pos) view returns((uint32,uint224))
func (_GovernanceToken *GovernanceTokenCaller) Checkpoints(opts *bind.CallOpts, account common.Address, pos uint32) (ERC20VotesCheckpoint, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "checkpoints", account, pos)

	if err != nil {
		return *new(ERC20VotesCheckpoint), err
	}

	out0 := *abi.ConvertType(out[0], new(ERC20VotesCheckpoint)).(*ERC20VotesCheckpoint)

	return out0, err

}

// Checkpoints is a free data retrieval call binding the contract method 0xf1127ed8.
//
// Solidity: function checkpoints(address account, uint32 pos) view returns((uint32,uint224))
func (_GovernanceToken *GovernanceTokenSession) Checkpoints(account common.Address, pos uint32) (ERC20VotesCheckpoint, error) {
	return _GovernanceToken.Contract.Checkpoints(&_GovernanceToken.CallOpts, account, pos)
}

// Checkpoints is a free data retrieval call binding the contract method 0xf1127ed8.
//
// Solidity: function checkpoints(address account, uint32 pos) view returns((uint32,uint224))
func (_GovernanceToken *GovernanceTokenCallerSession) Checkpoints(account common.Address, pos uint32) (ERC20VotesCheckpoint, error) {
	return _GovernanceToken.Contract.Checkpoints(&_GovernanceToken.CallOpts, account, pos)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_GovernanceToken *GovernanceTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_GovernanceToken *GovernanceTokenSession) Decimals() (uint8, error) {
	return _GovernanceToken.Contract.Decimals(&_GovernanceToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_GovernanceToken *GovernanceTokenCallerSession) Decimals() (uint8, error) {
	return _GovernanceToken.Contract.Decimals(&_GovernanceToken.CallOpts)
}

// Delegates is a free data retrieval call binding the contract method 0x587cde1e.
//
// Solidity: function delegates(address account) view returns(address)
func (_GovernanceToken *GovernanceTokenCaller) Delegates(opts *bind.CallOpts, account common.Address) (common.Address, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "delegates", account)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Delegates is a free data retrieval call binding the contract method 0x587cde1e.
//
// Solidity: function delegates(address account) view returns(address)
func (_GovernanceToken *GovernanceTokenSession) Delegates(account common.Address) (common.Address, error) {
	return _GovernanceToken.Contract.Delegates(&_GovernanceToken.CallOpts, account)
}

// Delegates is a free data retrieval call binding the contract method 0x587cde1e.
//
// Solidity: function delegates(address account) view returns(address)
func (_GovernanceToken *GovernanceTokenCallerSession) Delegates(account common.Address) (common.Address, error) {
	return _GovernanceToken.Contract.Delegates(&_GovernanceToken.CallOpts, account)
}

// GetPastTotalSupply is a free data retrieval call binding the contract method 0x8e539e8c.
//
// Solidity: function getPastTotalSupply(uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) GetPastTotalSupply(opts *bind.CallOpts, blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "getPastTotalSupply", blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPastTotalSupply is a free data retrieval call binding the contract method 0x8e539e8c.
//
// Solidity: function getPastTotalSupply(uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) GetPastTotalSupply(blockNumber *big.Int) (*big.Int, error) {
	return _GovernanceToken.Contract.GetPastTotalSupply(&_GovernanceToken.CallOpts, blockNumber)
}

// GetPastTotalSupply is a free data retrieval call binding the contract method 0x8e539e8c.
//
// Solidity: function getPastTotalSupply(uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) GetPastTotalSupply(blockNumber *big.Int) (*big.Int, error) {
	return _GovernanceToken.Contract.GetPastTotalSupply(&_GovernanceToken.CallOpts, blockNumber)
}

// GetPastVotes is a free data retrieval call binding the contract method 0x3a46b1a8.
//
// Solidity: function getPastVotes(address account, uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) GetPastVotes(opts *bind.CallOpts, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "getPastVotes", account, blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetPastVotes is a free data retrieval call binding the contract method 0x3a46b1a8.
//
// Solidity: function getPastVotes(address account, uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) GetPastVotes(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return _GovernanceToken.Contract.GetPastVotes(&_GovernanceToken.CallOpts, account, blockNumber)
}

// GetPastVotes is a free data retrieval call binding the contract method 0x3a46b1a8.
//
// Solidity: function getPastVotes(address account, uint256 blockNumber) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) GetPastVotes(account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return _GovernanceToken.Contract.GetPastVotes(&_GovernanceToken.CallOpts, account, blockNumber)
}

// GetVotes is a free data retrieval call binding the contract method 0x9ab24eb0.
//
// Solidity: function getVotes(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) GetVotes(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "getVotes", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetVotes is a free data retrieval call binding the contract method 0x9ab24eb0.
//
// Solidity: function getVotes(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) GetVotes(account common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.GetVotes(&_GovernanceToken.CallOpts, account)
}

// GetVotes is a free data retrieval call binding the contract method 0x9ab24eb0.
//
// Solidity: function getVotes(address account) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) GetVotes(account common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.GetVotes(&_GovernanceToken.CallOpts, account)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_GovernanceToken *GovernanceTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_GovernanceToken *GovernanceTokenSession) Name() (string, error) {
	return _GovernanceToken.Contract.Name(&_GovernanceToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_GovernanceToken *GovernanceTokenCallerSession) Name() (string, error) {
	return _GovernanceToken.Contract.Name(&_GovernanceToken.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) Nonces(opts *bind.CallOpts, owner common.Address) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "nonces", owner)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) Nonces(owner common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.Nonces(&_GovernanceToken.CallOpts, owner)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address owner) view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) Nonces(owner common.Address) (*big.Int, error) {
	return _GovernanceToken.Contract.Nonces(&_GovernanceToken.CallOpts, owner)
}

// NumCheckpoints is a free data retrieval call binding the contract method 0x6fcfff45.
//
// Solidity: function numCheckpoints(address account) view returns(uint32)
func (_GovernanceToken *GovernanceTokenCaller) NumCheckpoints(opts *bind.CallOpts, account common.Address) (uint32, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "numCheckpoints", account)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// NumCheckpoints is a free data retrieval call binding the contract method 0x6fcfff45.
//
// Solidity: function numCheckpoints(address account) view returns(uint32)
func (_GovernanceToken *GovernanceTokenSession) NumCheckpoints(account common.Address) (uint32, error) {
	return _GovernanceToken.Contract.NumCheckpoints(&_GovernanceToken.CallOpts, account)
}

// NumCheckpoints is a free data retrieval call binding the contract method 0x6fcfff45.
//
// Solidity: function numCheckpoints(address account) view returns(uint32)
func (_GovernanceToken *GovernanceTokenCallerSession) NumCheckpoints(account common.Address) (uint32, error) {
	return _GovernanceToken.Contract.NumCheckpoints(&_GovernanceToken.CallOpts, account)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GovernanceToken *GovernanceTokenCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GovernanceToken *GovernanceTokenSession) Owner() (common.Address, error) {
	return _GovernanceToken.Contract.Owner(&_GovernanceToken.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_GovernanceToken *GovernanceTokenCallerSession) Owner() (common.Address, error) {
	return _GovernanceToken.Contract.Owner(&_GovernanceToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_GovernanceToken *GovernanceTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_GovernanceToken *GovernanceTokenSession) Symbol() (string, error) {
	return _GovernanceToken.Contract.Symbol(&_GovernanceToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_GovernanceToken *GovernanceTokenCallerSession) Symbol() (string, error) {
	return _GovernanceToken.Contract.Symbol(&_GovernanceToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_GovernanceToken *GovernanceTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GovernanceToken.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_GovernanceToken *GovernanceTokenSession) TotalSupply() (*big.Int, error) {
	return _GovernanceToken.Contract.TotalSupply(&_GovernanceToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_GovernanceToken *GovernanceTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _GovernanceToken.Contract.TotalSupply(&_GovernanceToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Approve(&_GovernanceToken.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Approve(&_GovernanceToken.TransactOpts, spender, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenTransactor) Burn(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "burn", amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Burn(&_GovernanceToken.TransactOpts, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Burn(&_GovernanceToken.TransactOpts, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenTransactor) BurnFrom(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "burnFrom", account, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenSession) BurnFrom(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.BurnFrom(&_GovernanceToken.TransactOpts, account, amount)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 amount) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) BurnFrom(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.BurnFrom(&_GovernanceToken.TransactOpts, account, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.DecreaseAllowance(&_GovernanceToken.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.DecreaseAllowance(&_GovernanceToken.TransactOpts, spender, subtractedValue)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address delegatee) returns()
func (_GovernanceToken *GovernanceTokenTransactor) Delegate(opts *bind.TransactOpts, delegatee common.Address) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "delegate", delegatee)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address delegatee) returns()
func (_GovernanceToken *GovernanceTokenSession) Delegate(delegatee common.Address) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Delegate(&_GovernanceToken.TransactOpts, delegatee)
}

// Delegate is a paid mutator transaction binding the contract method 0x5c19a95c.
//
// Solidity: function delegate(address delegatee) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) Delegate(delegatee common.Address) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Delegate(&_GovernanceToken.TransactOpts, delegatee)
}

// DelegateBySig is a paid mutator transaction binding the contract method 0xc3cda520.
//
// Solidity: function delegateBySig(address delegatee, uint256 nonce, uint256 expiry, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenTransactor) DelegateBySig(opts *bind.TransactOpts, delegatee common.Address, nonce *big.Int, expiry *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "delegateBySig", delegatee, nonce, expiry, v, r, s)
}

// DelegateBySig is a paid mutator transaction binding the contract method 0xc3cda520.
//
// Solidity: function delegateBySig(address delegatee, uint256 nonce, uint256 expiry, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenSession) DelegateBySig(delegatee common.Address, nonce *big.Int, expiry *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.Contract.DelegateBySig(&_GovernanceToken.TransactOpts, delegatee, nonce, expiry, v, r, s)
}

// DelegateBySig is a paid mutator transaction binding the contract method 0xc3cda520.
//
// Solidity: function delegateBySig(address delegatee, uint256 nonce, uint256 expiry, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) DelegateBySig(delegatee common.Address, nonce *big.Int, expiry *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.Contract.DelegateBySig(&_GovernanceToken.TransactOpts, delegatee, nonce, expiry, v, r, s)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.IncreaseAllowance(&_GovernanceToken.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.IncreaseAllowance(&_GovernanceToken.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _account, uint256 _amount) returns()
func (_GovernanceToken *GovernanceTokenTransactor) Mint(opts *bind.TransactOpts, _account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "mint", _account, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _account, uint256 _amount) returns()
func (_GovernanceToken *GovernanceTokenSession) Mint(_account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Mint(&_GovernanceToken.TransactOpts, _account, _amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address _account, uint256 _amount) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) Mint(_account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Mint(&_GovernanceToken.TransactOpts, _account, _amount)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenTransactor) Permit(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "permit", owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Permit(&_GovernanceToken.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Permit(&_GovernanceToken.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GovernanceToken *GovernanceTokenTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GovernanceToken *GovernanceTokenSession) RenounceOwnership() (*types.Transaction, error) {
	return _GovernanceToken.Contract.RenounceOwnership(&_GovernanceToken.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _GovernanceToken.Contract.RenounceOwnership(&_GovernanceToken.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Transfer(&_GovernanceToken.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.Transfer(&_GovernanceToken.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.TransferFrom(&_GovernanceToken.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_GovernanceToken *GovernanceTokenTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _GovernanceToken.Contract.TransferFrom(&_GovernanceToken.TransactOpts, from, to, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GovernanceToken *GovernanceTokenTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _GovernanceToken.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GovernanceToken *GovernanceTokenSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GovernanceToken.Contract.TransferOwnership(&_GovernanceToken.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_GovernanceToken *GovernanceTokenTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _GovernanceToken.Contract.TransferOwnership(&_GovernanceToken.TransactOpts, newOwner)
}

// GovernanceTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the GovernanceToken contract.
type GovernanceTokenApprovalIterator struct {
	Event *GovernanceTokenApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GovernanceTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GovernanceTokenApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GovernanceTokenApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GovernanceTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GovernanceTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GovernanceTokenApproval represents a Approval event raised by the GovernanceToken contract.
type GovernanceTokenApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*GovernanceTokenApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _GovernanceToken.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenApprovalIterator{contract: _GovernanceToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *GovernanceTokenApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _GovernanceToken.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GovernanceTokenApproval)
				if err := _GovernanceToken.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) ParseApproval(log types.Log) (*GovernanceTokenApproval, error) {
	event := new(GovernanceTokenApproval)
	if err := _GovernanceToken.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GovernanceTokenDelegateChangedIterator is returned from FilterDelegateChanged and is used to iterate over the raw logs and unpacked data for DelegateChanged events raised by the GovernanceToken contract.
type GovernanceTokenDelegateChangedIterator struct {
	Event *GovernanceTokenDelegateChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GovernanceTokenDelegateChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GovernanceTokenDelegateChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GovernanceTokenDelegateChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GovernanceTokenDelegateChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GovernanceTokenDelegateChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GovernanceTokenDelegateChanged represents a DelegateChanged event raised by the GovernanceToken contract.
type GovernanceTokenDelegateChanged struct {
	Delegator    common.Address
	FromDelegate common.Address
	ToDelegate   common.Address
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDelegateChanged is a free log retrieval operation binding the contract event 0x3134e8a2e6d97e929a7e54011ea5485d7d196dd5f0ba4d4ef95803e8e3fc257f.
//
// Solidity: event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate)
func (_GovernanceToken *GovernanceTokenFilterer) FilterDelegateChanged(opts *bind.FilterOpts, delegator []common.Address, fromDelegate []common.Address, toDelegate []common.Address) (*GovernanceTokenDelegateChangedIterator, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var fromDelegateRule []interface{}
	for _, fromDelegateItem := range fromDelegate {
		fromDelegateRule = append(fromDelegateRule, fromDelegateItem)
	}
	var toDelegateRule []interface{}
	for _, toDelegateItem := range toDelegate {
		toDelegateRule = append(toDelegateRule, toDelegateItem)
	}

	logs, sub, err := _GovernanceToken.contract.FilterLogs(opts, "DelegateChanged", delegatorRule, fromDelegateRule, toDelegateRule)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenDelegateChangedIterator{contract: _GovernanceToken.contract, event: "DelegateChanged", logs: logs, sub: sub}, nil
}

// WatchDelegateChanged is a free log subscription operation binding the contract event 0x3134e8a2e6d97e929a7e54011ea5485d7d196dd5f0ba4d4ef95803e8e3fc257f.
//
// Solidity: event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate)
func (_GovernanceToken *GovernanceTokenFilterer) WatchDelegateChanged(opts *bind.WatchOpts, sink chan<- *GovernanceTokenDelegateChanged, delegator []common.Address, fromDelegate []common.Address, toDelegate []common.Address) (event.Subscription, error) {

	var delegatorRule []interface{}
	for _, delegatorItem := range delegator {
		delegatorRule = append(delegatorRule, delegatorItem)
	}
	var fromDelegateRule []interface{}
	for _, fromDelegateItem := range fromDelegate {
		fromDelegateRule = append(fromDelegateRule, fromDelegateItem)
	}
	var toDelegateRule []interface{}
	for _, toDelegateItem := range toDelegate {
		toDelegateRule = append(toDelegateRule, toDelegateItem)
	}

	logs, sub, err := _GovernanceToken.contract.WatchLogs(opts, "DelegateChanged", delegatorRule, fromDelegateRule, toDelegateRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GovernanceTokenDelegateChanged)
				if err := _GovernanceToken.contract.UnpackLog(event, "DelegateChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDelegateChanged is a log parse operation binding the contract event 0x3134e8a2e6d97e929a7e54011ea5485d7d196dd5f0ba4d4ef95803e8e3fc257f.
//
// Solidity: event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate)
func (_GovernanceToken *GovernanceTokenFilterer) ParseDelegateChanged(log types.Log) (*GovernanceTokenDelegateChanged, error) {
	event := new(GovernanceTokenDelegateChanged)
	if err := _GovernanceToken.contract.UnpackLog(event, "DelegateChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GovernanceTokenDelegateVotesChangedIterator is returned from FilterDelegateVotesChanged and is used to iterate over the raw logs and unpacked data for DelegateVotesChanged events raised by the GovernanceToken contract.
type GovernanceTokenDelegateVotesChangedIterator struct {
	Event *GovernanceTokenDelegateVotesChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GovernanceTokenDelegateVotesChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GovernanceTokenDelegateVotesChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GovernanceTokenDelegateVotesChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GovernanceTokenDelegateVotesChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GovernanceTokenDelegateVotesChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GovernanceTokenDelegateVotesChanged represents a DelegateVotesChanged event raised by the GovernanceToken contract.
type GovernanceTokenDelegateVotesChanged struct {
	Delegate        common.Address
	PreviousBalance *big.Int
	NewBalance      *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterDelegateVotesChanged is a free log retrieval operation binding the contract event 0xdec2bacdd2f05b59de34da9b523dff8be42e5e38e818c82fdb0bae774387a724.
//
// Solidity: event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance)
func (_GovernanceToken *GovernanceTokenFilterer) FilterDelegateVotesChanged(opts *bind.FilterOpts, delegate []common.Address) (*GovernanceTokenDelegateVotesChangedIterator, error) {

	var delegateRule []interface{}
	for _, delegateItem := range delegate {
		delegateRule = append(delegateRule, delegateItem)
	}

	logs, sub, err := _GovernanceToken.contract.FilterLogs(opts, "DelegateVotesChanged", delegateRule)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenDelegateVotesChangedIterator{contract: _GovernanceToken.contract, event: "DelegateVotesChanged", logs: logs, sub: sub}, nil
}

// WatchDelegateVotesChanged is a free log subscription operation binding the contract event 0xdec2bacdd2f05b59de34da9b523dff8be42e5e38e818c82fdb0bae774387a724.
//
// Solidity: event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance)
func (_GovernanceToken *GovernanceTokenFilterer) WatchDelegateVotesChanged(opts *bind.WatchOpts, sink chan<- *GovernanceTokenDelegateVotesChanged, delegate []common.Address) (event.Subscription, error) {

	var delegateRule []interface{}
	for _, delegateItem := range delegate {
		delegateRule = append(delegateRule, delegateItem)
	}

	logs, sub, err := _GovernanceToken.contract.WatchLogs(opts, "DelegateVotesChanged", delegateRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GovernanceTokenDelegateVotesChanged)
				if err := _GovernanceToken.contract.UnpackLog(event, "DelegateVotesChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDelegateVotesChanged is a log parse operation binding the contract event 0xdec2bacdd2f05b59de34da9b523dff8be42e5e38e818c82fdb0bae774387a724.
//
// Solidity: event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance)
func (_GovernanceToken *GovernanceTokenFilterer) ParseDelegateVotesChanged(log types.Log) (*GovernanceTokenDelegateVotesChanged, error) {
	event := new(GovernanceTokenDelegateVotesChanged)
	if err := _GovernanceToken.contract.UnpackLog(event, "DelegateVotesChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GovernanceTokenOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the GovernanceToken contract.
type GovernanceTokenOwnershipTransferredIterator struct {
	Event *GovernanceTokenOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GovernanceTokenOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GovernanceTokenOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GovernanceTokenOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GovernanceTokenOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GovernanceTokenOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GovernanceTokenOwnershipTransferred represents a OwnershipTransferred event raised by the GovernanceToken contract.
type GovernanceTokenOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GovernanceToken *GovernanceTokenFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*GovernanceTokenOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GovernanceToken.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenOwnershipTransferredIterator{contract: _GovernanceToken.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GovernanceToken *GovernanceTokenFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *GovernanceTokenOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _GovernanceToken.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GovernanceTokenOwnershipTransferred)
				if err := _GovernanceToken.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_GovernanceToken *GovernanceTokenFilterer) ParseOwnershipTransferred(log types.Log) (*GovernanceTokenOwnershipTransferred, error) {
	event := new(GovernanceTokenOwnershipTransferred)
	if err := _GovernanceToken.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// GovernanceTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the GovernanceToken contract.
type GovernanceTokenTransferIterator struct {
	Event *GovernanceTokenTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *GovernanceTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(GovernanceTokenTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(GovernanceTokenTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *GovernanceTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *GovernanceTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// GovernanceTokenTransfer represents a Transfer event raised by the GovernanceToken contract.
type GovernanceTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*GovernanceTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _GovernanceToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &GovernanceTokenTransferIterator{contract: _GovernanceToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *GovernanceTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _GovernanceToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(GovernanceTokenTransfer)
				if err := _GovernanceToken.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_GovernanceToken *GovernanceTokenFilterer) ParseTransfer(log types.Log) (*GovernanceTokenTransfer, error) {
	event := new(GovernanceTokenTransfer)
	if err := _GovernanceToken.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

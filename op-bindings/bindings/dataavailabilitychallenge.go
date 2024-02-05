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

// DataAvailabilityChallengeMetaData contains all meta data concerning the DataAvailabilityChallenge contract.
var DataAvailabilityChallengeMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"balances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bondSize\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"challenge\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"challengeWindow\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"challenges\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"lockedBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startBlock\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"resolvedBlock\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"fixedResolutionCost\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getChallengeStatus\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumChallengeStatus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_challengeWindow\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_resolveWindow\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_bondSize\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_resolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolve\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"preImage\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolveWindow\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"resolverRefundPercentage\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setBondSize\",\"inputs\":[{\"name\":\"_bondSize\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setResolverRefundPercentage\",\"inputs\":[{\"name\":\"_resolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unlockBond\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"variableResolutionCost\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"BalanceChanged\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"balance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ChallengeStatusChanged\",\"inputs\":[{\"name\":\"challengedHash\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"status\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumChallengeStatus\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RequiredBondSizeChanged\",\"inputs\":[{\"name\":\"challengeWindow\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ResolverRefundPercentageChanged\",\"inputs\":[{\"name\":\"resolverRefundPercentage\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BondTooLow\",\"inputs\":[{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"required\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ChallengeExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeNotActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeNotExpired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeWindowNotOpen\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidInputData\",\"inputs\":[{\"name\":\"providedDataHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"expectedHash\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidResolverRefundPercentage\",\"inputs\":[{\"name\":\"invalidResolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"WithdrawalFailed\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b5061151d806100206000396000f3fe60806040526004361061016e5760003560e01c8063861a1412116100cb578063c459f8091161007f578063d7d04e5411610059578063d7d04e541461043d578063f2fde38b1461045d578063f92ad2191461047d57600080fd5b8063c459f8091461037a578063c4ee20d41461039a578063d0e30db01461043557600080fd5b80638ecb85e1116100b05780638ecb85e1146103215780639561185214610337578063b740a2db1461034d57600080fd5b8063861a1412146102d65780638da5cb5b146102ec57600080fd5b8063336409fd1161012257806354fd4d501161010757806354fd4d50146102555780637099c581146102ab578063715018a6146102c157600080fd5b8063336409fd146102205780633ccfd60b1461024057600080fd5b806321cf39ee1161015357806321cf39ee146101b557806323c30f59146101de57806327e235e3146101f357600080fd5b806302b2f7c7146101825780630b1a73f41461019557600080fd5b3661017d5761017b61049d565b005b600080fd5b61017b6101903660046111ff565b61050b565b3480156101a157600080fd5b5061017b6101b0366004611221565b610713565b3480156101c157600080fd5b506101cb60665481565b6040519081526020015b60405180910390f35b3480156101ea57600080fd5b506101cb601481565b3480156101ff57600080fd5b506101cb61020e3660046112ca565b60696020526000908152604090205481565b34801561022c57600080fd5b5061017b61023b3660046112ec565b610844565b34801561024c57600080fd5b5061017b61088f565b34801561026157600080fd5b5061029e6040518060400160405280600581526020017f302e302e3000000000000000000000000000000000000000000000000000000081525081565b6040516101d59190611305565b3480156102b757600080fd5b506101cb60675481565b3480156102cd57600080fd5b5061017b6108ed565b3480156102e257600080fd5b506101cb60655481565b3480156102f857600080fd5b5060335460405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101d5565b34801561032d57600080fd5b506101cb60685481565b34801561034357600080fd5b506101cb61aca881565b34801561035957600080fd5b5061036d6103683660046111ff565b610901565b6040516101d591906113a7565b34801561038657600080fd5b5061017b6103953660046111ff565b6109a9565b3480156103a657600080fd5b506103fe6103b53660046111ff565b606a602090815260009283526040808420909152908252902080546001820154600283015460039093015473ffffffffffffffffffffffffffffffffffffffff90921692909184565b6040805173ffffffffffffffffffffffffffffffffffffffff909516855260208501939093529183015260608201526080016101d5565b61017b61049d565b34801561044957600080fd5b5061017b6104583660046112ec565b610abe565b34801561046957600080fd5b5061017b6104783660046112ca565b610b01565b34801561048957600080fd5b5061017b6104983660046113e8565b610bb8565b33600090815260696020526040812080543492906104bc908490611459565b909155505033600081815260696020908152604091829020548251938452908301527fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a1565b61051361049d565b6067543360009081526069602052604090205410156105845733600090815260696020526040908190205460675491517e0155b500000000000000000000000000000000000000000000000000000000815261057b9290600401918252602082015260400190565b60405180910390fd5b60006105908383610901565b60038111156105a1576105a1611378565b146105d8576040517f9bb6c64e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6105e182610d74565b610617576040517ff9e0d1f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6067543360009081526069602052604081208054909190610639908490611471565b909155505060408051608081018252338152606754602080830191825243838501908152600060608501818152888252606a8452868220888352909352859020935184547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff909116178455915160018085019190915591516002840155516003909201919091559051839183917f73b78891d84bab8633915b22168a5ed8a2f0b86fbaf9733698fbacea9a2b11f891610707916113a7565b60405180910390a35050565b8181604051610723929190611488565b60405180910390208314610781578181604051610741929190611488565b6040519081900381207f3b7d737200000000000000000000000000000000000000000000000000000000825260048201526024810184905260440161057b565b600161078d8585610901565b600381111561079e5761079e611378565b146107d5576040517fbeb11d3b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000848152606a60209081526040808320868452909152908190204360038201559051859085907f73b78891d84bab8633915b22168a5ed8a2f0b86fbaf9733698fbacea9a2b11f89061082a906002906113a7565b60405180910390a361083d818333610d96565b5050505050565b61084c610fa7565b606481111561088a576040517f16aa4e800000000000000000000000000000000000000000000000000000000081526004810182905260240161057b565b606855565b336000818152606960205260408120805490829055916108b0905a84611028565b9050806108e9576040517f27fcd9d100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050565b6108f5610fa7565b6108ff600061103e565b565b6000828152606a6020908152604080832084845282528083208151608081018352815473ffffffffffffffffffffffffffffffffffffffff168082526001830154948201949094526002820154928101929092526003015460608201529061096d5760009150506109a3565b6060810151156109815760029150506109a3565b61098e81604001516110b5565b1561099d5760019150506109a3565b60039150505b92915050565b60036109b58383610901565b60038111156109c6576109c6611378565b146109fd576040517f151f07fe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000828152606a6020908152604080832084845282528083206001810154815473ffffffffffffffffffffffffffffffffffffffff1685526069909352908320805491939091610a4e908490611459565b9091555050600060018201819055815473ffffffffffffffffffffffffffffffffffffffff1680825260696020908152604092839020548351928352908201527fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a1505050565b610ac6610fa7565b60678190556040518181527f4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae9060200160405180910390a150565b610b09610fa7565b73ffffffffffffffffffffffffffffffffffffffff8116610bac576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161057b565b610bb58161103e565b50565b600054610100900460ff1615808015610bd85750600054600160ff909116105b80610bf25750303b158015610bf2575060005460ff166001145b610c7e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161057b565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610cdc57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610ce46110c5565b60658590556066849055610cf783610abe565b610d0082610844565b610d098661103e565b8015610d6c57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050505050565b600081431180156109a35750606554610d8d9083611459565b43111592915050565b6001830154835473ffffffffffffffffffffffffffffffffffffffff1660003a610dc1601487611498565b610dcd9061aca8611459565b610dd79190611498565b905080831115610e8957610deb8184611471565b73ffffffffffffffffffffffffffffffffffffffff831660009081526069602052604081208054909190610e20908490611459565b909155505073ffffffffffffffffffffffffffffffffffffffff82166000818152606960209081526040918290205482519384529083015291935083917fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a15b6000606460685483610e9b9190611498565b610ea591906114d5565b905083811115610eb25750825b8015610f5f5773ffffffffffffffffffffffffffffffffffffffff851660009081526069602052604081208054839290610eed908490611459565b90915550610efd90508185611471565b73ffffffffffffffffffffffffffffffffffffffff8616600081815260696020908152604091829020548251938452908301529195507fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a15b8315610f945760405160009085156108fc0290869083818181858288f19350505050158015610f92573d6000803e3d6000fd5b505b6000876001018190555050505050505050565b60335473ffffffffffffffffffffffffffffffffffffffff1633146108ff576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161057b565b600080600080600080868989f195945050505050565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600060665482610d8d9190611459565b600054610100900460ff1661115c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161057b565b6108ff600054610100900460ff166111f6576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161057b565b6108ff3361103e565b6000806040838503121561121257600080fd5b50508035926020909101359150565b6000806000806060858703121561123757600080fd5b8435935060208501359250604085013567ffffffffffffffff8082111561125d57600080fd5b818701915087601f83011261127157600080fd5b81358181111561128057600080fd5b88602082850101111561129257600080fd5b95989497505060200194505050565b803573ffffffffffffffffffffffffffffffffffffffff811681146112c557600080fd5b919050565b6000602082840312156112dc57600080fd5b6112e5826112a1565b9392505050565b6000602082840312156112fe57600080fd5b5035919050565b600060208083528351808285015260005b8181101561133257858101830151858201604001528201611316565b81811115611344576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b60208101600483106113e2577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b600080600080600060a0868803121561140057600080fd5b611409866112a1565b97602087013597506040870135966060810135965060800135945092505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000821982111561146c5761146c61142a565b500190565b6000828210156114835761148361142a565b500390565b8183823760009101908152919050565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff04831182151516156114d0576114d061142a565b500290565b60008261150b577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b50049056fea164736f6c634300080f000a",
}

// DataAvailabilityChallengeABI is the input ABI used to generate the binding from.
// Deprecated: Use DataAvailabilityChallengeMetaData.ABI instead.
var DataAvailabilityChallengeABI = DataAvailabilityChallengeMetaData.ABI

// DataAvailabilityChallengeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DataAvailabilityChallengeMetaData.Bin instead.
var DataAvailabilityChallengeBin = DataAvailabilityChallengeMetaData.Bin

// DeployDataAvailabilityChallenge deploys a new Ethereum contract, binding an instance of DataAvailabilityChallenge to it.
func DeployDataAvailabilityChallenge(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DataAvailabilityChallenge, error) {
	parsed, err := DataAvailabilityChallengeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DataAvailabilityChallengeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DataAvailabilityChallenge{DataAvailabilityChallengeCaller: DataAvailabilityChallengeCaller{contract: contract}, DataAvailabilityChallengeTransactor: DataAvailabilityChallengeTransactor{contract: contract}, DataAvailabilityChallengeFilterer: DataAvailabilityChallengeFilterer{contract: contract}}, nil
}

// DataAvailabilityChallenge is an auto generated Go binding around an Ethereum contract.
type DataAvailabilityChallenge struct {
	DataAvailabilityChallengeCaller     // Read-only binding to the contract
	DataAvailabilityChallengeTransactor // Write-only binding to the contract
	DataAvailabilityChallengeFilterer   // Log filterer for contract events
}

// DataAvailabilityChallengeCaller is an auto generated read-only Go binding around an Ethereum contract.
type DataAvailabilityChallengeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataAvailabilityChallengeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DataAvailabilityChallengeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataAvailabilityChallengeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DataAvailabilityChallengeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DataAvailabilityChallengeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DataAvailabilityChallengeSession struct {
	Contract     *DataAvailabilityChallenge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts              // Call options to use throughout this session
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DataAvailabilityChallengeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DataAvailabilityChallengeCallerSession struct {
	Contract *DataAvailabilityChallengeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                    // Call options to use throughout this session
}

// DataAvailabilityChallengeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DataAvailabilityChallengeTransactorSession struct {
	Contract     *DataAvailabilityChallengeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                    // Transaction auth options to use throughout this session
}

// DataAvailabilityChallengeRaw is an auto generated low-level Go binding around an Ethereum contract.
type DataAvailabilityChallengeRaw struct {
	Contract *DataAvailabilityChallenge // Generic contract binding to access the raw methods on
}

// DataAvailabilityChallengeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DataAvailabilityChallengeCallerRaw struct {
	Contract *DataAvailabilityChallengeCaller // Generic read-only contract binding to access the raw methods on
}

// DataAvailabilityChallengeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DataAvailabilityChallengeTransactorRaw struct {
	Contract *DataAvailabilityChallengeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDataAvailabilityChallenge creates a new instance of DataAvailabilityChallenge, bound to a specific deployed contract.
func NewDataAvailabilityChallenge(address common.Address, backend bind.ContractBackend) (*DataAvailabilityChallenge, error) {
	contract, err := bindDataAvailabilityChallenge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallenge{DataAvailabilityChallengeCaller: DataAvailabilityChallengeCaller{contract: contract}, DataAvailabilityChallengeTransactor: DataAvailabilityChallengeTransactor{contract: contract}, DataAvailabilityChallengeFilterer: DataAvailabilityChallengeFilterer{contract: contract}}, nil
}

// NewDataAvailabilityChallengeCaller creates a new read-only instance of DataAvailabilityChallenge, bound to a specific deployed contract.
func NewDataAvailabilityChallengeCaller(address common.Address, caller bind.ContractCaller) (*DataAvailabilityChallengeCaller, error) {
	contract, err := bindDataAvailabilityChallenge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeCaller{contract: contract}, nil
}

// NewDataAvailabilityChallengeTransactor creates a new write-only instance of DataAvailabilityChallenge, bound to a specific deployed contract.
func NewDataAvailabilityChallengeTransactor(address common.Address, transactor bind.ContractTransactor) (*DataAvailabilityChallengeTransactor, error) {
	contract, err := bindDataAvailabilityChallenge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeTransactor{contract: contract}, nil
}

// NewDataAvailabilityChallengeFilterer creates a new log filterer instance of DataAvailabilityChallenge, bound to a specific deployed contract.
func NewDataAvailabilityChallengeFilterer(address common.Address, filterer bind.ContractFilterer) (*DataAvailabilityChallengeFilterer, error) {
	contract, err := bindDataAvailabilityChallenge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeFilterer{contract: contract}, nil
}

// bindDataAvailabilityChallenge binds a generic wrapper to an already deployed contract.
func bindDataAvailabilityChallenge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DataAvailabilityChallengeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DataAvailabilityChallenge.Contract.DataAvailabilityChallengeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.DataAvailabilityChallengeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.DataAvailabilityChallengeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DataAvailabilityChallenge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.contract.Transact(opts, method, params...)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.Balances(&_DataAvailabilityChallenge.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.Balances(&_DataAvailabilityChallenge.CallOpts, arg0)
}

// BondSize is a free data retrieval call binding the contract method 0x7099c581.
//
// Solidity: function bondSize() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) BondSize(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "bondSize")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BondSize is a free data retrieval call binding the contract method 0x7099c581.
//
// Solidity: function bondSize() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) BondSize() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.BondSize(&_DataAvailabilityChallenge.CallOpts)
}

// BondSize is a free data retrieval call binding the contract method 0x7099c581.
//
// Solidity: function bondSize() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) BondSize() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.BondSize(&_DataAvailabilityChallenge.CallOpts)
}

// ChallengeWindow is a free data retrieval call binding the contract method 0x861a1412.
//
// Solidity: function challengeWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) ChallengeWindow(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "challengeWindow")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengeWindow is a free data retrieval call binding the contract method 0x861a1412.
//
// Solidity: function challengeWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) ChallengeWindow() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ChallengeWindow(&_DataAvailabilityChallenge.CallOpts)
}

// ChallengeWindow is a free data retrieval call binding the contract method 0x861a1412.
//
// Solidity: function challengeWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) ChallengeWindow() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ChallengeWindow(&_DataAvailabilityChallenge.CallOpts)
}

// Challenges is a free data retrieval call binding the contract method 0xc4ee20d4.
//
// Solidity: function challenges(uint256 , bytes32 ) view returns(address challenger, uint256 lockedBond, uint256 startBlock, uint256 resolvedBlock)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) Challenges(opts *bind.CallOpts, arg0 *big.Int, arg1 [32]byte) (struct {
	Challenger    common.Address
	LockedBond    *big.Int
	StartBlock    *big.Int
	ResolvedBlock *big.Int
}, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "challenges", arg0, arg1)

	outstruct := new(struct {
		Challenger    common.Address
		LockedBond    *big.Int
		StartBlock    *big.Int
		ResolvedBlock *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Challenger = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.LockedBond = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.StartBlock = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ResolvedBlock = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Challenges is a free data retrieval call binding the contract method 0xc4ee20d4.
//
// Solidity: function challenges(uint256 , bytes32 ) view returns(address challenger, uint256 lockedBond, uint256 startBlock, uint256 resolvedBlock)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Challenges(arg0 *big.Int, arg1 [32]byte) (struct {
	Challenger    common.Address
	LockedBond    *big.Int
	StartBlock    *big.Int
	ResolvedBlock *big.Int
}, error) {
	return _DataAvailabilityChallenge.Contract.Challenges(&_DataAvailabilityChallenge.CallOpts, arg0, arg1)
}

// Challenges is a free data retrieval call binding the contract method 0xc4ee20d4.
//
// Solidity: function challenges(uint256 , bytes32 ) view returns(address challenger, uint256 lockedBond, uint256 startBlock, uint256 resolvedBlock)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) Challenges(arg0 *big.Int, arg1 [32]byte) (struct {
	Challenger    common.Address
	LockedBond    *big.Int
	StartBlock    *big.Int
	ResolvedBlock *big.Int
}, error) {
	return _DataAvailabilityChallenge.Contract.Challenges(&_DataAvailabilityChallenge.CallOpts, arg0, arg1)
}

// FixedResolutionCost is a free data retrieval call binding the contract method 0x95611852.
//
// Solidity: function fixedResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) FixedResolutionCost(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "fixedResolutionCost")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FixedResolutionCost is a free data retrieval call binding the contract method 0x95611852.
//
// Solidity: function fixedResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) FixedResolutionCost() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.FixedResolutionCost(&_DataAvailabilityChallenge.CallOpts)
}

// FixedResolutionCost is a free data retrieval call binding the contract method 0x95611852.
//
// Solidity: function fixedResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) FixedResolutionCost() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.FixedResolutionCost(&_DataAvailabilityChallenge.CallOpts)
}

// GetChallengeStatus is a free data retrieval call binding the contract method 0xb740a2db.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes32 challengedHash) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) GetChallengeStatus(opts *bind.CallOpts, challengedBlockNumber *big.Int, challengedHash [32]byte) (uint8, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "getChallengeStatus", challengedBlockNumber, challengedHash)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetChallengeStatus is a free data retrieval call binding the contract method 0xb740a2db.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes32 challengedHash) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) GetChallengeStatus(challengedBlockNumber *big.Int, challengedHash [32]byte) (uint8, error) {
	return _DataAvailabilityChallenge.Contract.GetChallengeStatus(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedHash)
}

// GetChallengeStatus is a free data retrieval call binding the contract method 0xb740a2db.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes32 challengedHash) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) GetChallengeStatus(challengedBlockNumber *big.Int, challengedHash [32]byte) (uint8, error) {
	return _DataAvailabilityChallenge.Contract.GetChallengeStatus(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedHash)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Owner() (common.Address, error) {
	return _DataAvailabilityChallenge.Contract.Owner(&_DataAvailabilityChallenge.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) Owner() (common.Address, error) {
	return _DataAvailabilityChallenge.Contract.Owner(&_DataAvailabilityChallenge.CallOpts)
}

// ResolveWindow is a free data retrieval call binding the contract method 0x21cf39ee.
//
// Solidity: function resolveWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) ResolveWindow(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "resolveWindow")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ResolveWindow is a free data retrieval call binding the contract method 0x21cf39ee.
//
// Solidity: function resolveWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) ResolveWindow() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ResolveWindow(&_DataAvailabilityChallenge.CallOpts)
}

// ResolveWindow is a free data retrieval call binding the contract method 0x21cf39ee.
//
// Solidity: function resolveWindow() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) ResolveWindow() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ResolveWindow(&_DataAvailabilityChallenge.CallOpts)
}

// ResolverRefundPercentage is a free data retrieval call binding the contract method 0x8ecb85e1.
//
// Solidity: function resolverRefundPercentage() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) ResolverRefundPercentage(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "resolverRefundPercentage")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ResolverRefundPercentage is a free data retrieval call binding the contract method 0x8ecb85e1.
//
// Solidity: function resolverRefundPercentage() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) ResolverRefundPercentage() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ResolverRefundPercentage(&_DataAvailabilityChallenge.CallOpts)
}

// ResolverRefundPercentage is a free data retrieval call binding the contract method 0x8ecb85e1.
//
// Solidity: function resolverRefundPercentage() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) ResolverRefundPercentage() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.ResolverRefundPercentage(&_DataAvailabilityChallenge.CallOpts)
}

// VariableResolutionCost is a free data retrieval call binding the contract method 0x23c30f59.
//
// Solidity: function variableResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) VariableResolutionCost(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "variableResolutionCost")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VariableResolutionCost is a free data retrieval call binding the contract method 0x23c30f59.
//
// Solidity: function variableResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) VariableResolutionCost() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.VariableResolutionCost(&_DataAvailabilityChallenge.CallOpts)
}

// VariableResolutionCost is a free data retrieval call binding the contract method 0x23c30f59.
//
// Solidity: function variableResolutionCost() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) VariableResolutionCost() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.VariableResolutionCost(&_DataAvailabilityChallenge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Version() (string, error) {
	return _DataAvailabilityChallenge.Contract.Version(&_DataAvailabilityChallenge.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) Version() (string, error) {
	return _DataAvailabilityChallenge.Contract.Version(&_DataAvailabilityChallenge.CallOpts)
}

// Challenge is a paid mutator transaction binding the contract method 0x02b2f7c7.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes32 challengedHash) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Challenge(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "challenge", challengedBlockNumber, challengedHash)
}

// Challenge is a paid mutator transaction binding the contract method 0x02b2f7c7.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes32 challengedHash) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Challenge(challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Challenge(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash)
}

// Challenge is a paid mutator transaction binding the contract method 0x02b2f7c7.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes32 challengedHash) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Challenge(challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Challenge(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Deposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "deposit")
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Deposit() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Deposit(&_DataAvailabilityChallenge.TransactOpts)
}

// Deposit is a paid mutator transaction binding the contract method 0xd0e30db0.
//
// Solidity: function deposit() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Deposit() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Deposit(&_DataAvailabilityChallenge.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0xf92ad219.
//
// Solidity: function initialize(address _owner, uint256 _challengeWindow, uint256 _resolveWindow, uint256 _bondSize, uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address, _challengeWindow *big.Int, _resolveWindow *big.Int, _bondSize *big.Int, _resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "initialize", _owner, _challengeWindow, _resolveWindow, _bondSize, _resolverRefundPercentage)
}

// Initialize is a paid mutator transaction binding the contract method 0xf92ad219.
//
// Solidity: function initialize(address _owner, uint256 _challengeWindow, uint256 _resolveWindow, uint256 _bondSize, uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Initialize(_owner common.Address, _challengeWindow *big.Int, _resolveWindow *big.Int, _bondSize *big.Int, _resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Initialize(&_DataAvailabilityChallenge.TransactOpts, _owner, _challengeWindow, _resolveWindow, _bondSize, _resolverRefundPercentage)
}

// Initialize is a paid mutator transaction binding the contract method 0xf92ad219.
//
// Solidity: function initialize(address _owner, uint256 _challengeWindow, uint256 _resolveWindow, uint256 _bondSize, uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Initialize(_owner common.Address, _challengeWindow *big.Int, _resolveWindow *big.Int, _bondSize *big.Int, _resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Initialize(&_DataAvailabilityChallenge.TransactOpts, _owner, _challengeWindow, _resolveWindow, _bondSize, _resolverRefundPercentage)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) RenounceOwnership() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.RenounceOwnership(&_DataAvailabilityChallenge.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.RenounceOwnership(&_DataAvailabilityChallenge.TransactOpts)
}

// Resolve is a paid mutator transaction binding the contract method 0x0b1a73f4.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes32 challengedHash, bytes preImage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Resolve(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedHash [32]byte, preImage []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "resolve", challengedBlockNumber, challengedHash, preImage)
}

// Resolve is a paid mutator transaction binding the contract method 0x0b1a73f4.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes32 challengedHash, bytes preImage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Resolve(challengedBlockNumber *big.Int, challengedHash [32]byte, preImage []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Resolve(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash, preImage)
}

// Resolve is a paid mutator transaction binding the contract method 0x0b1a73f4.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes32 challengedHash, bytes preImage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Resolve(challengedBlockNumber *big.Int, challengedHash [32]byte, preImage []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Resolve(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash, preImage)
}

// SetBondSize is a paid mutator transaction binding the contract method 0xd7d04e54.
//
// Solidity: function setBondSize(uint256 _bondSize) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) SetBondSize(opts *bind.TransactOpts, _bondSize *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "setBondSize", _bondSize)
}

// SetBondSize is a paid mutator transaction binding the contract method 0xd7d04e54.
//
// Solidity: function setBondSize(uint256 _bondSize) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) SetBondSize(_bondSize *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.SetBondSize(&_DataAvailabilityChallenge.TransactOpts, _bondSize)
}

// SetBondSize is a paid mutator transaction binding the contract method 0xd7d04e54.
//
// Solidity: function setBondSize(uint256 _bondSize) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) SetBondSize(_bondSize *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.SetBondSize(&_DataAvailabilityChallenge.TransactOpts, _bondSize)
}

// SetResolverRefundPercentage is a paid mutator transaction binding the contract method 0x336409fd.
//
// Solidity: function setResolverRefundPercentage(uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) SetResolverRefundPercentage(opts *bind.TransactOpts, _resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "setResolverRefundPercentage", _resolverRefundPercentage)
}

// SetResolverRefundPercentage is a paid mutator transaction binding the contract method 0x336409fd.
//
// Solidity: function setResolverRefundPercentage(uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) SetResolverRefundPercentage(_resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.SetResolverRefundPercentage(&_DataAvailabilityChallenge.TransactOpts, _resolverRefundPercentage)
}

// SetResolverRefundPercentage is a paid mutator transaction binding the contract method 0x336409fd.
//
// Solidity: function setResolverRefundPercentage(uint256 _resolverRefundPercentage) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) SetResolverRefundPercentage(_resolverRefundPercentage *big.Int) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.SetResolverRefundPercentage(&_DataAvailabilityChallenge.TransactOpts, _resolverRefundPercentage)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.TransferOwnership(&_DataAvailabilityChallenge.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.TransferOwnership(&_DataAvailabilityChallenge.TransactOpts, newOwner)
}

// UnlockBond is a paid mutator transaction binding the contract method 0xc459f809.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes32 challengedHash) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) UnlockBond(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "unlockBond", challengedBlockNumber, challengedHash)
}

// UnlockBond is a paid mutator transaction binding the contract method 0xc459f809.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes32 challengedHash) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) UnlockBond(challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.UnlockBond(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash)
}

// UnlockBond is a paid mutator transaction binding the contract method 0xc459f809.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes32 challengedHash) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) UnlockBond(challengedBlockNumber *big.Int, challengedHash [32]byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.UnlockBond(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedHash)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Withdraw() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Withdraw(&_DataAvailabilityChallenge.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Withdraw() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Withdraw(&_DataAvailabilityChallenge.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Receive() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Receive(&_DataAvailabilityChallenge.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Receive() (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Receive(&_DataAvailabilityChallenge.TransactOpts)
}

// DataAvailabilityChallengeBalanceChangedIterator is returned from FilterBalanceChanged and is used to iterate over the raw logs and unpacked data for BalanceChanged events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeBalanceChangedIterator struct {
	Event *DataAvailabilityChallengeBalanceChanged // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeBalanceChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeBalanceChanged)
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
		it.Event = new(DataAvailabilityChallengeBalanceChanged)
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
func (it *DataAvailabilityChallengeBalanceChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeBalanceChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeBalanceChanged represents a BalanceChanged event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeBalanceChanged struct {
	Account common.Address
	Balance *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterBalanceChanged is a free log retrieval operation binding the contract event 0xa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05.
//
// Solidity: event BalanceChanged(address account, uint256 balance)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterBalanceChanged(opts *bind.FilterOpts) (*DataAvailabilityChallengeBalanceChangedIterator, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "BalanceChanged")
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeBalanceChangedIterator{contract: _DataAvailabilityChallenge.contract, event: "BalanceChanged", logs: logs, sub: sub}, nil
}

// WatchBalanceChanged is a free log subscription operation binding the contract event 0xa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05.
//
// Solidity: event BalanceChanged(address account, uint256 balance)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchBalanceChanged(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeBalanceChanged) (event.Subscription, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "BalanceChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeBalanceChanged)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "BalanceChanged", log); err != nil {
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

// ParseBalanceChanged is a log parse operation binding the contract event 0xa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05.
//
// Solidity: event BalanceChanged(address account, uint256 balance)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseBalanceChanged(log types.Log) (*DataAvailabilityChallengeBalanceChanged, error) {
	event := new(DataAvailabilityChallengeBalanceChanged)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "BalanceChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DataAvailabilityChallengeChallengeStatusChangedIterator is returned from FilterChallengeStatusChanged and is used to iterate over the raw logs and unpacked data for ChallengeStatusChanged events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeChallengeStatusChangedIterator struct {
	Event *DataAvailabilityChallengeChallengeStatusChanged // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeChallengeStatusChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeChallengeStatusChanged)
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
		it.Event = new(DataAvailabilityChallengeChallengeStatusChanged)
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
func (it *DataAvailabilityChallengeChallengeStatusChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeChallengeStatusChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeChallengeStatusChanged represents a ChallengeStatusChanged event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeChallengeStatusChanged struct {
	ChallengedHash        [32]byte
	ChallengedBlockNumber *big.Int
	Status                uint8
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterChallengeStatusChanged is a free log retrieval operation binding the contract event 0x73b78891d84bab8633915b22168a5ed8a2f0b86fbaf9733698fbacea9a2b11f8.
//
// Solidity: event ChallengeStatusChanged(bytes32 indexed challengedHash, uint256 indexed challengedBlockNumber, uint8 status)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterChallengeStatusChanged(opts *bind.FilterOpts, challengedHash [][32]byte, challengedBlockNumber []*big.Int) (*DataAvailabilityChallengeChallengeStatusChangedIterator, error) {

	var challengedHashRule []interface{}
	for _, challengedHashItem := range challengedHash {
		challengedHashRule = append(challengedHashRule, challengedHashItem)
	}
	var challengedBlockNumberRule []interface{}
	for _, challengedBlockNumberItem := range challengedBlockNumber {
		challengedBlockNumberRule = append(challengedBlockNumberRule, challengedBlockNumberItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "ChallengeStatusChanged", challengedHashRule, challengedBlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeChallengeStatusChangedIterator{contract: _DataAvailabilityChallenge.contract, event: "ChallengeStatusChanged", logs: logs, sub: sub}, nil
}

// WatchChallengeStatusChanged is a free log subscription operation binding the contract event 0x73b78891d84bab8633915b22168a5ed8a2f0b86fbaf9733698fbacea9a2b11f8.
//
// Solidity: event ChallengeStatusChanged(bytes32 indexed challengedHash, uint256 indexed challengedBlockNumber, uint8 status)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchChallengeStatusChanged(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeChallengeStatusChanged, challengedHash [][32]byte, challengedBlockNumber []*big.Int) (event.Subscription, error) {

	var challengedHashRule []interface{}
	for _, challengedHashItem := range challengedHash {
		challengedHashRule = append(challengedHashRule, challengedHashItem)
	}
	var challengedBlockNumberRule []interface{}
	for _, challengedBlockNumberItem := range challengedBlockNumber {
		challengedBlockNumberRule = append(challengedBlockNumberRule, challengedBlockNumberItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "ChallengeStatusChanged", challengedHashRule, challengedBlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeChallengeStatusChanged)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "ChallengeStatusChanged", log); err != nil {
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

// ParseChallengeStatusChanged is a log parse operation binding the contract event 0x73b78891d84bab8633915b22168a5ed8a2f0b86fbaf9733698fbacea9a2b11f8.
//
// Solidity: event ChallengeStatusChanged(bytes32 indexed challengedHash, uint256 indexed challengedBlockNumber, uint8 status)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseChallengeStatusChanged(log types.Log) (*DataAvailabilityChallengeChallengeStatusChanged, error) {
	event := new(DataAvailabilityChallengeChallengeStatusChanged)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "ChallengeStatusChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DataAvailabilityChallengeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeInitializedIterator struct {
	Event *DataAvailabilityChallengeInitialized // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeInitialized)
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
		it.Event = new(DataAvailabilityChallengeInitialized)
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
func (it *DataAvailabilityChallengeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeInitialized represents a Initialized event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterInitialized(opts *bind.FilterOpts) (*DataAvailabilityChallengeInitializedIterator, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeInitializedIterator{contract: _DataAvailabilityChallenge.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeInitialized) (event.Subscription, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeInitialized)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseInitialized(log types.Log) (*DataAvailabilityChallengeInitialized, error) {
	event := new(DataAvailabilityChallengeInitialized)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DataAvailabilityChallengeOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeOwnershipTransferredIterator struct {
	Event *DataAvailabilityChallengeOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeOwnershipTransferred)
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
		it.Event = new(DataAvailabilityChallengeOwnershipTransferred)
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
func (it *DataAvailabilityChallengeOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeOwnershipTransferred represents a OwnershipTransferred event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DataAvailabilityChallengeOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeOwnershipTransferredIterator{contract: _DataAvailabilityChallenge.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeOwnershipTransferred)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseOwnershipTransferred(log types.Log) (*DataAvailabilityChallengeOwnershipTransferred, error) {
	event := new(DataAvailabilityChallengeOwnershipTransferred)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DataAvailabilityChallengeRequiredBondSizeChangedIterator is returned from FilterRequiredBondSizeChanged and is used to iterate over the raw logs and unpacked data for RequiredBondSizeChanged events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeRequiredBondSizeChangedIterator struct {
	Event *DataAvailabilityChallengeRequiredBondSizeChanged // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeRequiredBondSizeChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeRequiredBondSizeChanged)
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
		it.Event = new(DataAvailabilityChallengeRequiredBondSizeChanged)
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
func (it *DataAvailabilityChallengeRequiredBondSizeChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeRequiredBondSizeChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeRequiredBondSizeChanged represents a RequiredBondSizeChanged event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeRequiredBondSizeChanged struct {
	ChallengeWindow *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterRequiredBondSizeChanged is a free log retrieval operation binding the contract event 0x4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae.
//
// Solidity: event RequiredBondSizeChanged(uint256 challengeWindow)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterRequiredBondSizeChanged(opts *bind.FilterOpts) (*DataAvailabilityChallengeRequiredBondSizeChangedIterator, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "RequiredBondSizeChanged")
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeRequiredBondSizeChangedIterator{contract: _DataAvailabilityChallenge.contract, event: "RequiredBondSizeChanged", logs: logs, sub: sub}, nil
}

// WatchRequiredBondSizeChanged is a free log subscription operation binding the contract event 0x4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae.
//
// Solidity: event RequiredBondSizeChanged(uint256 challengeWindow)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchRequiredBondSizeChanged(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeRequiredBondSizeChanged) (event.Subscription, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "RequiredBondSizeChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeRequiredBondSizeChanged)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "RequiredBondSizeChanged", log); err != nil {
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

// ParseRequiredBondSizeChanged is a log parse operation binding the contract event 0x4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae.
//
// Solidity: event RequiredBondSizeChanged(uint256 challengeWindow)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseRequiredBondSizeChanged(log types.Log) (*DataAvailabilityChallengeRequiredBondSizeChanged, error) {
	event := new(DataAvailabilityChallengeRequiredBondSizeChanged)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "RequiredBondSizeChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DataAvailabilityChallengeResolverRefundPercentageChangedIterator is returned from FilterResolverRefundPercentageChanged and is used to iterate over the raw logs and unpacked data for ResolverRefundPercentageChanged events raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeResolverRefundPercentageChangedIterator struct {
	Event *DataAvailabilityChallengeResolverRefundPercentageChanged // Event containing the contract specifics and raw log

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
func (it *DataAvailabilityChallengeResolverRefundPercentageChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DataAvailabilityChallengeResolverRefundPercentageChanged)
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
		it.Event = new(DataAvailabilityChallengeResolverRefundPercentageChanged)
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
func (it *DataAvailabilityChallengeResolverRefundPercentageChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DataAvailabilityChallengeResolverRefundPercentageChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DataAvailabilityChallengeResolverRefundPercentageChanged represents a ResolverRefundPercentageChanged event raised by the DataAvailabilityChallenge contract.
type DataAvailabilityChallengeResolverRefundPercentageChanged struct {
	ResolverRefundPercentage *big.Int
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterResolverRefundPercentageChanged is a free log retrieval operation binding the contract event 0xbbd8605de8f773fedb355c2fecd4b6b2e16a10a44e6676a63b375ac8f693758d.
//
// Solidity: event ResolverRefundPercentageChanged(uint256 resolverRefundPercentage)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterResolverRefundPercentageChanged(opts *bind.FilterOpts) (*DataAvailabilityChallengeResolverRefundPercentageChangedIterator, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "ResolverRefundPercentageChanged")
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeResolverRefundPercentageChangedIterator{contract: _DataAvailabilityChallenge.contract, event: "ResolverRefundPercentageChanged", logs: logs, sub: sub}, nil
}

// WatchResolverRefundPercentageChanged is a free log subscription operation binding the contract event 0xbbd8605de8f773fedb355c2fecd4b6b2e16a10a44e6676a63b375ac8f693758d.
//
// Solidity: event ResolverRefundPercentageChanged(uint256 resolverRefundPercentage)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchResolverRefundPercentageChanged(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeResolverRefundPercentageChanged) (event.Subscription, error) {

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "ResolverRefundPercentageChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DataAvailabilityChallengeResolverRefundPercentageChanged)
				if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "ResolverRefundPercentageChanged", log); err != nil {
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

// ParseResolverRefundPercentageChanged is a log parse operation binding the contract event 0xbbd8605de8f773fedb355c2fecd4b6b2e16a10a44e6676a63b375ac8f693758d.
//
// Solidity: event ResolverRefundPercentageChanged(uint256 resolverRefundPercentage)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) ParseResolverRefundPercentageChanged(log types.Log) (*DataAvailabilityChallengeResolverRefundPercentageChanged, error) {
	event := new(DataAvailabilityChallengeResolverRefundPercentageChanged)
	if err := _DataAvailabilityChallenge.contract.UnpackLog(event, "ResolverRefundPercentageChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

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

// Challenge is an auto generated low-level Go binding around an user-defined struct.
type Challenge struct {
	Challenger    common.Address
	LockedBond    *big.Int
	StartBlock    *big.Int
	ResolvedBlock *big.Int
}

// DataAvailabilityChallengeMetaData contains all meta data concerning the DataAvailabilityChallenge contract.
var DataAvailabilityChallengeMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"balances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bondSize\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"challenge\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"challengeWindow\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"deposit\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"fixedResolutionCost\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getChallenge\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structChallenge\",\"components\":[{\"name\":\"challenger\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"lockedBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startBlock\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"resolvedBlock\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getChallengeStatus\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumChallengeStatus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_challengeWindow\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_resolveWindow\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_bondSize\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_resolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolve\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"resolveData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolveWindow\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"resolverRefundPercentage\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setBondSize\",\"inputs\":[{\"name\":\"_bondSize\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setResolverRefundPercentage\",\"inputs\":[{\"name\":\"_resolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unlockBond\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validateCommitment\",\"inputs\":[{\"name\":\"commitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"variableResolutionCost\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"variableResolutionCostPrecision\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdraw\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"BalanceChanged\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"balance\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ChallengeStatusChanged\",\"inputs\":[{\"name\":\"challengedBlockNumber\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"challengedCommitment\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"},{\"name\":\"status\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"enumChallengeStatus\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RequiredBondSizeChanged\",\"inputs\":[{\"name\":\"challengeWindow\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ResolverRefundPercentageChanged\",\"inputs\":[{\"name\":\"resolverRefundPercentage\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"BondTooLow\",\"inputs\":[{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"required\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ChallengeExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeNotActive\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeNotExpired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ChallengeWindowNotOpen\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidCommitmentLength\",\"inputs\":[{\"name\":\"commitmentType\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"expectedLength\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"actualLength\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"InvalidInputData\",\"inputs\":[{\"name\":\"providedDataCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"expectedCommitment\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"type\":\"error\",\"name\":\"InvalidResolverRefundPercentage\",\"inputs\":[{\"name\":\"invalidResolverRefundPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"UnknownCommitmentType\",\"inputs\":[{\"name\":\"commitmentType\",\"type\":\"uint8\",\"internalType\":\"uint8\"}]},{\"type\":\"error\",\"name\":\"WithdrawalFailed\",\"inputs\":[]}]",
	Bin: "0x60806040523480156200001157600080fd5b506200002461dead60008080806200002a565b62000392565b600054610100900460ff16158080156200004b5750600054600160ff909116105b806200007b575062000068306200018c60201b6200100e1760201c565b1580156200007b575060005460ff166001145b620000e45760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b60648201526084015b60405180910390fd5b6000805460ff19166001179055801562000108576000805461ff0019166101001790555b620001126200019b565b60658590556066849055620001278362000203565b620001328262000248565b6200013d866200027d565b801562000184576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050505050565b6001600160a01b03163b151590565b600054610100900460ff16620001f75760405162461bcd60e51b815260206004820152602b602482015260008051602062001d7783398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000db565b62000201620002cf565b565b6200020d62000336565b60678190556040518181527f4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae9060200160405180910390a150565b6200025262000336565b60648111156200027857604051622d549d60e71b815260048101829052602401620000db565b606855565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff166200032b5760405162461bcd60e51b815260206004820152602b602482015260008051602062001d7783398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000db565b62000201336200027d565b6033546001600160a01b03163314620002015760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401620000db565b6119d580620003a26000396000f3fe6080604052600436106101845760003560e01c8063848afb3d116100d6578063956118521161007f578063d7d04e5411610059578063d7d04e5414610459578063f2fde38b14610479578063f92ad2191461049957600080fd5b80639561185214610427578063a03aafbf1461043e578063d0e30db01461045157600080fd5b80638ecb85e1116100b05780638ecb85e1146103d157806393988233146103e757806393fb19441461040757600080fd5b8063848afb3d1461031d578063861a1412146103865780638da5cb5b1461039c57600080fd5b80634ebaf3ce11610138578063715018a611610112578063715018a6146102bb57806379e8a8b3146102d05780637ae929d9146102fd57600080fd5b80634ebaf3ce1461023957806354fd4d501461024f5780637099c581146102a557600080fd5b806327e235e31161016957806327e235e3146101d7578063336409fd146102045780633ccfd60b1461022457600080fd5b806321cf39ee1461019857806323c30f59146101c157600080fd5b36610193576101916104b9565b005b600080fd5b3480156101a457600080fd5b506101ae60665481565b6040519081526020015b60405180910390f35b3480156101cd57600080fd5b506101ae61410081565b3480156101e357600080fd5b506101ae6101f2366004611539565b60696020526000908152604090205481565b34801561021057600080fd5b5061019161021f366004611554565b610527565b34801561023057600080fd5b50610191610577565b34801561024557600080fd5b506101ae6103e881565b34801561025b57600080fd5b506102986040518060400160405280600581526020017f312e302e3000000000000000000000000000000000000000000000000000000081525081565b6040516101b891906115d8565b3480156102b157600080fd5b506101ae60675481565b3480156102c757600080fd5b50610191610611565b3480156102dc57600080fd5b506102f06102eb366004611634565b610625565b6040516101b891906116ea565b34801561030957600080fd5b506101916103183660046116f8565b6106e8565b34801561032957600080fd5b5061033d610338366004611634565b610884565b6040516101b89190815173ffffffffffffffffffffffffffffffffffffffff16815260208083015190820152604080830151908201526060918201519181019190915260800190565b34801561039257600080fd5b506101ae60655481565b3480156103a857600080fd5b5060335460405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101b8565b3480156103dd57600080fd5b506101ae60685481565b3480156103f357600080fd5b50610191610402366004611634565b610940565b34801561041357600080fd5b50610191610422366004611772565b610a72565b34801561043357600080fd5b506101ae62011cdd81565b61019161044c366004611634565b610b0f565b6101916104b9565b34801561046557600080fd5b50610191610474366004611554565b610d58565b34801561048557600080fd5b50610191610494366004611539565b610d9b565b3480156104a557600080fd5b506101916104b43660046117b4565b610e52565b33600090815260696020526040812080543492906104d8908490611825565b909155505033600081815260696020908152604091829020548251938452908301527fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a1565b61052f61102a565b6064811115610572576040517f16aa4e80000000000000000000000000000000000000000000000000000000008152600481018290526024015b60405180910390fd5b606855565b336000818152606960209081526040808320805490849055815194855291840192909252917fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a160006105d4335a846110ab565b90508061060d576040517f27fcd9d100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5050565b61061961102a565b61062360006110c1565b565b6000838152606a60205260408082209051829190610646908690869061183d565b9081526040805160209281900383018120608082018352805473ffffffffffffffffffffffffffffffffffffffff16808352600182015494830194909452600281015492820192909252600390910154606082015291506106ab5760009150506106e1565b6060810151156106bf5760029150506106e1565b6106cc8160400151611138565b156106db5760019150506106e1565b60039150505b9392505050565b6106f28484610a72565b60016106ff868686610625565b600381111561071057610710611680565b14610747576040517fbeb11d3b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006107538585611151565b9050606060ff82166107a15761079e84848080601f01602080910402602001604051908101604052809392919081815260200183838082843760009201919091525061116992505050565b90505b85856040516107b192919061183d565b60405180910390208180519060200120146107fe578086866040517f1a0bbf9f00000000000000000000000000000000000000000000000000000000815260040161056993929190611896565b6000878152606a6020526040808220905161081c908990899061183d565b908152604051908190036020018120436003820155915088907fc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad90610867908a908a906002906118c6565b60405180910390a261087a81853361119c565b5050505050505050565b6108c56040518060800160405280600073ffffffffffffffffffffffffffffffffffffffff1681526020016000815260200160008152602001600081525090565b6000848152606a60205260409081902090516108e4908590859061183d565b908152604080519182900360209081018320608084018352805473ffffffffffffffffffffffffffffffffffffffff16845260018101549184019190915260028101549183019190915260030154606082015290509392505050565b600361094d848484610625565b600381111561095e5761095e611680565b14610995576040517f151f07fe00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000838152606a602052604080822090516109b3908590859061183d565b90815260408051602092819003830190206001810154815473ffffffffffffffffffffffffffffffffffffffff16600090815260699094529183208054919450919290610a01908490611825565b9091555050600060018201819055815473ffffffffffffffffffffffffffffffffffffffff1680825260696020908152604092839020548351928352908201527fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a150505050565b6000610a7e8383611151565b905060ff8116610ad85760218214610ad3576040517ffd9a7e5b000000000000000000000000000000000000000000000000000000008152600060048201526021602482015260448101839052606401610569565b505050565b6040517f81ff071300000000000000000000000000000000000000000000000000000000815260ff82166004820152602401610569565b610b198282610a72565b610b216104b9565b606754336000908152606960205260409020541015610b895733600090815260696020526040908190205460675491517e0155b50000000000000000000000000000000000000000000000000000000081526105699290600401918252602082015260400190565b6000610b96848484610625565b6003811115610ba757610ba7611680565b14610bde576040517f9bb6c64e00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b610be7836113bc565b610c1d576040517ff9e0d1f300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6067543360009081526069602052604081208054909190610c3f9084906118f1565b9250508190555060405180608001604052803373ffffffffffffffffffffffffffffffffffffffff16815260200160675481526020014381526020016000815250606a60008581526020019081526020016000208383604051610ca392919061183d565b9081526040805160209281900383018120845181547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff9091161781559284015160018085019190915591840151600284015560609093015160039092019190915584917fc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad91610d4b91869186916118c6565b60405180910390a2505050565b610d6061102a565b60678190556040518181527f4468d695a0389e5f9e8ef0c9aee6d84e74cc0d0e0a28c8413badb54697d1bbae9060200160405180910390a150565b610da361102a565b73ffffffffffffffffffffffffffffffffffffffff8116610e46576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f64647265737300000000000000000000000000000000000000000000000000006064820152608401610569565b610e4f816110c1565b50565b600054610100900460ff1615808015610e725750600054600160ff909116105b80610e8c5750303b158015610e8c575060005460ff166001145b610f18576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610569565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790558015610f7657600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b610f7e6113d6565b60658590556066849055610f9183610d58565b610f9a82610527565b610fa3866110c1565b801561100657600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050505050565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b60335473ffffffffffffffffffffffffffffffffffffffff163314610623576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610569565b600080600080600080868989f195945050505050565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6000606654826111489190611825565b43111592915050565b600061115d8284611908565b60f81c90505b92915050565b80516020918201206040805160009381019390935260218084019290925280518084039092018252604190920190915290565b6001830154835473ffffffffffffffffffffffffffffffffffffffff166000486103e86111cb61410088611950565b6111d5919061198d565b6111e29062011cdd611825565b6111ec9190611950565b90508083111561129e5761120081846118f1565b73ffffffffffffffffffffffffffffffffffffffff831660009081526069602052604081208054909190611235908490611825565b909155505073ffffffffffffffffffffffffffffffffffffffff82166000818152606960209081526040918290205482519384529083015291935083917fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a15b60006064606854836112b09190611950565b6112ba919061198d565b9050838111156112c75750825b80156113745773ffffffffffffffffffffffffffffffffffffffff851660009081526069602052604081208054839290611302908490611825565b90915550611312905081856118f1565b73ffffffffffffffffffffffffffffffffffffffff8616600081815260696020908152604091829020548251938452908301529195507fa448afda7ea1e3a7a10fcab0c29fe9a9dd85791503bf0171f281521551c7ec05910160405180910390a15b83156113a95760405160009085156108fc0290869083818181858288f193505050501580156113a7573d6000803e3d6000fd5b505b6000876001018190555050505050505050565b600081431015801561116357506065546111489083611825565b600054610100900460ff1661146d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610569565b610623600054610100900460ff16611507576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610569565b610623336110c1565b803573ffffffffffffffffffffffffffffffffffffffff8116811461153457600080fd5b919050565b60006020828403121561154b57600080fd5b6106e182611510565b60006020828403121561156657600080fd5b5035919050565b6000815180845260005b8181101561159357602081850181015186830182015201611577565b818111156115a5576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006106e1602083018461156d565b60008083601f8401126115fd57600080fd5b50813567ffffffffffffffff81111561161557600080fd5b60208301915083602082850101111561162d57600080fd5b9250929050565b60008060006040848603121561164957600080fd5b83359250602084013567ffffffffffffffff81111561166757600080fd5b611673868287016115eb565b9497909650939450505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b600481106116e6577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b9052565b6020810161116382846116af565b60008060008060006060868803121561171057600080fd5b85359450602086013567ffffffffffffffff8082111561172f57600080fd5b61173b89838a016115eb565b9096509450604088013591508082111561175457600080fd5b50611761888289016115eb565b969995985093965092949392505050565b6000806020838503121561178557600080fd5b823567ffffffffffffffff81111561179c57600080fd5b6117a8858286016115eb565b90969095509350505050565b600080600080600060a086880312156117cc57600080fd5b6117d586611510565b97602087013597506040870135966060810135965060800135945092505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115611838576118386117f6565b500190565b8183823760009101908152919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b6040815260006118a9604083018661156d565b82810360208401526118bc81858761184d565b9695505050505050565b6040815260006118da60408301858761184d565b90506118e960208301846116af565b949350505050565b600082821015611903576119036117f6565b500390565b7fff0000000000000000000000000000000000000000000000000000000000000081358181169160018510156119485780818660010360031b1b83161692505b505092915050565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611988576119886117f6565b500290565b6000826119c3577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b50049056fea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
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

// GetChallenge is a free data retrieval call binding the contract method 0x848afb3d.
//
// Solidity: function getChallenge(uint256 challengedBlockNumber, bytes challengedCommitment) view returns((address,uint256,uint256,uint256))
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) GetChallenge(opts *bind.CallOpts, challengedBlockNumber *big.Int, challengedCommitment []byte) (Challenge, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "getChallenge", challengedBlockNumber, challengedCommitment)

	if err != nil {
		return *new(Challenge), err
	}

	out0 := *abi.ConvertType(out[0], new(Challenge)).(*Challenge)

	return out0, err

}

// GetChallenge is a free data retrieval call binding the contract method 0x848afb3d.
//
// Solidity: function getChallenge(uint256 challengedBlockNumber, bytes challengedCommitment) view returns((address,uint256,uint256,uint256))
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) GetChallenge(challengedBlockNumber *big.Int, challengedCommitment []byte) (Challenge, error) {
	return _DataAvailabilityChallenge.Contract.GetChallenge(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedCommitment)
}

// GetChallenge is a free data retrieval call binding the contract method 0x848afb3d.
//
// Solidity: function getChallenge(uint256 challengedBlockNumber, bytes challengedCommitment) view returns((address,uint256,uint256,uint256))
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) GetChallenge(challengedBlockNumber *big.Int, challengedCommitment []byte) (Challenge, error) {
	return _DataAvailabilityChallenge.Contract.GetChallenge(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedCommitment)
}

// GetChallengeStatus is a free data retrieval call binding the contract method 0x79e8a8b3.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes challengedCommitment) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) GetChallengeStatus(opts *bind.CallOpts, challengedBlockNumber *big.Int, challengedCommitment []byte) (uint8, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "getChallengeStatus", challengedBlockNumber, challengedCommitment)

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetChallengeStatus is a free data retrieval call binding the contract method 0x79e8a8b3.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes challengedCommitment) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) GetChallengeStatus(challengedBlockNumber *big.Int, challengedCommitment []byte) (uint8, error) {
	return _DataAvailabilityChallenge.Contract.GetChallengeStatus(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedCommitment)
}

// GetChallengeStatus is a free data retrieval call binding the contract method 0x79e8a8b3.
//
// Solidity: function getChallengeStatus(uint256 challengedBlockNumber, bytes challengedCommitment) view returns(uint8)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) GetChallengeStatus(challengedBlockNumber *big.Int, challengedCommitment []byte) (uint8, error) {
	return _DataAvailabilityChallenge.Contract.GetChallengeStatus(&_DataAvailabilityChallenge.CallOpts, challengedBlockNumber, challengedCommitment)
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

// ValidateCommitment is a free data retrieval call binding the contract method 0x93fb1944.
//
// Solidity: function validateCommitment(bytes commitment) pure returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) ValidateCommitment(opts *bind.CallOpts, commitment []byte) error {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "validateCommitment", commitment)

	if err != nil {
		return err
	}

	return err

}

// ValidateCommitment is a free data retrieval call binding the contract method 0x93fb1944.
//
// Solidity: function validateCommitment(bytes commitment) pure returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) ValidateCommitment(commitment []byte) error {
	return _DataAvailabilityChallenge.Contract.ValidateCommitment(&_DataAvailabilityChallenge.CallOpts, commitment)
}

// ValidateCommitment is a free data retrieval call binding the contract method 0x93fb1944.
//
// Solidity: function validateCommitment(bytes commitment) pure returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) ValidateCommitment(commitment []byte) error {
	return _DataAvailabilityChallenge.Contract.ValidateCommitment(&_DataAvailabilityChallenge.CallOpts, commitment)
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

// VariableResolutionCostPrecision is a free data retrieval call binding the contract method 0x4ebaf3ce.
//
// Solidity: function variableResolutionCostPrecision() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCaller) VariableResolutionCostPrecision(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DataAvailabilityChallenge.contract.Call(opts, &out, "variableResolutionCostPrecision")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VariableResolutionCostPrecision is a free data retrieval call binding the contract method 0x4ebaf3ce.
//
// Solidity: function variableResolutionCostPrecision() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) VariableResolutionCostPrecision() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.VariableResolutionCostPrecision(&_DataAvailabilityChallenge.CallOpts)
}

// VariableResolutionCostPrecision is a free data retrieval call binding the contract method 0x4ebaf3ce.
//
// Solidity: function variableResolutionCostPrecision() view returns(uint256)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeCallerSession) VariableResolutionCostPrecision() (*big.Int, error) {
	return _DataAvailabilityChallenge.Contract.VariableResolutionCostPrecision(&_DataAvailabilityChallenge.CallOpts)
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

// Challenge is a paid mutator transaction binding the contract method 0xa03aafbf.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes challengedCommitment) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Challenge(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "challenge", challengedBlockNumber, challengedCommitment)
}

// Challenge is a paid mutator transaction binding the contract method 0xa03aafbf.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes challengedCommitment) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Challenge(challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Challenge(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment)
}

// Challenge is a paid mutator transaction binding the contract method 0xa03aafbf.
//
// Solidity: function challenge(uint256 challengedBlockNumber, bytes challengedCommitment) payable returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Challenge(challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Challenge(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment)
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

// Resolve is a paid mutator transaction binding the contract method 0x7ae929d9.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes challengedCommitment, bytes resolveData) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) Resolve(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedCommitment []byte, resolveData []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "resolve", challengedBlockNumber, challengedCommitment, resolveData)
}

// Resolve is a paid mutator transaction binding the contract method 0x7ae929d9.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes challengedCommitment, bytes resolveData) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) Resolve(challengedBlockNumber *big.Int, challengedCommitment []byte, resolveData []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Resolve(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment, resolveData)
}

// Resolve is a paid mutator transaction binding the contract method 0x7ae929d9.
//
// Solidity: function resolve(uint256 challengedBlockNumber, bytes challengedCommitment, bytes resolveData) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) Resolve(challengedBlockNumber *big.Int, challengedCommitment []byte, resolveData []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.Resolve(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment, resolveData)
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

// UnlockBond is a paid mutator transaction binding the contract method 0x93988233.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes challengedCommitment) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactor) UnlockBond(opts *bind.TransactOpts, challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.contract.Transact(opts, "unlockBond", challengedBlockNumber, challengedCommitment)
}

// UnlockBond is a paid mutator transaction binding the contract method 0x93988233.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes challengedCommitment) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeSession) UnlockBond(challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.UnlockBond(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment)
}

// UnlockBond is a paid mutator transaction binding the contract method 0x93988233.
//
// Solidity: function unlockBond(uint256 challengedBlockNumber, bytes challengedCommitment) returns()
func (_DataAvailabilityChallenge *DataAvailabilityChallengeTransactorSession) UnlockBond(challengedBlockNumber *big.Int, challengedCommitment []byte) (*types.Transaction, error) {
	return _DataAvailabilityChallenge.Contract.UnlockBond(&_DataAvailabilityChallenge.TransactOpts, challengedBlockNumber, challengedCommitment)
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
	ChallengedBlockNumber *big.Int
	ChallengedCommitment  []byte
	Status                uint8
	Raw                   types.Log // Blockchain specific contextual infos
}

// FilterChallengeStatusChanged is a free log retrieval operation binding the contract event 0xc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad.
//
// Solidity: event ChallengeStatusChanged(uint256 indexed challengedBlockNumber, bytes challengedCommitment, uint8 status)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) FilterChallengeStatusChanged(opts *bind.FilterOpts, challengedBlockNumber []*big.Int) (*DataAvailabilityChallengeChallengeStatusChangedIterator, error) {

	var challengedBlockNumberRule []interface{}
	for _, challengedBlockNumberItem := range challengedBlockNumber {
		challengedBlockNumberRule = append(challengedBlockNumberRule, challengedBlockNumberItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.FilterLogs(opts, "ChallengeStatusChanged", challengedBlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &DataAvailabilityChallengeChallengeStatusChangedIterator{contract: _DataAvailabilityChallenge.contract, event: "ChallengeStatusChanged", logs: logs, sub: sub}, nil
}

// WatchChallengeStatusChanged is a free log subscription operation binding the contract event 0xc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad.
//
// Solidity: event ChallengeStatusChanged(uint256 indexed challengedBlockNumber, bytes challengedCommitment, uint8 status)
func (_DataAvailabilityChallenge *DataAvailabilityChallengeFilterer) WatchChallengeStatusChanged(opts *bind.WatchOpts, sink chan<- *DataAvailabilityChallengeChallengeStatusChanged, challengedBlockNumber []*big.Int) (event.Subscription, error) {

	var challengedBlockNumberRule []interface{}
	for _, challengedBlockNumberItem := range challengedBlockNumber {
		challengedBlockNumberRule = append(challengedBlockNumberRule, challengedBlockNumberItem)
	}

	logs, sub, err := _DataAvailabilityChallenge.contract.WatchLogs(opts, "ChallengeStatusChanged", challengedBlockNumberRule)
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

// ParseChallengeStatusChanged is a log parse operation binding the contract event 0xc5d8c630ba2fdacb1db24c4599df78c7fb8cf97b5aecde34939597f6697bb1ad.
//
// Solidity: event ChallengeStatusChanged(uint256 indexed challengedBlockNumber, bytes challengedCommitment, uint8 status)
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

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
	_ = abi.ConvertType
)

// DisputeGameFactoryGameSearchResult is an auto generated low-level Go binding around an user-defined struct.
type DisputeGameFactoryGameSearchResult struct {
	Index     *big.Int
	Metadata  [32]byte
	Timestamp uint64
	RootClaim [32]byte
	ExtraData []byte
}

// DisputeGameFactoryMetaData contains all meta data concerning the DisputeGameFactory contract.
var DisputeGameFactoryMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"create\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"findLatestGames\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_start\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_n\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"games_\",\"type\":\"tuple[]\",\"internalType\":\"structDisputeGameFactory.GameSearchResult[]\",\"components\":[{\"name\":\"index\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"metadata\",\"type\":\"bytes32\",\"internalType\":\"GameId\"},{\"name\":\"timestamp\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"},{\"name\":\"rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameArgs\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"GameType\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameAtIndex\",\"inputs\":[{\"name\":\"_index\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"gameType_\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"timestamp_\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"},{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameCount\",\"inputs\":[],\"outputs\":[{\"name\":\"gameCount_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameImpls\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"GameType\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"games\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"proxy_\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"},{\"name\":\"timestamp_\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getGameUUID\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_extraData\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"uuid_\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"initBonds\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"GameType\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"proxyAdmin\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIProxyAdmin\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"proxyAdminOwner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setImplementation\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_impl\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setImplementation\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_impl\",\"type\":\"address\",\"internalType\":\"contractIDisputeGame\"},{\"name\":\"_args\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInitBond\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"},{\"name\":\"_initBond\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"DisputeGameCreated\",\"inputs\":[{\"name\":\"disputeProxy\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"gameType\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"GameType\"},{\"name\":\"rootClaim\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"Claim\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ImplementationArgsSet\",\"inputs\":[{\"name\":\"gameType\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"GameType\"},{\"name\":\"args\",\"type\":\"bytes\",\"indexed\":false,\"internalType\":\"bytes\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ImplementationSet\",\"inputs\":[{\"name\":\"impl\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"gameType\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"GameType\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"InitBondUpdated\",\"inputs\":[{\"name\":\"gameType\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"GameType\"},{\"name\":\"newBond\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"GameAlreadyExists\",\"inputs\":[{\"name\":\"uuid\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}]},{\"type\":\"error\",\"name\":\"IncorrectBondAmount\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NoImplementation\",\"inputs\":[{\"name\":\"gameType\",\"type\":\"uint32\",\"internalType\":\"GameType\"}]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_NotProxyAdmin\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_NotProxyAdminOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_NotResolvedDelegateProxy\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_NotSharedProxyAdminOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ProxyAdminOwnedBase_ProxyAdminNotFound\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ReinitializableBase_ZeroInitVersion\",\"inputs\":[]}]",
	Bin: "0x60a060405234801561001057600080fd5b50600160805261001e610023565b6100e3565b600054610100900460ff161561008f5760405162461bcd60e51b815260206004820152602760248201527f496e697469616c697a61626c653a20636f6e747261637420697320696e697469604482015266616c697a696e6760c81b606482015260840160405180910390fd5b60005460ff90811610156100e1576000805460ff191660ff9081179091556040519081527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b565b608051611f79610105600039600081816102540152610fc80152611f796000f3fe60806040526004361061015f5760003560e01c8063715018a6116100c0578063b107095711610074578063c4d66de811610059578063c4d66de81461049e578063dad544e0146104be578063f2fde38b146104d357600080fd5b8063b10709571461041d578063bb8aa1fc1461043d57600080fd5b806382ecf2f6116100a557806382ecf2f6146103bf5780638da5cb5b146103d257806396cd9720146103fd57600080fd5b8063715018a61461038a57806374cc86ac1461039f57600080fd5b80633e47158c1161011757806354fd4d50116100fc57806354fd4d50146102b25780635f0150cb146103085780636593dc6e1461035d57600080fd5b80633e47158c1461027e5780634d1975b41461029357600080fd5b80631e334240116101485780631e334240146101f3578063254bd6831461021357806338d38c971461024057600080fd5b806314f6b1a3146101645780631b685b9e14610186575b600080fd5b34801561017057600080fd5b5061018461017f3660046116b9565b6104f3565b005b34801561019257600080fd5b506101c96101a13660046116f0565b60656020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b3480156101ff57600080fd5b5061018461020e36600461170b565b61057d565b34801561021f57600080fd5b5061023361022e366004611735565b6105c9565b6040516101ea91906117e2565b34801561024c57600080fd5b5060405160ff7f00000000000000000000000000000000000000000000000000000000000000001681526020016101ea565b34801561028a57600080fd5b506101c961080d565b34801561029f57600080fd5b506068545b6040519081526020016101ea565b3480156102be57600080fd5b506102fb6040518060400160405280600581526020017f312e342e3000000000000000000000000000000000000000000000000000000081525081565b6040516101ea919061189f565b34801561031457600080fd5b506103286103233660046118fb565b610a18565b6040805173ffffffffffffffffffffffffffffffffffffffff909316835267ffffffffffffffff9091166020830152016101ea565b34801561036957600080fd5b506102a46103783660046116f0565b60666020526000908152604090205481565b34801561039657600080fd5b50610184610a6b565b3480156103ab57600080fd5b506102fb6103ba3660046116f0565b610a7f565b6101c96103cd3660046118fb565b610b19565b3480156103de57600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff166101c9565b34801561040957600080fd5b506102a46104183660046118fb565b610e24565b34801561042957600080fd5b50610184610438366004611955565b610e5d565b34801561044957600080fd5b5061045d6104583660046119a0565b610f5d565b6040805163ffffffff909416845267ffffffffffffffff909216602084015273ffffffffffffffffffffffffffffffffffffffff16908201526060016101ea565b3480156104aa57600080fd5b506101846104b93660046119b9565b610fc6565b3480156104ca57600080fd5b506101c961113e565b3480156104df57600080fd5b506101846104ee3660046119b9565b6111bb565b6104fb611272565b63ffffffff821660008181526065602052604080822080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff8616908117909155905190917fff513d80e2c7fa487608f70a618dfbc0cf415699dc69588c747e8c71566c88de91a35050565b610585611272565b63ffffffff8216600081815260666020526040808220849055518392917f74d6665c4b26d5596a5aa13d3014e0c06af4d322075a797f87b03cd4c5bc91ca91a35050565b606854606090831015806105db575081155b610806575060408051600583901b8101602001909152825b8381116108045760006068828154811061060f5761060f6119d6565b600091825260209091200154905060e081901c60a082901c67ffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff831663ffffffff891683036107d5576001865101865260008173ffffffffffffffffffffffffffffffffffffffff1663609d33346040518163ffffffff1660e01b8152600401600060405180830381865afa1580156106a9573d6000803e3d6000fd5b505050506040513d6000823e601f3d9081017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe01682016040526106ef9190810190611a34565b905060008273ffffffffffffffffffffffffffffffffffffffff1663bcef3b556040518163ffffffff1660e01b8152600401602060405180830381865afa15801561073e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107629190611aff565b90506040518060a001604052808881526020018781526020018567ffffffffffffffff168152602001828152602001838152508860018a516107a49190611b47565b815181106107b4576107b46119d6565b6020026020010181905250888851106107d257505050505050610804565b50505b50507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff90920191506105f39050565b505b9392505050565b6000806108387fb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d61035490565b905073ffffffffffffffffffffffffffffffffffffffff81161561085b57919050565b6040518060400160405280601a81526020017f4f564d5f4c3143726f7373446f6d61696e4d657373656e67657200000000000081525051600261089e9190611b5e565b604080513060208201526000918101919091527f4f564d5f4c3143726f7373446f6d61696e4d657373656e67657200000000000091909117906108f9906060015b604051602081830303815290604052805190602001205490565b14610930576040517f54e433cd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60408051306020820152600191810191909152600090610952906060016108df565b905073ffffffffffffffffffffffffffffffffffffffff8116156109e6578073ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156109bb573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109df9190611b9b565b9250505090565b6040517f332144db00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000806000610a2987878787610e24565b60009081526067602052604090205473ffffffffffffffffffffffffffffffffffffffff81169860a09190911c67ffffffffffffffff16975095505050505050565b610a73611272565b610a7d60006112f3565b565b60696020526000908152604090208054610a9890611bb8565b80601f0160208091040260200160405190810160405280929190818152602001828054610ac490611bb8565b8015610b115780601f10610ae657610100808354040283529160200191610b11565b820191906000526020600020905b815481529060010190602001808311610af457829003601f168201915b505050505081565b63ffffffff841660009081526065602052604081205473ffffffffffffffffffffffffffffffffffffffff1680610b89576040517f031c6de400000000000000000000000000000000000000000000000000000000815263ffffffff871660048201526024015b60405180910390fd5b63ffffffff86166000908152606660205260409020543414610bd7576040517f8620aa1900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000610be4600143611b47565b63ffffffff8816600090815260696020526040902080549140925090610c0990611bb8565b9050600003610c8057610c793387838888604051602001610c2e959493929190611c0b565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905273ffffffffffffffffffffffffffffffffffffffff84169061136a565b9250610cc1565b610cbe3387838a8989606960008f63ffffffff1663ffffffff168152602001908152602001600020604051602001610c2e9796959493929190611c58565b92505b8273ffffffffffffffffffffffffffffffffffffffff16638129fc1c346040518263ffffffff1660e01b81526004016000604051808303818588803b158015610d0957600080fd5b505af1158015610d1d573d6000803e3d6000fd5b50505050506000610d3088888888610e24565b60008181526067602052604090205490915015610d7c576040517f014f6fe500000000000000000000000000000000000000000000000000000000815260048101829052602401610b80565b60004260a01b60e08a901b178517600083815260676020526040808220839055606880546001810182559083527fa2153420d844928b4421650203c77babc8b33d7f2e7b450e2966db0c220977530183905551919250899163ffffffff8c169173ffffffffffffffffffffffffffffffffffffffff8916917f5b565efe82411da98814f356d0e7bcb8f0219b8d970307c5afb4a6903a8b2e359190a450505050949350505050565b600084848484604051602001610e3d9493929190611db6565b604051602081830303815290604052805190602001209050949350505050565b610e65611272565b63ffffffff8416600090815260656020908152604080832080547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff881617905560699091529020610ecd828483611e35565b5060405163ffffffff85169073ffffffffffffffffffffffffffffffffffffffff8516907fff513d80e2c7fa487608f70a618dfbc0cf415699dc69588c747e8c71566c88de90600090a38363ffffffff167fa47fcdf075d680d3817bfca7973b373e1a5f6cfc3b444748299cc2b83d8348f98383604051610f4f929190611f50565b60405180910390a250505050565b600080600080600080610fb660688881548110610f7c57610f7c6119d6565b906000526020600020015460e081901c9160a082901c67ffffffffffffffff169173ffffffffffffffffffffffffffffffffffffffff1690565b9199909850909650945050505050565b7f0000000000000000000000000000000000000000000000000000000000000000600054610100900460ff16158015611006575060005460ff8083169116105b611092576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a65640000000000000000000000000000000000006064820152608401610b80565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00001660ff8316176101001790556110cb611378565b6110d36113f9565b6110dc826112f3565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff821681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b600061114861080d565b73ffffffffffffffffffffffffffffffffffffffff16638da5cb5b6040518163ffffffff1660e01b8152600401602060405180830381865afa158015611192573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111b69190611b9b565b905090565b6111c3611272565b73ffffffffffffffffffffffffffffffffffffffff8116611266576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f64647265737300000000000000000000000000000000000000000000000000006064820152608401610b80565b61126f816112f3565b50565b60335473ffffffffffffffffffffffffffffffffffffffff163314610a7d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401610b80565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600061080660008484611498565b3361138161080d565b73ffffffffffffffffffffffffffffffffffffffff16141580156113c25750336113a961113e565b73ffffffffffffffffffffffffffffffffffffffff1614155b15610a7d576040517fc4050a2600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600054610100900460ff16611490576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610b80565b610a7d6115de565b600060608203516040830351602084035184518060208701018051600283016c5af43d3d93803e606057fd5bf3895289600d8a035278593da1005b363d3d373d3d3d3d610000806062363936013d738160481b1760218a03527f9e4ac34f21c619cefc926c8bd93b54bf5a39c7ab2127a895af1cc0691d7e3dff603a8a035272fd6100003d81600a3d39f336602c57343d527f6062820160781b1761ff9e82106059018a03528060f01b8352606c8101604c8a038cf0975050866115645763301164256000526004601cfd5b905285527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08501527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc08401527fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa09092019190915292915050565b600054610100900460ff16611675576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e670000000000000000000000000000000000000000006064820152608401610b80565b610a7d336112f3565b803563ffffffff8116811461169257600080fd5b919050565b73ffffffffffffffffffffffffffffffffffffffff8116811461126f57600080fd5b600080604083850312156116cc57600080fd5b6116d58361167e565b915060208301356116e581611697565b809150509250929050565b60006020828403121561170257600080fd5b6108068261167e565b6000806040838503121561171e57600080fd5b6117278361167e565b946020939093013593505050565b60008060006060848603121561174a57600080fd5b6117538461167e565b95602085013595506040909401359392505050565b60005b8381101561178357818101518382015260200161176b565b83811115611792576000848401525b50505050565b600081518084526117b0816020860160208601611768565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60006020808301818452808551808352604092508286019150828160051b87010184880160005b83811015611891578883037fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc001855281518051845287810151888501528681015167ffffffffffffffff16878501526060808201519085015260809081015160a09185018290529061187d81860183611798565b968901969450505090860190600101611809565b509098975050505050505050565b6020815260006108066020830184611798565b60008083601f8401126118c457600080fd5b50813567ffffffffffffffff8111156118dc57600080fd5b6020830191508360208285010111156118f457600080fd5b9250929050565b6000806000806060858703121561191157600080fd5b61191a8561167e565b935060208501359250604085013567ffffffffffffffff81111561193d57600080fd5b611949878288016118b2565b95989497509550505050565b6000806000806060858703121561196b57600080fd5b6119748561167e565b9350602085013561198481611697565b9250604085013567ffffffffffffffff81111561193d57600080fd5b6000602082840312156119b257600080fd5b5035919050565b6000602082840312156119cb57600080fd5b813561080681611697565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600060208284031215611a4657600080fd5b815167ffffffffffffffff80821115611a5e57600080fd5b818401915084601f830112611a7257600080fd5b815181811115611a8457611a84611a05565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908382118183101715611aca57611aca611a05565b81604052828152876020848701011115611ae357600080fd5b611af4836020830160208801611768565b979650505050505050565b600060208284031215611b1157600080fd5b5051919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611b5957611b59611b18565b500390565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611b9657611b96611b18565b500290565b600060208284031215611bad57600080fd5b815161080681611697565b600181811c90821680611bcc57607f821691505b602082108103611c05577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7fffffffffffffffffffffffffffffffffffffffff0000000000000000000000008660601b1681528460148201528360348201528183605483013760009101605401908152949350505050565b7fffffffffffffffffffffffffffffffffffffffff0000000000000000000000008860601b1681528660148201528560348201527fffffffff000000000000000000000000000000000000000000000000000000008560e01b166054820152600060588486828501378483018181016000815260008654611cd881611bb8565b60018281168015611cf05760018114611d2557611d58565b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00841686528783151584028801019450611d58565b8a60005260208060002060005b85811015611d4d5781548a82018c0152908401908201611d32565b505050878388010194505b50929f9e505050505050505050505050505050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b63ffffffff85168152836020820152606060408201526000611ddc606083018486611d6d565b9695505050505050565b601f821115611e3057600081815260208120601f850160051c81016020861015611e0d5750805b601f850160051c820191505b81811015611e2c57828155600101611e19565b5050505b505050565b67ffffffffffffffff831115611e4d57611e4d611a05565b611e6183611e5b8354611bb8565b83611de6565b6000601f841160018114611eb35760008515611e7d5750838201355b7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600387901b1c1916600186901b178355611f49565b6000838152602090207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0861690835b82811015611f025786850135825560209485019460019092019101611ee2565b5086821015611f3d577fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60f88860031b161c19848701351681555b505060018560011b0183555b5050505050565b602081526000611f64602083018486611d6d565b94935050505056fea164736f6c634300080f000a",
}

// DisputeGameFactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use DisputeGameFactoryMetaData.ABI instead.
var DisputeGameFactoryABI = DisputeGameFactoryMetaData.ABI

// DisputeGameFactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DisputeGameFactoryMetaData.Bin instead.
var DisputeGameFactoryBin = DisputeGameFactoryMetaData.Bin

// DeployDisputeGameFactory deploys a new Ethereum contract, binding an instance of DisputeGameFactory to it.
func DeployDisputeGameFactory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *DisputeGameFactory, error) {
	parsed, err := DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DisputeGameFactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DisputeGameFactory{DisputeGameFactoryCaller: DisputeGameFactoryCaller{contract: contract}, DisputeGameFactoryTransactor: DisputeGameFactoryTransactor{contract: contract}, DisputeGameFactoryFilterer: DisputeGameFactoryFilterer{contract: contract}}, nil
}

// DisputeGameFactory is an auto generated Go binding around an Ethereum contract.
type DisputeGameFactory struct {
	DisputeGameFactoryCaller     // Read-only binding to the contract
	DisputeGameFactoryTransactor // Write-only binding to the contract
	DisputeGameFactoryFilterer   // Log filterer for contract events
}

// DisputeGameFactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type DisputeGameFactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DisputeGameFactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DisputeGameFactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DisputeGameFactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DisputeGameFactorySession struct {
	Contract     *DisputeGameFactory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts       // Call options to use throughout this session
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// DisputeGameFactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DisputeGameFactoryCallerSession struct {
	Contract *DisputeGameFactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts             // Call options to use throughout this session
}

// DisputeGameFactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DisputeGameFactoryTransactorSession struct {
	Contract     *DisputeGameFactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// DisputeGameFactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type DisputeGameFactoryRaw struct {
	Contract *DisputeGameFactory // Generic contract binding to access the raw methods on
}

// DisputeGameFactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DisputeGameFactoryCallerRaw struct {
	Contract *DisputeGameFactoryCaller // Generic read-only contract binding to access the raw methods on
}

// DisputeGameFactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DisputeGameFactoryTransactorRaw struct {
	Contract *DisputeGameFactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDisputeGameFactory creates a new instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactory(address common.Address, backend bind.ContractBackend) (*DisputeGameFactory, error) {
	contract, err := bindDisputeGameFactory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactory{DisputeGameFactoryCaller: DisputeGameFactoryCaller{contract: contract}, DisputeGameFactoryTransactor: DisputeGameFactoryTransactor{contract: contract}, DisputeGameFactoryFilterer: DisputeGameFactoryFilterer{contract: contract}}, nil
}

// NewDisputeGameFactoryCaller creates a new read-only instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryCaller(address common.Address, caller bind.ContractCaller) (*DisputeGameFactoryCaller, error) {
	contract, err := bindDisputeGameFactory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryCaller{contract: contract}, nil
}

// NewDisputeGameFactoryTransactor creates a new write-only instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*DisputeGameFactoryTransactor, error) {
	contract, err := bindDisputeGameFactory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryTransactor{contract: contract}, nil
}

// NewDisputeGameFactoryFilterer creates a new log filterer instance of DisputeGameFactory, bound to a specific deployed contract.
func NewDisputeGameFactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*DisputeGameFactoryFilterer, error) {
	contract, err := bindDisputeGameFactory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryFilterer{contract: contract}, nil
}

// bindDisputeGameFactory binds a generic wrapper to an already deployed contract.
func bindDisputeGameFactory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DisputeGameFactory.Contract.DisputeGameFactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.DisputeGameFactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DisputeGameFactory *DisputeGameFactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.DisputeGameFactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DisputeGameFactory *DisputeGameFactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DisputeGameFactory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DisputeGameFactory *DisputeGameFactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DisputeGameFactory *DisputeGameFactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.contract.Transact(opts, method, params...)
}

// FindLatestGames is a free data retrieval call binding the contract method 0x254bd683.
//
// Solidity: function findLatestGames(uint32 _gameType, uint256 _start, uint256 _n) view returns((uint256,bytes32,uint64,bytes32,bytes)[] games_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) FindLatestGames(opts *bind.CallOpts, _gameType uint32, _start *big.Int, _n *big.Int) ([]DisputeGameFactoryGameSearchResult, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "findLatestGames", _gameType, _start, _n)

	if err != nil {
		return *new([]DisputeGameFactoryGameSearchResult), err
	}

	out0 := *abi.ConvertType(out[0], new([]DisputeGameFactoryGameSearchResult)).(*[]DisputeGameFactoryGameSearchResult)

	return out0, err

}

// FindLatestGames is a free data retrieval call binding the contract method 0x254bd683.
//
// Solidity: function findLatestGames(uint32 _gameType, uint256 _start, uint256 _n) view returns((uint256,bytes32,uint64,bytes32,bytes)[] games_)
func (_DisputeGameFactory *DisputeGameFactorySession) FindLatestGames(_gameType uint32, _start *big.Int, _n *big.Int) ([]DisputeGameFactoryGameSearchResult, error) {
	return _DisputeGameFactory.Contract.FindLatestGames(&_DisputeGameFactory.CallOpts, _gameType, _start, _n)
}

// FindLatestGames is a free data retrieval call binding the contract method 0x254bd683.
//
// Solidity: function findLatestGames(uint32 _gameType, uint256 _start, uint256 _n) view returns((uint256,bytes32,uint64,bytes32,bytes)[] games_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) FindLatestGames(_gameType uint32, _start *big.Int, _n *big.Int) ([]DisputeGameFactoryGameSearchResult, error) {
	return _DisputeGameFactory.Contract.FindLatestGames(&_DisputeGameFactory.CallOpts, _gameType, _start, _n)
}

// GameArgs is a free data retrieval call binding the contract method 0x74cc86ac.
//
// Solidity: function gameArgs(uint32 ) view returns(bytes)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameArgs(opts *bind.CallOpts, arg0 uint32) ([]byte, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameArgs", arg0)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GameArgs is a free data retrieval call binding the contract method 0x74cc86ac.
//
// Solidity: function gameArgs(uint32 ) view returns(bytes)
func (_DisputeGameFactory *DisputeGameFactorySession) GameArgs(arg0 uint32) ([]byte, error) {
	return _DisputeGameFactory.Contract.GameArgs(&_DisputeGameFactory.CallOpts, arg0)
}

// GameArgs is a free data retrieval call binding the contract method 0x74cc86ac.
//
// Solidity: function gameArgs(uint32 ) view returns(bytes)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameArgs(arg0 uint32) ([]byte, error) {
	return _DisputeGameFactory.Contract.GameArgs(&_DisputeGameFactory.CallOpts, arg0)
}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint32 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameAtIndex(opts *bind.CallOpts, _index *big.Int) (struct {
	GameType  uint32
	Timestamp uint64
	Proxy     common.Address
}, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameAtIndex", _index)

	outstruct := new(struct {
		GameType  uint32
		Timestamp uint64
		Proxy     common.Address
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.GameType = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Proxy = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)

	return *outstruct, err

}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint32 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactorySession) GameAtIndex(_index *big.Int) (struct {
	GameType  uint32
	Timestamp uint64
	Proxy     common.Address
}, error) {
	return _DisputeGameFactory.Contract.GameAtIndex(&_DisputeGameFactory.CallOpts, _index)
}

// GameAtIndex is a free data retrieval call binding the contract method 0xbb8aa1fc.
//
// Solidity: function gameAtIndex(uint256 _index) view returns(uint32 gameType_, uint64 timestamp_, address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameAtIndex(_index *big.Int) (struct {
	GameType  uint32
	Timestamp uint64
	Proxy     common.Address
}, error) {
	return _DisputeGameFactory.Contract.GameAtIndex(&_DisputeGameFactory.CallOpts, _index)
}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactorySession) GameCount() (*big.Int, error) {
	return _DisputeGameFactory.Contract.GameCount(&_DisputeGameFactory.CallOpts)
}

// GameCount is a free data retrieval call binding the contract method 0x4d1975b4.
//
// Solidity: function gameCount() view returns(uint256 gameCount_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameCount() (*big.Int, error) {
	return _DisputeGameFactory.Contract.GameCount(&_DisputeGameFactory.CallOpts)
}

// GameImpls is a free data retrieval call binding the contract method 0x1b685b9e.
//
// Solidity: function gameImpls(uint32 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GameImpls(opts *bind.CallOpts, arg0 uint32) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "gameImpls", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GameImpls is a free data retrieval call binding the contract method 0x1b685b9e.
//
// Solidity: function gameImpls(uint32 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) GameImpls(arg0 uint32) (common.Address, error) {
	return _DisputeGameFactory.Contract.GameImpls(&_DisputeGameFactory.CallOpts, arg0)
}

// GameImpls is a free data retrieval call binding the contract method 0x1b685b9e.
//
// Solidity: function gameImpls(uint32 ) view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GameImpls(arg0 uint32) (common.Address, error) {
	return _DisputeGameFactory.Contract.GameImpls(&_DisputeGameFactory.CallOpts, arg0)
}

// Games is a free data retrieval call binding the contract method 0x5f0150cb.
//
// Solidity: function games(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Games(opts *bind.CallOpts, _gameType uint32, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "games", _gameType, _rootClaim, _extraData)

	outstruct := new(struct {
		Proxy     common.Address
		Timestamp uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Proxy = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// Games is a free data retrieval call binding the contract method 0x5f0150cb.
//
// Solidity: function games(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactorySession) Games(_gameType uint32, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	return _DisputeGameFactory.Contract.Games(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// Games is a free data retrieval call binding the contract method 0x5f0150cb.
//
// Solidity: function games(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) view returns(address proxy_, uint64 timestamp_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Games(_gameType uint32, _rootClaim [32]byte, _extraData []byte) (struct {
	Proxy     common.Address
	Timestamp uint64
}, error) {
	return _DisputeGameFactory.Contract.Games(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// GetGameUUID is a free data retrieval call binding the contract method 0x96cd9720.
//
// Solidity: function getGameUUID(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactoryCaller) GetGameUUID(opts *bind.CallOpts, _gameType uint32, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "getGameUUID", _gameType, _rootClaim, _extraData)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetGameUUID is a free data retrieval call binding the contract method 0x96cd9720.
//
// Solidity: function getGameUUID(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactorySession) GetGameUUID(_gameType uint32, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	return _DisputeGameFactory.Contract.GetGameUUID(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// GetGameUUID is a free data retrieval call binding the contract method 0x96cd9720.
//
// Solidity: function getGameUUID(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) pure returns(bytes32 uuid_)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) GetGameUUID(_gameType uint32, _rootClaim [32]byte, _extraData []byte) ([32]byte, error) {
	return _DisputeGameFactory.Contract.GetGameUUID(&_DisputeGameFactory.CallOpts, _gameType, _rootClaim, _extraData)
}

// InitBonds is a free data retrieval call binding the contract method 0x6593dc6e.
//
// Solidity: function initBonds(uint32 ) view returns(uint256)
func (_DisputeGameFactory *DisputeGameFactoryCaller) InitBonds(opts *bind.CallOpts, arg0 uint32) (*big.Int, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "initBonds", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// InitBonds is a free data retrieval call binding the contract method 0x6593dc6e.
//
// Solidity: function initBonds(uint32 ) view returns(uint256)
func (_DisputeGameFactory *DisputeGameFactorySession) InitBonds(arg0 uint32) (*big.Int, error) {
	return _DisputeGameFactory.Contract.InitBonds(&_DisputeGameFactory.CallOpts, arg0)
}

// InitBonds is a free data retrieval call binding the contract method 0x6593dc6e.
//
// Solidity: function initBonds(uint32 ) view returns(uint256)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) InitBonds(arg0 uint32) (*big.Int, error) {
	return _DisputeGameFactory.Contract.InitBonds(&_DisputeGameFactory.CallOpts, arg0)
}

// InitVersion is a free data retrieval call binding the contract method 0x38d38c97.
//
// Solidity: function initVersion() view returns(uint8)
func (_DisputeGameFactory *DisputeGameFactoryCaller) InitVersion(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "initVersion")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// InitVersion is a free data retrieval call binding the contract method 0x38d38c97.
//
// Solidity: function initVersion() view returns(uint8)
func (_DisputeGameFactory *DisputeGameFactorySession) InitVersion() (uint8, error) {
	return _DisputeGameFactory.Contract.InitVersion(&_DisputeGameFactory.CallOpts)
}

// InitVersion is a free data retrieval call binding the contract method 0x38d38c97.
//
// Solidity: function initVersion() view returns(uint8)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) InitVersion() (uint8, error) {
	return _DisputeGameFactory.Contract.InitVersion(&_DisputeGameFactory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) Owner() (common.Address, error) {
	return _DisputeGameFactory.Contract.Owner(&_DisputeGameFactory.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Owner() (common.Address, error) {
	return _DisputeGameFactory.Contract.Owner(&_DisputeGameFactory.CallOpts)
}

// ProxyAdmin is a free data retrieval call binding the contract method 0x3e47158c.
//
// Solidity: function proxyAdmin() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) ProxyAdmin(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "proxyAdmin")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProxyAdmin is a free data retrieval call binding the contract method 0x3e47158c.
//
// Solidity: function proxyAdmin() view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) ProxyAdmin() (common.Address, error) {
	return _DisputeGameFactory.Contract.ProxyAdmin(&_DisputeGameFactory.CallOpts)
}

// ProxyAdmin is a free data retrieval call binding the contract method 0x3e47158c.
//
// Solidity: function proxyAdmin() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) ProxyAdmin() (common.Address, error) {
	return _DisputeGameFactory.Contract.ProxyAdmin(&_DisputeGameFactory.CallOpts)
}

// ProxyAdminOwner is a free data retrieval call binding the contract method 0xdad544e0.
//
// Solidity: function proxyAdminOwner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCaller) ProxyAdminOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "proxyAdminOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProxyAdminOwner is a free data retrieval call binding the contract method 0xdad544e0.
//
// Solidity: function proxyAdminOwner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactorySession) ProxyAdminOwner() (common.Address, error) {
	return _DisputeGameFactory.Contract.ProxyAdminOwner(&_DisputeGameFactory.CallOpts)
}

// ProxyAdminOwner is a free data retrieval call binding the contract method 0xdad544e0.
//
// Solidity: function proxyAdminOwner() view returns(address)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) ProxyAdminOwner() (common.Address, error) {
	return _DisputeGameFactory.Contract.ProxyAdminOwner(&_DisputeGameFactory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactoryCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _DisputeGameFactory.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactorySession) Version() (string, error) {
	return _DisputeGameFactory.Contract.Version(&_DisputeGameFactory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_DisputeGameFactory *DisputeGameFactoryCallerSession) Version() (string, error) {
	return _DisputeGameFactory.Contract.Version(&_DisputeGameFactory.CallOpts)
}

// Create is a paid mutator transaction binding the contract method 0x82ecf2f6.
//
// Solidity: function create(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) payable returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryTransactor) Create(opts *bind.TransactOpts, _gameType uint32, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "create", _gameType, _rootClaim, _extraData)
}

// Create is a paid mutator transaction binding the contract method 0x82ecf2f6.
//
// Solidity: function create(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) payable returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactorySession) Create(_gameType uint32, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Create(&_DisputeGameFactory.TransactOpts, _gameType, _rootClaim, _extraData)
}

// Create is a paid mutator transaction binding the contract method 0x82ecf2f6.
//
// Solidity: function create(uint32 _gameType, bytes32 _rootClaim, bytes _extraData) payable returns(address proxy_)
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) Create(_gameType uint32, _rootClaim [32]byte, _extraData []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Create(&_DisputeGameFactory.TransactOpts, _gameType, _rootClaim, _extraData)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) Initialize(opts *bind.TransactOpts, _owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "initialize", _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) Initialize(_owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Initialize(&_DisputeGameFactory.TransactOpts, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _owner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) Initialize(_owner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.Initialize(&_DisputeGameFactory.TransactOpts, _owner)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactorySession) RenounceOwnership() (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.RenounceOwnership(&_DisputeGameFactory.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.RenounceOwnership(&_DisputeGameFactory.TransactOpts)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x14f6b1a3.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) SetImplementation(opts *bind.TransactOpts, _gameType uint32, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "setImplementation", _gameType, _impl)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x14f6b1a3.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) SetImplementation(_gameType uint32, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation(&_DisputeGameFactory.TransactOpts, _gameType, _impl)
}

// SetImplementation is a paid mutator transaction binding the contract method 0x14f6b1a3.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) SetImplementation(_gameType uint32, _impl common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation(&_DisputeGameFactory.TransactOpts, _gameType, _impl)
}

// SetImplementation0 is a paid mutator transaction binding the contract method 0xb1070957.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl, bytes _args) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) SetImplementation0(opts *bind.TransactOpts, _gameType uint32, _impl common.Address, _args []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "setImplementation0", _gameType, _impl, _args)
}

// SetImplementation0 is a paid mutator transaction binding the contract method 0xb1070957.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl, bytes _args) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) SetImplementation0(_gameType uint32, _impl common.Address, _args []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation0(&_DisputeGameFactory.TransactOpts, _gameType, _impl, _args)
}

// SetImplementation0 is a paid mutator transaction binding the contract method 0xb1070957.
//
// Solidity: function setImplementation(uint32 _gameType, address _impl, bytes _args) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) SetImplementation0(_gameType uint32, _impl common.Address, _args []byte) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetImplementation0(&_DisputeGameFactory.TransactOpts, _gameType, _impl, _args)
}

// SetInitBond is a paid mutator transaction binding the contract method 0x1e334240.
//
// Solidity: function setInitBond(uint32 _gameType, uint256 _initBond) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) SetInitBond(opts *bind.TransactOpts, _gameType uint32, _initBond *big.Int) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "setInitBond", _gameType, _initBond)
}

// SetInitBond is a paid mutator transaction binding the contract method 0x1e334240.
//
// Solidity: function setInitBond(uint32 _gameType, uint256 _initBond) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) SetInitBond(_gameType uint32, _initBond *big.Int) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetInitBond(&_DisputeGameFactory.TransactOpts, _gameType, _initBond)
}

// SetInitBond is a paid mutator transaction binding the contract method 0x1e334240.
//
// Solidity: function setInitBond(uint32 _gameType, uint256 _initBond) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) SetInitBond(_gameType uint32, _initBond *big.Int) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.SetInitBond(&_DisputeGameFactory.TransactOpts, _gameType, _initBond)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactorySession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.TransferOwnership(&_DisputeGameFactory.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_DisputeGameFactory *DisputeGameFactoryTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _DisputeGameFactory.Contract.TransferOwnership(&_DisputeGameFactory.TransactOpts, newOwner)
}

// DisputeGameFactoryDisputeGameCreatedIterator is returned from FilterDisputeGameCreated and is used to iterate over the raw logs and unpacked data for DisputeGameCreated events raised by the DisputeGameFactory contract.
type DisputeGameFactoryDisputeGameCreatedIterator struct {
	Event *DisputeGameFactoryDisputeGameCreated // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryDisputeGameCreated)
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
		it.Event = new(DisputeGameFactoryDisputeGameCreated)
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
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryDisputeGameCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryDisputeGameCreated represents a DisputeGameCreated event raised by the DisputeGameFactory contract.
type DisputeGameFactoryDisputeGameCreated struct {
	DisputeProxy common.Address
	GameType     uint32
	RootClaim    [32]byte
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDisputeGameCreated is a free log retrieval operation binding the contract event 0x5b565efe82411da98814f356d0e7bcb8f0219b8d970307c5afb4a6903a8b2e35.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint32 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterDisputeGameCreated(opts *bind.FilterOpts, disputeProxy []common.Address, gameType []uint32, rootClaim [][32]byte) (*DisputeGameFactoryDisputeGameCreatedIterator, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryDisputeGameCreatedIterator{contract: _DisputeGameFactory.contract, event: "DisputeGameCreated", logs: logs, sub: sub}, nil
}

// WatchDisputeGameCreated is a free log subscription operation binding the contract event 0x5b565efe82411da98814f356d0e7bcb8f0219b8d970307c5afb4a6903a8b2e35.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint32 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchDisputeGameCreated(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryDisputeGameCreated, disputeProxy []common.Address, gameType []uint32, rootClaim [][32]byte) (event.Subscription, error) {

	var disputeProxyRule []interface{}
	for _, disputeProxyItem := range disputeProxy {
		disputeProxyRule = append(disputeProxyRule, disputeProxyItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var rootClaimRule []interface{}
	for _, rootClaimItem := range rootClaim {
		rootClaimRule = append(rootClaimRule, rootClaimItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "DisputeGameCreated", disputeProxyRule, gameTypeRule, rootClaimRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryDisputeGameCreated)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
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

// ParseDisputeGameCreated is a log parse operation binding the contract event 0x5b565efe82411da98814f356d0e7bcb8f0219b8d970307c5afb4a6903a8b2e35.
//
// Solidity: event DisputeGameCreated(address indexed disputeProxy, uint32 indexed gameType, bytes32 indexed rootClaim)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseDisputeGameCreated(log types.Log) (*DisputeGameFactoryDisputeGameCreated, error) {
	event := new(DisputeGameFactoryDisputeGameCreated)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "DisputeGameCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryImplementationArgsSetIterator is returned from FilterImplementationArgsSet and is used to iterate over the raw logs and unpacked data for ImplementationArgsSet events raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationArgsSetIterator struct {
	Event *DisputeGameFactoryImplementationArgsSet // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryImplementationArgsSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryImplementationArgsSet)
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
		it.Event = new(DisputeGameFactoryImplementationArgsSet)
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
func (it *DisputeGameFactoryImplementationArgsSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryImplementationArgsSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryImplementationArgsSet represents a ImplementationArgsSet event raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationArgsSet struct {
	GameType uint32
	Args     []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterImplementationArgsSet is a free log retrieval operation binding the contract event 0xa47fcdf075d680d3817bfca7973b373e1a5f6cfc3b444748299cc2b83d8348f9.
//
// Solidity: event ImplementationArgsSet(uint32 indexed gameType, bytes args)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterImplementationArgsSet(opts *bind.FilterOpts, gameType []uint32) (*DisputeGameFactoryImplementationArgsSetIterator, error) {

	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "ImplementationArgsSet", gameTypeRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryImplementationArgsSetIterator{contract: _DisputeGameFactory.contract, event: "ImplementationArgsSet", logs: logs, sub: sub}, nil
}

// WatchImplementationArgsSet is a free log subscription operation binding the contract event 0xa47fcdf075d680d3817bfca7973b373e1a5f6cfc3b444748299cc2b83d8348f9.
//
// Solidity: event ImplementationArgsSet(uint32 indexed gameType, bytes args)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchImplementationArgsSet(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryImplementationArgsSet, gameType []uint32) (event.Subscription, error) {

	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "ImplementationArgsSet", gameTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryImplementationArgsSet)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationArgsSet", log); err != nil {
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

// ParseImplementationArgsSet is a log parse operation binding the contract event 0xa47fcdf075d680d3817bfca7973b373e1a5f6cfc3b444748299cc2b83d8348f9.
//
// Solidity: event ImplementationArgsSet(uint32 indexed gameType, bytes args)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseImplementationArgsSet(log types.Log) (*DisputeGameFactoryImplementationArgsSet, error) {
	event := new(DisputeGameFactoryImplementationArgsSet)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationArgsSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryImplementationSetIterator is returned from FilterImplementationSet and is used to iterate over the raw logs and unpacked data for ImplementationSet events raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationSetIterator struct {
	Event *DisputeGameFactoryImplementationSet // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryImplementationSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryImplementationSet)
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
		it.Event = new(DisputeGameFactoryImplementationSet)
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
func (it *DisputeGameFactoryImplementationSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryImplementationSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryImplementationSet represents a ImplementationSet event raised by the DisputeGameFactory contract.
type DisputeGameFactoryImplementationSet struct {
	Impl     common.Address
	GameType uint32
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterImplementationSet is a free log retrieval operation binding the contract event 0xff513d80e2c7fa487608f70a618dfbc0cf415699dc69588c747e8c71566c88de.
//
// Solidity: event ImplementationSet(address indexed impl, uint32 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterImplementationSet(opts *bind.FilterOpts, impl []common.Address, gameType []uint32) (*DisputeGameFactoryImplementationSetIterator, error) {

	var implRule []interface{}
	for _, implItem := range impl {
		implRule = append(implRule, implItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "ImplementationSet", implRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryImplementationSetIterator{contract: _DisputeGameFactory.contract, event: "ImplementationSet", logs: logs, sub: sub}, nil
}

// WatchImplementationSet is a free log subscription operation binding the contract event 0xff513d80e2c7fa487608f70a618dfbc0cf415699dc69588c747e8c71566c88de.
//
// Solidity: event ImplementationSet(address indexed impl, uint32 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchImplementationSet(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryImplementationSet, impl []common.Address, gameType []uint32) (event.Subscription, error) {

	var implRule []interface{}
	for _, implItem := range impl {
		implRule = append(implRule, implItem)
	}
	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "ImplementationSet", implRule, gameTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryImplementationSet)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationSet", log); err != nil {
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

// ParseImplementationSet is a log parse operation binding the contract event 0xff513d80e2c7fa487608f70a618dfbc0cf415699dc69588c747e8c71566c88de.
//
// Solidity: event ImplementationSet(address indexed impl, uint32 indexed gameType)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseImplementationSet(log types.Log) (*DisputeGameFactoryImplementationSet, error) {
	event := new(DisputeGameFactoryImplementationSet)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "ImplementationSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryInitBondUpdatedIterator is returned from FilterInitBondUpdated and is used to iterate over the raw logs and unpacked data for InitBondUpdated events raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitBondUpdatedIterator struct {
	Event *DisputeGameFactoryInitBondUpdated // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryInitBondUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryInitBondUpdated)
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
		it.Event = new(DisputeGameFactoryInitBondUpdated)
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
func (it *DisputeGameFactoryInitBondUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryInitBondUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryInitBondUpdated represents a InitBondUpdated event raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitBondUpdated struct {
	GameType uint32
	NewBond  *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterInitBondUpdated is a free log retrieval operation binding the contract event 0x74d6665c4b26d5596a5aa13d3014e0c06af4d322075a797f87b03cd4c5bc91ca.
//
// Solidity: event InitBondUpdated(uint32 indexed gameType, uint256 indexed newBond)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterInitBondUpdated(opts *bind.FilterOpts, gameType []uint32, newBond []*big.Int) (*DisputeGameFactoryInitBondUpdatedIterator, error) {

	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var newBondRule []interface{}
	for _, newBondItem := range newBond {
		newBondRule = append(newBondRule, newBondItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "InitBondUpdated", gameTypeRule, newBondRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryInitBondUpdatedIterator{contract: _DisputeGameFactory.contract, event: "InitBondUpdated", logs: logs, sub: sub}, nil
}

// WatchInitBondUpdated is a free log subscription operation binding the contract event 0x74d6665c4b26d5596a5aa13d3014e0c06af4d322075a797f87b03cd4c5bc91ca.
//
// Solidity: event InitBondUpdated(uint32 indexed gameType, uint256 indexed newBond)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchInitBondUpdated(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryInitBondUpdated, gameType []uint32, newBond []*big.Int) (event.Subscription, error) {

	var gameTypeRule []interface{}
	for _, gameTypeItem := range gameType {
		gameTypeRule = append(gameTypeRule, gameTypeItem)
	}
	var newBondRule []interface{}
	for _, newBondItem := range newBond {
		newBondRule = append(newBondRule, newBondItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "InitBondUpdated", gameTypeRule, newBondRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryInitBondUpdated)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "InitBondUpdated", log); err != nil {
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

// ParseInitBondUpdated is a log parse operation binding the contract event 0x74d6665c4b26d5596a5aa13d3014e0c06af4d322075a797f87b03cd4c5bc91ca.
//
// Solidity: event InitBondUpdated(uint32 indexed gameType, uint256 indexed newBond)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseInitBondUpdated(log types.Log) (*DisputeGameFactoryInitBondUpdated, error) {
	event := new(DisputeGameFactoryInitBondUpdated)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "InitBondUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitializedIterator struct {
	Event *DisputeGameFactoryInitialized // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryInitialized)
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
		it.Event = new(DisputeGameFactoryInitialized)
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
func (it *DisputeGameFactoryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryInitialized represents a Initialized event raised by the DisputeGameFactory contract.
type DisputeGameFactoryInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterInitialized(opts *bind.FilterOpts) (*DisputeGameFactoryInitializedIterator, error) {

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryInitializedIterator{contract: _DisputeGameFactory.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryInitialized) (event.Subscription, error) {

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryInitialized)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseInitialized(log types.Log) (*DisputeGameFactoryInitialized, error) {
	event := new(DisputeGameFactoryInitialized)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DisputeGameFactoryOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the DisputeGameFactory contract.
type DisputeGameFactoryOwnershipTransferredIterator struct {
	Event *DisputeGameFactoryOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *DisputeGameFactoryOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DisputeGameFactoryOwnershipTransferred)
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
		it.Event = new(DisputeGameFactoryOwnershipTransferred)
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
func (it *DisputeGameFactoryOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DisputeGameFactoryOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DisputeGameFactoryOwnershipTransferred represents a OwnershipTransferred event raised by the DisputeGameFactory contract.
type DisputeGameFactoryOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*DisputeGameFactoryOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &DisputeGameFactoryOwnershipTransferredIterator{contract: _DisputeGameFactory.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_DisputeGameFactory *DisputeGameFactoryFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *DisputeGameFactoryOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _DisputeGameFactory.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DisputeGameFactoryOwnershipTransferred)
				if err := _DisputeGameFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_DisputeGameFactory *DisputeGameFactoryFilterer) ParseOwnershipTransferred(log types.Log) (*DisputeGameFactoryOwnershipTransferred, error) {
	event := new(DisputeGameFactoryOwnershipTransferred)
	if err := _DisputeGameFactory.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

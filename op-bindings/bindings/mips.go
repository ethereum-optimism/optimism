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

// MIPSMetaData contains all meta data concerning the MIPS contract.
var MIPSMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"_oracle\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BRK_START\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"oracle_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"output_\",\"type\":\"bytes32\"},{\"internalType\":\"uint8\",\"name\":\"exitCode_\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051611da1380380611da183398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051611d1061009160003960008181608501526114d80152611d106000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063f8e0cb96146100af575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60405173ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610062565b6100c26100bd366004611c15565b6100d9565b6040805192835260ff909116602083015201610062565b6000806100e4611b49565b608081146100f157600080fd5b6040516106001461010157600080fd5b6064871461010e57600080fd5b610184851461011c57600080fd5b8635608052602087013560a052604087013560e090811c60c09081526044890135821c82526048890135821c61010052604c890135821c610120526050890135821c61014052605489013590911c61016052605888013560f890811c610180526059890135901c6101a052605a880135901c6101c0526102006101e0819052606288019060005b60208110156101c757823560e01c82526004909201916020909101906001016101a3565b505050806101200151156101e7576101dd610626565b925092505061061d565b6101408101805160010167ffffffffffffffff169052606081015160009061020f90826106d8565b9050603f601a82901c16600281148061022e57508063ffffffff166003145b1561027d576102718163ffffffff1660021461024b57601f61024e565b60005b60ff166002610264856303ffffff16601a610794565b63ffffffff16901b610807565b9450945050505061061d565b6101608301516000908190601f601086901c81169190601587901c16602081106102a9576102a9611c81565b602002015192508063ffffffff851615806102ca57508463ffffffff16601c145b15610301578661016001518263ffffffff16602081106102ec576102ec611c81565b6020020151925050601f600b86901c166103bd565b60208563ffffffff161015610363578463ffffffff16600c148061032b57508463ffffffff16600d145b8061033c57508463ffffffff16600e145b1561034d578561ffff1692506103bd565b61035c8661ffff166010610794565b92506103bd565b60288563ffffffff1610158061037f57508463ffffffff166022145b8061039057508463ffffffff166026145b156103bd578661016001518263ffffffff16602081106103b2576103b2611c81565b602002015192508190505b60048563ffffffff16101580156103da575060088563ffffffff16105b806103eb57508463ffffffff166001145b1561040c576103fc85878487610907565b985098505050505050505061061d565b63ffffffff60006020878316106104715761042c8861ffff166010610794565b9095019463fffffffc86166104428160016106d8565b915060288863ffffffff161015801561046257508763ffffffff16603014155b1561046f57809250600093505b505b600061047f89888885610b1c565b63ffffffff9081169150603f8a169089161580156104a4575060088163ffffffff1610155b80156104b65750601c8163ffffffff16105b15610594578063ffffffff16600814806104d657508063ffffffff166009145b1561050f576104fb8163ffffffff166008146104f257856104f5565b60005b89610807565b9c509c50505050505050505050505061061d565b8063ffffffff16600a0361052f576104fb858963ffffffff8a16156111d1565b8063ffffffff16600b03610550576104fb858963ffffffff8a1615156111d1565b8063ffffffff16600c03610566576104fb6112c5565b60108163ffffffff16101580156105835750601c8163ffffffff16105b15610594576104fb818989886117fd565b8863ffffffff1660381480156105af575063ffffffff861615155b156105e45760018b61016001518763ffffffff16602081106105d3576105d3611c81565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff1461060157610601846001846119fc565b61060d858360016111d1565b9c509c5050505050505050505050505b94509492505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c51605482015261019f5160588201526101bf5160598201526101d851605a82015260009081906102009060628101835b60208110156106bb57601c8401518252602090930192600490910190600101610697565b506000815281810382a08190039020610180519094909350915050565b6000806106e483611aa0565b905060038416156106f457600080fd5b6020810190358460051c8160005b601b81101561075a5760208501943583821c600116801561072a576001811461073f57610750565b60008481526020839052604090209350610750565b600082815260208590526040902093505b5050600101610702565b50608051915081811461077557630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b01826107f15760006107f3565b815b90861663ffffffff16179250505092915050565b600080610812611b49565b60809050806060015160040163ffffffff16816080015163ffffffff161461089b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff9081169093528683169052908616156108f157806008018261016001518763ffffffff16602081106108e0576108e0611c81565b63ffffffff90921660209290920201525b6108f9610626565b9350935050505b9250929050565b600080610912611b49565b608090506000816060015160040163ffffffff16826080015163ffffffff1614610998576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f740000000000000000000000006044820152606401610892565b8763ffffffff16600414806109b357508763ffffffff166005145b15610a2f5760008261016001518763ffffffff16602081106109d7576109d7611c81565b602002015190508063ffffffff168663ffffffff161480156109ff57508863ffffffff166004145b80610a2757508063ffffffff168663ffffffff1614158015610a2757508863ffffffff166005145b915050610aac565b8763ffffffff16600603610a4c5760008560030b13159050610aac565b8763ffffffff16600703610a685760008560030b139050610aac565b8763ffffffff16600103610aac57601f601088901c166000819003610a915760008660030b1291505b8063ffffffff16600103610aaa5760008660030b121591505b505b606082018051608084015163ffffffff169091528115610af2576002610ad78961ffff166010610794565b63ffffffff90811690911b8201600401166080840152610b04565b60808301805160040163ffffffff1690525b610b0c610626565b9450945050505094509492505050565b6000603f601a86901c81169086166020821015610eea5760088263ffffffff1610158015610b505750600f8263ffffffff16105b15610bf0578163ffffffff16600803610b6b57506020610beb565b8163ffffffff16600903610b8157506021610beb565b8163ffffffff16600a03610b975750602a610beb565b8163ffffffff16600b03610bad5750602b610beb565b8163ffffffff16600c03610bc357506024610beb565b8163ffffffff16600d03610bd957506025610beb565b8163ffffffff16600e03610beb575060265b600091505b8163ffffffff16600003610e3e57601f600688901c16602063ffffffff83161015610d185760088263ffffffff1610610c2e578693505050506111c9565b8163ffffffff16600003610c515763ffffffff86811691161b92506111c9915050565b8163ffffffff16600203610c745763ffffffff86811691161c92506111c9915050565b8163ffffffff16600303610ca857610c9e8163ffffffff168763ffffffff16901c82602003610794565b93505050506111c9565b8163ffffffff16600403610ccb575050505063ffffffff8216601f84161b6111c9565b8163ffffffff16600603610cee575050505063ffffffff8216601f84161c6111c9565b8163ffffffff16600703610d1857610c9e8763ffffffff168763ffffffff16901c88602003610794565b8163ffffffff1660201480610d3357508163ffffffff166021145b15610d455785870193505050506111c9565b8163ffffffff1660221480610d6057508163ffffffff166023145b15610d725785870393505050506111c9565b8163ffffffff16602403610d8d5785871693505050506111c9565b8163ffffffff16602503610da85785871793505050506111c9565b8163ffffffff16602603610dc35785871893505050506111c9565b8163ffffffff16602703610dde5750505050828217196111c9565b8163ffffffff16602a03610e10578560030b8760030b12610e00576000610e03565b60015b60ff1693505050506111c9565b8163ffffffff16602b03610e38578563ffffffff168763ffffffff1610610e00576000610e03565b50611167565b8163ffffffff16600f03610e605760108563ffffffff16901b925050506111c9565b8163ffffffff16601c03610ee5578063ffffffff16600203610e87575050508282026111c9565b8063ffffffff1660201480610ea257508063ffffffff166021145b15610ee5578063ffffffff16602003610eb9579419945b60005b6380000000871615610edb576401fffffffe600197881b169601610ebc565b92506111c9915050565b611167565b60288263ffffffff16101561104d578163ffffffff16602003610f3657610f2d8660031660080260180363ffffffff168563ffffffff16901c60ff166008610794565b925050506111c9565b8163ffffffff16602103610f6b57610f2d8660021660080260100363ffffffff168563ffffffff16901c61ffff166010610794565b8163ffffffff16602203610f9b5750505063ffffffff60086003851602811681811b198416918316901b176111c9565b8163ffffffff16602303610fb35783925050506111c9565b8163ffffffff16602403610fe6578560031660080260180363ffffffff168463ffffffff16901c60ff16925050506111c9565b8163ffffffff1660250361101a578560021660080260100363ffffffff168463ffffffff16901c61ffff16925050506111c9565b8163ffffffff16602603610ee55750505063ffffffff60086003851602601803811681811c198416918316901c176111c9565b8163ffffffff166028036110845750505060ff63ffffffff60086003861602601803811682811b9091188316918416901b176111c9565b8163ffffffff166029036110bc5750505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b176111c9565b8163ffffffff16602a036110ec5750505063ffffffff60086003851602811681811c198316918416901c176111c9565b8163ffffffff16602b036111045784925050506111c9565b8163ffffffff16602e036111375750505063ffffffff60086003851602601803811681811b198316918416901b176111c9565b8163ffffffff1660300361114f5783925050506111c9565b8163ffffffff166038036111675784925050506111c9565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e000000000000000000000000006044820152606401610892565b949350505050565b6000806111dc611b49565b506080602063ffffffff87161061124f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c69642072656769737465720000000000000000000000000000000000006044820152606401610892565b63ffffffff8616158015906112615750835b1561129557848161016001518763ffffffff166020811061128457611284611c81565b63ffffffff90921660209290920201525b60808101805163ffffffff808216606085015260049091011690526112b8610626565b9250925050935093915050565b6000806112d0611b49565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa0361134a5781610fff81161561131957610fff811661100003015b8363ffffffff166000036113405760e08801805163ffffffff838201169091529550611344565b8395505b506117bc565b8563ffffffff16610fcd0361136557634000000094506117bc565b8563ffffffff166110180361137d57600194506117bc565b8563ffffffff16611096036113b557600161012088015260ff83166101008801526113a6610626565b98509850505050505050509091565b8563ffffffff16610fa30361161f5763ffffffff8316156117bc577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff8416016115d95760006114108363fffffffc1660016106d8565b60208901519091508060001a60010361147d5761147a81600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b90505b6040808a015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810183905263ffffffff9091166024820152600090819073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e03110e1906044016040805180830381865afa15801561151e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906115429190611cb0565b9150915060038616806004038281101561155a578092505b5081861015611567578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506115be8663fffffffc166001866119fc565b60408b018051820163ffffffff169052975061161a92505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff84160161160e578094506117bc565b63ffffffff9450600993505b6117bc565b8563ffffffff16610fa4036117105763ffffffff831660011480611649575063ffffffff83166002145b8061165a575063ffffffff83166004145b15611667578094506117bc565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff84160161160e5760006116a78363fffffffc1660016106d8565b602089015190915060038416600403838110156116c2578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b176020880152600060408801529350836117bc565b8563ffffffff16610fd7036117bc578163ffffffff166003036117b05763ffffffff83161580611746575063ffffffff83166005145b80611757575063ffffffff83166003145b1561176557600094506117bc565b63ffffffff831660011480611780575063ffffffff83166002145b80611791575063ffffffff83166006145b806117a2575063ffffffff83166004145b1561160e57600194506117bc565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b015260040190911690526113a6610626565b600080611808611b49565b506080600063ffffffff8816601003611826575060c081015161198f565b8763ffffffff166011036118455763ffffffff871660c083015261198f565b8763ffffffff1660120361185e575060a081015161198f565b8763ffffffff1660130361187d5763ffffffff871660a083015261198f565b8763ffffffff166018036118b15763ffffffff600388810b9088900b02602081901c821660c08501521660a083015261198f565b8763ffffffff166019036118e25763ffffffff87811681881602602081901c821660c08501521660a083015261198f565b8763ffffffff16601a03611938578560030b8760030b8161190557611905611cd4565b0763ffffffff1660c0830152600386810b9088900b8161192757611927611cd4565b0563ffffffff1660a083015261198f565b8763ffffffff16601b0361198f578563ffffffff168763ffffffff168161196157611961611cd4565b0663ffffffff90811660c08401528681169088168161198257611982611cd4565b0463ffffffff1660a08301525b63ffffffff8516156119ca57808261016001518663ffffffff16602081106119b9576119b9611c81565b63ffffffff90921660209290920201525b60808201805163ffffffff808216606086015260049091011690526119ed610626565b93509350505094509492505050565b6000611a0783611aa0565b90506003841615611a1757600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611a955760208401933582821c6001168015611a655760018114611a7a57611a8b565b60008581526020839052604090209450611a8b565b600082815260208690526040902094505b5050600101611a3d565b505060805250505050565b60ff811661038002610184810190369061050401811015611b43576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f61746100000000000000000000000000000000000000000000000000000000006064820152608401610892565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611baf611bb4565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611be557600080fd5b50813567ffffffffffffffff811115611bfd57600080fd5b60208301915083602082850101111561090057600080fd5b60008060008060408587031215611c2b57600080fd5b843567ffffffffffffffff80821115611c4357600080fd5b611c4f88838901611bd3565b90965094506020870135915080821115611c6857600080fd5b50611c7587828801611bd3565b95989497509550505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611cc357600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
}

// MIPSABI is the input ABI used to generate the binding from.
// Deprecated: Use MIPSMetaData.ABI instead.
var MIPSABI = MIPSMetaData.ABI

// MIPSBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MIPSMetaData.Bin instead.
var MIPSBin = MIPSMetaData.Bin

// DeployMIPS deploys a new Ethereum contract, binding an instance of MIPS to it.
func DeployMIPS(auth *bind.TransactOpts, backend bind.ContractBackend, _oracle common.Address) (common.Address, *types.Transaction, *MIPS, error) {
	parsed, err := MIPSMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MIPSBin), backend, _oracle)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MIPS{MIPSCaller: MIPSCaller{contract: contract}, MIPSTransactor: MIPSTransactor{contract: contract}, MIPSFilterer: MIPSFilterer{contract: contract}}, nil
}

// MIPS is an auto generated Go binding around an Ethereum contract.
type MIPS struct {
	MIPSCaller     // Read-only binding to the contract
	MIPSTransactor // Write-only binding to the contract
	MIPSFilterer   // Log filterer for contract events
}

// MIPSCaller is an auto generated read-only Go binding around an Ethereum contract.
type MIPSCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MIPSTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MIPSTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MIPSFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MIPSFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MIPSSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MIPSSession struct {
	Contract     *MIPS             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MIPSCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MIPSCallerSession struct {
	Contract *MIPSCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// MIPSTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MIPSTransactorSession struct {
	Contract     *MIPSTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MIPSRaw is an auto generated low-level Go binding around an Ethereum contract.
type MIPSRaw struct {
	Contract *MIPS // Generic contract binding to access the raw methods on
}

// MIPSCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MIPSCallerRaw struct {
	Contract *MIPSCaller // Generic read-only contract binding to access the raw methods on
}

// MIPSTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MIPSTransactorRaw struct {
	Contract *MIPSTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMIPS creates a new instance of MIPS, bound to a specific deployed contract.
func NewMIPS(address common.Address, backend bind.ContractBackend) (*MIPS, error) {
	contract, err := bindMIPS(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MIPS{MIPSCaller: MIPSCaller{contract: contract}, MIPSTransactor: MIPSTransactor{contract: contract}, MIPSFilterer: MIPSFilterer{contract: contract}}, nil
}

// NewMIPSCaller creates a new read-only instance of MIPS, bound to a specific deployed contract.
func NewMIPSCaller(address common.Address, caller bind.ContractCaller) (*MIPSCaller, error) {
	contract, err := bindMIPS(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MIPSCaller{contract: contract}, nil
}

// NewMIPSTransactor creates a new write-only instance of MIPS, bound to a specific deployed contract.
func NewMIPSTransactor(address common.Address, transactor bind.ContractTransactor) (*MIPSTransactor, error) {
	contract, err := bindMIPS(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MIPSTransactor{contract: contract}, nil
}

// NewMIPSFilterer creates a new log filterer instance of MIPS, bound to a specific deployed contract.
func NewMIPSFilterer(address common.Address, filterer bind.ContractFilterer) (*MIPSFilterer, error) {
	contract, err := bindMIPS(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MIPSFilterer{contract: contract}, nil
}

// bindMIPS binds a generic wrapper to an already deployed contract.
func bindMIPS(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MIPSABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MIPS *MIPSRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MIPS.Contract.MIPSCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MIPS *MIPSRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MIPS.Contract.MIPSTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MIPS *MIPSRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MIPS.Contract.MIPSTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MIPS *MIPSCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _MIPS.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MIPS *MIPSTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MIPS.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MIPS *MIPSTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MIPS.Contract.contract.Transact(opts, method, params...)
}

// BRKSTART is a free data retrieval call binding the contract method 0x155633fe.
//
// Solidity: function BRK_START() view returns(uint32)
func (_MIPS *MIPSCaller) BRKSTART(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _MIPS.contract.Call(opts, &out, "BRK_START")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BRKSTART is a free data retrieval call binding the contract method 0x155633fe.
//
// Solidity: function BRK_START() view returns(uint32)
func (_MIPS *MIPSSession) BRKSTART() (uint32, error) {
	return _MIPS.Contract.BRKSTART(&_MIPS.CallOpts)
}

// BRKSTART is a free data retrieval call binding the contract method 0x155633fe.
//
// Solidity: function BRK_START() view returns(uint32)
func (_MIPS *MIPSCallerSession) BRKSTART() (uint32, error) {
	return _MIPS.Contract.BRKSTART(&_MIPS.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address oracle_)
func (_MIPS *MIPSCaller) Oracle(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _MIPS.contract.Call(opts, &out, "oracle")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address oracle_)
func (_MIPS *MIPSSession) Oracle() (common.Address, error) {
	return _MIPS.Contract.Oracle(&_MIPS.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address oracle_)
func (_MIPS *MIPSCallerSession) Oracle() (common.Address, error) {
	return _MIPS.Contract.Oracle(&_MIPS.CallOpts)
}

// Step is a paid mutator transaction binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32 output_, uint8 exitCode_)
func (_MIPS *MIPSTransactor) Step(opts *bind.TransactOpts, stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.contract.Transact(opts, "step", stateData, proof)
}

// Step is a paid mutator transaction binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32 output_, uint8 exitCode_)
func (_MIPS *MIPSSession) Step(stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, stateData, proof)
}

// Step is a paid mutator transaction binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32 output_, uint8 exitCode_)
func (_MIPS *MIPSTransactorSession) Step(stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, stateData, proof)
}

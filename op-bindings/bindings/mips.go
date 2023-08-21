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
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"_oracle\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BRK_START\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"oracle_\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051611d67380380611d6783398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051611cd6610091600039600081816085015261149c0152611cd66000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063f8e0cb96146100af575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60405173ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610062565b6100c26100bd366004611bdb565b6100d0565b604051908152602001610062565b60006100da611b08565b608081146100e757600080fd5b604051610600146100f757600080fd5b6064861461010457600080fd5b610184841461011257600080fd5b8535608052602086013560a052604086013560e090811c60c09081526044880135821c82526048880135821c61010052604c880135821c610120526050880135821c61014052605488013590911c61016052605887013560f890811c610180526059880135901c6101a052605a870135901c6101c0526102006101e0819052606287019060005b60208110156101bd57823560e01c8252600490920191602090910190600101610199565b505050806101200151156101db576101d3610619565b915050610611565b6101408101805160010167ffffffffffffffff169052606081015160009061020390826106c1565b9050603f601a82901c16600281148061022257508063ffffffff166003145b156102775760006002836303ffffff1663ffffffff16901b846080015163f00000001617905061026c8263ffffffff1660021461026057601f610263565b60005b60ff168261077d565b945050505050610611565b6101608301516000908190601f601086901c81169190601587901c16602081106102a3576102a3611c47565b602002015192508063ffffffff851615806102c457508463ffffffff16601c145b156102fb578661016001518263ffffffff16602081106102e6576102e6611c47565b6020020151925050601f600b86901c166103b7565b60208563ffffffff16101561035d578463ffffffff16600c148061032557508463ffffffff16600d145b8061033657508463ffffffff16600e145b15610347578561ffff1692506103b7565b6103568661ffff166010610877565b92506103b7565b60288563ffffffff1610158061037957508463ffffffff166022145b8061038a57508463ffffffff166026145b156103b7578661016001518263ffffffff16602081106103ac576103ac611c47565b602002015192508190505b60048563ffffffff16101580156103d4575060088563ffffffff16105b806103e557508463ffffffff166001145b15610404576103f6858784876108ea565b975050505050505050610611565b63ffffffff6000602087831610610469576104248861ffff166010610877565b9095019463fffffffc861661043a8160016106c1565b915060288863ffffffff161015801561045a57508763ffffffff16603014155b1561046757809250600093505b505b600061047789888885610afa565b63ffffffff9081169150603f8a1690891615801561049c575060088163ffffffff1610155b80156104ae5750601c8163ffffffff16105b1561058a578063ffffffff16600814806104ce57508063ffffffff166009145b15610505576104f38163ffffffff166008146104ea57856104ed565b60005b8961077d565b9b505050505050505050505050610611565b8063ffffffff16600a03610525576104f3858963ffffffff8a16156111a7565b8063ffffffff16600b03610546576104f3858963ffffffff8a1615156111a7565b8063ffffffff16600c0361055c576104f361128d565b60108163ffffffff16101580156105795750601c8163ffffffff16105b1561058a576104f3818989886117c1565b8863ffffffff1660381480156105a5575063ffffffff861615155b156105da5760018b61016001518763ffffffff16602081106105c9576105c9611c47565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff146105f7576105f7846001846119bb565b610603858360016111a7565b9b5050505050505050505050505b949350505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c51605482015261019f5160588201526101bf5160598201526101d851605a8201526000906102009060628101835b60208110156106ac57601c8401518252602090930192600490910190600101610688565b506000815281810382a0819003902092915050565b6000806106cd83611a5f565b905060038416156106dd57600080fd5b6020810190358460051c8160005b601b8110156107435760208501943583821c6001168015610713576001811461072857610739565b60008481526020839052604090209350610739565b600082815260208590526040902093505b50506001016106eb565b50608051915081811461075e57630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b6000610787611b08565b60809050806060015160040163ffffffff16816080015163ffffffff1614610810576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff90811690935285831690529085161561086657806008018261016001518663ffffffff166020811061085557610855611c47565b63ffffffff90921660209290920201525b61086e610619565b95945050505050565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b01826108d45760006108d6565b815b90861663ffffffff16179250505092915050565b60006108f4611b08565b608090506000816060015160040163ffffffff16826080015163ffffffff161461097a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f740000000000000000000000006044820152606401610807565b8663ffffffff166004148061099557508663ffffffff166005145b15610a115760008261016001518663ffffffff16602081106109b9576109b9611c47565b602002015190508063ffffffff168563ffffffff161480156109e157508763ffffffff166004145b80610a0957508063ffffffff168563ffffffff1614158015610a0957508763ffffffff166005145b915050610a8e565b8663ffffffff16600603610a2e5760008460030b13159050610a8e565b8663ffffffff16600703610a4a5760008460030b139050610a8e565b8663ffffffff16600103610a8e57601f601087901c166000819003610a735760008560030b1291505b8063ffffffff16600103610a8c5760008560030b121591505b505b606082018051608084015163ffffffff169091528115610ad4576002610ab98861ffff166010610877565b63ffffffff90811690911b8201600401166080840152610ae6565b60808301805160040163ffffffff1690525b610aee610619565b98975050505050505050565b6000603f601a86901c81169086166020821015610ec85760088263ffffffff1610158015610b2e5750600f8263ffffffff16105b15610bce578163ffffffff16600803610b4957506020610bc9565b8163ffffffff16600903610b5f57506021610bc9565b8163ffffffff16600a03610b755750602a610bc9565b8163ffffffff16600b03610b8b5750602b610bc9565b8163ffffffff16600c03610ba157506024610bc9565b8163ffffffff16600d03610bb757506025610bc9565b8163ffffffff16600e03610bc9575060265b600091505b8163ffffffff16600003610e1c57601f600688901c16602063ffffffff83161015610cf65760088263ffffffff1610610c0c57869350505050610611565b8163ffffffff16600003610c2f5763ffffffff86811691161b9250610611915050565b8163ffffffff16600203610c525763ffffffff86811691161c9250610611915050565b8163ffffffff16600303610c8657610c7c8163ffffffff168763ffffffff16901c82602003610877565b9350505050610611565b8163ffffffff16600403610ca9575050505063ffffffff8216601f84161b610611565b8163ffffffff16600603610ccc575050505063ffffffff8216601f84161c610611565b8163ffffffff16600703610cf657610c7c8763ffffffff168763ffffffff16901c88602003610877565b8163ffffffff1660201480610d1157508163ffffffff166021145b15610d23578587019350505050610611565b8163ffffffff1660221480610d3e57508163ffffffff166023145b15610d50578587039350505050610611565b8163ffffffff16602403610d6b578587169350505050610611565b8163ffffffff16602503610d86578587179350505050610611565b8163ffffffff16602603610da1578587189350505050610611565b8163ffffffff16602703610dbc575050505082821719610611565b8163ffffffff16602a03610dee578560030b8760030b12610dde576000610de1565b60015b60ff169350505050610611565b8163ffffffff16602b03610e16578563ffffffff168763ffffffff1610610dde576000610de1565b50611145565b8163ffffffff16600f03610e3e5760108563ffffffff16901b92505050610611565b8163ffffffff16601c03610ec3578063ffffffff16600203610e6557505050828202610611565b8063ffffffff1660201480610e8057508063ffffffff166021145b15610ec3578063ffffffff16602003610e97579419945b60005b6380000000871615610eb9576401fffffffe600197881b169601610e9a565b9250610611915050565b611145565b60288263ffffffff16101561102b578163ffffffff16602003610f1457610f0b8660031660080260180363ffffffff168563ffffffff16901c60ff166008610877565b92505050610611565b8163ffffffff16602103610f4957610f0b8660021660080260100363ffffffff168563ffffffff16901c61ffff166010610877565b8163ffffffff16602203610f795750505063ffffffff60086003851602811681811b198416918316901b17610611565b8163ffffffff16602303610f91578392505050610611565b8163ffffffff16602403610fc4578560031660080260180363ffffffff168463ffffffff16901c60ff1692505050610611565b8163ffffffff16602503610ff8578560021660080260100363ffffffff168463ffffffff16901c61ffff1692505050610611565b8163ffffffff16602603610ec35750505063ffffffff60086003851602601803811681811c198416918316901c17610611565b8163ffffffff166028036110625750505060ff63ffffffff60086003861602601803811682811b9091188316918416901b17610611565b8163ffffffff1660290361109a5750505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b17610611565b8163ffffffff16602a036110ca5750505063ffffffff60086003851602811681811c198316918416901c17610611565b8163ffffffff16602b036110e2578492505050610611565b8163ffffffff16602e036111155750505063ffffffff60086003851602601803811681811b198316918416901b17610611565b8163ffffffff1660300361112d578392505050610611565b8163ffffffff16603803611145578492505050610611565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e000000000000000000000000006044820152606401610807565b60006111b1611b08565b506080602063ffffffff861610611224576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c69642072656769737465720000000000000000000000000000000000006044820152606401610807565b63ffffffff8516158015906112365750825b1561126a57838161016001518663ffffffff166020811061125957611259611c47565b63ffffffff90921660209290920201525b60808101805163ffffffff8082166060850152600490910116905261086e610619565b6000611297611b08565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036113115781610fff8116156112e057610fff811661100003015b8363ffffffff166000036113075760e08801805163ffffffff83820116909152955061130b565b8395505b50611780565b8563ffffffff16610fcd0361132c5763400000009450611780565b8563ffffffff16611018036113445760019450611780565b8563ffffffff166110960361137957600161012088015260ff831661010088015261136d610619565b97505050505050505090565b8563ffffffff16610fa3036115e35763ffffffff831615611780577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff84160161159d5760006113d48363fffffffc1660016106c1565b60208901519091508060001a6001036114415761143e81600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b90505b6040808a015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810183905263ffffffff9091166024820152600090819073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e03110e1906044016040805180830381865afa1580156114e2573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906115069190611c76565b9150915060038616806004038281101561151e578092505b508186101561152b578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506115828663fffffffc166001866119bb565b60408b018051820163ffffffff16905297506115de92505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff8416016115d257809450611780565b63ffffffff9450600993505b611780565b8563ffffffff16610fa4036116d45763ffffffff83166001148061160d575063ffffffff83166002145b8061161e575063ffffffff83166004145b1561162b57809450611780565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff8416016115d257600061166b8363fffffffc1660016106c1565b60208901519091506003841660040383811015611686578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b17602088015260006040880152935083611780565b8563ffffffff16610fd703611780578163ffffffff166003036117745763ffffffff8316158061170a575063ffffffff83166005145b8061171b575063ffffffff83166003145b156117295760009450611780565b63ffffffff831660011480611744575063ffffffff83166002145b80611755575063ffffffff83166006145b80611766575063ffffffff83166004145b156115d25760019450611780565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b0152600401909116905261136d610619565b60006117cb611b08565b506080600063ffffffff87166010036117e9575060c0810151611952565b8663ffffffff166011036118085763ffffffff861660c0830152611952565b8663ffffffff16601203611821575060a0810151611952565b8663ffffffff166013036118405763ffffffff861660a0830152611952565b8663ffffffff166018036118745763ffffffff600387810b9087900b02602081901c821660c08501521660a0830152611952565b8663ffffffff166019036118a55763ffffffff86811681871602602081901c821660c08501521660a0830152611952565b8663ffffffff16601a036118fb578460030b8660030b816118c8576118c8611c9a565b0763ffffffff1660c0830152600385810b9087900b816118ea576118ea611c9a565b0563ffffffff1660a0830152611952565b8663ffffffff16601b03611952578463ffffffff168663ffffffff168161192457611924611c9a565b0663ffffffff90811660c08401528581169087168161194557611945611c9a565b0463ffffffff1660a08301525b63ffffffff84161561198d57808261016001518563ffffffff166020811061197c5761197c611c47565b63ffffffff90921660209290920201525b60808201805163ffffffff808216606086015260049091011690526119b0610619565b979650505050505050565b60006119c683611a5f565b905060038416156119d657600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611a545760208401933582821c6001168015611a245760018114611a3957611a4a565b60008581526020839052604090209450611a4a565b600082815260208690526040902094505b50506001016119fc565b505060805250505050565b60ff811661038002610184810190369061050401811015611b02576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f61746100000000000000000000000000000000000000000000000000000000006064820152608401610807565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611b6e611b73565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611ba457600080fd5b50813567ffffffffffffffff811115611bbc57600080fd5b602083019150836020828501011115611bd457600080fd5b9250929050565b60008060008060408587031215611bf157600080fd5b843567ffffffffffffffff80821115611c0957600080fd5b611c1588838901611b92565b90965094506020870135915080821115611c2e57600080fd5b50611c3b87828801611b92565b95989497509550505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611c8957600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
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
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32)
func (_MIPS *MIPSTransactor) Step(opts *bind.TransactOpts, stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.contract.Transact(opts, "step", stateData, proof)
}

// Step is a paid mutator transaction binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32)
func (_MIPS *MIPSSession) Step(stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, stateData, proof)
}

// Step is a paid mutator transaction binding the contract method 0xf8e0cb96.
//
// Solidity: function step(bytes stateData, bytes proof) returns(bytes32)
func (_MIPS *MIPSTransactorSession) Step(stateData []byte, proof []byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, stateData, proof)
}

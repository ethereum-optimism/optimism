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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_oracle\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"BRK_START\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"oracle\",\"inputs\":[],\"outputs\":[{\"name\":\"oracle_\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"step\",\"inputs\":[{\"name\":\"_stateData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051611ec2380380611ec283398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051611e3161009160003960008181608501526115ef0152611e316000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063e14ced32146100af575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60405173ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610062565b6100c26100bd366004611d2e565b6100d0565b604051908152602001610062565b60006100da611c5b565b608081146100e757600080fd5b604051610600146100f757600080fd5b6084871461010457600080fd5b6101a4851461011257600080fd5b8635608052602087013560a052604087013560e090811c60c09081526044890135821c82526048890135821c61010052604c890135821c610120526050890135821c61014052605489013590911c61016052605888013560f890811c610180526059890135901c6101a052605a880135901c6101c0526102006101e0819052606288019060005b60208110156101bd57823560e01c8252600490920191602090910190600101610199565b505050806101200151156101db576101d361061b565b915050610612565b6101408101805160010167ffffffffffffffff16905260608101516000906102039082610737565b9050603f601a82901c16600281148061022257508063ffffffff166003145b156102775760006002836303ffffff1663ffffffff16901b846080015163f00000001617905061026c8263ffffffff1660021461026057601f610263565b60005b60ff16826107f3565b945050505050610612565b6101608301516000908190601f601086901c81169190601587901c16602081106102a3576102a3611da2565b602002015192508063ffffffff851615806102c457508463ffffffff16601c145b156102fb578661016001518263ffffffff16602081106102e6576102e6611da2565b6020020151925050601f600b86901c166103b7565b60208563ffffffff16101561035d578463ffffffff16600c148061032557508463ffffffff16600d145b8061033657508463ffffffff16600e145b15610347578561ffff1692506103b7565b6103568661ffff1660106108e4565b92506103b7565b60288563ffffffff1610158061037957508463ffffffff166022145b8061038a57508463ffffffff166026145b156103b7578661016001518263ffffffff16602081106103ac576103ac611da2565b602002015192508190505b60048563ffffffff16101580156103d4575060088563ffffffff16105b806103e557508463ffffffff166001145b15610404576103f685878487610957565b975050505050505050610612565b63ffffffff6000602087831610610469576104248861ffff1660106108e4565b9095019463fffffffc861661043a816001610737565b915060288863ffffffff161015801561045a57508763ffffffff16603014155b1561046757809250600093505b505b600061047789888885610b67565b63ffffffff9081169150603f8a1690891615801561049c575060088163ffffffff1610155b80156104ae5750601c8163ffffffff16105b1561058b578063ffffffff16600814806104ce57508063ffffffff166009145b15610505576104f38163ffffffff166008146104ea57856104ed565b60005b896107f3565b9b505050505050505050505050610612565b8063ffffffff16600a03610525576104f3858963ffffffff8a16156112f7565b8063ffffffff16600b03610546576104f3858963ffffffff8a1615156112f7565b8063ffffffff16600c0361055d576104f38d6113dd565b60108163ffffffff161015801561057a5750601c8163ffffffff16105b1561058b576104f381898988611914565b8863ffffffff1660381480156105a6575063ffffffff861615155b156105db5760018b61016001518763ffffffff16602081106105ca576105ca611da2565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff146105f8576105f884600184611b0e565b610604858360016112f7565b9b5050505050505050505050505b95945050505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c5160548201526101805161019f5160588301526101a0516101bf5160598401526101d851605a840152600092610200929091606283019190855b60208110156106ba57601c8601518452602090950194600490930192600101610696565b506000835283830384a06000945080600181146106da5760039550610702565b8280156106f257600181146106fb5760029650610700565b60009650610700565b600196505b505b50505081900390207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1660f89190911b17919050565b60008061074383611bb2565b9050600384161561075357600080fd5b6020810190358460051c8160005b601b8110156107b95760208501943583821c6001168015610789576001811461079e576107af565b600084815260208390526040902093506107af565b600082815260208590526040902093505b5050600101610761565b5060805191508181146107d457630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b60006107fd611c5b565b60809050806060015160040163ffffffff16816080015163ffffffff1614610886576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff9081169093528583169052908516156108dc57806008018261016001518663ffffffff16602081106108cb576108cb611da2565b63ffffffff90921660209290920201525b61061261061b565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b0182610941576000610943565b815b90861663ffffffff16179250505092915050565b6000610961611c5b565b608090506000816060015160040163ffffffff16826080015163ffffffff16146109e7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f74000000000000000000000000604482015260640161087d565b8663ffffffff1660041480610a0257508663ffffffff166005145b15610a7e5760008261016001518663ffffffff1660208110610a2657610a26611da2565b602002015190508063ffffffff168563ffffffff16148015610a4e57508763ffffffff166004145b80610a7657508063ffffffff168563ffffffff1614158015610a7657508763ffffffff166005145b915050610afb565b8663ffffffff16600603610a9b5760008460030b13159050610afb565b8663ffffffff16600703610ab75760008460030b139050610afb565b8663ffffffff16600103610afb57601f601087901c166000819003610ae05760008560030b1291505b8063ffffffff16600103610af95760008560030b121591505b505b606082018051608084015163ffffffff169091528115610b41576002610b268861ffff1660106108e4565b63ffffffff90811690911b8201600401166080840152610b53565b60808301805160040163ffffffff1690525b610b5b61061b565b98975050505050505050565b6000603f601a86901c16801580610b96575060088163ffffffff1610158015610b965750600f8163ffffffff16105b15610fec57603f86168160088114610bdd5760098114610be657600a8114610bef57600b8114610bf857600c8114610c0157600d8114610c0a57600e8114610c1357610c18565b60209150610c18565b60219150610c18565b602a9150610c18565b602b9150610c18565b60249150610c18565b60259150610c18565b602691505b508063ffffffff16600003610c3f5750505063ffffffff8216601f600686901c161b6112ef565b8063ffffffff16600203610c655750505063ffffffff8216601f600686901c161c6112ef565b8063ffffffff16600303610c9b57601f600688901c16610c9163ffffffff8716821c60208390036108e4565b93505050506112ef565b8063ffffffff16600403610cbd5750505063ffffffff8216601f84161b6112ef565b8063ffffffff16600603610cdf5750505063ffffffff8216601f84161c6112ef565b8063ffffffff16600703610d1257610d098663ffffffff168663ffffffff16901c876020036108e4565b925050506112ef565b8063ffffffff16600803610d2a5785925050506112ef565b8063ffffffff16600903610d425785925050506112ef565b8063ffffffff16600a03610d5a5785925050506112ef565b8063ffffffff16600b03610d725785925050506112ef565b8063ffffffff16600c03610d8a5785925050506112ef565b8063ffffffff16600f03610da25785925050506112ef565b8063ffffffff16601003610dba5785925050506112ef565b8063ffffffff16601103610dd25785925050506112ef565b8063ffffffff16601203610dea5785925050506112ef565b8063ffffffff16601303610e025785925050506112ef565b8063ffffffff16601803610e1a5785925050506112ef565b8063ffffffff16601903610e325785925050506112ef565b8063ffffffff16601a03610e4a5785925050506112ef565b8063ffffffff16601b03610e625785925050506112ef565b8063ffffffff16602003610e7b575050508282016112ef565b8063ffffffff16602103610e94575050508282016112ef565b8063ffffffff16602203610ead575050508183036112ef565b8063ffffffff16602303610ec6575050508183036112ef565b8063ffffffff16602403610edf575050508282166112ef565b8063ffffffff16602503610ef8575050508282176112ef565b8063ffffffff16602603610f11575050508282186112ef565b8063ffffffff16602703610f2b57505050828217196112ef565b8063ffffffff16602a03610f5c578460030b8660030b12610f4d576000610f50565b60015b60ff16925050506112ef565b8063ffffffff16602b03610f84578463ffffffff168663ffffffff1610610f4d576000610f50565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e00000000000000000000000000604482015260640161087d565b50610f84565b8063ffffffff16601c0361107057603f86166002819003611012575050508282026112ef565b8063ffffffff166020148061102d57508063ffffffff166021145b15610fe6578063ffffffff16602003611044579419945b60005b6380000000871615611066576401fffffffe600197881b169601611047565b92506112ef915050565b8063ffffffff16600f0361109257505065ffffffff0000601083901b166112ef565b8063ffffffff166020036110ce576110c68560031660080260180363ffffffff168463ffffffff16901c60ff1660086108e4565b9150506112ef565b8063ffffffff16602103611103576110c68560021660080260100363ffffffff168463ffffffff16901c61ffff1660106108e4565b8063ffffffff1660220361113257505063ffffffff60086003851602811681811b198416918316901b176112ef565b8063ffffffff1660230361114957829150506112ef565b8063ffffffff1660240361117b578460031660080260180363ffffffff168363ffffffff16901c60ff169150506112ef565b8063ffffffff166025036111ae578460021660080260100363ffffffff168363ffffffff16901c61ffff169150506112ef565b8063ffffffff166026036111e057505063ffffffff60086003851602601803811681811c198416918316901c176112ef565b8063ffffffff1660280361121657505060ff63ffffffff60086003861602601803811682811b9091188316918416901b176112ef565b8063ffffffff1660290361124d57505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b176112ef565b8063ffffffff16602a0361127c57505063ffffffff60086003851602811681811c198316918416901c176112ef565b8063ffffffff16602b0361129357839150506112ef565b8063ffffffff16602e036112c557505063ffffffff60086003851602601803811681811b198316918416901b176112ef565b8063ffffffff166030036112dc57829150506112ef565b8063ffffffff16603803610f8457839150505b949350505050565b6000611301611c5b565b506080602063ffffffff861610611374576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c6964207265676973746572000000000000000000000000000000000000604482015260640161087d565b63ffffffff8516158015906113865750825b156113ba57838161016001518663ffffffff16602081106113a9576113a9611da2565b63ffffffff90921660209290920201525b60808101805163ffffffff8082166060850152600490910116905261061261061b565b60006113e7611c5b565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036114615781610fff81161561143057610fff811661100003015b8363ffffffff166000036114575760e08801805163ffffffff83820116909152955061145b565b8395505b506118d3565b8563ffffffff16610fcd0361147c57634000000094506118d3565b8563ffffffff166110180361149457600194506118d3565b8563ffffffff16611096036114ca57600161012088015260ff83166101008801526114bd61061b565b9998505050505050505050565b8563ffffffff16610fa3036117365763ffffffff8316156118d3577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff8416016116f05760006115258363fffffffc166001610737565b60208901519091508060001a60010361159457604080516000838152336020528d83526060902091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790505b6040808a015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810183905263ffffffff9091166024820152600090819073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e03110e1906044016040805180830381865afa158015611635573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116599190611dd1565b91509150600386168060040382811015611671578092505b508186101561167e578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506116d58663fffffffc16600186611b0e565b60408b018051820163ffffffff169052975061173192505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff841601611725578094506118d3565b63ffffffff9450600993505b6118d3565b8563ffffffff16610fa4036118275763ffffffff831660011480611760575063ffffffff83166002145b80611771575063ffffffff83166004145b1561177e578094506118d3565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff8416016117255760006117be8363fffffffc166001610737565b602089015190915060038416600403838110156117d9578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b176020880152600060408801529350836118d3565b8563ffffffff16610fd7036118d3578163ffffffff166003036118c75763ffffffff8316158061185d575063ffffffff83166005145b8061186e575063ffffffff83166003145b1561187c57600094506118d3565b63ffffffff831660011480611897575063ffffffff83166002145b806118a8575063ffffffff83166006145b806118b9575063ffffffff83166004145b1561172557600194506118d3565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b015260040190911690526114bd61061b565b600061191e611c5b565b506080600063ffffffff871660100361193c575060c0810151611aa5565b8663ffffffff1660110361195b5763ffffffff861660c0830152611aa5565b8663ffffffff16601203611974575060a0810151611aa5565b8663ffffffff166013036119935763ffffffff861660a0830152611aa5565b8663ffffffff166018036119c75763ffffffff600387810b9087900b02602081901c821660c08501521660a0830152611aa5565b8663ffffffff166019036119f85763ffffffff86811681871602602081901c821660c08501521660a0830152611aa5565b8663ffffffff16601a03611a4e578460030b8660030b81611a1b57611a1b611df5565b0763ffffffff1660c0830152600385810b9087900b81611a3d57611a3d611df5565b0563ffffffff1660a0830152611aa5565b8663ffffffff16601b03611aa5578463ffffffff168663ffffffff1681611a7757611a77611df5565b0663ffffffff90811660c084015285811690871681611a9857611a98611df5565b0463ffffffff1660a08301525b63ffffffff841615611ae057808261016001518563ffffffff1660208110611acf57611acf611da2565b63ffffffff90921660209290920201525b60808201805163ffffffff80821660608601526004909101169052611b0361061b565b979650505050505050565b6000611b1983611bb2565b90506003841615611b2957600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611ba75760208401933582821c6001168015611b775760018114611b8c57611b9d565b60008581526020839052604090209450611b9d565b600082815260208690526040902094505b5050600101611b4f565b505060805250505050565b60ff8116610380026101a4810190369061052401811015611c55576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f6174610000000000000000000000000000000000000000000000000000000000606482015260840161087d565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611cc1611cc6565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611cf757600080fd5b50813567ffffffffffffffff811115611d0f57600080fd5b602083019150836020828501011115611d2757600080fd5b9250929050565b600080600080600060608688031215611d4657600080fd5b853567ffffffffffffffff80821115611d5e57600080fd5b611d6a89838a01611ce5565b90975095506020880135915080821115611d8357600080fd5b50611d9088828901611ce5565b96999598509660400135949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611de457600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
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

// Step is a paid mutator transaction binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes _proof, bytes32 _localContext) returns(bytes32)
func (_MIPS *MIPSTransactor) Step(opts *bind.TransactOpts, _stateData []byte, _proof []byte, _localContext [32]byte) (*types.Transaction, error) {
	return _MIPS.contract.Transact(opts, "step", _stateData, _proof, _localContext)
}

// Step is a paid mutator transaction binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes _proof, bytes32 _localContext) returns(bytes32)
func (_MIPS *MIPSSession) Step(_stateData []byte, _proof []byte, _localContext [32]byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, _stateData, _proof, _localContext)
}

// Step is a paid mutator transaction binding the contract method 0xe14ced32.
//
// Solidity: function step(bytes _stateData, bytes _proof, bytes32 _localContext) returns(bytes32)
func (_MIPS *MIPSTransactorSession) Step(_stateData []byte, _proof []byte, _localContext [32]byte) (*types.Transaction, error) {
	return _MIPS.Contract.Step(&_MIPS.TransactOpts, _stateData, _proof, _localContext)
}

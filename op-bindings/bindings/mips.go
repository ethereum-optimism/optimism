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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_oracle\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"BRK_START\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"oracle\",\"inputs\":[],\"outputs\":[{\"name\":\"oracle_\",\"type\":\"address\",\"internalType\":\"contractIPreimageOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"step\",\"inputs\":[{\"name\":\"_stateData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x60a060405234801561001057600080fd5b50604051611f89380380611f8983398101604081905261002f91610040565b6001600160a01b0316608052610070565b60006020828403121561005257600080fd5b81516001600160a01b038116811461006957600080fd5b9392505050565b608051611ef86100916000396000818160d901526116430152611ef86000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c8063155633fe1461005157806354fd4d50146100765780637dc0d1d0146100bf578063e14ced3214610103575b600080fd5b61005c634000000081565b60405163ffffffff90911681526020015b60405180910390f35b6100b26040518060400160405280600581526020017f302e312e3000000000000000000000000000000000000000000000000000000081525081565b60405161006d9190611d39565b60405173ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016815260200161006d565b610116610111366004611df5565b610124565b60405190815260200161006d565b600061012e611caf565b6080811461013b57600080fd5b6040516106001461014b57600080fd5b6084871461015857600080fd5b6101a4851461016657600080fd5b8635608052602087013560a052604087013560e090811c60c09081526044890135821c82526048890135821c61010052604c890135821c610120526050890135821c61014052605489013590911c61016052605888013560f890811c610180526059890135901c6101a052605a880135901c6101c0526102006101e0819052606288019060005b602081101561021157823560e01c82526004909201916020909101906001016101ed565b5050508061012001511561022f5761022761066f565b915050610666565b6101408101805160010167ffffffffffffffff1690526060810151600090610257908261078b565b9050603f601a82901c16600281148061027657508063ffffffff166003145b156102cb5760006002836303ffffff1663ffffffff16901b846080015163f0000000161790506102c08263ffffffff166002146102b457601f6102b7565b60005b60ff1682610847565b945050505050610666565b6101608301516000908190601f601086901c81169190601587901c16602081106102f7576102f7611e69565b602002015192508063ffffffff8516158061031857508463ffffffff16601c145b1561034f578661016001518263ffffffff166020811061033a5761033a611e69565b6020020151925050601f600b86901c1661040b565b60208563ffffffff1610156103b1578463ffffffff16600c148061037957508463ffffffff16600d145b8061038a57508463ffffffff16600e145b1561039b578561ffff16925061040b565b6103aa8661ffff166010610938565b925061040b565b60288563ffffffff161015806103cd57508463ffffffff166022145b806103de57508463ffffffff166026145b1561040b578661016001518263ffffffff166020811061040057610400611e69565b602002015192508190505b60048563ffffffff1610158015610428575060088563ffffffff16105b8061043957508463ffffffff166001145b156104585761044a858784876109ab565b975050505050505050610666565b63ffffffff60006020878316106104bd576104788861ffff166010610938565b9095019463fffffffc861661048e81600161078b565b915060288863ffffffff16101580156104ae57508763ffffffff16603014155b156104bb57809250600093505b505b60006104cb89888885610bbb565b63ffffffff9081169150603f8a169089161580156104f0575060088163ffffffff1610155b80156105025750601c8163ffffffff16105b156105df578063ffffffff166008148061052257508063ffffffff166009145b15610559576105478163ffffffff1660081461053e5785610541565b60005b89610847565b9b505050505050505050505050610666565b8063ffffffff16600a0361057957610547858963ffffffff8a161561134b565b8063ffffffff16600b0361059a57610547858963ffffffff8a16151561134b565b8063ffffffff16600c036105b1576105478d611431565b60108163ffffffff16101580156105ce5750601c8163ffffffff16105b156105df5761054781898988611968565b8863ffffffff1660381480156105fa575063ffffffff861615155b1561062f5760018b61016001518763ffffffff166020811061061e5761061e611e69565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff1461064c5761064c84600184611b62565b6106588583600161134b565b9b5050505050505050505050505b95945050505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c5160548201526101805161019f5160588301526101a0516101bf5160598401526101d851605a840152600092610200929091606283019190855b602081101561070e57601c86015184526020909501946004909301926001016106ea565b506000835283830384a060009450806001811461072e5760039550610756565b828015610746576001811461074f5760029650610754565b60009650610754565b600196505b505b50505081900390207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1660f89190911b17919050565b60008061079783611c06565b905060038416156107a757600080fd5b6020810190358460051c8160005b601b81101561080d5760208501943583821c60011680156107dd57600181146107f257610803565b60008481526020839052604090209350610803565b600082815260208590526040902093505b50506001016107b5565b50608051915081811461082857630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b6000610851611caf565b60809050806060015160040163ffffffff16816080015163ffffffff16146108da576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff90811690935285831690529085161561093057806008018261016001518663ffffffff166020811061091f5761091f611e69565b63ffffffff90921660209290920201525b61066661066f565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b0182610995576000610997565b815b90861663ffffffff16179250505092915050565b60006109b5611caf565b608090506000816060015160040163ffffffff16826080015163ffffffff1614610a3b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f7400000000000000000000000060448201526064016108d1565b8663ffffffff1660041480610a5657508663ffffffff166005145b15610ad25760008261016001518663ffffffff1660208110610a7a57610a7a611e69565b602002015190508063ffffffff168563ffffffff16148015610aa257508763ffffffff166004145b80610aca57508063ffffffff168563ffffffff1614158015610aca57508763ffffffff166005145b915050610b4f565b8663ffffffff16600603610aef5760008460030b13159050610b4f565b8663ffffffff16600703610b0b5760008460030b139050610b4f565b8663ffffffff16600103610b4f57601f601087901c166000819003610b345760008560030b1291505b8063ffffffff16600103610b4d5760008560030b121591505b505b606082018051608084015163ffffffff169091528115610b95576002610b7a8861ffff166010610938565b63ffffffff90811690911b8201600401166080840152610ba7565b60808301805160040163ffffffff1690525b610baf61066f565b98975050505050505050565b6000603f601a86901c16801580610bea575060088163ffffffff1610158015610bea5750600f8163ffffffff16105b1561104057603f86168160088114610c315760098114610c3a57600a8114610c4357600b8114610c4c57600c8114610c5557600d8114610c5e57600e8114610c6757610c6c565b60209150610c6c565b60219150610c6c565b602a9150610c6c565b602b9150610c6c565b60249150610c6c565b60259150610c6c565b602691505b508063ffffffff16600003610c935750505063ffffffff8216601f600686901c161b611343565b8063ffffffff16600203610cb95750505063ffffffff8216601f600686901c161c611343565b8063ffffffff16600303610cef57601f600688901c16610ce563ffffffff8716821c6020839003610938565b9350505050611343565b8063ffffffff16600403610d115750505063ffffffff8216601f84161b611343565b8063ffffffff16600603610d335750505063ffffffff8216601f84161c611343565b8063ffffffff16600703610d6657610d5d8663ffffffff168663ffffffff16901c87602003610938565b92505050611343565b8063ffffffff16600803610d7e578592505050611343565b8063ffffffff16600903610d96578592505050611343565b8063ffffffff16600a03610dae578592505050611343565b8063ffffffff16600b03610dc6578592505050611343565b8063ffffffff16600c03610dde578592505050611343565b8063ffffffff16600f03610df6578592505050611343565b8063ffffffff16601003610e0e578592505050611343565b8063ffffffff16601103610e26578592505050611343565b8063ffffffff16601203610e3e578592505050611343565b8063ffffffff16601303610e56578592505050611343565b8063ffffffff16601803610e6e578592505050611343565b8063ffffffff16601903610e86578592505050611343565b8063ffffffff16601a03610e9e578592505050611343565b8063ffffffff16601b03610eb6578592505050611343565b8063ffffffff16602003610ecf57505050828201611343565b8063ffffffff16602103610ee857505050828201611343565b8063ffffffff16602203610f0157505050818303611343565b8063ffffffff16602303610f1a57505050818303611343565b8063ffffffff16602403610f3357505050828216611343565b8063ffffffff16602503610f4c57505050828217611343565b8063ffffffff16602603610f6557505050828218611343565b8063ffffffff16602703610f7f5750505082821719611343565b8063ffffffff16602a03610fb0578460030b8660030b12610fa1576000610fa4565b60015b60ff1692505050611343565b8063ffffffff16602b03610fd8578463ffffffff168663ffffffff1610610fa1576000610fa4565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e0000000000000000000000000060448201526064016108d1565b50610fd8565b8063ffffffff16601c036110c457603f8616600281900361106657505050828202611343565b8063ffffffff166020148061108157508063ffffffff166021145b1561103a578063ffffffff16602003611098579419945b60005b63800000008716156110ba576401fffffffe600197881b16960161109b565b9250611343915050565b8063ffffffff16600f036110e657505065ffffffff0000601083901b16611343565b8063ffffffff166020036111225761111a8560031660080260180363ffffffff168463ffffffff16901c60ff166008610938565b915050611343565b8063ffffffff166021036111575761111a8560021660080260100363ffffffff168463ffffffff16901c61ffff166010610938565b8063ffffffff1660220361118657505063ffffffff60086003851602811681811b198416918316901b17611343565b8063ffffffff1660230361119d5782915050611343565b8063ffffffff166024036111cf578460031660080260180363ffffffff168363ffffffff16901c60ff16915050611343565b8063ffffffff16602503611202578460021660080260100363ffffffff168363ffffffff16901c61ffff16915050611343565b8063ffffffff1660260361123457505063ffffffff60086003851602601803811681811c198416918316901c17611343565b8063ffffffff1660280361126a57505060ff63ffffffff60086003861602601803811682811b9091188316918416901b17611343565b8063ffffffff166029036112a157505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b17611343565b8063ffffffff16602a036112d057505063ffffffff60086003851602811681811c198316918416901c17611343565b8063ffffffff16602b036112e75783915050611343565b8063ffffffff16602e0361131957505063ffffffff60086003851602601803811681811b198316918416901b17611343565b8063ffffffff166030036113305782915050611343565b8063ffffffff16603803610fd857839150505b949350505050565b6000611355611caf565b506080602063ffffffff8616106113c8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c696420726567697374657200000000000000000000000000000000000060448201526064016108d1565b63ffffffff8516158015906113da5750825b1561140e57838161016001518663ffffffff16602081106113fd576113fd611e69565b63ffffffff90921660209290920201525b60808101805163ffffffff8082166060850152600490910116905261066661066f565b600061143b611caf565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036114b55781610fff81161561148457610fff811661100003015b8363ffffffff166000036114ab5760e08801805163ffffffff8382011690915295506114af565b8395505b50611927565b8563ffffffff16610fcd036114d05763400000009450611927565b8563ffffffff16611018036114e85760019450611927565b8563ffffffff166110960361151e57600161012088015260ff831661010088015261151161066f565b9998505050505050505050565b8563ffffffff16610fa30361178a5763ffffffff831615611927577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff8416016117445760006115798363fffffffc16600161078b565b60208901519091508060001a6001036115e857604080516000838152336020528d83526060902091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790505b6040808a015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810183905263ffffffff9091166024820152600090819073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063e03110e1906044016040805180830381865afa158015611689573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116ad9190611e98565b915091506003861680600403828110156116c5578092505b50818610156116d2578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506117298663fffffffc16600186611b62565b60408b018051820163ffffffff169052975061178592505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff84160161177957809450611927565b63ffffffff9450600993505b611927565b8563ffffffff16610fa40361187b5763ffffffff8316600114806117b4575063ffffffff83166002145b806117c5575063ffffffff83166004145b156117d257809450611927565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff8416016117795760006118128363fffffffc16600161078b565b6020890151909150600384166004038381101561182d578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b17602088015260006040880152935083611927565b8563ffffffff16610fd703611927578163ffffffff1660030361191b5763ffffffff831615806118b1575063ffffffff83166005145b806118c2575063ffffffff83166003145b156118d05760009450611927565b63ffffffff8316600114806118eb575063ffffffff83166002145b806118fc575063ffffffff83166006145b8061190d575063ffffffff83166004145b156117795760019450611927565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b0152600401909116905261151161066f565b6000611972611caf565b506080600063ffffffff8716601003611990575060c0810151611af9565b8663ffffffff166011036119af5763ffffffff861660c0830152611af9565b8663ffffffff166012036119c8575060a0810151611af9565b8663ffffffff166013036119e75763ffffffff861660a0830152611af9565b8663ffffffff16601803611a1b5763ffffffff600387810b9087900b02602081901c821660c08501521660a0830152611af9565b8663ffffffff16601903611a4c5763ffffffff86811681871602602081901c821660c08501521660a0830152611af9565b8663ffffffff16601a03611aa2578460030b8660030b81611a6f57611a6f611ebc565b0763ffffffff1660c0830152600385810b9087900b81611a9157611a91611ebc565b0563ffffffff1660a0830152611af9565b8663ffffffff16601b03611af9578463ffffffff168663ffffffff1681611acb57611acb611ebc565b0663ffffffff90811660c084015285811690871681611aec57611aec611ebc565b0463ffffffff1660a08301525b63ffffffff841615611b3457808261016001518563ffffffff1660208110611b2357611b23611e69565b63ffffffff90921660209290920201525b60808201805163ffffffff80821660608601526004909101169052611b5761066f565b979650505050505050565b6000611b6d83611c06565b90506003841615611b7d57600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611bfb5760208401933582821c6001168015611bcb5760018114611be057611bf1565b60008581526020839052604090209450611bf1565b600082815260208690526040902094505b5050600101611ba3565b505060805250505050565b60ff8116610380026101a4810190369061052401811015611ca9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f617461000000000000000000000000000000000000000000000000000000000060648201526084016108d1565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611d15611d1a565b905290565b6040518061040001604052806020906020820280368337509192915050565b600060208083528351808285015260005b81811015611d6657858101830151858201604001528201611d4a565b81811115611d78576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60008083601f840112611dbe57600080fd5b50813567ffffffffffffffff811115611dd657600080fd5b602083019150836020828501011115611dee57600080fd5b9250929050565b600080600080600060608688031215611e0d57600080fd5b853567ffffffffffffffff80821115611e2557600080fd5b611e3189838a01611dac565b90975095506020880135915080821115611e4a57600080fd5b50611e5788828901611dac565b96999598509660400135949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611eab57600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
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

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MIPS *MIPSCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _MIPS.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MIPS *MIPSSession) Version() (string, error) {
	return _MIPS.Contract.Version(&_MIPS.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_MIPS *MIPSCallerSession) Version() (string, error) {
	return _MIPS.Contract.Version(&_MIPS.CallOpts)
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

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
	ABI: "[{\"inputs\":[],\"name\":\"BRK_START\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"oracle\",\"outputs\":[{\"internalType\":\"contractIPreimageOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proof\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611ba0806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063f8e0cb96146100b0575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60005461008b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610062565b6100c36100be366004611aa5565b6100d1565b604051908152602001610062565b60006100db6119d2565b608081146100e857600080fd5b604051610600146100f857600080fd5b6064861461010557600080fd5b610166841461011357600080fd5b8535608052602086013560a052604086013560e090811c60c09081526044880135821c82526048880135821c61010052604c880135821c610120526050880135821c61014052605488013590911c61016052605887013560f890811c610180526059880135901c6101a052605a870135901c6101c0526102006101e0819052606287019060005b60208110156101be57823560e01c825260049092019160209091019060010161019a565b505050806101200151156101dc576101d4610612565b91505061060a565b6101408101805160010167ffffffffffffffff169052606081015160009061020490826106ba565b9050603f601a82901c16600281148061022357508063ffffffff166003145b15610270576102668163ffffffff1660021461024057601f610243565b60005b60ff166002610259856303ffffff16601a610776565b63ffffffff16901b6107e9565b935050505061060a565b6101608301516000908190601f601086901c81169190601587901c166020811061029c5761029c611b11565b602002015192508063ffffffff851615806102bd57508463ffffffff16601c145b156102f4578661016001518263ffffffff16602081106102df576102df611b11565b6020020151925050601f600b86901c166103b0565b60208563ffffffff161015610356578463ffffffff16600c148061031e57508463ffffffff16600d145b8061032f57508463ffffffff16600e145b15610340578561ffff1692506103b0565b61034f8661ffff166010610776565b92506103b0565b60288563ffffffff1610158061037257508463ffffffff166022145b8061038357508463ffffffff166026145b156103b0578661016001518263ffffffff16602081106103a5576103a5611b11565b602002015192508190505b60048563ffffffff16101580156103cd575060088563ffffffff16105b806103de57508463ffffffff166001145b156103fd576103ef8587848761085a565b97505050505050505061060a565b63ffffffff60006020878316106104625761041d8861ffff166010610776565b9095019463fffffffc86166104338160016106ba565b915060288863ffffffff161015801561045357508763ffffffff16603014155b1561046057809250600093505b505b6000610470898888856109e9565b63ffffffff9081169150603f8a16908916158015610495575060088163ffffffff1610155b80156104a75750601c8163ffffffff16105b15610583578063ffffffff16600814806104c757508063ffffffff166009145b156104fe576104ec8163ffffffff166008146104e357856104e6565b60005b896107e9565b9b50505050505050505050505061060a565b8063ffffffff16600a0361051e576104ec858963ffffffff8a1615611091565b8063ffffffff16600b0361053f576104ec858963ffffffff8a161515611091565b8063ffffffff16600c03610555576104ec611177565b60108163ffffffff16101580156105725750601c8163ffffffff16105b15610583576104ec8189898861168b565b8863ffffffff16603814801561059e575063ffffffff861615155b156105d35760018b61016001518763ffffffff16602081106105c2576105c2611b11565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff146105f0576105f084600184611885565b6105fc85836001611091565b9b5050505050505050505050505b949350505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c51605482015261019f5160588201526101bf5160598201526101d851605a8201526000906102009060628101835b60208110156106a557601c8401518252602090930192600490910190600101610681565b506000815281810382a0819003902092915050565b6000806106c683611929565b905060038416156106d657600080fd5b6020810190358460051c8160005b601b81101561073c5760208501943583821c600116801561070c576001811461072157610732565b60008481526020839052604090209350610732565b600082815260208590526040902093505b50506001016106e4565b50608051915081811461075757630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b01826107d35760006107d5565b815b90861663ffffffff16179250505092915050565b60006107f36119d2565b5060e08051610100805163ffffffff908116909352848316905260809185161561084957806008018261016001518663ffffffff166020811061083857610838611b11565b63ffffffff90921660209290920201525b610851610612565b95945050505050565b60006108646119d2565b5060806000600463ffffffff8816148061088457508663ffffffff166005145b156109005760008261016001518663ffffffff16602081106108a8576108a8611b11565b602002015190508063ffffffff168563ffffffff161480156108d057508763ffffffff166004145b806108f857508063ffffffff168563ffffffff16141580156108f857508763ffffffff166005145b91505061097d565b8663ffffffff1660060361091d5760008460030b1315905061097d565b8663ffffffff166007036109395760008460030b13905061097d565b8663ffffffff1660010361097d57601f601087901c1660008190036109625760008560030b1291505b8063ffffffff1660010361097b5760008560030b121591505b505b606082018051608084015163ffffffff1690915281156109c35760026109a88861ffff166010610776565b63ffffffff90811690911b82016004011660808401526109d5565b60808301805160040163ffffffff1690525b6109dd610612565b98975050505050505050565b6000603f601a86901c81169086166020821015610dad5760088263ffffffff1610158015610a1d5750600f8263ffffffff16105b15610abd578163ffffffff16600803610a3857506020610ab8565b8163ffffffff16600903610a4e57506021610ab8565b8163ffffffff16600a03610a645750602a610ab8565b8163ffffffff16600b03610a7a5750602b610ab8565b8163ffffffff16600c03610a9057506024610ab8565b8163ffffffff16600d03610aa657506025610ab8565b8163ffffffff16600e03610ab8575060265b600091505b8163ffffffff16600003610d0157601f600688901c16602063ffffffff83161015610bdb5760088263ffffffff1610610afb5786935050505061060a565b8163ffffffff16600003610b1e5763ffffffff86811691161b925061060a915050565b8163ffffffff16600203610b415763ffffffff86811691161c925061060a915050565b8163ffffffff16600303610b6b576102668163ffffffff168763ffffffff16901c82602003610776565b8163ffffffff16600403610b8e575050505063ffffffff8216601f84161b61060a565b8163ffffffff16600603610bb1575050505063ffffffff8216601f84161c61060a565b8163ffffffff16600703610bdb576102668763ffffffff168763ffffffff16901c88602003610776565b8163ffffffff1660201480610bf657508163ffffffff166021145b15610c0857858701935050505061060a565b8163ffffffff1660221480610c2357508163ffffffff166023145b15610c3557858703935050505061060a565b8163ffffffff16602403610c5057858716935050505061060a565b8163ffffffff16602503610c6b57858717935050505061060a565b8163ffffffff16602603610c8657858718935050505061060a565b8163ffffffff16602703610ca157505050508282171961060a565b8163ffffffff16602a03610cd3578560030b8760030b12610cc3576000610cc6565b60015b60ff16935050505061060a565b8163ffffffff16602b03610cfb578563ffffffff168763ffffffff1610610cc3576000610cc6565b5061102a565b8163ffffffff16600f03610d235760108563ffffffff16901b9250505061060a565b8163ffffffff16601c03610da8578063ffffffff16600203610d4a5750505082820261060a565b8063ffffffff1660201480610d6557508063ffffffff166021145b15610da8578063ffffffff16602003610d7c579419945b60005b6380000000871615610d9e576401fffffffe600197881b169601610d7f565b925061060a915050565b61102a565b60288263ffffffff161015610f10578163ffffffff16602003610df957610df08660031660080260180363ffffffff168563ffffffff16901c60ff166008610776565b9250505061060a565b8163ffffffff16602103610e2e57610df08660021660080260100363ffffffff168563ffffffff16901c61ffff166010610776565b8163ffffffff16602203610e5e5750505063ffffffff60086003851602811681811b198416918316901b1761060a565b8163ffffffff16602303610e7657839250505061060a565b8163ffffffff16602403610ea9578560031660080260180363ffffffff168463ffffffff16901c60ff169250505061060a565b8163ffffffff16602503610edd578560021660080260100363ffffffff168463ffffffff16901c61ffff169250505061060a565b8163ffffffff16602603610da85750505063ffffffff60086003851602601803811681811c198416918316901c1761060a565b8163ffffffff16602803610f475750505060ff63ffffffff60086003861602601803811682811b9091188316918416901b1761060a565b8163ffffffff16602903610f7f5750505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b1761060a565b8163ffffffff16602a03610faf5750505063ffffffff60086003851602811681811c198316918416901c1761060a565b8163ffffffff16602b03610fc757849250505061060a565b8163ffffffff16602e03610ffa5750505063ffffffff60086003851602601803811681811b198316918416901b1761060a565b8163ffffffff1660300361101257839250505061060a565b8163ffffffff1660380361102a57849250505061060a565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e0000000000000000000000000060448201526064015b60405180910390fd5b600061109b6119d2565b506080602063ffffffff86161061110e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c69642072656769737465720000000000000000000000000000000000006044820152606401611088565b63ffffffff8516158015906111205750825b1561115457838161016001518663ffffffff166020811061114357611143611b11565b63ffffffff90921660209290920201525b60808101805163ffffffff80821660608501526004909101169052610851610612565b60006111816119d2565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036111fb5781610fff8116156111ca57610fff811661100003015b8363ffffffff166000036111f15760e08801805163ffffffff8382011690915295506111f5565b8395505b5061164a565b8563ffffffff16610fcd03611216576340000000945061164a565b8563ffffffff166110180361122e576001945061164a565b8563ffffffff166110960361126357600161012088015260ff8316610100880152611257610612565b97505050505050505090565b8563ffffffff16610fa3036114ad5763ffffffff83161561164a577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff8416016114675760006112be8363fffffffc1660016106ba565b60208901519091508060001a60010361132b5761132881600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b90505b6000805460408b81015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810185905263ffffffff9091166024820152829173ffffffffffffffffffffffffffffffffffffffff169063e03110e1906044016040805180830381865afa1580156113ac573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113d09190611b40565b915091506003861680600403828110156113e8578092505b50818610156113f5578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b0391508119811690508381198716179550505061144c8663fffffffc16600186611885565b60408b018051820163ffffffff16905297506114a892505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff84160161149c5780945061164a565b63ffffffff9450600993505b61164a565b8563ffffffff16610fa40361159e5763ffffffff8316600114806114d7575063ffffffff83166002145b806114e8575063ffffffff83166004145b156114f55780945061164a565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff84160161149c5760006115358363fffffffc1660016106ba565b60208901519091506003841660040383811015611550578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b1760208801526000604088015293508361164a565b8563ffffffff16610fd70361164a578163ffffffff1660030361163e5763ffffffff831615806115d4575063ffffffff83166005145b806115e5575063ffffffff83166003145b156115f3576000945061164a565b63ffffffff83166001148061160e575063ffffffff83166002145b8061161f575063ffffffff83166006145b80611630575063ffffffff83166004145b1561149c576001945061164a565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b01526004019091169052611257610612565b60006116956119d2565b506080600063ffffffff87166010036116b3575060c081015161181c565b8663ffffffff166011036116d25763ffffffff861660c083015261181c565b8663ffffffff166012036116eb575060a081015161181c565b8663ffffffff1660130361170a5763ffffffff861660a083015261181c565b8663ffffffff1660180361173e5763ffffffff600387810b9087900b02602081901c821660c08501521660a083015261181c565b8663ffffffff1660190361176f5763ffffffff86811681871602602081901c821660c08501521660a083015261181c565b8663ffffffff16601a036117c5578460030b8660030b8161179257611792611b64565b0763ffffffff1660c0830152600385810b9087900b816117b4576117b4611b64565b0563ffffffff1660a083015261181c565b8663ffffffff16601b0361181c578463ffffffff168663ffffffff16816117ee576117ee611b64565b0663ffffffff90811660c08401528581169087168161180f5761180f611b64565b0463ffffffff1660a08301525b63ffffffff84161561185757808261016001518563ffffffff166020811061184657611846611b11565b63ffffffff90921660209290920201525b60808201805163ffffffff8082166060860152600490910116905261187a610612565b979650505050505050565b600061189083611929565b905060038416156118a057600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b81101561191e5760208401933582821c60011680156118ee576001811461190357611914565b60008581526020839052604090209450611914565b600082815260208690526040902094505b50506001016118c6565b505060805250505050565b60ff81166103800261016681019036906104e6018110156119cc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f61746100000000000000000000000000000000000000000000000000000000006064820152608401611088565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611a38611a3d565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611a6e57600080fd5b50813567ffffffffffffffff811115611a8657600080fd5b602083019150836020828501011115611a9e57600080fd5b9250929050565b60008060008060408587031215611abb57600080fd5b843567ffffffffffffffff80821115611ad357600080fd5b611adf88838901611a5c565b90965094506020870135915080821115611af857600080fd5b50611b0587828801611a5c565b95989497509550505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611b5357600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
}

// MIPSABI is the input ABI used to generate the binding from.
// Deprecated: Use MIPSMetaData.ABI instead.
var MIPSABI = MIPSMetaData.ABI

// MIPSBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MIPSMetaData.Bin instead.
var MIPSBin = MIPSMetaData.Bin

// DeployMIPS deploys a new Ethereum contract, binding an instance of MIPS to it.
func DeployMIPS(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MIPS, error) {
	parsed, err := MIPSMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MIPSBin), backend)
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
// Solidity: function oracle() view returns(address)
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
// Solidity: function oracle() view returns(address)
func (_MIPS *MIPSSession) Oracle() (common.Address, error) {
	return _MIPS.Contract.Oracle(&_MIPS.CallOpts)
}

// Oracle is a free data retrieval call binding the contract method 0x7dc0d1d0.
//
// Solidity: function oracle() view returns(address)
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

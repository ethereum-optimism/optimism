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
	Bin: "0x608060405234801561001057600080fd5b50611ca5806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c8063155633fe146100465780637dc0d1d01461006b578063f8e0cb96146100b0575b600080fd5b610051634000000081565b60405163ffffffff90911681526020015b60405180910390f35b60005461008b9073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610062565b6100c36100be366004611baa565b6100d1565b604051908152602001610062565b60006100db611ad7565b608081146100e857600080fd5b604051610600146100f857600080fd5b6064861461010557600080fd5b610166841461011357600080fd5b8535608052602086013560a052604086013560e090811c60c09081526044880135821c82526048880135821c61010052604c880135821c610120526050880135821c61014052605488013590911c61016052605887013560f890811c610180526059880135901c6101a052605a870135901c6101c0526102006101e0819052606287019060005b60208110156101be57823560e01c825260049092019160209091019060010161019a565b505050806101200151156101dc576101d4610612565b91505061060a565b6101408101805160010167ffffffffffffffff169052606081015160009061020490826106ba565b9050603f601a82901c16600281148061022357508063ffffffff166003145b15610270576102668163ffffffff1660021461024057601f610243565b60005b60ff166002610259856303ffffff16601a610776565b63ffffffff16901b6107e9565b935050505061060a565b6101608301516000908190601f601086901c81169190601587901c166020811061029c5761029c611c16565b602002015192508063ffffffff851615806102bd57508463ffffffff16601c145b156102f4578661016001518263ffffffff16602081106102df576102df611c16565b6020020151925050601f600b86901c166103b0565b60208563ffffffff161015610356578463ffffffff16600c148061031e57508463ffffffff16600d145b8061032f57508463ffffffff16600e145b15610340578561ffff1692506103b0565b61034f8661ffff166010610776565b92506103b0565b60288563ffffffff1610158061037257508463ffffffff166022145b8061038357508463ffffffff166026145b156103b0578661016001518263ffffffff16602081106103a5576103a5611c16565b602002015192508190505b60048563ffffffff16101580156103cd575060088563ffffffff16105b806103de57508463ffffffff166001145b156103fd576103ef858784876108e3565b97505050505050505061060a565b63ffffffff60006020878316106104625761041d8861ffff166010610776565b9095019463fffffffc86166104338160016106ba565b915060288863ffffffff161015801561045357508763ffffffff16603014155b1561046057809250600093505b505b600061047089888885610af3565b63ffffffff9081169150603f8a16908916158015610495575060088163ffffffff1610155b80156104a75750601c8163ffffffff16105b15610583578063ffffffff16600814806104c757508063ffffffff166009145b156104fe576104ec8163ffffffff166008146104e357856104e6565b60005b896107e9565b9b50505050505050505050505061060a565b8063ffffffff16600a0361051e576104ec858963ffffffff8a1615611196565b8063ffffffff16600b0361053f576104ec858963ffffffff8a161515611196565b8063ffffffff16600c03610555576104ec61127c565b60108163ffffffff16101580156105725750601c8163ffffffff16105b15610583576104ec81898988611790565b8863ffffffff16603814801561059e575063ffffffff861615155b156105d35760018b61016001518763ffffffff16602081106105c2576105c2611c16565b63ffffffff90921660209290920201525b8363ffffffff1663ffffffff146105f0576105f08460018461198a565b6105fc85836001611196565b9b5050505050505050505050505b949350505050565b60408051608051815260a051602082015260dc519181019190915260fc51604482015261011c51604882015261013c51604c82015261015c51605082015261017c51605482015261019f5160588201526101bf5160598201526101d851605a8201526000906102009060628101835b60208110156106a557601c8401518252602090930192600490910190600101610681565b506000815281810382a0819003902092915050565b6000806106c683611a2e565b905060038416156106d657600080fd5b6020810190358460051c8160005b601b81101561073c5760208501943583821c600116801561070c576001811461072157610732565b60008481526020839052604090209350610732565b600082815260208590526040902093505b50506001016106e4565b50608051915081811461075757630badf00d60005260206000fd5b5050601f94909416601c0360031b9390931c63ffffffff169392505050565b600063ffffffff8381167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80850183169190911c821615159160016020869003821681901b830191861691821b92911b01826107d35760006107d5565b815b90861663ffffffff16179250505092915050565b60006107f3611ad7565b60809050806060015160040163ffffffff16816080015163ffffffff161461087c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601260248201527f6a756d7020696e2064656c617920736c6f74000000000000000000000000000060448201526064015b60405180910390fd5b60608101805160808301805163ffffffff9081169093528583169052908516156108d257806008018261016001518663ffffffff16602081106108c1576108c1611c16565b63ffffffff90921660209290920201525b6108da610612565b95945050505050565b60006108ed611ad7565b608090506000816060015160040163ffffffff16826080015163ffffffff1614610973576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f6272616e636820696e2064656c617920736c6f740000000000000000000000006044820152606401610873565b8663ffffffff166004148061098e57508663ffffffff166005145b15610a0a5760008261016001518663ffffffff16602081106109b2576109b2611c16565b602002015190508063ffffffff168563ffffffff161480156109da57508763ffffffff166004145b80610a0257508063ffffffff168563ffffffff1614158015610a0257508763ffffffff166005145b915050610a87565b8663ffffffff16600603610a275760008460030b13159050610a87565b8663ffffffff16600703610a435760008460030b139050610a87565b8663ffffffff16600103610a8757601f601087901c166000819003610a6c5760008560030b1291505b8063ffffffff16600103610a855760008560030b121591505b505b606082018051608084015163ffffffff169091528115610acd576002610ab28861ffff166010610776565b63ffffffff90811690911b8201600401166080840152610adf565b60808301805160040163ffffffff1690525b610ae7610612565b98975050505050505050565b6000603f601a86901c81169086166020821015610eb75760088263ffffffff1610158015610b275750600f8263ffffffff16105b15610bc7578163ffffffff16600803610b4257506020610bc2565b8163ffffffff16600903610b5857506021610bc2565b8163ffffffff16600a03610b6e5750602a610bc2565b8163ffffffff16600b03610b845750602b610bc2565b8163ffffffff16600c03610b9a57506024610bc2565b8163ffffffff16600d03610bb057506025610bc2565b8163ffffffff16600e03610bc2575060265b600091505b8163ffffffff16600003610e0b57601f600688901c16602063ffffffff83161015610ce55760088263ffffffff1610610c055786935050505061060a565b8163ffffffff16600003610c285763ffffffff86811691161b925061060a915050565b8163ffffffff16600203610c4b5763ffffffff86811691161c925061060a915050565b8163ffffffff16600303610c75576102668163ffffffff168763ffffffff16901c82602003610776565b8163ffffffff16600403610c98575050505063ffffffff8216601f84161b61060a565b8163ffffffff16600603610cbb575050505063ffffffff8216601f84161c61060a565b8163ffffffff16600703610ce5576102668763ffffffff168763ffffffff16901c88602003610776565b8163ffffffff1660201480610d0057508163ffffffff166021145b15610d1257858701935050505061060a565b8163ffffffff1660221480610d2d57508163ffffffff166023145b15610d3f57858703935050505061060a565b8163ffffffff16602403610d5a57858716935050505061060a565b8163ffffffff16602503610d7557858717935050505061060a565b8163ffffffff16602603610d9057858718935050505061060a565b8163ffffffff16602703610dab57505050508282171961060a565b8163ffffffff16602a03610ddd578560030b8760030b12610dcd576000610dd0565b60015b60ff16935050505061060a565b8163ffffffff16602b03610e05578563ffffffff168763ffffffff1610610dcd576000610dd0565b50611134565b8163ffffffff16600f03610e2d5760108563ffffffff16901b9250505061060a565b8163ffffffff16601c03610eb2578063ffffffff16600203610e545750505082820261060a565b8063ffffffff1660201480610e6f57508063ffffffff166021145b15610eb2578063ffffffff16602003610e86579419945b60005b6380000000871615610ea8576401fffffffe600197881b169601610e89565b925061060a915050565b611134565b60288263ffffffff16101561101a578163ffffffff16602003610f0357610efa8660031660080260180363ffffffff168563ffffffff16901c60ff166008610776565b9250505061060a565b8163ffffffff16602103610f3857610efa8660021660080260100363ffffffff168563ffffffff16901c61ffff166010610776565b8163ffffffff16602203610f685750505063ffffffff60086003851602811681811b198416918316901b1761060a565b8163ffffffff16602303610f8057839250505061060a565b8163ffffffff16602403610fb3578560031660080260180363ffffffff168463ffffffff16901c60ff169250505061060a565b8163ffffffff16602503610fe7578560021660080260100363ffffffff168463ffffffff16901c61ffff169250505061060a565b8163ffffffff16602603610eb25750505063ffffffff60086003851602601803811681811c198416918316901c1761060a565b8163ffffffff166028036110515750505060ff63ffffffff60086003861602601803811682811b9091188316918416901b1761060a565b8163ffffffff166029036110895750505061ffff63ffffffff60086002861602601003811682811b9091188316918416901b1761060a565b8163ffffffff16602a036110b95750505063ffffffff60086003851602811681811c198316918416901c1761060a565b8163ffffffff16602b036110d157849250505061060a565b8163ffffffff16602e036111045750505063ffffffff60086003851602601803811681811b198316918416901b1761060a565b8163ffffffff1660300361111c57839250505061060a565b8163ffffffff1660380361113457849250505061060a565b6040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601360248201527f696e76616c696420696e737472756374696f6e000000000000000000000000006044820152606401610873565b60006111a0611ad7565b506080602063ffffffff861610611213576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152600e60248201527f76616c69642072656769737465720000000000000000000000000000000000006044820152606401610873565b63ffffffff8516158015906112255750825b1561125957838161016001518663ffffffff166020811061124857611248611c16565b63ffffffff90921660209290920201525b60808101805163ffffffff808216606085015260049091011690526108da610612565b6000611286611ad7565b506101e051604081015160808083015160a084015160c09094015191936000928392919063ffffffff8616610ffa036113005781610fff8116156112cf57610fff811661100003015b8363ffffffff166000036112f65760e08801805163ffffffff8382011690915295506112fa565b8395505b5061174f565b8563ffffffff16610fcd0361131b576340000000945061174f565b8563ffffffff1661101803611333576001945061174f565b8563ffffffff166110960361136857600161012088015260ff831661010088015261135c610612565b97505050505050505090565b8563ffffffff16610fa3036115b25763ffffffff83161561174f577ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffb63ffffffff84160161156c5760006113c38363fffffffc1660016106ba565b60208901519091508060001a6001036114305761142d81600090815233602052604090207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b90505b6000805460408b81015190517fe03110e10000000000000000000000000000000000000000000000000000000081526004810185905263ffffffff9091166024820152829173ffffffffffffffffffffffffffffffffffffffff169063e03110e1906044016040805180830381865afa1580156114b1573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114d59190611c45565b915091506003861680600403828110156114ed578092505b50818610156114fa578591505b8260088302610100031c9250826008828460040303021b9250600180600883600403021b036001806008858560040303021b039150811981169050838119871617955050506115518663fffffffc1660018661198a565b60408b018051820163ffffffff16905297506115ad92505050565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd63ffffffff8416016115a15780945061174f565b63ffffffff9450600993505b61174f565b8563ffffffff16610fa4036116a35763ffffffff8316600114806115dc575063ffffffff83166002145b806115ed575063ffffffff83166004145b156115fa5780945061174f565b7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa63ffffffff8416016115a157600061163a8363fffffffc1660016106ba565b60208901519091506003841660040383811015611655578093505b83900360089081029290921c7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff600193850293841b0116911b1760208801526000604088015293508361174f565b8563ffffffff16610fd70361174f578163ffffffff166003036117435763ffffffff831615806116d9575063ffffffff83166005145b806116ea575063ffffffff83166003145b156116f8576000945061174f565b63ffffffff831660011480611713575063ffffffff83166002145b80611724575063ffffffff83166006145b80611735575063ffffffff83166004145b156115a1576001945061174f565b63ffffffff9450601693505b6101608701805163ffffffff808816604090920191909152905185821660e09091015260808801805180831660608b0152600401909116905261135c610612565b600061179a611ad7565b506080600063ffffffff87166010036117b8575060c0810151611921565b8663ffffffff166011036117d75763ffffffff861660c0830152611921565b8663ffffffff166012036117f0575060a0810151611921565b8663ffffffff1660130361180f5763ffffffff861660a0830152611921565b8663ffffffff166018036118435763ffffffff600387810b9087900b02602081901c821660c08501521660a0830152611921565b8663ffffffff166019036118745763ffffffff86811681871602602081901c821660c08501521660a0830152611921565b8663ffffffff16601a036118ca578460030b8660030b8161189757611897611c69565b0763ffffffff1660c0830152600385810b9087900b816118b9576118b9611c69565b0563ffffffff1660a0830152611921565b8663ffffffff16601b03611921578463ffffffff168663ffffffff16816118f3576118f3611c69565b0663ffffffff90811660c08401528581169087168161191457611914611c69565b0463ffffffff1660a08301525b63ffffffff84161561195c57808261016001518563ffffffff166020811061194b5761194b611c16565b63ffffffff90921660209290920201525b60808201805163ffffffff8082166060860152600490910116905261197f610612565b979650505050505050565b600061199583611a2e565b905060038416156119a557600080fd5b6020810190601f8516601c0360031b83811b913563ffffffff90911b1916178460051c60005b601b811015611a235760208401933582821c60011680156119f35760018114611a0857611a19565b60008581526020839052604090209450611a19565b600082815260208690526040902094505b50506001016119cb565b505060805250505050565b60ff81166103800261016681019036906104e601811015611ad1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f636865636b207468617420746865726520697320656e6f7567682063616c6c6460448201527f61746100000000000000000000000000000000000000000000000000000000006064820152608401610873565b50919050565b6040805161018081018252600080825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101919091526101608101611b3d611b42565b905290565b6040518061040001604052806020906020820280368337509192915050565b60008083601f840112611b7357600080fd5b50813567ffffffffffffffff811115611b8b57600080fd5b602083019150836020828501011115611ba357600080fd5b9250929050565b60008060008060408587031215611bc057600080fd5b843567ffffffffffffffff80821115611bd857600080fd5b611be488838901611b61565b90965094506020870135915080821115611bfd57600080fd5b50611c0a87828801611b61565b95989497509550505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60008060408385031215611c5857600080fd5b505080516020909101519092909150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fdfea164736f6c634300080f000a",
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

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

// PreimageOracleMetaData contains all meta data concerning the PreimageOracle contract.
var PreimageOracleMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"absorbLargePreimagePart\",\"inputs\":[{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_finalize\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initLargeKeccak256Preimage\",\"inputs\":[{\"name\":\"_offset\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"_claimedSize\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"largePreimageMeta\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"offset\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"claimedSize\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"size\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"preimagePart\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"loadKeccak256PreimagePart\",\"inputs\":[{\"name\":\"_partOffset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_preimage\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"loadLocalData\",\"inputs\":[{\"name\":\"_ident\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_word\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_size\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_partOffset\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"key_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"preimageLengths\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"preimagePartOk\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"preimageParts\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"readPreimage\",\"inputs\":[{\"name\":\"_key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"dat_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"datLen_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"squeezeLargePreimagePart\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stateMatrices\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"error\",\"name\":\"InvalidClaimedSize\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidInputLength\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PartOffsetOOB\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b50611cb2806100206000396000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c806397777bf911610081578063e03110e11161005b578063e03110e114610274578063e15926111461029c578063fef2b4ed146102af57600080fd5b806397777bf914610222578063983f564e1461024e578063bc2621971461026157600080fd5b806361238bde116100b257806361238bde146101af5780637a341a70146101da5780638542cf50146101e457600080fd5b806340fa225c146100ce57806352f0f3ad1461018e575b600080fd5b6101486100dc3660046119d9565b600360205260009081526040902080546001909101546fffffffffffffffffffffffffffffffff82169167ffffffffffffffff70010000000000000000000000000000000082048116927801000000000000000000000000000000000000000000000000909204169084565b604080516fffffffffffffffffffffffffffffffff95909516855267ffffffffffffffff9384166020860152919092169083015260608201526080015b60405180910390f35b6101a161019c3660046119f4565b6102cf565b604051908152602001610185565b6101a16101bd366004611a2f565b600160209081526000928352604080842090915290825290205481565b6101e26103a4565b005b6102126101f2366004611a2f565b600260209081526000928352604080842090915290825290205460ff1681565b6040519015158152602001610185565b610235610230366004611a51565b610646565b60405167ffffffffffffffff9091168152602001610185565b6101e261025c366004611ac4565b610689565b6101e261026f366004611b20565b6109af565b610287610282366004611a2f565b610ab9565b60408051928352602083019190915201610185565b6101e26102aa366004611b7b565b610baa565b6101a16102bd366004611bc7565b60006020819052908152604090205481565b60006102db8686610cb3565b90506102e8836008611c0f565b8211806102f55750602083115b1561032c576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000602081815260c085901b82526008959095528251828252600286526040808320858452875280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660019081179091558484528752808320948352938652838220558181529384905292205592915050565b60408051336000908152600460209081528382206103408401948590529193839291830191906019908287855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116103d157905050505091909252505033600090815260036020908152604091829020825160808101845281546fffffffffffffffffffffffffffffffff8116825267ffffffffffffffff7001000000000000000000000000000000008204811694830194909452780100000000000000000000000000000000000000000000000090049092169282018390526001015460608201529192506104b2906008611c27565b67ffffffffffffffff1681600001516fffffffffffffffffffffffffffffffff16111561050b576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b806020015167ffffffffffffffff16816040015167ffffffffffffffff1614610560576040517f0354d0e700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600061056b83610d60565b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f020000000000000000000000000000000000000000000000000000000000000017600081815260026020908152604080832086516fffffffffffffffffffffffffffffffff908116855290835281842080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915560608801518686529084528285208851909216855290835281842055948501519282528190529290922067ffffffffffffffff9092169091555050565b6004602052816000526040600020816019811061066257600080fd5b60049182820401919006600802915091509054906101000a900467ffffffffffffffff1681565b6000610696608884611c53565b15905080806106a25750815b6106d8576040517f7db491eb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b606082156106f1576106ea8585610e3f565b905061072b565b84848080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509293505050505b6040805133600090815260046020908152838220610340840194859052858201949293928392830191906019908287855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff168152602001906008019060208260070104928301926001038202915080841161075c575050509290935250503360009081526003602052604090208054929350917801000000000000000000000000000000000000000000000000810467ffffffffffffffff1691506fffffffffffffffffffffffffffffffff16600881108015610804575081155b15610858578254700100000000000000000000000000000000900460c01b7fffffffffffffffff00000000000000000000000000000000000000000000000016600052845160085280516001840155610884565b818110158015610870575061086d8983611c0f565b81105b156108845760018282038601810151908401555b60408051608880825260c0820190925260009160208201818036833701905050905060005b8a811015610916578087018051602084015260208101516040840152604081015160608401526060810151608084015267ffffffffffffffff60c01b60808201511660a0840152506108fb8683610ec8565b61090486611020565b61090f608882611c0f565b90506108a9565b5084513360009081526004602052604090206109339160196118bc565b5033600090815260036020526040902080548b919060189061097c9084907801000000000000000000000000000000000000000000000000900467ffffffffffffffff16611c27565b92506101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055505050505050505050505050565b604080516080810182526fffffffffffffffffffffffffffffffff808516825267ffffffffffffffff808516602080850191825260008587018181526060870182815233835260039093529690209451855492519651841678010000000000000000000000000000000000000000000000000277ffffffffffffffffffffffffffffffffffffffffffffffff97909416700100000000000000000000000000000000027fffffffffffffffff00000000000000000000000000000000000000000000000090931694169390931717939093169290921781559051600190910155610a97611964565b8051336000908152600460205260409020610ab39160196118bc565b50505050565b6000828152600260209081526040808320848452909152812054819060ff16610b42576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f7072652d696d616765206d757374206578697374000000000000000000000000604482015260640160405180910390fd5b5060008381526020818152604090912054610b5e816008611c0f565b610b69856020611c0f565b10610b875783610b7a826008611c0f565b610b849190611c8e565b91505b506000938452600160209081526040808620948652939052919092205492909150565b60443560008060088301861115610bc95763fe2549876000526004601cfd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001760008181526002602090815260408083208b8452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915584845282528083209a83529981528982209390935590815290819052959095209190915550505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff831617610d59818360408051600093845233602052918152606090922091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b9392505050565b6000610de3565b66ff00ff00ff00ff8160081c1667ff00ff00ff00ff00610d918360081b67ffffffffffffffff1690565b1617905065ffff0000ffff8160101c1667ffff0000ffff0000610dbe8360101b67ffffffffffffffff1690565b1617905060008160201c610ddc8360201b67ffffffffffffffff1690565b1792915050565b60808201516020830190610dfb90610d67565b610d67565b6040820151610e0990610d67565b60401b17610e21610df660018460059190911b015190565b825160809190911b90610e3390610d67565b60c01b17179392505050565b6060604051905081602082018181018286833760888306808015610e9d576088829003850160808582017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff01536001845160001a1784538652610eaf565b60018353608060878401536088850186525b5050505050601f19603f82510116810160405292915050565b6088815114610ed657600080fd5b6020810160208301610f54565b8260031b8083015190508060001a8160011a60081b178160021a60101b8260031a60181b17178160041a60201b8260051a60281b178260061a60301b8360071a60381b1717179050610ab381610f3f868560059190911b015190565b1867ffffffffffffffff16600586901b840152565b610f6060008383610ee3565b610f6c60018383610ee3565b610f7860028383610ee3565b610f8460038383610ee3565b610f9060048383610ee3565b610f9c60058383610ee3565b610fa860068383610ee3565b610fb460078383610ee3565b610fc060088383610ee3565b610fcc60098383610ee3565b610fd8600a8383610ee3565b610fe4600b8383610ee3565b610ff0600c8383610ee3565b610ffc600d8383610ee3565b611008600e8383610ee3565b611014600f8383610ee3565b610ab360108383610ee3565b6040805178010000000000008082800000000000808a8000000080008000602082015279808b00000000800000018000000080008081800000000000800991810191909152788a00000000000000880000000080008009000000008000000a60608201527b8000808b800000000000008b8000000000008089800000000000800360808201527f80000000000080028000000000000080000000000000800a800000008000000a60a08201527f800000008000808180000000000080800000000080000001800000008000800860c082015260009060e0016040516020818303038152906040529050602082016020820161179c565b6102808101516101e082015161014083015160a0840151845118189118186102a082015161020083015161016084015160c0850151602086015118189118186102c083015161022084015161018085015160e0860151604087015118189118186102e08401516102408501516101a0860151610100870151606088015118189118186103008501516102608601516101c0870151610120880151608089015118189118188084603f1c6111d38660011b67ffffffffffffffff1690565b18188584603f1c6111ee8660011b67ffffffffffffffff1690565b18188584603f1c6112098660011b67ffffffffffffffff1690565b181895508483603f1c6112268560011b67ffffffffffffffff1690565b181894508387603f1c6112438960011b67ffffffffffffffff1690565b60208b01518b51861867ffffffffffffffff168c5291189190911897508118600181901b603f9190911c18935060c08801518118601481901c602c9190911b1867ffffffffffffffff1660208901526101208801518718602c81901c60149190911b1867ffffffffffffffff1660c08901526102c08801518618600381901c603d9190911b1867ffffffffffffffff166101208901526101c08801518718601981901c60279190911b1867ffffffffffffffff166102c08901526102808801518218602e81901c60129190911b1867ffffffffffffffff166101c089015260408801518618600281901c603e9190911b1867ffffffffffffffff166102808901526101808801518618601581901c602b9190911b1867ffffffffffffffff1660408901526101a08801518518602781901c60199190911b1867ffffffffffffffff166101808901526102608801518718603881901c60089190911b1867ffffffffffffffff166101a08901526102e08801518518600881901c60389190911b1867ffffffffffffffff166102608901526101e08801518218601781901c60299190911b1867ffffffffffffffff166102e089015260808801518718602581901c601b9190911b1867ffffffffffffffff166101e08901526103008801518718603281901c600e9190911b1867ffffffffffffffff1660808901526102a08801518118603e81901c60029190911b1867ffffffffffffffff166103008901526101008801518518600981901c60379190911b1867ffffffffffffffff166102a08901526102008801518118601381901c602d9190911b1867ffffffffffffffff1661010089015260a08801518218601c81901c60249190911b1867ffffffffffffffff1661020089015260608801518518602481901c601c9190911b1867ffffffffffffffff1660a08901526102408801518518602b81901c60159190911b1867ffffffffffffffff1660608901526102208801518618603181901c600f9190911b1867ffffffffffffffff166102408901526101608801518118603681901c600a9190911b1867ffffffffffffffff166102208901525060e08701518518603a81901c60069190911b1867ffffffffffffffff166101608801526101408701518118603d81901c60039190911b1867ffffffffffffffff1660e0880152505067ffffffffffffffff81166101408601525050505050565b6115c381611116565b805160208201805160408401805160608601805160808801805167ffffffffffffffff871986168a188116808c528619851689188216909952831982169095188516909552841988169091188316909152941990921618811690925260a08301805160c0808601805160e0880180516101008a0180516101208c018051861985168a188d16909a528319821686188c16909652801989169092188a169092528619861618881690529219909216909218841690526101408401805161016086018051610180880180516101a08a0180516101c08c0180518619851689188d169099528319821686188c16909652801988169092188a169092528519851618881690529119909116909118841690526101e08401805161020086018051610220880180516102408a0180516102608c0180518619851689188d169099528319821686188c16909652801988169092188a16909252851985161888169052911990911690911884169052610280840180516102a0860180516102c0880180516102e08a0180516103008c0180518619851689188d169099528319821686188c16909652801988169092188a16909252851985161888169052911990911690911884169052600386901b850151901c9081189091168252610ab3565b6117a8600082846115ba565b6117b4600182846115ba565b6117c0600282846115ba565b6117cc600382846115ba565b6117d8600482846115ba565b6117e4600582846115ba565b6117f0600682846115ba565b6117fc600782846115ba565b611808600882846115ba565b611814600982846115ba565b611820600a82846115ba565b61182c600b82846115ba565b611838600c82846115ba565b611844600d82846115ba565b611850600e82846115ba565b61185c600f82846115ba565b611868601082846115ba565b611874601182846115ba565b611880601282846115ba565b61188c601382846115ba565b611898601482846115ba565b6118a4601582846115ba565b6118b0601682846115ba565b610ab3601782846115ba565b6007830191839082156119545791602002820160005b8382111561191e57835183826101000a81548167ffffffffffffffff021916908367ffffffffffffffff16021790555092602001926008016020816007010492830192600103026118d2565b80156119525782816101000a81549067ffffffffffffffff021916905560080160208160070104928301926001030261191e565b505b5061196092915061197c565b5090565b6040518060200160405280611977611991565b905290565b5b80821115611960576000815560010161197d565b6040518061032001604052806019906020820280368337509192915050565b803573ffffffffffffffffffffffffffffffffffffffff811681146119d457600080fd5b919050565b6000602082840312156119eb57600080fd5b610d59826119b0565b600080600080600060a08688031215611a0c57600080fd5b505083359560208501359550604085013594606081013594506080013592509050565b60008060408385031215611a4257600080fd5b50508035926020909101359150565b60008060408385031215611a6457600080fd5b611a6d836119b0565b946020939093013593505050565b60008083601f840112611a8d57600080fd5b50813567ffffffffffffffff811115611aa557600080fd5b602083019150836020828501011115611abd57600080fd5b9250929050565b600080600060408486031215611ad957600080fd5b833567ffffffffffffffff811115611af057600080fd5b611afc86828701611a7b565b90945092505060208401358015158114611b1557600080fd5b809150509250925092565b60008060408385031215611b3357600080fd5b82356fffffffffffffffffffffffffffffffff81168114611b5357600080fd5b9150602083013567ffffffffffffffff81168114611b7057600080fd5b809150509250929050565b600080600060408486031215611b9057600080fd5b83359250602084013567ffffffffffffffff811115611bae57600080fd5b611bba86828701611a7b565b9497909650939450505050565b600060208284031215611bd957600080fd5b5035919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115611c2257611c22611be0565b500190565b600067ffffffffffffffff808316818516808303821115611c4a57611c4a611be0565b01949350505050565b600082611c89577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500690565b600082821015611ca057611ca0611be0565b50039056fea164736f6c634300080f000a",
}

// PreimageOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use PreimageOracleMetaData.ABI instead.
var PreimageOracleABI = PreimageOracleMetaData.ABI

// PreimageOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PreimageOracleMetaData.Bin instead.
var PreimageOracleBin = PreimageOracleMetaData.Bin

// DeployPreimageOracle deploys a new Ethereum contract, binding an instance of PreimageOracle to it.
func DeployPreimageOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *PreimageOracle, error) {
	parsed, err := PreimageOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PreimageOracleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PreimageOracle{PreimageOracleCaller: PreimageOracleCaller{contract: contract}, PreimageOracleTransactor: PreimageOracleTransactor{contract: contract}, PreimageOracleFilterer: PreimageOracleFilterer{contract: contract}}, nil
}

// PreimageOracle is an auto generated Go binding around an Ethereum contract.
type PreimageOracle struct {
	PreimageOracleCaller     // Read-only binding to the contract
	PreimageOracleTransactor // Write-only binding to the contract
	PreimageOracleFilterer   // Log filterer for contract events
}

// PreimageOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type PreimageOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PreimageOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PreimageOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PreimageOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PreimageOracleSession struct {
	Contract     *PreimageOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PreimageOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PreimageOracleCallerSession struct {
	Contract *PreimageOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// PreimageOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PreimageOracleTransactorSession struct {
	Contract     *PreimageOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// PreimageOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type PreimageOracleRaw struct {
	Contract *PreimageOracle // Generic contract binding to access the raw methods on
}

// PreimageOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PreimageOracleCallerRaw struct {
	Contract *PreimageOracleCaller // Generic read-only contract binding to access the raw methods on
}

// PreimageOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PreimageOracleTransactorRaw struct {
	Contract *PreimageOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPreimageOracle creates a new instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracle(address common.Address, backend bind.ContractBackend) (*PreimageOracle, error) {
	contract, err := bindPreimageOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PreimageOracle{PreimageOracleCaller: PreimageOracleCaller{contract: contract}, PreimageOracleTransactor: PreimageOracleTransactor{contract: contract}, PreimageOracleFilterer: PreimageOracleFilterer{contract: contract}}, nil
}

// NewPreimageOracleCaller creates a new read-only instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleCaller(address common.Address, caller bind.ContractCaller) (*PreimageOracleCaller, error) {
	contract, err := bindPreimageOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleCaller{contract: contract}, nil
}

// NewPreimageOracleTransactor creates a new write-only instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*PreimageOracleTransactor, error) {
	contract, err := bindPreimageOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleTransactor{contract: contract}, nil
}

// NewPreimageOracleFilterer creates a new log filterer instance of PreimageOracle, bound to a specific deployed contract.
func NewPreimageOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*PreimageOracleFilterer, error) {
	contract, err := bindPreimageOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PreimageOracleFilterer{contract: contract}, nil
}

// bindPreimageOracle binds a generic wrapper to an already deployed contract.
func bindPreimageOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PreimageOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PreimageOracle *PreimageOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PreimageOracle.Contract.PreimageOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PreimageOracle *PreimageOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PreimageOracle.Contract.PreimageOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PreimageOracle *PreimageOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PreimageOracle.Contract.PreimageOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PreimageOracle *PreimageOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PreimageOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PreimageOracle *PreimageOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PreimageOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PreimageOracle *PreimageOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PreimageOracle.Contract.contract.Transact(opts, method, params...)
}

// LargePreimageMeta is a free data retrieval call binding the contract method 0x40fa225c.
//
// Solidity: function largePreimageMeta(address ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleCaller) LargePreimageMeta(opts *bind.CallOpts, arg0 common.Address) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "largePreimageMeta", arg0)

	outstruct := new(struct {
		Offset       *big.Int
		ClaimedSize  uint64
		Size         uint64
		PreimagePart [32]byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Offset = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.ClaimedSize = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Size = *abi.ConvertType(out[2], new(uint64)).(*uint64)
	outstruct.PreimagePart = *abi.ConvertType(out[3], new([32]byte)).(*[32]byte)

	return *outstruct, err

}

// LargePreimageMeta is a free data retrieval call binding the contract method 0x40fa225c.
//
// Solidity: function largePreimageMeta(address ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleSession) LargePreimageMeta(arg0 common.Address) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	return _PreimageOracle.Contract.LargePreimageMeta(&_PreimageOracle.CallOpts, arg0)
}

// LargePreimageMeta is a free data retrieval call binding the contract method 0x40fa225c.
//
// Solidity: function largePreimageMeta(address ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleCallerSession) LargePreimageMeta(arg0 common.Address) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	return _PreimageOracle.Contract.LargePreimageMeta(&_PreimageOracle.CallOpts, arg0)
}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleCaller) PreimageLengths(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimageLengths", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleSession) PreimageLengths(arg0 [32]byte) (*big.Int, error) {
	return _PreimageOracle.Contract.PreimageLengths(&_PreimageOracle.CallOpts, arg0)
}

// PreimageLengths is a free data retrieval call binding the contract method 0xfef2b4ed.
//
// Solidity: function preimageLengths(bytes32 ) view returns(uint256)
func (_PreimageOracle *PreimageOracleCallerSession) PreimageLengths(arg0 [32]byte) (*big.Int, error) {
	return _PreimageOracle.Contract.PreimageLengths(&_PreimageOracle.CallOpts, arg0)
}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleCaller) PreimagePartOk(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) (bool, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimagePartOk", arg0, arg1)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleSession) PreimagePartOk(arg0 [32]byte, arg1 *big.Int) (bool, error) {
	return _PreimageOracle.Contract.PreimagePartOk(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimagePartOk is a free data retrieval call binding the contract method 0x8542cf50.
//
// Solidity: function preimagePartOk(bytes32 , uint256 ) view returns(bool)
func (_PreimageOracle *PreimageOracleCallerSession) PreimagePartOk(arg0 [32]byte, arg1 *big.Int) (bool, error) {
	return _PreimageOracle.Contract.PreimagePartOk(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleCaller) PreimageParts(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "preimageParts", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleSession) PreimageParts(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _PreimageOracle.Contract.PreimageParts(&_PreimageOracle.CallOpts, arg0, arg1)
}

// PreimageParts is a free data retrieval call binding the contract method 0x61238bde.
//
// Solidity: function preimageParts(bytes32 , uint256 ) view returns(bytes32)
func (_PreimageOracle *PreimageOracleCallerSession) PreimageParts(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _PreimageOracle.Contract.PreimageParts(&_PreimageOracle.CallOpts, arg0, arg1)
}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 _key, uint256 _offset) view returns(bytes32 dat_, uint256 datLen_)
func (_PreimageOracle *PreimageOracleCaller) ReadPreimage(opts *bind.CallOpts, _key [32]byte, _offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "readPreimage", _key, _offset)

	outstruct := new(struct {
		Dat    [32]byte
		DatLen *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Dat = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.DatLen = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 _key, uint256 _offset) view returns(bytes32 dat_, uint256 datLen_)
func (_PreimageOracle *PreimageOracleSession) ReadPreimage(_key [32]byte, _offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	return _PreimageOracle.Contract.ReadPreimage(&_PreimageOracle.CallOpts, _key, _offset)
}

// ReadPreimage is a free data retrieval call binding the contract method 0xe03110e1.
//
// Solidity: function readPreimage(bytes32 _key, uint256 _offset) view returns(bytes32 dat_, uint256 datLen_)
func (_PreimageOracle *PreimageOracleCallerSession) ReadPreimage(_key [32]byte, _offset *big.Int) (struct {
	Dat    [32]byte
	DatLen *big.Int
}, error) {
	return _PreimageOracle.Contract.ReadPreimage(&_PreimageOracle.CallOpts, _key, _offset)
}

// StateMatrices is a free data retrieval call binding the contract method 0x97777bf9.
//
// Solidity: function stateMatrices(address , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleCaller) StateMatrices(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (uint64, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "stateMatrices", arg0, arg1)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// StateMatrices is a free data retrieval call binding the contract method 0x97777bf9.
//
// Solidity: function stateMatrices(address , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleSession) StateMatrices(arg0 common.Address, arg1 *big.Int) (uint64, error) {
	return _PreimageOracle.Contract.StateMatrices(&_PreimageOracle.CallOpts, arg0, arg1)
}

// StateMatrices is a free data retrieval call binding the contract method 0x97777bf9.
//
// Solidity: function stateMatrices(address , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleCallerSession) StateMatrices(arg0 common.Address, arg1 *big.Int) (uint64, error) {
	return _PreimageOracle.Contract.StateMatrices(&_PreimageOracle.CallOpts, arg0, arg1)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0x983f564e.
//
// Solidity: function absorbLargePreimagePart(bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleTransactor) AbsorbLargePreimagePart(opts *bind.TransactOpts, _data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "absorbLargePreimagePart", _data, _finalize)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0x983f564e.
//
// Solidity: function absorbLargePreimagePart(bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleSession) AbsorbLargePreimagePart(_data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.Contract.AbsorbLargePreimagePart(&_PreimageOracle.TransactOpts, _data, _finalize)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0x983f564e.
//
// Solidity: function absorbLargePreimagePart(bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) AbsorbLargePreimagePart(_data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.Contract.AbsorbLargePreimagePart(&_PreimageOracle.TransactOpts, _data, _finalize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0xbc262197.
//
// Solidity: function initLargeKeccak256Preimage(uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleTransactor) InitLargeKeccak256Preimage(opts *bind.TransactOpts, _offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "initLargeKeccak256Preimage", _offset, _claimedSize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0xbc262197.
//
// Solidity: function initLargeKeccak256Preimage(uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleSession) InitLargeKeccak256Preimage(_offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.Contract.InitLargeKeccak256Preimage(&_PreimageOracle.TransactOpts, _offset, _claimedSize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0xbc262197.
//
// Solidity: function initLargeKeccak256Preimage(uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) InitLargeKeccak256Preimage(_offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.Contract.InitLargeKeccak256Preimage(&_PreimageOracle.TransactOpts, _offset, _claimedSize)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 _partOffset, bytes _preimage) returns()
func (_PreimageOracle *PreimageOracleTransactor) LoadKeccak256PreimagePart(opts *bind.TransactOpts, _partOffset *big.Int, _preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "loadKeccak256PreimagePart", _partOffset, _preimage)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 _partOffset, bytes _preimage) returns()
func (_PreimageOracle *PreimageOracleSession) LoadKeccak256PreimagePart(_partOffset *big.Int, _preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadKeccak256PreimagePart(&_PreimageOracle.TransactOpts, _partOffset, _preimage)
}

// LoadKeccak256PreimagePart is a paid mutator transaction binding the contract method 0xe1592611.
//
// Solidity: function loadKeccak256PreimagePart(uint256 _partOffset, bytes _preimage) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) LoadKeccak256PreimagePart(_partOffset *big.Int, _preimage []byte) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadKeccak256PreimagePart(&_PreimageOracle.TransactOpts, _partOffset, _preimage)
}

// LoadLocalData is a paid mutator transaction binding the contract method 0x52f0f3ad.
//
// Solidity: function loadLocalData(uint256 _ident, bytes32 _localContext, bytes32 _word, uint256 _size, uint256 _partOffset) returns(bytes32 key_)
func (_PreimageOracle *PreimageOracleTransactor) LoadLocalData(opts *bind.TransactOpts, _ident *big.Int, _localContext [32]byte, _word [32]byte, _size *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "loadLocalData", _ident, _localContext, _word, _size, _partOffset)
}

// LoadLocalData is a paid mutator transaction binding the contract method 0x52f0f3ad.
//
// Solidity: function loadLocalData(uint256 _ident, bytes32 _localContext, bytes32 _word, uint256 _size, uint256 _partOffset) returns(bytes32 key_)
func (_PreimageOracle *PreimageOracleSession) LoadLocalData(_ident *big.Int, _localContext [32]byte, _word [32]byte, _size *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadLocalData(&_PreimageOracle.TransactOpts, _ident, _localContext, _word, _size, _partOffset)
}

// LoadLocalData is a paid mutator transaction binding the contract method 0x52f0f3ad.
//
// Solidity: function loadLocalData(uint256 _ident, bytes32 _localContext, bytes32 _word, uint256 _size, uint256 _partOffset) returns(bytes32 key_)
func (_PreimageOracle *PreimageOracleTransactorSession) LoadLocalData(_ident *big.Int, _localContext [32]byte, _word [32]byte, _size *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.LoadLocalData(&_PreimageOracle.TransactOpts, _ident, _localContext, _word, _size, _partOffset)
}

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x7a341a70.
//
// Solidity: function squeezeLargePreimagePart() returns()
func (_PreimageOracle *PreimageOracleTransactor) SqueezeLargePreimagePart(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "squeezeLargePreimagePart")
}

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x7a341a70.
//
// Solidity: function squeezeLargePreimagePart() returns()
func (_PreimageOracle *PreimageOracleSession) SqueezeLargePreimagePart() (*types.Transaction, error) {
	return _PreimageOracle.Contract.SqueezeLargePreimagePart(&_PreimageOracle.TransactOpts)
}

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x7a341a70.
//
// Solidity: function squeezeLargePreimagePart() returns()
func (_PreimageOracle *PreimageOracleTransactorSession) SqueezeLargePreimagePart() (*types.Transaction, error) {
	return _PreimageOracle.Contract.SqueezeLargePreimagePart(&_PreimageOracle.TransactOpts)
}

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
	ABI: "[{\"type\":\"function\",\"name\":\"absorbLargePreimagePart\",\"inputs\":[{\"name\":\"_contextKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_finalize\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"initLargeKeccak256Preimage\",\"inputs\":[{\"name\":\"_contextKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_offset\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"_claimedSize\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"largePreimageMeta\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"offset\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"claimedSize\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"size\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"preimagePart\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"loadKeccak256PreimagePart\",\"inputs\":[{\"name\":\"_partOffset\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_preimage\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"loadLocalData\",\"inputs\":[{\"name\":\"_ident\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_word\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_size\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_partOffset\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"key_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"preimageLengths\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"preimagePartOk\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"preimageParts\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"readPreimage\",\"inputs\":[{\"name\":\"_key\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_offset\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"dat_\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"datLen_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"squeezeLargePreimagePart\",\"inputs\":[{\"name\":\"_contextKey\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stateMatrices\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"error\",\"name\":\"InvalidClaimedSize\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidInputLength\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"PartOffsetOOB\",\"inputs\":[]}]",
	Bin: "0x608060405234801561001057600080fd5b50611df4806100206000396000f3fe608060405234801561001057600080fd5b50600436106100c95760003560e01c80639ec5011811610081578063e03110e11161005b578063e03110e11461028a578063e1592611146102b2578063fef2b4ed146102c557600080fd5b80639ec501181461019e578063a2c3e141146101b1578063d709ebef146101c457600080fd5b806355fbfaea116100b257806355fbfaea1461012057806361238bde146101355780638542cf501461016057600080fd5b80633a7492de146100ce57806352f0f3ad146100ff575b600080fd5b6100e16100dc366004611af2565b6102e5565b60405167ffffffffffffffff90911681526020015b60405180910390f35b61011261010d366004611b25565b610335565b6040519081526020016100f6565b61013361012e366004611b60565b61040a565b005b610112610143366004611b79565b600160209081526000928352604080842090915290825290205481565b61018e61016e366004611b79565b600260209081526000928352604080842090915290825290205460ff1681565b60405190151581526020016100f6565b6101336101ac366004611b9b565b6106bd565b6101336101bf366004611c48565b6107db565b6102496101d2366004611cac565b6003602090815260009283526040808420909152908252902080546001909101546fffffffffffffffffffffffffffffffff82169167ffffffffffffffff70010000000000000000000000000000000082048116927801000000000000000000000000000000000000000000000000909204169084565b604080516fffffffffffffffffffffffffffffffff95909516855267ffffffffffffffff9384166020860152919092169083015260608201526080016100f6565b61029d610298366004611b79565b610bd1565b604080519283526020830191909152016100f6565b6101336102c0366004611cd6565b610cc2565b6101126102d3366004611b60565b60006020819052908152604090205481565b6004602052826000526040600020602052816000526040600020816019811061030d57600080fd5b6004918282040191900660080292509250509054906101000a900467ffffffffffffffff1681565b60006103418686610dcb565b905061034e836008611d51565b82118061035b5750602083115b15610392576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000602081815260c085901b82526008959095528251828252600286526040808320858452875280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660019081179091558484528752808320948352938652838220558181529384905292205592915050565b604080513360009081526004602090815283822085835281528382206103408401948590529193839291830191906019908287855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff168152602001906008019060208260070104928301926001038202915080841161043f579050505050919092525050336000908152600360209081526040808320868452825291829020825160808101845281546fffffffffffffffffffffffffffffffff8116825267ffffffffffffffff700100000000000000000000000000000000820481169483019490945278010000000000000000000000000000000000000000000000009004909216928201839052600101546060820152919250610528906008611d69565b67ffffffffffffffff1681600001516fffffffffffffffffffffffffffffffff161115610581576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b806020015167ffffffffffffffff16816040015167ffffffffffffffff16146105d6576040517f0354d0e700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006105e183610e78565b7effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f020000000000000000000000000000000000000000000000000000000000000017600081815260026020908152604080832086516fffffffffffffffffffffffffffffffff908116855290835281842080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915560608801518686529084528285208851909216855290835281842055948501519282528190529290922067ffffffffffffffff909216909155505050565b604080516080810182526fffffffffffffffffffffffffffffffff808516825267ffffffffffffffff8085166020808501918252600085870181815260608701828152338352600384528883208c84529093529690209451855492519651841678010000000000000000000000000000000000000000000000000277ffffffffffffffffffffffffffffffffffffffffffffffff97909416700100000000000000000000000000000000027fffffffffffffffff000000000000000000000000000000000000000000000000909316941693909317179390931692909217815590516001909101556107ad6119d5565b805133600090815260046020908152604080832088845290915290206107d49160196119ed565b5050505050565b60006107e8608884611d95565b15905080806107f45750815b61082a576040517f7db491eb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b606082156108435761083c8585610f57565b905061087d565b84848080601f0160208091040260200160405190810160405280939291908181526020018383808284376000920191909152509293505050505b60408051336000908152600460209081528382208a835281528382206103408401948590529193839291830191906019908287855b82829054906101000a900467ffffffffffffffff1667ffffffffffffffff16815260200190600801906020826007010492830192600103820291508084116108b2575050509290935250503360009081526003602090815260408083208c845290915290208054929350917801000000000000000000000000000000000000000000000000810467ffffffffffffffff1691506fffffffffffffffffffffffffffffffff16600881108015610965575081155b156109b9578254700100000000000000000000000000000000900460c01b7fffffffffffffffff00000000000000000000000000000000000000000000000016600052883560085280516001840155610a56565b600881101580156109d75750816109d1600883611dd0565b91508110155b80156109eb57506109e88883611d51565b81105b15610a565760006109fc8383611dd0565b905088610a0a826020611d51565b10158015610a16575087155b15610a4d576040517ffe25498700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b89013560018401555b60408051608880825260c0820190925260208701916000919060208201818036833701905050905060005b8751811015610ae5578083018051602084015260208101516040840152604081015160608401526060810151608084015267ffffffffffffffff60c01b60808201511660a084015250610ad48783610fe0565b610add8761113b565b608801610a81565b508560000151600460003373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008e8152602001908152602001600020906019610b499291906119ed565b503360009081526003602090815260408083208f8452909152902080548b9190601890610b9d9084907801000000000000000000000000000000000000000000000000900467ffffffffffffffff16611d69565b92506101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550505050505050505050505050565b6000828152600260209081526040808320848452909152812054819060ff16610c5a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601460248201527f7072652d696d616765206d757374206578697374000000000000000000000000604482015260640160405180910390fd5b5060008381526020818152604090912054610c76816008611d51565b610c81856020611d51565b10610c9f5783610c92826008611d51565b610c9c9190611dd0565b91505b506000938452600160209081526040808620948652939052919092205492909150565b60443560008060088301861115610ce15763fe2549876000526004601cfd5b60c083901b6080526088838682378087017ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80151908490207effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f02000000000000000000000000000000000000000000000000000000000000001760008181526002602090815260408083208b8452825280832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600190811790915584845282528083209a83529981528982209390935590815290819052959095209190915550505050565b7f01000000000000000000000000000000000000000000000000000000000000007effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff831617610e71818360408051600093845233602052918152606090922091527effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01000000000000000000000000000000000000000000000000000000000000001790565b9392505050565b6000610efb565b66ff00ff00ff00ff8160081c1667ff00ff00ff00ff00610ea98360081b67ffffffffffffffff1690565b1617905065ffff0000ffff8160101c1667ffff0000ffff0000610ed68360101b67ffffffffffffffff1690565b1617905060008160201c610ef48360201b67ffffffffffffffff1690565b1792915050565b60808201516020830190610f1390610e7f565b610e7f565b6040820151610f2190610e7f565b60401b17610f39610f0e60018460059190911b015190565b825160809190911b90610f4b90610e7f565b60c01b17179392505050565b6060604051905081602082018181018286833760888306808015610fb5576088829003850160808582017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff01536001845160001a1784538652610fc7565b60018353608060878401536088850186525b5050505050601f19603f82510116810160405292915050565b6088815114610fee57600080fd5b602081016020830161106f565b8260031b8201518060001a8160011a60081b178160021a60101b8260031a60181b17178160041a60201b8260051a60281b178260061a60301b8360071a60381b171717905061106981611054868560059190911b015190565b1867ffffffffffffffff16600586901b840152565b50505050565b61107b60008383610ffb565b61108760018383610ffb565b61109360028383610ffb565b61109f60038383610ffb565b6110ab60048383610ffb565b6110b760058383610ffb565b6110c360068383610ffb565b6110cf60078383610ffb565b6110db60088383610ffb565b6110e760098383610ffb565b6110f3600a8383610ffb565b6110ff600b8383610ffb565b61110b600c8383610ffb565b611117600d8383610ffb565b611123600e8383610ffb565b61112f600f8383610ffb565b61106960108383610ffb565b6040805178010000000000008082800000000000808a8000000080008000602082015279808b00000000800000018000000080008081800000000000800991810191909152788a00000000000000880000000080008009000000008000000a60608201527b8000808b800000000000008b8000000000008089800000000000800360808201527f80000000000080028000000000000080000000000000800a800000008000000a60a08201527f800000008000808180000000000080800000000080000001800000008000800860c082015260009060e001604051602081830303815290604052905060208201602082016118b5565b6102808101516101e082015161014083015160a0840151845118189118186102a082015161020083015161016084015160c0850151602086015118189118186102c083015161022084015161018085015160e0860151604087015118189118186102e08401516102408501516101a0860151610100870151606088015118189118186103008501516102608601516101c0870151610120880151608089015118189118188084603f1c6112ee8660011b67ffffffffffffffff1690565b18188584603f1c6113098660011b67ffffffffffffffff1690565b18188584603f1c6113248660011b67ffffffffffffffff1690565b181895508483603f1c6113418560011b67ffffffffffffffff1690565b181894508387603f1c61135e8960011b67ffffffffffffffff1690565b60208b01518b51861867ffffffffffffffff168c5291189190911897508118600181901b603f9190911c18935060c08801518118601481901c602c9190911b1867ffffffffffffffff1660208901526101208801518718602c81901c60149190911b1867ffffffffffffffff1660c08901526102c08801518618600381901c603d9190911b1867ffffffffffffffff166101208901526101c08801518718601981901c60279190911b1867ffffffffffffffff166102c08901526102808801518218602e81901c60129190911b1867ffffffffffffffff166101c089015260408801518618600281901c603e9190911b1867ffffffffffffffff166102808901526101808801518618601581901c602b9190911b1867ffffffffffffffff1660408901526101a08801518518602781901c60199190911b1867ffffffffffffffff166101808901526102608801518718603881901c60089190911b1867ffffffffffffffff166101a08901526102e08801518518600881901c60389190911b1867ffffffffffffffff166102608901526101e08801518218601781901c60299190911b1867ffffffffffffffff166102e089015260808801518718602581901c601b9190911b1867ffffffffffffffff166101e08901526103008801518718603281901c600e9190911b1867ffffffffffffffff1660808901526102a08801518118603e81901c60029190911b1867ffffffffffffffff166103008901526101008801518518600981901c60379190911b1867ffffffffffffffff166102a08901526102008801518118601381901c602d9190911b1867ffffffffffffffff1661010089015260a08801518218601c81901c60249190911b1867ffffffffffffffff1661020089015260608801518518602481901c601c9190911b1867ffffffffffffffff1660a08901526102408801518518602b81901c60159190911b1867ffffffffffffffff1660608901526102208801518618603181901c600f9190911b1867ffffffffffffffff166102408901526101608801518118603681901c600a9190911b1867ffffffffffffffff166102208901525060e08701518518603a81901c60069190911b1867ffffffffffffffff166101608801526101408701518118603d81901c60039190911b1867ffffffffffffffff1660e0880152505067ffffffffffffffff81166101408601526107d4565b6116dc81611231565b805160208201805160408401805160608601805160808801805167ffffffffffffffff871986168a188116808c528619851689188216909952831982169095188516909552841988169091188316909152941990921618811690925260a08301805160c0808601805160e0880180516101008a0180516101208c018051861985168a188d16909a528319821686188c16909652801989169092188a169092528619861618881690529219909216909218841690526101408401805161016086018051610180880180516101a08a0180516101c08c0180518619851689188d169099528319821686188c16909652801988169092188a169092528519851618881690529119909116909118841690526101e08401805161020086018051610220880180516102408a0180516102608c0180518619851689188d169099528319821686188c16909652801988169092188a16909252851985161888169052911990911690911884169052610280840180516102a0860180516102c0880180516102e08a0180516103008c0180518619851689188d169099528319821686188c16909652801988169092188a16909252851985161888169052911990911690911884169052600386901b850151901c9081189091168252611069565b6118c1600082846116d3565b6118cd600182846116d3565b6118d9600282846116d3565b6118e5600382846116d3565b6118f1600482846116d3565b6118fd600582846116d3565b611909600682846116d3565b611915600782846116d3565b611921600882846116d3565b61192d600982846116d3565b611939600a82846116d3565b611945600b82846116d3565b611951600c82846116d3565b61195d600d82846116d3565b611969600e82846116d3565b611975600f82846116d3565b611981601082846116d3565b61198d601182846116d3565b611999601282846116d3565b6119a5601382846116d3565b6119b1601482846116d3565b6119bd601582846116d3565b6119c9601682846116d3565b611069601782846116d3565b60405180602001604052806119e8611a95565b905290565b600783019183908215611a855791602002820160005b83821115611a4f57835183826101000a81548167ffffffffffffffff021916908367ffffffffffffffff1602179055509260200192600801602081600701049283019260010302611a03565b8015611a835782816101000a81549067ffffffffffffffff0219169055600801602081600701049283019260010302611a4f565b505b50611a91929150611ab4565b5090565b6040518061032001604052806019906020820280368337509192915050565b5b80821115611a915760008155600101611ab5565b803573ffffffffffffffffffffffffffffffffffffffff81168114611aed57600080fd5b919050565b600080600060608486031215611b0757600080fd5b611b1084611ac9565b95602085013595506040909401359392505050565b600080600080600060a08688031215611b3d57600080fd5b505083359560208501359550604085013594606081013594506080013592509050565b600060208284031215611b7257600080fd5b5035919050565b60008060408385031215611b8c57600080fd5b50508035926020909101359150565b600080600060608486031215611bb057600080fd5b8335925060208401356fffffffffffffffffffffffffffffffff81168114611bd757600080fd5b9150604084013567ffffffffffffffff81168114611bf457600080fd5b809150509250925092565b60008083601f840112611c1157600080fd5b50813567ffffffffffffffff811115611c2957600080fd5b602083019150836020828501011115611c4157600080fd5b9250929050565b60008060008060608587031215611c5e57600080fd5b84359350602085013567ffffffffffffffff811115611c7c57600080fd5b611c8887828801611bff565b90945092505060408501358015158114611ca157600080fd5b939692955090935050565b60008060408385031215611cbf57600080fd5b611cc883611ac9565b946020939093013593505050565b600080600060408486031215611ceb57600080fd5b83359250602084013567ffffffffffffffff811115611d0957600080fd5b611d1586828701611bff565b9497909650939450505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115611d6457611d64611d22565b500190565b600067ffffffffffffffff808316818516808303821115611d8c57611d8c611d22565b01949350505050565b600082611dcb577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500690565b600082821015611de257611de2611d22565b50039056fea164736f6c634300080f000a",
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

// LargePreimageMeta is a free data retrieval call binding the contract method 0xd709ebef.
//
// Solidity: function largePreimageMeta(address , uint256 ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleCaller) LargePreimageMeta(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "largePreimageMeta", arg0, arg1)

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

// LargePreimageMeta is a free data retrieval call binding the contract method 0xd709ebef.
//
// Solidity: function largePreimageMeta(address , uint256 ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleSession) LargePreimageMeta(arg0 common.Address, arg1 *big.Int) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	return _PreimageOracle.Contract.LargePreimageMeta(&_PreimageOracle.CallOpts, arg0, arg1)
}

// LargePreimageMeta is a free data retrieval call binding the contract method 0xd709ebef.
//
// Solidity: function largePreimageMeta(address , uint256 ) view returns(uint128 offset, uint64 claimedSize, uint64 size, bytes32 preimagePart)
func (_PreimageOracle *PreimageOracleCallerSession) LargePreimageMeta(arg0 common.Address, arg1 *big.Int) (struct {
	Offset       *big.Int
	ClaimedSize  uint64
	Size         uint64
	PreimagePart [32]byte
}, error) {
	return _PreimageOracle.Contract.LargePreimageMeta(&_PreimageOracle.CallOpts, arg0, arg1)
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

// StateMatrices is a free data retrieval call binding the contract method 0x3a7492de.
//
// Solidity: function stateMatrices(address , uint256 , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleCaller) StateMatrices(opts *bind.CallOpts, arg0 common.Address, arg1 *big.Int, arg2 *big.Int) (uint64, error) {
	var out []interface{}
	err := _PreimageOracle.contract.Call(opts, &out, "stateMatrices", arg0, arg1, arg2)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// StateMatrices is a free data retrieval call binding the contract method 0x3a7492de.
//
// Solidity: function stateMatrices(address , uint256 , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleSession) StateMatrices(arg0 common.Address, arg1 *big.Int, arg2 *big.Int) (uint64, error) {
	return _PreimageOracle.Contract.StateMatrices(&_PreimageOracle.CallOpts, arg0, arg1, arg2)
}

// StateMatrices is a free data retrieval call binding the contract method 0x3a7492de.
//
// Solidity: function stateMatrices(address , uint256 , uint256 ) view returns(uint64)
func (_PreimageOracle *PreimageOracleCallerSession) StateMatrices(arg0 common.Address, arg1 *big.Int, arg2 *big.Int) (uint64, error) {
	return _PreimageOracle.Contract.StateMatrices(&_PreimageOracle.CallOpts, arg0, arg1, arg2)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0xa2c3e141.
//
// Solidity: function absorbLargePreimagePart(uint256 _contextKey, bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleTransactor) AbsorbLargePreimagePart(opts *bind.TransactOpts, _contextKey *big.Int, _data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "absorbLargePreimagePart", _contextKey, _data, _finalize)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0xa2c3e141.
//
// Solidity: function absorbLargePreimagePart(uint256 _contextKey, bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleSession) AbsorbLargePreimagePart(_contextKey *big.Int, _data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.Contract.AbsorbLargePreimagePart(&_PreimageOracle.TransactOpts, _contextKey, _data, _finalize)
}

// AbsorbLargePreimagePart is a paid mutator transaction binding the contract method 0xa2c3e141.
//
// Solidity: function absorbLargePreimagePart(uint256 _contextKey, bytes _data, bool _finalize) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) AbsorbLargePreimagePart(_contextKey *big.Int, _data []byte, _finalize bool) (*types.Transaction, error) {
	return _PreimageOracle.Contract.AbsorbLargePreimagePart(&_PreimageOracle.TransactOpts, _contextKey, _data, _finalize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0x9ec50118.
//
// Solidity: function initLargeKeccak256Preimage(uint256 _contextKey, uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleTransactor) InitLargeKeccak256Preimage(opts *bind.TransactOpts, _contextKey *big.Int, _offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "initLargeKeccak256Preimage", _contextKey, _offset, _claimedSize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0x9ec50118.
//
// Solidity: function initLargeKeccak256Preimage(uint256 _contextKey, uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleSession) InitLargeKeccak256Preimage(_contextKey *big.Int, _offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.Contract.InitLargeKeccak256Preimage(&_PreimageOracle.TransactOpts, _contextKey, _offset, _claimedSize)
}

// InitLargeKeccak256Preimage is a paid mutator transaction binding the contract method 0x9ec50118.
//
// Solidity: function initLargeKeccak256Preimage(uint256 _contextKey, uint128 _offset, uint64 _claimedSize) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) InitLargeKeccak256Preimage(_contextKey *big.Int, _offset *big.Int, _claimedSize uint64) (*types.Transaction, error) {
	return _PreimageOracle.Contract.InitLargeKeccak256Preimage(&_PreimageOracle.TransactOpts, _contextKey, _offset, _claimedSize)
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

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x55fbfaea.
//
// Solidity: function squeezeLargePreimagePart(uint256 _contextKey) returns()
func (_PreimageOracle *PreimageOracleTransactor) SqueezeLargePreimagePart(opts *bind.TransactOpts, _contextKey *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.contract.Transact(opts, "squeezeLargePreimagePart", _contextKey)
}

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x55fbfaea.
//
// Solidity: function squeezeLargePreimagePart(uint256 _contextKey) returns()
func (_PreimageOracle *PreimageOracleSession) SqueezeLargePreimagePart(_contextKey *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.SqueezeLargePreimagePart(&_PreimageOracle.TransactOpts, _contextKey)
}

// SqueezeLargePreimagePart is a paid mutator transaction binding the contract method 0x55fbfaea.
//
// Solidity: function squeezeLargePreimagePart(uint256 _contextKey) returns()
func (_PreimageOracle *PreimageOracleTransactorSession) SqueezeLargePreimagePart(_contextKey *big.Int) (*types.Transaction, error) {
	return _PreimageOracle.Contract.SqueezeLargePreimagePart(&_PreimageOracle.TransactOpts, _contextKey)
}

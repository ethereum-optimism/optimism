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

// GasPriceOracleMetaData contains all meta data concerning the GasPriceOracle contract.
var GasPriceOracleMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"DECIMALS\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseFee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"baseFeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blobBaseFee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"blobBaseFeeScalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"gasPrice\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getL1Fee\",\"inputs\":[{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getL1FeeUpperBound\",\"inputs\":[{\"name\":\"_unsignedTxSize\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getL1GasUsed\",\"inputs\":[{\"name\":\"_data\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isEcotone\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isFjord\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l1BaseFee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"overhead\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"scalar\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setEcotone\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setFjord\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x608060405234801561001057600080fd5b50611951806100206000396000f3fe608060405234801561001057600080fd5b50600436106101365760003560e01c80636ef25c3a116100b2578063de26c4a111610081578063f45e65d811610066578063f45e65d81461025b578063f820614014610263578063fe173b971461020d57600080fd5b8063de26c4a114610235578063f1c7a58b1461024857600080fd5b80636ef25c3a1461020d5780638e98b10614610213578063960e3a231461021b578063c59859181461022d57600080fd5b806349948e0e11610109578063519b4bd3116100ee578063519b4bd31461019f57806354fd4d50146101a757806368d5dca6146101f057600080fd5b806349948e0e1461016f5780634ef6e2241461018257600080fd5b80630c18c1621461013b57806322b90ab3146101565780632e0f262514610160578063313ce56714610168575b600080fd5b61014361026b565b6040519081526020015b60405180910390f35b61015e61038c565b005b610143600681565b6006610143565b61014361017d3660046113c6565b6105af565b60005461018f9060ff1681565b604051901515815260200161014d565b610143610600565b6101e36040518060400160405280600581526020017f312e332e3000000000000000000000000000000000000000000000000000000081525081565b60405161014d9190611495565b6101f8610661565b60405163ffffffff909116815260200161014d565b48610143565b61015e6106e6565b60005461018f90610100900460ff1681565b6101f861097a565b6101436102433660046113c6565b6109db565b610143610256366004611508565b610ad5565b610143610ba5565b610143610c98565b6000805460ff1615610304576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602860248201527f47617350726963654f7261636c653a206f76657268656164282920697320646560448201527f707265636174656400000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16638b239f736040518163ffffffff1660e01b8152600401602060405180830381865afa158015610363573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906103879190611521565b905090565b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff1663e591b2826040518163ffffffff1660e01b8152600401602060405180830381865afa1580156103eb573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061040f919061153a565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146104ef576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604160248201527f47617350726963654f7261636c653a206f6e6c7920746865206465706f73697460448201527f6f72206163636f756e742063616e2073657420697345636f746f6e6520666c6160648201527f6700000000000000000000000000000000000000000000000000000000000000608482015260a4016102fb565b60005460ff1615610582576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f47617350726963654f7261636c653a2045636f746f6e6520616c72656164792060448201527f616374697665000000000000000000000000000000000000000000000000000060648201526084016102fb565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001179055565b60008054610100900460ff16156105e3576105dd6105cc83610cf9565b516105d890604461159f565b61101e565b92915050565b60005460ff16156105f7576105dd826110b0565b6105dd82611154565b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16635cf249696040518163ffffffff1660e01b8152600401602060405180830381865afa158015610363573d6000803e3d6000fd5b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff166368d5dca66040518163ffffffff1660e01b8152600401602060405180830381865afa1580156106c2573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061038791906115b7565b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff1663e591b2826040518163ffffffff1660e01b8152600401602060405180830381865afa158015610745573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610769919061153a565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610823576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603f60248201527f47617350726963654f7261636c653a206f6e6c7920746865206465706f73697460448201527f6f72206163636f756e742063616e20736574206973466a6f726420666c61670060648201526084016102fb565b60005460ff166108b5576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603960248201527f47617350726963654f7261636c653a20466a6f72642063616e206f6e6c79206260448201527f65206163746976617465642061667465722045636f746f6e650000000000000060648201526084016102fb565b600054610100900460ff161561094c576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f47617350726963654f7261636c653a20466a6f726420616c726561647920616360448201527f746976650000000000000000000000000000000000000000000000000000000060648201526084016102fb565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16610100179055565b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff1663c59859186040518163ffffffff1660e01b8152600401602060405180830381865afa1580156106c2573d6000803e3d6000fd5b60008054610100900460ff1615610a2257620f4240610a0d6109fc84610cf9565b51610a0890604461159f565b6112a8565b610a189060106115dd565b6105dd919061161a565b6000610a2d83611307565b60005490915060ff1615610a415792915050565b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16638b239f736040518163ffffffff1660e01b8152600401602060405180830381865afa158015610aa0573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610ac49190611521565b610ace908261159f565b9392505050565b60008054610100900460ff16610b6d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f47617350726963654f7261636c653a206765744c314665655570706572426f7560448201527f6e64206f6e6c7920737570706f72747320466a6f72640000000000000000000060648201526084016102fb565b6000610b7a60ff8461161a565b610b84908461159f565b610b8f90601061159f565b610b9a90604461159f565b9050610ace8161101e565b6000805460ff1615610c39576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f47617350726963654f7261636c653a207363616c61722829206973206465707260448201527f656361746564000000000000000000000000000000000000000000000000000060648201526084016102fb565b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16639e8c49666040518163ffffffff1660e01b8152600401602060405180830381865afa158015610363573d6000803e3d6000fd5b600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff1663f82061406040518163ffffffff1660e01b8152600401602060405180830381865afa158015610363573d6000803e3d6000fd5b6060610e90565b818153600101919050565b600082840393505b83811015610ace5782810151828201511860001a1590930292600101610d13565b825b60208210610d80578251610d4b601f83610d00565b52602092909201917fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe090910190602101610d36565b8115610ace578251610d956001840383610d00565b520160010192915050565b60006001830392505b6101078210610de157610dd38360ff16610dce60fd610dce8760081c60e00189610d00565b610d00565b935061010682039150610da9565b60078210610e0e57610e078360ff16610dce60078503610dce8760081c60e00189610d00565b9050610ace565b610e278360ff16610dce8560081c8560051b0187610d00565b949350505050565b610e88828203610e6c610e5c84600081518060001a8160011a60081b178160021a60101b17915050919050565b639e3779b90260131c611fff1690565b8060021b6040510182815160e01c1860e01b8151188152505050565b600101919050565b6180003860405139618000604051016020830180600d8551820103826002015b81811015610fc3576000805b50508051604051600082901a600183901a60081b1760029290921a60101b91909117639e3779b9810260111c617ffc16909101805160e081811c878603811890911b90911890915284019081830390848410610f185750610f53565b600184019350611fff8211610f4d578251600081901a600182901a60081b1760029190911a60101b178103610f4d5750610f53565b50610ebc565b838310610f61575050610fc3565b60018303925085831115610f7f57610f7c8787888603610d34565b96505b610f93600985016003850160038501610d0b565b9150610fa0878284610da0565b965050610fb884610fb386848601610e2f565b610e2f565b915050809350610eb0565b5050610fd58383848851850103610d34565b925050506040519150618000820180820391508183526020830160005b8381101561100a578281015182820152602001610ff2565b506000920191825250602001604052919050565b60008061102a836112a8565b90506000611036610c98565b61103e610661565b63ffffffff1661104e91906115dd565b611056610600565b61105e61097a565b611069906010611655565b63ffffffff1661107991906115dd565b611083919061159f565b9050611091600660026115dd565b61109c90600a6117a1565b6110a682846115dd565b610e27919061161a565b6000806110bc83611307565b905060006110c8610600565b6110d061097a565b6110db906010611655565b63ffffffff166110eb91906115dd565b905060006110f7610c98565b6110ff610661565b63ffffffff1661110f91906115dd565b9050600061111d828461159f565b61112790856115dd565b90506111356006600a6117a1565b6111409060106115dd565b61114a908261161a565b9695505050505050565b60008061116083611307565b9050600073420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16639e8c49666040518163ffffffff1660e01b8152600401602060405180830381865afa1580156111c3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906111e79190611521565b6111ef610600565b73420000000000000000000000000000000000001573ffffffffffffffffffffffffffffffffffffffff16638b239f736040518163ffffffff1660e01b8152600401602060405180830381865afa15801561124e573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112729190611521565b61127c908561159f565b61128691906115dd565b61129091906115dd565b905061129e6006600a6117a1565b610e27908261161a565b6000806112b883620cc3946115dd565b6112e2907ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd7632006117ad565b90506112f26064620f4240611821565b8112156105dd57610ace6064620f4240611821565b80516000908190815b8181101561138a5784818151811061132a5761132a6118dd565b01602001517fff000000000000000000000000000000000000000000000000000000000000001660000361136a5761136360048461159f565b9250611378565b61137560108461159f565b92505b806113828161190c565b915050611310565b50610e278261044061159f565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000602082840312156113d857600080fd5b813567ffffffffffffffff808211156113f057600080fd5b818401915084601f83011261140457600080fd5b81358181111561141657611416611397565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f0116810190838211818310171561145c5761145c611397565b8160405282815287602084870101111561147557600080fd5b826020860160208301376000928101602001929092525095945050505050565b600060208083528351808285015260005b818110156114c2578581018301518582016040015282016114a6565b818111156114d4576000604083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe016929092016040019392505050565b60006020828403121561151a57600080fd5b5035919050565b60006020828403121561153357600080fd5b5051919050565b60006020828403121561154c57600080fd5b815173ffffffffffffffffffffffffffffffffffffffff81168114610ace57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156115b2576115b2611570565b500190565b6000602082840312156115c957600080fd5b815163ffffffff81168114610ace57600080fd5b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff048311821515161561161557611615611570565b500290565b600082611650577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b500490565b600063ffffffff8083168185168183048111821515161561167857611678611570565b02949350505050565b600181815b808511156116da57817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff048211156116c0576116c0611570565b808516156116cd57918102915b93841c9390800290611686565b509250929050565b6000826116f1575060016105dd565b816116fe575060006105dd565b8160018114611714576002811461171e5761173a565b60019150506105dd565b60ff84111561172f5761172f611570565b50506001821b6105dd565b5060208310610133831016604e8410600b841016171561175d575081810a6105dd565b6117678383611681565b807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0482111561179957611799611570565b029392505050565b6000610ace83836116e2565b6000808212827f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038413811516156117e7576117e7611570565b827f800000000000000000000000000000000000000000000000000000000000000003841281161561181b5761181b611570565b50500190565b60007f7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff60008413600084138583048511828216161561186257611862611570565b7f8000000000000000000000000000000000000000000000000000000000000000600087128682058812818416161561189d5761189d611570565b600087129250878205871284841616156118b9576118b9611570565b878505871281841616156118cf576118cf611570565b505050929093029392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361193d5761193d611570565b506001019056fea164736f6c634300080f000a",
}

// GasPriceOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use GasPriceOracleMetaData.ABI instead.
var GasPriceOracleABI = GasPriceOracleMetaData.ABI

// GasPriceOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use GasPriceOracleMetaData.Bin instead.
var GasPriceOracleBin = GasPriceOracleMetaData.Bin

// DeployGasPriceOracle deploys a new Ethereum contract, binding an instance of GasPriceOracle to it.
func DeployGasPriceOracle(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *GasPriceOracle, error) {
	parsed, err := GasPriceOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(GasPriceOracleBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &GasPriceOracle{GasPriceOracleCaller: GasPriceOracleCaller{contract: contract}, GasPriceOracleTransactor: GasPriceOracleTransactor{contract: contract}, GasPriceOracleFilterer: GasPriceOracleFilterer{contract: contract}}, nil
}

// GasPriceOracle is an auto generated Go binding around an Ethereum contract.
type GasPriceOracle struct {
	GasPriceOracleCaller     // Read-only binding to the contract
	GasPriceOracleTransactor // Write-only binding to the contract
	GasPriceOracleFilterer   // Log filterer for contract events
}

// GasPriceOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type GasPriceOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type GasPriceOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type GasPriceOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// GasPriceOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type GasPriceOracleSession struct {
	Contract     *GasPriceOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// GasPriceOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type GasPriceOracleCallerSession struct {
	Contract *GasPriceOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// GasPriceOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type GasPriceOracleTransactorSession struct {
	Contract     *GasPriceOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// GasPriceOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type GasPriceOracleRaw struct {
	Contract *GasPriceOracle // Generic contract binding to access the raw methods on
}

// GasPriceOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type GasPriceOracleCallerRaw struct {
	Contract *GasPriceOracleCaller // Generic read-only contract binding to access the raw methods on
}

// GasPriceOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type GasPriceOracleTransactorRaw struct {
	Contract *GasPriceOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewGasPriceOracle creates a new instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracle(address common.Address, backend bind.ContractBackend) (*GasPriceOracle, error) {
	contract, err := bindGasPriceOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracle{GasPriceOracleCaller: GasPriceOracleCaller{contract: contract}, GasPriceOracleTransactor: GasPriceOracleTransactor{contract: contract}, GasPriceOracleFilterer: GasPriceOracleFilterer{contract: contract}}, nil
}

// NewGasPriceOracleCaller creates a new read-only instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleCaller(address common.Address, caller bind.ContractCaller) (*GasPriceOracleCaller, error) {
	contract, err := bindGasPriceOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleCaller{contract: contract}, nil
}

// NewGasPriceOracleTransactor creates a new write-only instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*GasPriceOracleTransactor, error) {
	contract, err := bindGasPriceOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleTransactor{contract: contract}, nil
}

// NewGasPriceOracleFilterer creates a new log filterer instance of GasPriceOracle, bound to a specific deployed contract.
func NewGasPriceOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*GasPriceOracleFilterer, error) {
	contract, err := bindGasPriceOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &GasPriceOracleFilterer{contract: contract}, nil
}

// bindGasPriceOracle binds a generic wrapper to an already deployed contract.
func bindGasPriceOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(GasPriceOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasPriceOracle *GasPriceOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasPriceOracle.Contract.GasPriceOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasPriceOracle *GasPriceOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.GasPriceOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasPriceOracle *GasPriceOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.GasPriceOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_GasPriceOracle *GasPriceOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _GasPriceOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_GasPriceOracle *GasPriceOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_GasPriceOracle *GasPriceOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _GasPriceOracle.Contract.contract.Transact(opts, method, params...)
}

// DECIMALS is a free data retrieval call binding the contract method 0x2e0f2625.
//
// Solidity: function DECIMALS() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) DECIMALS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "DECIMALS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DECIMALS is a free data retrieval call binding the contract method 0x2e0f2625.
//
// Solidity: function DECIMALS() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) DECIMALS() (*big.Int, error) {
	return _GasPriceOracle.Contract.DECIMALS(&_GasPriceOracle.CallOpts)
}

// DECIMALS is a free data retrieval call binding the contract method 0x2e0f2625.
//
// Solidity: function DECIMALS() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) DECIMALS() (*big.Int, error) {
	return _GasPriceOracle.Contract.DECIMALS(&_GasPriceOracle.CallOpts)
}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "baseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BaseFee(&_GasPriceOracle.CallOpts)
}

// BaseFee is a free data retrieval call binding the contract method 0x6ef25c3a.
//
// Solidity: function baseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BaseFee(&_GasPriceOracle.CallOpts)
}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleCaller) BaseFeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "baseFeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleSession) BaseFeeScalar() (uint32, error) {
	return _GasPriceOracle.Contract.BaseFeeScalar(&_GasPriceOracle.CallOpts)
}

// BaseFeeScalar is a free data retrieval call binding the contract method 0xc5985918.
//
// Solidity: function baseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleCallerSession) BaseFeeScalar() (uint32, error) {
	return _GasPriceOracle.Contract.BaseFeeScalar(&_GasPriceOracle.CallOpts)
}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) BlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "blobBaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) BlobBaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BlobBaseFee(&_GasPriceOracle.CallOpts)
}

// BlobBaseFee is a free data retrieval call binding the contract method 0xf8206140.
//
// Solidity: function blobBaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) BlobBaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.BlobBaseFee(&_GasPriceOracle.CallOpts)
}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleCaller) BlobBaseFeeScalar(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "blobBaseFeeScalar")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleSession) BlobBaseFeeScalar() (uint32, error) {
	return _GasPriceOracle.Contract.BlobBaseFeeScalar(&_GasPriceOracle.CallOpts)
}

// BlobBaseFeeScalar is a free data retrieval call binding the contract method 0x68d5dca6.
//
// Solidity: function blobBaseFeeScalar() view returns(uint32)
func (_GasPriceOracle *GasPriceOracleCallerSession) BlobBaseFeeScalar() (uint32, error) {
	return _GasPriceOracle.Contract.BlobBaseFeeScalar(&_GasPriceOracle.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Decimals(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Decimals() (*big.Int, error) {
	return _GasPriceOracle.Contract.Decimals(&_GasPriceOracle.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Decimals() (*big.Int, error) {
	return _GasPriceOracle.Contract.Decimals(&_GasPriceOracle.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GasPrice(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "gasPrice")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GasPrice() (*big.Int, error) {
	return _GasPriceOracle.Contract.GasPrice(&_GasPriceOracle.CallOpts)
}

// GasPrice is a free data retrieval call binding the contract method 0xfe173b97.
//
// Solidity: function gasPrice() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GasPrice() (*big.Int, error) {
	return _GasPriceOracle.Contract.GasPrice(&_GasPriceOracle.CallOpts)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GetL1Fee(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "getL1Fee", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1Fee(&_GasPriceOracle.CallOpts, _data)
}

// GetL1Fee is a free data retrieval call binding the contract method 0x49948e0e.
//
// Solidity: function getL1Fee(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GetL1Fee(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1Fee(&_GasPriceOracle.CallOpts, _data)
}

// GetL1FeeUpperBound is a free data retrieval call binding the contract method 0xf1c7a58b.
//
// Solidity: function getL1FeeUpperBound(uint256 _unsignedTxSize) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GetL1FeeUpperBound(opts *bind.CallOpts, _unsignedTxSize *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "getL1FeeUpperBound", _unsignedTxSize)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1FeeUpperBound is a free data retrieval call binding the contract method 0xf1c7a58b.
//
// Solidity: function getL1FeeUpperBound(uint256 _unsignedTxSize) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GetL1FeeUpperBound(_unsignedTxSize *big.Int) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1FeeUpperBound(&_GasPriceOracle.CallOpts, _unsignedTxSize)
}

// GetL1FeeUpperBound is a free data retrieval call binding the contract method 0xf1c7a58b.
//
// Solidity: function getL1FeeUpperBound(uint256 _unsignedTxSize) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GetL1FeeUpperBound(_unsignedTxSize *big.Int) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1FeeUpperBound(&_GasPriceOracle.CallOpts, _unsignedTxSize)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) GetL1GasUsed(opts *bind.CallOpts, _data []byte) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "getL1GasUsed", _data)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1GasUsed(&_GasPriceOracle.CallOpts, _data)
}

// GetL1GasUsed is a free data retrieval call binding the contract method 0xde26c4a1.
//
// Solidity: function getL1GasUsed(bytes _data) view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) GetL1GasUsed(_data []byte) (*big.Int, error) {
	return _GasPriceOracle.Contract.GetL1GasUsed(&_GasPriceOracle.CallOpts, _data)
}

// IsEcotone is a free data retrieval call binding the contract method 0x4ef6e224.
//
// Solidity: function isEcotone() view returns(bool)
func (_GasPriceOracle *GasPriceOracleCaller) IsEcotone(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "isEcotone")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsEcotone is a free data retrieval call binding the contract method 0x4ef6e224.
//
// Solidity: function isEcotone() view returns(bool)
func (_GasPriceOracle *GasPriceOracleSession) IsEcotone() (bool, error) {
	return _GasPriceOracle.Contract.IsEcotone(&_GasPriceOracle.CallOpts)
}

// IsEcotone is a free data retrieval call binding the contract method 0x4ef6e224.
//
// Solidity: function isEcotone() view returns(bool)
func (_GasPriceOracle *GasPriceOracleCallerSession) IsEcotone() (bool, error) {
	return _GasPriceOracle.Contract.IsEcotone(&_GasPriceOracle.CallOpts)
}

// IsFjord is a free data retrieval call binding the contract method 0x960e3a23.
//
// Solidity: function isFjord() view returns(bool)
func (_GasPriceOracle *GasPriceOracleCaller) IsFjord(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "isFjord")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsFjord is a free data retrieval call binding the contract method 0x960e3a23.
//
// Solidity: function isFjord() view returns(bool)
func (_GasPriceOracle *GasPriceOracleSession) IsFjord() (bool, error) {
	return _GasPriceOracle.Contract.IsFjord(&_GasPriceOracle.CallOpts)
}

// IsFjord is a free data retrieval call binding the contract method 0x960e3a23.
//
// Solidity: function isFjord() view returns(bool)
func (_GasPriceOracle *GasPriceOracleCallerSession) IsFjord() (bool, error) {
	return _GasPriceOracle.Contract.IsFjord(&_GasPriceOracle.CallOpts)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) L1BaseFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "l1BaseFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) L1BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.L1BaseFee(&_GasPriceOracle.CallOpts)
}

// L1BaseFee is a free data retrieval call binding the contract method 0x519b4bd3.
//
// Solidity: function l1BaseFee() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) L1BaseFee() (*big.Int, error) {
	return _GasPriceOracle.Contract.L1BaseFee(&_GasPriceOracle.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Overhead(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "overhead")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Overhead() (*big.Int, error) {
	return _GasPriceOracle.Contract.Overhead(&_GasPriceOracle.CallOpts)
}

// Overhead is a free data retrieval call binding the contract method 0x0c18c162.
//
// Solidity: function overhead() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Overhead() (*big.Int, error) {
	return _GasPriceOracle.Contract.Overhead(&_GasPriceOracle.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCaller) Scalar(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "scalar")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleSession) Scalar() (*big.Int, error) {
	return _GasPriceOracle.Contract.Scalar(&_GasPriceOracle.CallOpts)
}

// Scalar is a free data retrieval call binding the contract method 0xf45e65d8.
//
// Solidity: function scalar() view returns(uint256)
func (_GasPriceOracle *GasPriceOracleCallerSession) Scalar() (*big.Int, error) {
	return _GasPriceOracle.Contract.Scalar(&_GasPriceOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _GasPriceOracle.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleSession) Version() (string, error) {
	return _GasPriceOracle.Contract.Version(&_GasPriceOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_GasPriceOracle *GasPriceOracleCallerSession) Version() (string, error) {
	return _GasPriceOracle.Contract.Version(&_GasPriceOracle.CallOpts)
}

// SetEcotone is a paid mutator transaction binding the contract method 0x22b90ab3.
//
// Solidity: function setEcotone() returns()
func (_GasPriceOracle *GasPriceOracleTransactor) SetEcotone(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "setEcotone")
}

// SetEcotone is a paid mutator transaction binding the contract method 0x22b90ab3.
//
// Solidity: function setEcotone() returns()
func (_GasPriceOracle *GasPriceOracleSession) SetEcotone() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetEcotone(&_GasPriceOracle.TransactOpts)
}

// SetEcotone is a paid mutator transaction binding the contract method 0x22b90ab3.
//
// Solidity: function setEcotone() returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) SetEcotone() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetEcotone(&_GasPriceOracle.TransactOpts)
}

// SetFjord is a paid mutator transaction binding the contract method 0x8e98b106.
//
// Solidity: function setFjord() returns()
func (_GasPriceOracle *GasPriceOracleTransactor) SetFjord(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _GasPriceOracle.contract.Transact(opts, "setFjord")
}

// SetFjord is a paid mutator transaction binding the contract method 0x8e98b106.
//
// Solidity: function setFjord() returns()
func (_GasPriceOracle *GasPriceOracleSession) SetFjord() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetFjord(&_GasPriceOracle.TransactOpts)
}

// SetFjord is a paid mutator transaction binding the contract method 0x8e98b106.
//
// Solidity: function setFjord() returns()
func (_GasPriceOracle *GasPriceOracleTransactorSession) SetFjord() (*types.Transaction, error) {
	return _GasPriceOracle.Contract.SetFjord(&_GasPriceOracle.TransactOpts)
}

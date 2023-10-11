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

// OptimismMintableERC20FactoryMetaData contains all meta data concerning the OptimismMintableERC20Factory contract.
var OptimismMintableERC20FactoryMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"deployer\",\"type\":\"address\"}],\"name\":\"OptimismMintableERC20Created\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"remoteToken\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"localToken\",\"type\":\"address\"}],\"name\":\"StandardL2TokenCreated\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"BRIDGE\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bridge\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"name\":\"createOptimismMintableERC20\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"}],\"name\":\"createOptimismMintableERC20WithDecimals\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_remoteToken\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"_symbol\",\"type\":\"string\"}],\"name\":\"createStandardL2Token\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_bridge\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b5061001b6000610020565b610118565b600054600390610100900460ff16158015610042575060005460ff8083169116105b6100a95760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b606482015260840160405180910390fd5b6000805461010060ff841661ffff19909216821717610100600160b01b03191661ff0019620100006001600160a01b0387160216179091556040519081527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6123cf806101276000396000f3fe60806040523480156200001157600080fd5b5060043610620000875760003560e01c8063c4d66de81162000062578063c4d66de81462000135578063ce5ac90f146200014e578063e78cea921462000165578063ee9a31a2146200018c57600080fd5b806354fd4d50146200008c578063896f93d114620000e15780638cf0629c146200011e575b600080fd5b620000c96040518060400160405280600581526020017f312e362e3000000000000000000000000000000000000000000000000000000081525081565b604051620000d89190620005d1565b60405180910390f35b620000f8620000f2366004620006f9565b620001b1565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001620000d8565b620000f86200012f36600462000776565b620001c8565b6200014c620001463660046200080d565b620003c6565b005b620000f86200015f366004620006f9565b62000544565b600054620000f89062010000900473ffffffffffffffffffffffffffffffffffffffff1681565b60005462010000900473ffffffffffffffffffffffffffffffffffffffff16620000f8565b6000620001c084848462000544565b949350505050565b600073ffffffffffffffffffffffffffffffffffffffff851662000273576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603f60248201527f4f7074696d69736d4d696e7461626c654552433230466163746f72793a206d7560448201527f73742070726f766964652072656d6f746520746f6b656e20616464726573730060648201526084015b60405180910390fd5b6000858585856040516020016200028e94939291906200082b565b604051602081830303815290604052805190602001209050600081600060029054906101000a900473ffffffffffffffffffffffffffffffffffffffff1688888888604051620002de9062000555565b620002ee95949392919062000885565b8190604051809103906000f59050801580156200030f573d6000803e3d6000fd5b5090508073ffffffffffffffffffffffffffffffffffffffff168773ffffffffffffffffffffffffffffffffffffffff167fceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf60405160405180910390a360405133815273ffffffffffffffffffffffffffffffffffffffff80891691908316907f52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb9060200160405180910390a39695505050505050565b600054600390610100900460ff16158015620003e9575060005460ff8083169116105b62000477576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016200026a565b6000805461010060ff84167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00009092168217177fffffffffffffffffffff000000000000000000000000000000000000000000ff167fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff6201000073ffffffffffffffffffffffffffffffffffffffff87160216179091556040519081527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15050565b6000620001c08484846012620001c8565b611ad880620008eb83390190565b6000815180845260005b818110156200058b576020818501810151868301820152016200056d565b818111156200059e576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000620005e6602083018462000563565b9392505050565b803573ffffffffffffffffffffffffffffffffffffffff811681146200061257600080fd5b919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b600082601f8301126200065857600080fd5b813567ffffffffffffffff8082111562000676576200067662000617565b604051601f83017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908282118183101715620006bf57620006bf62000617565b81604052838152866020858801011115620006d957600080fd5b836020870160208301376000602085830101528094505050505092915050565b6000806000606084860312156200070f57600080fd5b6200071a84620005ed565b9250602084013567ffffffffffffffff808211156200073857600080fd5b620007468783880162000646565b935060408601359150808211156200075d57600080fd5b506200076c8682870162000646565b9150509250925092565b600080600080608085870312156200078d57600080fd5b6200079885620005ed565b9350602085013567ffffffffffffffff80821115620007b657600080fd5b620007c48883890162000646565b94506040870135915080821115620007db57600080fd5b50620007ea8782880162000646565b925050606085013560ff811681146200080257600080fd5b939692955090935050565b6000602082840312156200082057600080fd5b620005e682620005ed565b73ffffffffffffffffffffffffffffffffffffffff851681526080602082015260006200085c608083018662000563565b828103604084015262000870818662000563565b91505060ff8316606083015295945050505050565b600073ffffffffffffffffffffffffffffffffffffffff808816835280871660208401525060a06040830152620008c060a083018662000563565b8281036060840152620008d4818662000563565b91505060ff83166080830152969550505050505056fe6101406040523480156200001257600080fd5b5060405162001ad838038062001ad8833981016040819052620000359162000178565b600160026000858560036200004b8382620002b3565b5060046200005a8282620002b3565b50505060809290925260a05260c0526001600160a01b0393841660e0529390921661010052505060ff16610120526200037f565b80516001600160a01b0381168114620000a657600080fd5b919050565b634e487b7160e01b600052604160045260246000fd5b600082601f830112620000d357600080fd5b81516001600160401b0380821115620000f057620000f0620000ab565b604051601f8301601f19908116603f011681019082821181831017156200011b576200011b620000ab565b816040528381526020925086838588010111156200013857600080fd5b600091505b838210156200015c57858201830151818301840152908201906200013d565b838211156200016e5760008385830101525b9695505050505050565b600080600080600060a086880312156200019157600080fd5b6200019c866200008e565b9450620001ac602087016200008e565b60408701519094506001600160401b0380821115620001ca57600080fd5b620001d889838a01620000c1565b94506060880151915080821115620001ef57600080fd5b50620001fe88828901620000c1565b925050608086015160ff811681146200021657600080fd5b809150509295509295909350565b600181811c908216806200023957607f821691505b6020821081036200025a57634e487b7160e01b600052602260045260246000fd5b50919050565b601f821115620002ae57600081815260208120601f850160051c81016020861015620002895750805b601f850160051c820191505b81811015620002aa5782815560010162000295565b5050505b505050565b81516001600160401b03811115620002cf57620002cf620000ab565b620002e781620002e0845462000224565b8462000260565b602080601f8311600181146200031f5760008415620003065750858301515b600019600386901b1c1916600185901b178555620002aa565b600085815260208120601f198616915b8281101562000350578886015182559484019460019091019084016200032f565b50858210156200036f5787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b60805160a05160c05160e05161010051610120516116ed620003eb6000396000610244015260008181610317015281816103ac015281816105f101526107cb0152600081816101a9015261033d0152600061075a015260006107310152600061070801526116ed6000f3fe608060405234801561001057600080fd5b50600436106101775760003560e01c806370a08231116100d8578063ae1f6aaf1161008c578063dd62ed3e11610066578063dd62ed3e14610361578063e78cea9214610315578063ee9a31a2146103a757600080fd5b8063ae1f6aaf14610315578063c01e1bd61461033b578063d6c0b2c41461033b57600080fd5b80639dc29fac116100bd5780639dc29fac146102dc578063a457c2d7146102ef578063a9059cbb1461030257600080fd5b806370a082311461029e57806395d89b41146102d457600080fd5b806323b872dd1161012f5780633950935111610114578063395093511461026e57806340c10f191461028157806354fd4d501461029657600080fd5b806323b872dd1461022a578063313ce5671461023d57600080fd5b806306fdde031161016057806306fdde03146101f0578063095ea7b31461020557806318160ddd1461021857600080fd5b806301ffc9a71461017c578063033964be146101a4575b600080fd5b61018f61018a366004611329565b6103ce565b60405190151581526020015b60405180910390f35b6101cb7f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161019b565b6101f86104bf565b60405161019b919061139e565b61018f610213366004611418565b610551565b6002545b60405190815260200161019b565b61018f610238366004611442565b610569565b60405160ff7f000000000000000000000000000000000000000000000000000000000000000016815260200161019b565b61018f61027c366004611418565b61058d565b61029461028f366004611418565b6105d9565b005b6101f8610701565b61021c6102ac36600461147e565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b6101f86107a4565b6102946102ea366004611418565b6107b3565b61018f6102fd366004611418565b6108ca565b61018f610310366004611418565b61099b565b7f00000000000000000000000000000000000000000000000000000000000000006101cb565b7f00000000000000000000000000000000000000000000000000000000000000006101cb565b61021c61036f366004611499565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260016020908152604080832093909416825291909152205490565b6101cb7f000000000000000000000000000000000000000000000000000000000000000081565b60007f01ffc9a7000000000000000000000000000000000000000000000000000000007f1d1d8b63000000000000000000000000000000000000000000000000000000007fec4fc8e3000000000000000000000000000000000000000000000000000000007fffffffff00000000000000000000000000000000000000000000000000000000851683148061048757507fffffffff00000000000000000000000000000000000000000000000000000000858116908316145b806104b657507fffffffff00000000000000000000000000000000000000000000000000000000858116908216145b95945050505050565b6060600380546104ce906114cc565b80601f01602080910402602001604051908101604052809291908181526020018280546104fa906114cc565b80156105475780601f1061051c57610100808354040283529160200191610547565b820191906000526020600020905b81548152906001019060200180831161052a57829003601f168201915b5050505050905090565b60003361055f8185856109a9565b5060019392505050565b600033610577858285610b5d565b610582858585610c34565b506001949350505050565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919061055f90829086906105d490879061154e565b6109a9565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016146106a3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4f7074696d69736d4d696e7461626c6545524332303a206f6e6c79206272696460448201527f67652063616e206d696e7420616e64206275726e00000000000000000000000060648201526084015b60405180910390fd5b6106ad8282610ee7565b8173ffffffffffffffffffffffffffffffffffffffff167f0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885826040516106f591815260200190565b60405180910390a25050565b606061072c7f0000000000000000000000000000000000000000000000000000000000000000611007565b6107557f0000000000000000000000000000000000000000000000000000000000000000611007565b61077e7f0000000000000000000000000000000000000000000000000000000000000000611007565b60405160200161079093929190611566565b604051602081830303815290604052905090565b6060600480546104ce906114cc565b3373ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001614610878576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603460248201527f4f7074696d69736d4d696e7461626c6545524332303a206f6e6c79206272696460448201527f67652063616e206d696e7420616e64206275726e000000000000000000000000606482015260840161069a565b6108828282611144565b8173ffffffffffffffffffffffffffffffffffffffff167fcc16f5dbb4873280815c1ee09dbd06736cffcc184412cf7a71a0fdb75d397ca5826040516106f591815260200190565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff871684529091528120549091908381101561098e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f000000000000000000000000000000000000000000000000000000606482015260840161069a565b61058282868684036109a9565b60003361055f818585610c34565b73ffffffffffffffffffffffffffffffffffffffff8316610a4b576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f7265737300000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff8216610aee576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f7373000000000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92591015b60405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610c2e5781811015610c21576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e6365000000604482015260640161069a565b610c2e84848484036109a9565b50505050565b73ffffffffffffffffffffffffffffffffffffffff8316610cd7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f6472657373000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff8216610d7a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f6573730000000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604090205481811015610e30576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e63650000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff808516600090815260208190526040808220858503905591851681529081208054849290610e7490849061154e565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610eda91815260200190565b60405180910390a3610c2e565b73ffffffffffffffffffffffffffffffffffffffff8216610f64576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015260640161069a565b8060026000828254610f76919061154e565b909155505073ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604081208054839290610fb090849061154e565b909155505060405181815273ffffffffffffffffffffffffffffffffffffffff8316906000907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9060200160405180910390a35050565b60608160000361104a57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611074578061105e816115dc565b915061106d9050600a83611643565b915061104e565b60008167ffffffffffffffff81111561108f5761108f611657565b6040519080825280601f01601f1916602001820160405280156110b9576020820181803683370190505b5090505b841561113c576110ce600183611686565b91506110db600a8661169d565b6110e690603061154e565b60f81b8183815181106110fb576110fb6116b1565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611135600a86611643565b94506110bd565b949350505050565b73ffffffffffffffffffffffffffffffffffffffff82166111e7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f7300000000000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff82166000908152602081905260409020548181101561129d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f6365000000000000000000000000000000000000000000000000000000000000606482015260840161069a565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604081208383039055600280548492906112d9908490611686565b909155505060405182815260009073ffffffffffffffffffffffffffffffffffffffff8516907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef90602001610b50565b60006020828403121561133b57600080fd5b81357fffffffff000000000000000000000000000000000000000000000000000000008116811461136b57600080fd5b9392505050565b60005b8381101561138d578181015183820152602001611375565b83811115610c2e5750506000910152565b60208152600082518060208401526113bd816040850160208701611372565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461141357600080fd5b919050565b6000806040838503121561142b57600080fd5b611434836113ef565b946020939093013593505050565b60008060006060848603121561145757600080fd5b611460846113ef565b925061146e602085016113ef565b9150604084013590509250925092565b60006020828403121561149057600080fd5b61136b826113ef565b600080604083850312156114ac57600080fd5b6114b5836113ef565b91506114c3602084016113ef565b90509250929050565b600181811c908216806114e057607f821691505b602082108103611519577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082198211156115615761156161151f565b500190565b60008451611578818460208901611372565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516115b4816001850160208a01611372565b600192019182015283516115cf816002840160208801611372565b0160020195945050505050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361160d5761160d61151f565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60008261165257611652611614565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000828210156116985761169861151f565b500390565b6000826116ac576116ac611614565b500690565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000aa164736f6c634300080f000a",
}

// OptimismMintableERC20FactoryABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismMintableERC20FactoryMetaData.ABI instead.
var OptimismMintableERC20FactoryABI = OptimismMintableERC20FactoryMetaData.ABI

// OptimismMintableERC20FactoryBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OptimismMintableERC20FactoryMetaData.Bin instead.
var OptimismMintableERC20FactoryBin = OptimismMintableERC20FactoryMetaData.Bin

// DeployOptimismMintableERC20Factory deploys a new Ethereum contract, binding an instance of OptimismMintableERC20Factory to it.
func DeployOptimismMintableERC20Factory(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *OptimismMintableERC20Factory, error) {
	parsed, err := OptimismMintableERC20FactoryMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OptimismMintableERC20FactoryBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &OptimismMintableERC20Factory{OptimismMintableERC20FactoryCaller: OptimismMintableERC20FactoryCaller{contract: contract}, OptimismMintableERC20FactoryTransactor: OptimismMintableERC20FactoryTransactor{contract: contract}, OptimismMintableERC20FactoryFilterer: OptimismMintableERC20FactoryFilterer{contract: contract}}, nil
}

// OptimismMintableERC20Factory is an auto generated Go binding around an Ethereum contract.
type OptimismMintableERC20Factory struct {
	OptimismMintableERC20FactoryCaller     // Read-only binding to the contract
	OptimismMintableERC20FactoryTransactor // Write-only binding to the contract
	OptimismMintableERC20FactoryFilterer   // Log filterer for contract events
}

// OptimismMintableERC20FactoryCaller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactoryTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactoryFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismMintableERC20FactoryFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismMintableERC20FactorySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismMintableERC20FactorySession struct {
	Contract     *OptimismMintableERC20Factory // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                 // Call options to use throughout this session
	TransactOpts bind.TransactOpts             // Transaction auth options to use throughout this session
}

// OptimismMintableERC20FactoryCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismMintableERC20FactoryCallerSession struct {
	Contract *OptimismMintableERC20FactoryCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                       // Call options to use throughout this session
}

// OptimismMintableERC20FactoryTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismMintableERC20FactoryTransactorSession struct {
	Contract     *OptimismMintableERC20FactoryTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                       // Transaction auth options to use throughout this session
}

// OptimismMintableERC20FactoryRaw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryRaw struct {
	Contract *OptimismMintableERC20Factory // Generic contract binding to access the raw methods on
}

// OptimismMintableERC20FactoryCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryCallerRaw struct {
	Contract *OptimismMintableERC20FactoryCaller // Generic read-only contract binding to access the raw methods on
}

// OptimismMintableERC20FactoryTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismMintableERC20FactoryTransactorRaw struct {
	Contract *OptimismMintableERC20FactoryTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismMintableERC20Factory creates a new instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20Factory(address common.Address, backend bind.ContractBackend) (*OptimismMintableERC20Factory, error) {
	contract, err := bindOptimismMintableERC20Factory(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20Factory{OptimismMintableERC20FactoryCaller: OptimismMintableERC20FactoryCaller{contract: contract}, OptimismMintableERC20FactoryTransactor: OptimismMintableERC20FactoryTransactor{contract: contract}, OptimismMintableERC20FactoryFilterer: OptimismMintableERC20FactoryFilterer{contract: contract}}, nil
}

// NewOptimismMintableERC20FactoryCaller creates a new read-only instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryCaller(address common.Address, caller bind.ContractCaller) (*OptimismMintableERC20FactoryCaller, error) {
	contract, err := bindOptimismMintableERC20Factory(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryCaller{contract: contract}, nil
}

// NewOptimismMintableERC20FactoryTransactor creates a new write-only instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryTransactor(address common.Address, transactor bind.ContractTransactor) (*OptimismMintableERC20FactoryTransactor, error) {
	contract, err := bindOptimismMintableERC20Factory(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryTransactor{contract: contract}, nil
}

// NewOptimismMintableERC20FactoryFilterer creates a new log filterer instance of OptimismMintableERC20Factory, bound to a specific deployed contract.
func NewOptimismMintableERC20FactoryFilterer(address common.Address, filterer bind.ContractFilterer) (*OptimismMintableERC20FactoryFilterer, error) {
	contract, err := bindOptimismMintableERC20Factory(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryFilterer{contract: contract}, nil
}

// bindOptimismMintableERC20Factory binds a generic wrapper to an already deployed contract.
func bindOptimismMintableERC20Factory(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OptimismMintableERC20FactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.OptimismMintableERC20FactoryTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismMintableERC20Factory.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.contract.Transact(opts, method, params...)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) BRIDGE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "BRIDGE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) BRIDGE() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.BRIDGE(&_OptimismMintableERC20Factory.CallOpts)
}

// BRIDGE is a free data retrieval call binding the contract method 0xee9a31a2.
//
// Solidity: function BRIDGE() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) BRIDGE() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.BRIDGE(&_OptimismMintableERC20Factory.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) Bridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "bridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Bridge() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.Bridge(&_OptimismMintableERC20Factory.CallOpts)
}

// Bridge is a free data retrieval call binding the contract method 0xe78cea92.
//
// Solidity: function bridge() view returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) Bridge() (common.Address, error) {
	return _OptimismMintableERC20Factory.Contract.Bridge(&_OptimismMintableERC20Factory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismMintableERC20Factory.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Version() (string, error) {
	return _OptimismMintableERC20Factory.Contract.Version(&_OptimismMintableERC20Factory.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryCallerSession) Version() (string, error) {
	return _OptimismMintableERC20Factory.Contract.Version(&_OptimismMintableERC20Factory.CallOpts)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateOptimismMintableERC20(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createOptimismMintableERC20", _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateOptimismMintableERC20(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20 is a paid mutator transaction binding the contract method 0xce5ac90f.
//
// Solidity: function createOptimismMintableERC20(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateOptimismMintableERC20(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateOptimismMintableERC20WithDecimals(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createOptimismMintableERC20WithDecimals", _remoteToken, _name, _symbol, _decimals)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateOptimismMintableERC20WithDecimals(_remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20WithDecimals(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol, _decimals)
}

// CreateOptimismMintableERC20WithDecimals is a paid mutator transaction binding the contract method 0x8cf0629c.
//
// Solidity: function createOptimismMintableERC20WithDecimals(address _remoteToken, string _name, string _symbol, uint8 _decimals) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateOptimismMintableERC20WithDecimals(_remoteToken common.Address, _name string, _symbol string, _decimals uint8) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateOptimismMintableERC20WithDecimals(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol, _decimals)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) CreateStandardL2Token(opts *bind.TransactOpts, _remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "createStandardL2Token", _remoteToken, _name, _symbol)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) CreateStandardL2Token(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateStandardL2Token(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// CreateStandardL2Token is a paid mutator transaction binding the contract method 0x896f93d1.
//
// Solidity: function createStandardL2Token(address _remoteToken, string _name, string _symbol) returns(address)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) CreateStandardL2Token(_remoteToken common.Address, _name string, _symbol string) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.CreateStandardL2Token(&_OptimismMintableERC20Factory.TransactOpts, _remoteToken, _name, _symbol)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactor) Initialize(opts *bind.TransactOpts, _bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.contract.Transact(opts, "initialize", _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactorySession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.Initialize(&_OptimismMintableERC20Factory.TransactOpts, _bridge)
}

// Initialize is a paid mutator transaction binding the contract method 0xc4d66de8.
//
// Solidity: function initialize(address _bridge) returns()
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryTransactorSession) Initialize(_bridge common.Address) (*types.Transaction, error) {
	return _OptimismMintableERC20Factory.Contract.Initialize(&_OptimismMintableERC20Factory.TransactOpts, _bridge)
}

// OptimismMintableERC20FactoryInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryInitializedIterator struct {
	Event *OptimismMintableERC20FactoryInitialized // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryInitialized)
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
		it.Event = new(OptimismMintableERC20FactoryInitialized)
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
func (it *OptimismMintableERC20FactoryInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryInitialized represents a Initialized event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterInitialized(opts *bind.FilterOpts) (*OptimismMintableERC20FactoryInitializedIterator, error) {

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryInitializedIterator{contract: _OptimismMintableERC20Factory.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryInitialized) (event.Subscription, error) {

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryInitialized)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseInitialized(log types.Log) (*OptimismMintableERC20FactoryInitialized, error) {
	event := new(OptimismMintableERC20FactoryInitialized)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator is returned from FilterOptimismMintableERC20Created and is used to iterate over the raw logs and unpacked data for OptimismMintableERC20Created events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator struct {
	Event *OptimismMintableERC20FactoryOptimismMintableERC20Created // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
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
		it.Event = new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
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
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryOptimismMintableERC20Created represents a OptimismMintableERC20Created event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryOptimismMintableERC20Created struct {
	LocalToken  common.Address
	RemoteToken common.Address
	Deployer    common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterOptimismMintableERC20Created is a free log retrieval operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterOptimismMintableERC20Created(opts *bind.FilterOpts, localToken []common.Address, remoteToken []common.Address) (*OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "OptimismMintableERC20Created", localTokenRule, remoteTokenRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryOptimismMintableERC20CreatedIterator{contract: _OptimismMintableERC20Factory.contract, event: "OptimismMintableERC20Created", logs: logs, sub: sub}, nil
}

// WatchOptimismMintableERC20Created is a free log subscription operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchOptimismMintableERC20Created(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryOptimismMintableERC20Created, localToken []common.Address, remoteToken []common.Address) (event.Subscription, error) {

	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}
	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "OptimismMintableERC20Created", localTokenRule, remoteTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "OptimismMintableERC20Created", log); err != nil {
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

// ParseOptimismMintableERC20Created is a log parse operation binding the contract event 0x52fe89dd5930f343d25650b62fd367bae47088bcddffd2a88350a6ecdd620cdb.
//
// Solidity: event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseOptimismMintableERC20Created(log types.Log) (*OptimismMintableERC20FactoryOptimismMintableERC20Created, error) {
	event := new(OptimismMintableERC20FactoryOptimismMintableERC20Created)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "OptimismMintableERC20Created", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismMintableERC20FactoryStandardL2TokenCreatedIterator is returned from FilterStandardL2TokenCreated and is used to iterate over the raw logs and unpacked data for StandardL2TokenCreated events raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryStandardL2TokenCreatedIterator struct {
	Event *OptimismMintableERC20FactoryStandardL2TokenCreated // Event containing the contract specifics and raw log

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
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismMintableERC20FactoryStandardL2TokenCreated)
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
		it.Event = new(OptimismMintableERC20FactoryStandardL2TokenCreated)
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
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismMintableERC20FactoryStandardL2TokenCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismMintableERC20FactoryStandardL2TokenCreated represents a StandardL2TokenCreated event raised by the OptimismMintableERC20Factory contract.
type OptimismMintableERC20FactoryStandardL2TokenCreated struct {
	RemoteToken common.Address
	LocalToken  common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterStandardL2TokenCreated is a free log retrieval operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) FilterStandardL2TokenCreated(opts *bind.FilterOpts, remoteToken []common.Address, localToken []common.Address) (*OptimismMintableERC20FactoryStandardL2TokenCreatedIterator, error) {

	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.FilterLogs(opts, "StandardL2TokenCreated", remoteTokenRule, localTokenRule)
	if err != nil {
		return nil, err
	}
	return &OptimismMintableERC20FactoryStandardL2TokenCreatedIterator{contract: _OptimismMintableERC20Factory.contract, event: "StandardL2TokenCreated", logs: logs, sub: sub}, nil
}

// WatchStandardL2TokenCreated is a free log subscription operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) WatchStandardL2TokenCreated(opts *bind.WatchOpts, sink chan<- *OptimismMintableERC20FactoryStandardL2TokenCreated, remoteToken []common.Address, localToken []common.Address) (event.Subscription, error) {

	var remoteTokenRule []interface{}
	for _, remoteTokenItem := range remoteToken {
		remoteTokenRule = append(remoteTokenRule, remoteTokenItem)
	}
	var localTokenRule []interface{}
	for _, localTokenItem := range localToken {
		localTokenRule = append(localTokenRule, localTokenItem)
	}

	logs, sub, err := _OptimismMintableERC20Factory.contract.WatchLogs(opts, "StandardL2TokenCreated", remoteTokenRule, localTokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismMintableERC20FactoryStandardL2TokenCreated)
				if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "StandardL2TokenCreated", log); err != nil {
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

// ParseStandardL2TokenCreated is a log parse operation binding the contract event 0xceeb8e7d520d7f3b65fc11a262b91066940193b05d4f93df07cfdced0eb551cf.
//
// Solidity: event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken)
func (_OptimismMintableERC20Factory *OptimismMintableERC20FactoryFilterer) ParseStandardL2TokenCreated(log types.Log) (*OptimismMintableERC20FactoryStandardL2TokenCreated, error) {
	event := new(OptimismMintableERC20FactoryStandardL2TokenCreated)
	if err := _OptimismMintableERC20Factory.contract.UnpackLog(event, "StandardL2TokenCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

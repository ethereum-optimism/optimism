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

// TypesOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type TypesOutputProposal struct {
	OutputRoot [32]byte
	Timestamp  *big.Int
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"OutputProposed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"OutputsDeleted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousProposer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newProposer\",\"type\":\"address\"}],\"name\":\"ProposerChanged\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"L2_BLOCK_TIME\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_BLOCK_NUMBER\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_TIMESTAMP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newProposer\",\"type\":\"address\"}],\"name\":\"changeProposer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"computeL2Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"deleteL2Outputs\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structTypes.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1BlockNumber\",\"type\":\"uint256\"}],\"name\":\"proposeL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proposer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101606040523480156200001257600080fd5b50604051620022d5380380620022d5833981016040819052620000359162000602565b6000608081905260a052600160c05242841115620000ce5760405162461bcd60e51b8152602060048201526044602482018190527f4c324f75747075744f7261636c653a207374617274696e67204c322074696d65908201527f7374616d70206d757374206265206c657373207468616e2063757272656e742060648201526374696d6560e01b608482015260a4015b60405180910390fd5b60e0879052610100859052610120849052610140839052620000f2868383620000ff565b505050505050506200066a565b600054610100900460ff1615808015620001205750600054600160ff909116105b806200015057506200013d30620002f260201b620013ed1760201c565b15801562000150575060005460ff166001145b620001b55760405162461bcd60e51b815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000c5565b6000805460ff191660011790558015620001d9576000805461ff0019166101001790555b816001600160a01b0316836001600160a01b031603620002515760405162461bcd60e51b81526020600482015260386024820152600080516020620022b583398151915260448201527f6265207468652073616d6520617320746865206f776e657200000000000000006064820152608401620000c5565b604080518082018252858152426020808301918252610100516000818152606790925293902091518255516001909101556066556200028f62000301565b6200029a8362000369565b620002a582620004d0565b8015620002ec576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50505050565b6001600160a01b03163b151590565b600054610100900460ff166200035d5760405162461bcd60e51b815260206004820152602b60248201526000805160206200229583398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000c5565b6200036762000522565b565b6200037362000589565b6001600160a01b038116620003f15760405162461bcd60e51b815260206004820152603760248201527f4c324f75747075744f7261636c653a206e65772070726f706f7365722063616e60448201527f6e6f7420626520746865207a65726f20616464726573730000000000000000006064820152608401620000c5565b6033546001600160a01b03166001600160a01b0316816001600160a01b031603620004745760405162461bcd60e51b81526020600482015260386024820152600080516020620022b583398151915260448201527f6265207468652073616d6520617320746865206f776e657200000000000000006064820152608401620000c5565b6065546040516001600160a01b038084169216907f3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd90600090a3606580546001600160a01b0319166001600160a01b0392909216919091179055565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff166200057e5760405162461bcd60e51b815260206004820152602b60248201526000805160206200229583398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000c5565b6200036733620004d0565b6033546001600160a01b03163314620003675760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401620000c5565b80516001600160a01b0381168114620005fd57600080fd5b919050565b600080600080600080600060e0888a0312156200061e57600080fd5b87519650602088015195506040880151945060608801519350608088015192506200064c60a08901620005e5565b91506200065c60c08901620005e5565b905092959891949750929550565b60805160a05160c05160e051610100516101205161014051611b836200071260003960008181610124015261127601526000818161016b01526112cf0152600081816101b50152818161087e01528181610eb001528181611064015281816111a0015261129a0152600081816101e901528181610a9b01528181611040015281816110a301526112fd01526000610419015260006103f0015260006103c70152611b836000f3fe60806040526004361061010d5760003560e01c806388b117b3116100a5578063a25ae55711610074578063d1de856c11610059578063d1de856c1461036b578063dcec33481461038b578063f2fde38b146103a057600080fd5b8063a25ae55714610303578063a8e4fb901461033e57600080fd5b806388b117b31461026457806389c44cbb146102845780638da5cb5b146102a45780639aaab648146102f057600080fd5b8063529933df116100e1578063529933df146101d757806354fd4d501461020b578063715018a61461022d57806372d5fe211461024457600080fd5b80622134cc1461011257806320e9fcd4146101595780634599c7881461018d5780634ab65d73146101a3575b600080fd5b34801561011e57600080fd5b506101467f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b34801561016557600080fd5b506101467f000000000000000000000000000000000000000000000000000000000000000081565b34801561019957600080fd5b5061014660665481565b3480156101af57600080fd5b506101467f000000000000000000000000000000000000000000000000000000000000000081565b3480156101e357600080fd5b506101467f000000000000000000000000000000000000000000000000000000000000000081565b34801561021757600080fd5b506102206103c0565b6040516101509190611855565b34801561023957600080fd5b50610242610463565b005b34801561025057600080fd5b5061024261025f3660046118cf565b610477565b34801561027057600080fd5b5061024261027f3660046118f1565b610688565b34801561029057600080fd5b5061024261029f36600461192d565b61093b565b3480156102b057600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610150565b6102426102fe366004611946565b610af1565b34801561030f57600080fd5b5061032361031e36600461192d565b610e9a565b60408051825181526020928301519281019290925201610150565b34801561034a57600080fd5b506065546102cb9073ffffffffffffffffffffffffffffffffffffffff1681565b34801561037757600080fd5b5061014661038636600461192d565b61119c565b34801561039757600080fd5b506101466112f9565b3480156103ac57600080fd5b506102426103bb3660046118cf565b61132e565b60606103eb7f0000000000000000000000000000000000000000000000000000000000000000611409565b6104147f0000000000000000000000000000000000000000000000000000000000000000611409565b61043d7f0000000000000000000000000000000000000000000000000000000000000000611409565b60405160200161044f93929190611978565b604051602081830303815290604052905090565b61046b61153e565b61047560006115bf565b565b61047f61153e565b73ffffffffffffffffffffffffffffffffffffffff8116610527576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324f75747075744f7261636c653a206e65772070726f706f7365722063616e60448201527f6e6f7420626520746865207a65726f206164647265737300000000000000000060648201526084015b60405180910390fd5b60335473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16036105fa576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603860248201527f4c324f75747075744f7261636c653a2070726f706f7365722063616e6e6f742060448201527f6265207468652073616d6520617320746865206f776e65720000000000000000606482015260840161051e565b60655460405173ffffffffffffffffffffffffffffffffffffffff8084169216907f3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd90600090a3606580547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b600054610100900460ff16158080156106a85750600054600160ff909116105b806106c25750303b1580156106c2575060005460ff166001145b61074e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161051e565b600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600117905580156107ac57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610867576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603860248201527f4c324f75747075744f7261636c653a2070726f706f7365722063616e6e6f742060448201527f6265207468652073616d6520617320746865206f776e65720000000000000000606482015260840161051e565b6040805180820182528581524260208083019182527f00000000000000000000000000000000000000000000000000000000000000006000818152606790925293902091518255516001909101556066556108c0611636565b6108c983610477565b6108d2826115bf565b801561093557600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b50505050565b61094361153e565b6000818152606760205260409020546109de576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f4c324f75747075744f7261636c653a2063616e6e6f742064656c65746520612060448201527f6e6f6e2d6578697374656e74206f757470757400000000000000000000000000606482015260840161051e565b606654811115610a96576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604360248201527f4c324f75747075744f7261636c653a2063616e6e6f742064656c657465206f7560448201527f747075747320616674657220746865206c617465737420626c6f636b206e756d60648201527f6265720000000000000000000000000000000000000000000000000000000000608482015260a40161051e565b610ac07f000000000000000000000000000000000000000000000000000000000000000082611a1d565b60665560405181907f21cbfd8ae20a7662b0a8aa9162fc9428f52c94b2e0ac268312628ea47f3f17eb90600090a250565b60655473ffffffffffffffffffffffffffffffffffffffff163314610b98576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324f75747075744f7261636c653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642062792070726f706f736572000000000000000000606482015260840161051e565b610ba06112f9565b8314610c54576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d626572206d757360448201527f7420626520657175616c20746f206e65787420657870656374656420626c6f6360648201527f6b206e756d626572000000000000000000000000000000000000000000000000608482015260a40161051e565b42610c5e8461119c565b10610ceb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324f75747075744f7261636c653a2063616e6e6f742070726f706f7365204c60448201527f32206f757470757420696e207468652066757475726500000000000000000000606482015260840161051e565b83610d78576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f4c324f75747075744f7261636c653a204c32206f75747075742070726f706f7360448201527f616c2063616e6e6f7420626520746865207a65726f2068617368000000000000606482015260840161051e565b8115610e345781814014610e34576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a20626c6f636b6861736820646f6573206e60448201527f6f74206d6174636820746865206861736820617420746865206578706563746560648201527f6420686569676874000000000000000000000000000000000000000000000000608482015260a40161051e565b6040805180820182528581524260208083018281526000888152606790925284822093518455516001909301929092556066869055915185929187917fc120f5e881491e6e212befa39e36b8f57d5eca31915f2e5d60a420f418caa6df9190a450505050565b60408051808201909152600080825260208201527f0000000000000000000000000000000000000000000000000000000000000000821015610f84576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604a60248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d6265722063616e60448201527f6e6f74206265206c657373207468616e20746865207374617274696e6720626c60648201527f6f636b206e756d62657200000000000000000000000000000000000000000000608482015260a40161051e565b60665482111561103c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604b60248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d6265722063616e60448201527f6e6f742062652067726561746572207468616e20746865206c6174657374206260648201527f6c6f636b206e756d626572000000000000000000000000000000000000000000608482015260a40161051e565b60007f00000000000000000000000000000000000000000000000000000000000000006110897f000000000000000000000000000000000000000000000000000000000000000085611a1d565b6110939190611a63565b9050600081156110d6576110c7827f0000000000000000000000000000000000000000000000000000000000000000611a1d565b6110d19085611a77565b6110d8565b835b600081815260676020908152604091829020825180840190935280548084526001909101549183019190915291925090611194576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603560248201527f4c324f75747075744f7261636c653a206e6f206f757470757420666f756e642060448201527f666f72207468617420626c6f636b206e756d6265720000000000000000000000606482015260840161051e565b949350505050565b60007f0000000000000000000000000000000000000000000000000000000000000000821015611274576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605360248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d626572206d757360448201527f742062652067726561746572207468616e206f7220657175616c20746f20737460648201527f617274696e6720626c6f636b206e756d62657200000000000000000000000000608482015260a40161051e565b7f00000000000000000000000000000000000000000000000000000000000000006112bf7f000000000000000000000000000000000000000000000000000000000000000084611a1d565b6112c99190611a8f565b6112f3907f0000000000000000000000000000000000000000000000000000000000000000611a77565b92915050565b60007f00000000000000000000000000000000000000000000000000000000000000006066546113299190611a77565b905090565b61133661153e565b60655473ffffffffffffffffffffffffffffffffffffffff908116908216036113e1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603860248201527f4c324f75747075744f7261636c653a206f776e65722063616e6e6f742062652060448201527f7468652073616d65206173207468652070726f706f7365720000000000000000606482015260840161051e565b6113ea816116d5565b50565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b60608160000361144c57505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611476578061146081611acc565b915061146f9050600a83611b04565b9150611450565b60008167ffffffffffffffff81111561149157611491611b18565b6040519080825280601f01601f1916602001820160405280156114bb576020820181803683370190505b5090505b8415611194576114d0600183611a1d565b91506114dd600a86611a63565b6114e8906030611a77565b60f81b8183815181106114fd576114fd611b47565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611537600a86611b04565b94506114bf565b60335473ffffffffffffffffffffffffffffffffffffffff163314610475576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161051e565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b600054610100900460ff166116cd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161051e565b610475611789565b6116dd61153e565b73ffffffffffffffffffffffffffffffffffffffff8116611780576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161051e565b6113ea816115bf565b600054610100900460ff16611820576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161051e565b610475336115bf565b60005b8381101561184457818101518382015260200161182c565b838111156109355750506000910152565b6020815260008251806020840152611874816040850160208701611829565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b803573ffffffffffffffffffffffffffffffffffffffff811681146118ca57600080fd5b919050565b6000602082840312156118e157600080fd5b6118ea826118a6565b9392505050565b60008060006060848603121561190657600080fd5b83359250611916602085016118a6565b9150611924604085016118a6565b90509250925092565b60006020828403121561193f57600080fd5b5035919050565b6000806000806080858703121561195c57600080fd5b5050823594602084013594506040840135936060013592509050565b6000845161198a818460208901611829565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516119c6816001850160208a01611829565b600192019182015283516119e1816002840160208801611829565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611a2f57611a2f6119ee565b500390565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082611a7257611a72611a34565b500690565b60008219821115611a8a57611a8a6119ee565b500190565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611ac757611ac76119ee565b500290565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203611afd57611afd6119ee565b5060010190565b600082611b1357611b13611a34565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e7472616374206973206e6f7420694c324f75747075744f7261636c653a2070726f706f7365722063616e6e6f7420",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _genesisL2Output [32]byte, _startingBlockNumber *big.Int, _startingTimestamp *big.Int, _l2BlockTime *big.Int, _proposer common.Address, _owner common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _genesisL2Output, _startingBlockNumber, _startingTimestamp, _l2BlockTime, _proposer, _owner)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// L2OutputOracle is an auto generated Go binding around an Ethereum contract.
type L2OutputOracle struct {
	L2OutputOracleCaller     // Read-only binding to the contract
	L2OutputOracleTransactor // Write-only binding to the contract
	L2OutputOracleFilterer   // Log filterer for contract events
}

// L2OutputOracleCaller is an auto generated read-only Go binding around an Ethereum contract.
type L2OutputOracleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type L2OutputOracleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// L2OutputOracleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type L2OutputOracleSession struct {
	Contract     *L2OutputOracle   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// L2OutputOracleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type L2OutputOracleCallerSession struct {
	Contract *L2OutputOracleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// L2OutputOracleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type L2OutputOracleTransactorSession struct {
	Contract     *L2OutputOracleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// L2OutputOracleRaw is an auto generated low-level Go binding around an Ethereum contract.
type L2OutputOracleRaw struct {
	Contract *L2OutputOracle // Generic contract binding to access the raw methods on
}

// L2OutputOracleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type L2OutputOracleCallerRaw struct {
	Contract *L2OutputOracleCaller // Generic read-only contract binding to access the raw methods on
}

// L2OutputOracleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type L2OutputOracleTransactorRaw struct {
	Contract *L2OutputOracleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewL2OutputOracle creates a new instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracle(address common.Address, backend bind.ContractBackend) (*L2OutputOracle, error) {
	contract, err := bindL2OutputOracle(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracle{L2OutputOracleCaller: L2OutputOracleCaller{contract: contract}, L2OutputOracleTransactor: L2OutputOracleTransactor{contract: contract}, L2OutputOracleFilterer: L2OutputOracleFilterer{contract: contract}}, nil
}

// NewL2OutputOracleCaller creates a new read-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleCaller(address common.Address, caller bind.ContractCaller) (*L2OutputOracleCaller, error) {
	contract, err := bindL2OutputOracle(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleCaller{contract: contract}, nil
}

// NewL2OutputOracleTransactor creates a new write-only instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleTransactor(address common.Address, transactor bind.ContractTransactor) (*L2OutputOracleTransactor, error) {
	contract, err := bindL2OutputOracle(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleTransactor{contract: contract}, nil
}

// NewL2OutputOracleFilterer creates a new log filterer instance of L2OutputOracle, bound to a specific deployed contract.
func NewL2OutputOracleFilterer(address common.Address, filterer bind.ContractFilterer) (*L2OutputOracleFilterer, error) {
	contract, err := bindL2OutputOracle(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleFilterer{contract: contract}, nil
}

// bindL2OutputOracle binds a generic wrapper to an already deployed contract.
func bindL2OutputOracle(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(L2OutputOracleABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.L2OutputOracleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.L2OutputOracleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_L2OutputOracle *L2OutputOracleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _L2OutputOracle.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_L2OutputOracle *L2OutputOracleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.contract.Transact(opts, method, params...)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) L2BLOCKTIME(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "L2_BLOCK_TIME")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// L2BLOCKTIME is a free data retrieval call binding the contract method 0x002134cc.
//
// Solidity: function L2_BLOCK_TIME() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2BLOCKTIME() (*big.Int, error) {
	return _L2OutputOracle.Contract.L2BLOCKTIME(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) STARTINGBLOCKNUMBER(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "STARTING_BLOCK_NUMBER")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) STARTINGBLOCKNUMBER() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKNUMBER(&_L2OutputOracle.CallOpts)
}

// STARTINGBLOCKNUMBER is a free data retrieval call binding the contract method 0x4ab65d73.
//
// Solidity: function STARTING_BLOCK_NUMBER() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) STARTINGBLOCKNUMBER() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGBLOCKNUMBER(&_L2OutputOracle.CallOpts)
}

// STARTINGTIMESTAMP is a free data retrieval call binding the contract method 0x20e9fcd4.
//
// Solidity: function STARTING_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) STARTINGTIMESTAMP(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "STARTING_TIMESTAMP")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// STARTINGTIMESTAMP is a free data retrieval call binding the contract method 0x20e9fcd4.
//
// Solidity: function STARTING_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) STARTINGTIMESTAMP() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGTIMESTAMP(&_L2OutputOracle.CallOpts)
}

// STARTINGTIMESTAMP is a free data retrieval call binding the contract method 0x20e9fcd4.
//
// Solidity: function STARTING_TIMESTAMP() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) STARTINGTIMESTAMP() (*big.Int, error) {
	return _L2OutputOracle.Contract.STARTINGTIMESTAMP(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) SUBMISSIONINTERVAL(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "SUBMISSION_INTERVAL")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// SUBMISSIONINTERVAL is a free data retrieval call binding the contract method 0x529933df.
//
// Solidity: function SUBMISSION_INTERVAL() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) SUBMISSIONINTERVAL() (*big.Int, error) {
	return _L2OutputOracle.Contract.SUBMISSIONINTERVAL(&_L2OutputOracle.CallOpts)
}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) ComputeL2Timestamp(opts *bind.CallOpts, _l2BlockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "computeL2Timestamp", _l2BlockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) ComputeL2Timestamp(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2Timestamp(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// ComputeL2Timestamp is a free data retrieval call binding the contract method 0xd1de856c.
//
// Solidity: function computeL2Timestamp(uint256 _l2BlockNumber) view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) ComputeL2Timestamp(_l2BlockNumber *big.Int) (*big.Int, error) {
	return _L2OutputOracle.Contract.ComputeL2Timestamp(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2BlockNumber)

	if err != nil {
		return *new(TypesOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(TypesOutputProposal)).(*TypesOutputProposal)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2BlockNumber *big.Int) (TypesOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) LatestBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "latestBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// LatestBlockNumber is a free data retrieval call binding the contract method 0x4599c788.
//
// Solidity: function latestBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) LatestBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.LatestBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) NextBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "nextBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// NextBlockNumber is a free data retrieval call binding the contract method 0xdcec3348.
//
// Solidity: function nextBlockNumber() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) NextBlockNumber() (*big.Int, error) {
	return _L2OutputOracle.Contract.NextBlockNumber(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Owner() (common.Address, error) {
	return _L2OutputOracle.Contract.Owner(&_L2OutputOracle.CallOpts)
}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Proposer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "proposer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Proposer() (common.Address, error) {
	return _L2OutputOracle.Contract.Proposer(&_L2OutputOracle.CallOpts)
}

// Proposer is a free data retrieval call binding the contract method 0xa8e4fb90.
//
// Solidity: function proposer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Proposer() (common.Address, error) {
	return _L2OutputOracle.Contract.Proposer(&_L2OutputOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleSession) Version() (string, error) {
	return _L2OutputOracle.Contract.Version(&_L2OutputOracle.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_L2OutputOracle *L2OutputOracleCallerSession) Version() (string, error) {
	return _L2OutputOracle.Contract.Version(&_L2OutputOracle.CallOpts)
}

// ChangeProposer is a paid mutator transaction binding the contract method 0x72d5fe21.
//
// Solidity: function changeProposer(address _newProposer) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) ChangeProposer(opts *bind.TransactOpts, _newProposer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "changeProposer", _newProposer)
}

// ChangeProposer is a paid mutator transaction binding the contract method 0x72d5fe21.
//
// Solidity: function changeProposer(address _newProposer) returns()
func (_L2OutputOracle *L2OutputOracleSession) ChangeProposer(_newProposer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ChangeProposer(&_L2OutputOracle.TransactOpts, _newProposer)
}

// ChangeProposer is a paid mutator transaction binding the contract method 0x72d5fe21.
//
// Solidity: function changeProposer(address _newProposer) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) ChangeProposer(_newProposer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ChangeProposer(&_L2OutputOracle.TransactOpts, _newProposer)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2BlockNumber) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Outputs(opts *bind.TransactOpts, _l2BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Outputs", _l2BlockNumber)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2BlockNumber) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Outputs(_l2BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Outputs(&_L2OutputOracle.TransactOpts, _l2BlockNumber)
}

// DeleteL2Outputs is a paid mutator transaction binding the contract method 0x89c44cbb.
//
// Solidity: function deleteL2Outputs(uint256 _l2BlockNumber) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Outputs(_l2BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Outputs(&_L2OutputOracle.TransactOpts, _l2BlockNumber)
}

// Initialize is a paid mutator transaction binding the contract method 0x88b117b3.
//
// Solidity: function initialize(bytes32 _genesisL2Output, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) Initialize(opts *bind.TransactOpts, _genesisL2Output [32]byte, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "initialize", _genesisL2Output, _proposer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0x88b117b3.
//
// Solidity: function initialize(bytes32 _genesisL2Output, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleSession) Initialize(_genesisL2Output [32]byte, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _proposer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0x88b117b3.
//
// Solidity: function initialize(bytes32 _genesisL2Output, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) Initialize(_genesisL2Output [32]byte, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _proposer, _owner)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) ProposeL2Output(opts *bind.TransactOpts, _outputRoot [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "proposeL2Output", _outputRoot, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) ProposeL2Output(_outputRoot [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ProposeL2Output(&_L2OutputOracle.TransactOpts, _outputRoot, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// ProposeL2Output is a paid mutator transaction binding the contract method 0x9aaab648.
//
// Solidity: function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) ProposeL2Output(_outputRoot [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ProposeL2Output(&_L2OutputOracle.TransactOpts, _outputRoot, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _L2OutputOracle.Contract.RenounceOwnership(&_L2OutputOracle.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L2OutputOracle *L2OutputOracleSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, _newOwner)
}

// L2OutputOracleInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the L2OutputOracle contract.
type L2OutputOracleInitializedIterator struct {
	Event *L2OutputOracleInitialized // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleInitialized)
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
		it.Event = new(L2OutputOracleInitialized)
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
func (it *L2OutputOracleInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleInitialized represents a Initialized event raised by the L2OutputOracle contract.
type L2OutputOracleInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterInitialized(opts *bind.FilterOpts) (*L2OutputOracleInitializedIterator, error) {

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleInitializedIterator{contract: _L2OutputOracle.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *L2OutputOracleInitialized) (event.Subscription, error) {

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleInitialized)
				if err := _L2OutputOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_L2OutputOracle *L2OutputOracleFilterer) ParseInitialized(log types.Log) (*L2OutputOracleInitialized, error) {
	event := new(L2OutputOracleInitialized)
	if err := _L2OutputOracle.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleOutputProposedIterator is returned from FilterOutputProposed and is used to iterate over the raw logs and unpacked data for OutputProposed events raised by the L2OutputOracle contract.
type L2OutputOracleOutputProposedIterator struct {
	Event *L2OutputOracleOutputProposed // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOutputProposedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOutputProposed)
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
		it.Event = new(L2OutputOracleOutputProposed)
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
func (it *L2OutputOracleOutputProposedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOutputProposedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOutputProposed represents a OutputProposed event raised by the L2OutputOracle contract.
type L2OutputOracleOutputProposed struct {
	OutputRoot    [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOutputProposed is a free log retrieval operation binding the contract event 0xc120f5e881491e6e212befa39e36b8f57d5eca31915f2e5d60a420f418caa6df.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOutputProposed(opts *bind.FilterOpts, outputRoot [][32]byte, l1Timestamp []*big.Int, l2BlockNumber []*big.Int) (*L2OutputOracleOutputProposedIterator, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l1TimestampRule []interface{}
	for _, l1TimestampItem := range l1Timestamp {
		l1TimestampRule = append(l1TimestampRule, l1TimestampItem)
	}
	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OutputProposed", outputRootRule, l1TimestampRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOutputProposedIterator{contract: _L2OutputOracle.contract, event: "OutputProposed", logs: logs, sub: sub}, nil
}

// WatchOutputProposed is a free log subscription operation binding the contract event 0xc120f5e881491e6e212befa39e36b8f57d5eca31915f2e5d60a420f418caa6df.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOutputProposed(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOutputProposed, outputRoot [][32]byte, l1Timestamp []*big.Int, l2BlockNumber []*big.Int) (event.Subscription, error) {

	var outputRootRule []interface{}
	for _, outputRootItem := range outputRoot {
		outputRootRule = append(outputRootRule, outputRootItem)
	}
	var l1TimestampRule []interface{}
	for _, l1TimestampItem := range l1Timestamp {
		l1TimestampRule = append(l1TimestampRule, l1TimestampItem)
	}
	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OutputProposed", outputRootRule, l1TimestampRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOutputProposed)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OutputProposed", log); err != nil {
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

// ParseOutputProposed is a log parse operation binding the contract event 0xc120f5e881491e6e212befa39e36b8f57d5eca31915f2e5d60a420f418caa6df.
//
// Solidity: event OutputProposed(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOutputProposed(log types.Log) (*L2OutputOracleOutputProposed, error) {
	event := new(L2OutputOracleOutputProposed)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OutputProposed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleOutputsDeletedIterator is returned from FilterOutputsDeleted and is used to iterate over the raw logs and unpacked data for OutputsDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleOutputsDeletedIterator struct {
	Event *L2OutputOracleOutputsDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOutputsDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOutputsDeleted)
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
		it.Event = new(L2OutputOracleOutputsDeleted)
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
func (it *L2OutputOracleOutputsDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOutputsDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOutputsDeleted represents a OutputsDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleOutputsDeleted struct {
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOutputsDeleted is a free log retrieval operation binding the contract event 0x21cbfd8ae20a7662b0a8aa9162fc9428f52c94b2e0ac268312628ea47f3f17eb.
//
// Solidity: event OutputsDeleted(uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOutputsDeleted(opts *bind.FilterOpts, l2BlockNumber []*big.Int) (*L2OutputOracleOutputsDeletedIterator, error) {

	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OutputsDeleted", l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOutputsDeletedIterator{contract: _L2OutputOracle.contract, event: "OutputsDeleted", logs: logs, sub: sub}, nil
}

// WatchOutputsDeleted is a free log subscription operation binding the contract event 0x21cbfd8ae20a7662b0a8aa9162fc9428f52c94b2e0ac268312628ea47f3f17eb.
//
// Solidity: event OutputsDeleted(uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOutputsDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOutputsDeleted, l2BlockNumber []*big.Int) (event.Subscription, error) {

	var l2BlockNumberRule []interface{}
	for _, l2BlockNumberItem := range l2BlockNumber {
		l2BlockNumberRule = append(l2BlockNumberRule, l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OutputsDeleted", l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOutputsDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OutputsDeleted", log); err != nil {
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

// ParseOutputsDeleted is a log parse operation binding the contract event 0x21cbfd8ae20a7662b0a8aa9162fc9428f52c94b2e0ac268312628ea47f3f17eb.
//
// Solidity: event OutputsDeleted(uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOutputsDeleted(log types.Log) (*L2OutputOracleOutputsDeleted, error) {
	event := new(L2OutputOracleOutputsDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OutputsDeleted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferredIterator struct {
	Event *L2OutputOracleOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOwnershipTransferred)
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
		it.Event = new(L2OutputOracleOwnershipTransferred)
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
func (it *L2OutputOracleOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOwnershipTransferred represents a OwnershipTransferred event raised by the L2OutputOracle contract.
type L2OutputOracleOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*L2OutputOracleOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOwnershipTransferredIterator{contract: _L2OutputOracle.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOwnershipTransferred)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOwnershipTransferred(log types.Log) (*L2OutputOracleOwnershipTransferred, error) {
	event := new(L2OutputOracleOwnershipTransferred)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleProposerChangedIterator is returned from FilterProposerChanged and is used to iterate over the raw logs and unpacked data for ProposerChanged events raised by the L2OutputOracle contract.
type L2OutputOracleProposerChangedIterator struct {
	Event *L2OutputOracleProposerChanged // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleProposerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleProposerChanged)
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
		it.Event = new(L2OutputOracleProposerChanged)
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
func (it *L2OutputOracleProposerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleProposerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleProposerChanged represents a ProposerChanged event raised by the L2OutputOracle contract.
type L2OutputOracleProposerChanged struct {
	PreviousProposer common.Address
	NewProposer      common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterProposerChanged is a free log retrieval operation binding the contract event 0x3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd.
//
// Solidity: event ProposerChanged(address indexed previousProposer, address indexed newProposer)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterProposerChanged(opts *bind.FilterOpts, previousProposer []common.Address, newProposer []common.Address) (*L2OutputOracleProposerChangedIterator, error) {

	var previousProposerRule []interface{}
	for _, previousProposerItem := range previousProposer {
		previousProposerRule = append(previousProposerRule, previousProposerItem)
	}
	var newProposerRule []interface{}
	for _, newProposerItem := range newProposer {
		newProposerRule = append(newProposerRule, newProposerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "ProposerChanged", previousProposerRule, newProposerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleProposerChangedIterator{contract: _L2OutputOracle.contract, event: "ProposerChanged", logs: logs, sub: sub}, nil
}

// WatchProposerChanged is a free log subscription operation binding the contract event 0x3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd.
//
// Solidity: event ProposerChanged(address indexed previousProposer, address indexed newProposer)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchProposerChanged(opts *bind.WatchOpts, sink chan<- *L2OutputOracleProposerChanged, previousProposer []common.Address, newProposer []common.Address) (event.Subscription, error) {

	var previousProposerRule []interface{}
	for _, previousProposerItem := range previousProposer {
		previousProposerRule = append(previousProposerRule, previousProposerItem)
	}
	var newProposerRule []interface{}
	for _, newProposerItem := range newProposer {
		newProposerRule = append(newProposerRule, newProposerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "ProposerChanged", previousProposerRule, newProposerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleProposerChanged)
				if err := _L2OutputOracle.contract.UnpackLog(event, "ProposerChanged", log); err != nil {
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

// ParseProposerChanged is a log parse operation binding the contract event 0x3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd.
//
// Solidity: event ProposerChanged(address indexed previousProposer, address indexed newProposer)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseProposerChanged(log types.Log) (*L2OutputOracleProposerChanged, error) {
	event := new(L2OutputOracleProposerChanged)
	if err := _L2OutputOracle.contract.UnpackLog(event, "ProposerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

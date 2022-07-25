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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"OutputDeleted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"OutputProposed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousProposer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newProposer\",\"type\":\"address\"}],\"name\":\"ProposerChanged\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"HISTORICAL_TOTAL_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_BLOCK_TIME\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_BLOCK_NUMBER\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_TIMESTAMP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newProposer\",\"type\":\"address\"}],\"name\":\"changeProposer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"computeL2Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structTypes.OutputProposal\",\"name\":\"_proposal\",\"type\":\"tuple\"}],\"name\":\"deleteL2Output\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structTypes.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_proposer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1BlockNumber\",\"type\":\"uint256\"}],\"name\":\"proposeL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proposer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101806040523480156200001257600080fd5b50604051620023d0380380620023d08339810160408190526200003591620005f4565b6000608081905260a052600160c052428310620000cd5760405162461bcd60e51b8152602060048201526044602482018190527f4c324f75747075744f7261636c653a20696e697469616c204c3220626c6f636b908201527f2074696d65206d757374206265206c657373207468616e2063757272656e742060648201526374696d6560e01b608482015260a4015b60405180910390fd5b60e0889052610100869052610120859052610140849052610160839052620000f88786848462000106565b505050505050505062000666565b6000620001146001620001cf565b905080156200012d576000805461ff0019166101001790555b604080518082018252868152426020808301918252600088815260679091529290922090518155905160019091015560668490556200016b620002e2565b62000176836200034a565b620001818262000514565b8015620001c8576000805461ff0019169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050505050565b60008054610100900460ff161562000264578160ff16600114801562000208575062000206306200056660201b620014181760201c565b155b6200025c5760405162461bcd60e51b815260206004820152602e60248201526000805160206200239083398151915260448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000c4565b506000919050565b60005460ff808416911610620002c35760405162461bcd60e51b815260206004820152602e60248201526000805160206200239083398151915260448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000c4565b506000805460ff191660ff92909216919091179055600190565b919050565b600054610100900460ff166200033e5760405162461bcd60e51b815260206004820152602b6024820152600080516020620023b083398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000c4565b6200034862000575565b565b6033546001600160a01b03163314620003a65760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401620000c4565b6001600160a01b038116620004245760405162461bcd60e51b815260206004820152603760248201527f4c324f75747075744f7261636c653a206e65772070726f706f7365722063616e60448201527f6e6f7420626520746865207a65726f20616464726573730000000000000000006064820152608401620000c4565b6033546001600160a01b03166001600160a01b0316816001600160a01b031603620004b85760405162461bcd60e51b815260206004820152603860248201527f4c324f75747075744f7261636c653a2070726f706f7365722063616e6e6f742060448201527f6265207468652073616d6520617320746865206f776e657200000000000000006064820152608401620000c4565b6065546040516001600160a01b038084169216907f3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd90600090a3606580546001600160a01b0319166001600160a01b0392909216919091179055565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6001600160a01b03163b151590565b600054610100900460ff16620005d15760405162461bcd60e51b815260206004820152602b6024820152600080516020620023b083398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000c4565b620003483362000514565b80516001600160a01b0381168114620002dd57600080fd5b600080600080600080600080610100898b0312156200061257600080fd5b885197506020890151965060408901519550606089015194506080890151935060a089015192506200064760c08a01620005dc565b91506200065760e08a01620005dc565b90509295985092959890939650565b60805160a05160c05160e05161010051610120516101405161016051611c7e620007126000396000818161013f01526111370152600081816101a801526111900152600081816101f201528181610e2901528181610f2501528181611061015261115b0152600061034b0152600081816102260152818161068901528181610f0101528181610f6401526112b70152600061070e015260006106e5015260006106bc0152611c7e6000f3fe6080604052600436106101285760003560e01c80638da5cb5b116100a5578063a8e4fb9011610074578063d20b1a5111610059578063d20b1a51146103ba578063dcec3348146103da578063f2fde38b146103ef57600080fd5b8063a8e4fb901461036d578063d1de856c1461039a57600080fd5b80638da5cb5b1461029f5780639aaab648146102eb578063a25ae557146102fe578063a4771aad1461033957600080fd5b80634ab65d73116100fc57806354fd4d50116100e157806354fd4d5014610248578063715018a61461026a57806372d5fe211461027f57600080fd5b80634ab65d73146101e0578063529933df1461021457600080fd5b80622134cc1461012d578063093b3d901461017457806320e9fcd4146101965780634599c788146101ca575b600080fd5b34801561013957600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b34801561018057600080fd5b5061019461018f3660046118d4565b61040f565b005b3480156101a257600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b3480156101d657600080fd5b5061016160665481565b3480156101ec57600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b34801561022057600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b34801561025457600080fd5b5061025d6106b5565b60405161016b919061197a565b34801561027657600080fd5b50610194610758565b34801561028b57600080fd5b5061019461029a3660046119ef565b6107e5565b3480156102ab57600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff165b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161016b565b6101946102f9366004611a11565b610a6a565b34801561030a57600080fd5b5061031e610319366004611a43565b610e13565b6040805182518152602092830151928101929092520161016b565b34801561034557600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b34801561037957600080fd5b506065546102c69073ffffffffffffffffffffffffffffffffffffffff1681565b3480156103a657600080fd5b506101616103b5366004611a43565b61105d565b3480156103c657600080fd5b506101946103d5366004611a5c565b6111ba565b3480156103e657600080fd5b506101616112b3565b3480156103fb57600080fd5b5061019461040a3660046119ef565b6112e8565b60335473ffffffffffffffffffffffffffffffffffffffff163314610495576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b6066546000908152606760209081526040918290208251808401909352805480845260019091015491830191909152825114610579576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604f60248201527f4c324f75747075744f7261636c653a206f757470757420726f6f7420746f206460448201527f656c65746520646f6573206e6f74206d6174636820746865206c61746573742060648201527f6f75747075742070726f706f73616c0000000000000000000000000000000000608482015260a40161048c565b8060200151826020015114610636576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604d60248201527f4c324f75747075744f7261636c653a2074696d657374616d7020746f2064656c60448201527f65746520646f6573206e6f74206d6174636820746865206c6174657374206f7560648201527f747075742070726f706f73616c00000000000000000000000000000000000000608482015260a40161048c565b606654602082015182516040517f11e942315215fbc11bf574b22ca610d001e704d870a2307833c188d31600b5c690600090a460668054600090815260676020526040812081815560010155546106ae907f000000000000000000000000000000000000000000000000000000000000000090611ad1565b6066555050565b60606106e07f0000000000000000000000000000000000000000000000000000000000000000611434565b6107097f0000000000000000000000000000000000000000000000000000000000000000611434565b6107327f0000000000000000000000000000000000000000000000000000000000000000611434565b60405160200161074493929190611ae8565b604051602081830303815290604052905090565b60335473ffffffffffffffffffffffffffffffffffffffff1633146107d9576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161048c565b6107e36000611569565b565b60335473ffffffffffffffffffffffffffffffffffffffff163314610866576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161048c565b73ffffffffffffffffffffffffffffffffffffffff8116610909576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324f75747075744f7261636c653a206e65772070726f706f7365722063616e60448201527f6e6f7420626520746865207a65726f2061646472657373000000000000000000606482015260840161048c565b60335473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16036109dc576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603860248201527f4c324f75747075744f7261636c653a2070726f706f7365722063616e6e6f742060448201527f6265207468652073616d6520617320746865206f776e65720000000000000000606482015260840161048c565b60655460405173ffffffffffffffffffffffffffffffffffffffff8084169216907f3d7728dc2838bb794606bd89f5a37930830b32060f69ee929bbfc59b669024dd90600090a3606580547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60655473ffffffffffffffffffffffffffffffffffffffff163314610b11576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324f75747075744f7261636c653a2066756e6374696f6e2063616e206f6e6c60448201527f792062652063616c6c65642062792070726f706f736572000000000000000000606482015260840161048c565b610b196112b3565b8314610bcd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d626572206d757360448201527f7420626520657175616c20746f206e65787420657870656374656420626c6f6360648201527f6b206e756d626572000000000000000000000000000000000000000000000000608482015260a40161048c565b42610bd78461105d565b10610c64576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324f75747075744f7261636c653a2063616e6e6f742070726f706f7365204c60448201527f32206f757470757420696e207468652066757475726500000000000000000000606482015260840161048c565b83610cf1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603a60248201527f4c324f75747075744f7261636c653a204c32206f75747075742070726f706f7360448201527f616c2063616e6e6f7420626520746865207a65726f2068617368000000000000606482015260840161048c565b8115610dad5781814014610dad576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604860248201527f4c324f75747075744f7261636c653a20626c6f636b6861736820646f6573206e60448201527f6f74206d6174636820746865206861736820617420746865206578706563746560648201527f6420686569676874000000000000000000000000000000000000000000000000608482015260a40161048c565b6040805180820182528581524260208083018281526000888152606790925284822093518455516001909301929092556066869055915185929187917fc120f5e881491e6e212befa39e36b8f57d5eca31915f2e5d60a420f418caa6df9190a450505050565b60408051808201909152600080825260208201527f0000000000000000000000000000000000000000000000000000000000000000821015610efd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604b60248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d6265722063616e60448201527f6e6f74206265206c657373207468616e20746865207374617274696e6720626c60648201527f6f636b206e756d6265722e000000000000000000000000000000000000000000608482015260a40161048c565b60007f0000000000000000000000000000000000000000000000000000000000000000610f4a7f000000000000000000000000000000000000000000000000000000000000000085611ad1565b610f549190611b8d565b905060008115610f9757610f88827f0000000000000000000000000000000000000000000000000000000000000000611ad1565b610f929085611ba1565b610f99565b835b600081815260676020908152604091829020825180840190935280548084526001909101549183019190915291925090611055576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324f75747075744f7261636c653a204e6f206f757470757420666f756e642060448201527f666f72207468617420626c6f636b206e756d6265722e00000000000000000000606482015260840161048c565b949350505050565b60007f0000000000000000000000000000000000000000000000000000000000000000821015611135576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605360248201527f4c324f75747075744f7261636c653a20626c6f636b206e756d626572206d757360448201527f742062652067726561746572207468616e206f7220657175616c20746f20737460648201527f617274696e6720626c6f636b206e756d62657200000000000000000000000000608482015260a40161048c565b7f00000000000000000000000000000000000000000000000000000000000000006111807f000000000000000000000000000000000000000000000000000000000000000084611ad1565b61118a9190611bb9565b6111b4907f0000000000000000000000000000000000000000000000000000000000000000611ba1565b92915050565b60006111c660016115e0565b905080156111fb57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b6040805180820182528681524260208083019182526000888152606790915292909220905181559051600190910155606684905561123761176b565b611240836107e5565b61124982611569565b80156112ac57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff169055604051600181527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b5050505050565b60007f00000000000000000000000000000000000000000000000000000000000000006066546112e39190611ba1565b905090565b60335473ffffffffffffffffffffffffffffffffffffffff163314611369576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572604482015260640161048c565b73ffffffffffffffffffffffffffffffffffffffff811661140c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f6464726573730000000000000000000000000000000000000000000000000000606482015260840161048c565b61141581611569565b50565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b60608160000361147757505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b81156114a1578061148b81611bf6565b915061149a9050600a83611c2e565b915061147b565b60008167ffffffffffffffff8111156114bc576114bc6118a5565b6040519080825280601f01601f1916602001820160405280156114e6576020820181803683370190505b5090505b8415611055576114fb600183611ad1565b9150611508600a86611b8d565b611513906030611ba1565b60f81b81838151811061152857611528611c42565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350611562600a86611c2e565b94506114ea565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60008054610100900460ff1615611697578160ff1660011480156116035750303b155b61168f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161048c565b506000919050565b60005460ff80841691161061172e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a6564000000000000000000000000000000000000606482015260840161048c565b50600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660ff92909216919091179055600190565b919050565b600054610100900460ff16611802576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161048c565b6107e3600054610100900460ff1661189c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e67000000000000000000000000000000000000000000606482015260840161048c565b6107e333611569565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6000604082840312156118e657600080fd5b6040516040810181811067ffffffffffffffff82111715611930577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052823581526020928301359281019290925250919050565b60005b8381101561196557818101518382015260200161194d565b83811115611974576000848401525b50505050565b602081526000825180602084015261199981604085016020870161194a565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461176657600080fd5b600060208284031215611a0157600080fd5b611a0a826119cb565b9392505050565b60008060008060808587031215611a2757600080fd5b5050823594602084013594506040840135936060013592509050565b600060208284031215611a5557600080fd5b5035919050565b60008060008060808587031215611a7257600080fd5b8435935060208501359250611a89604086016119cb565b9150611a97606086016119cb565b905092959194509250565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600082821015611ae357611ae3611aa2565b500390565b60008451611afa81846020890161194a565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551611b36816001850160208a0161194a565b60019201918201528351611b5181600284016020880161194a565b0160020195945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600082611b9c57611b9c611b5e565b500690565b60008219821115611bb457611bb4611aa2565b500190565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0483118215151615611bf157611bf1611aa2565b500290565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203611c2757611c27611aa2565b5060010190565b600082611c3d57611c3d611b5e565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a496e697469616c697a61626c653a20636f6e747261637420697320616c726561496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int, _startingBlockNumber *big.Int, _startingTimestamp *big.Int, _l2BlockTime *big.Int, _proposer common.Address, _owner common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _genesisL2Output, _historicalTotalBlocks, _startingBlockNumber, _startingTimestamp, _l2BlockTime, _proposer, _owner)
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

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCaller) HISTORICALTOTALBLOCKS(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "HISTORICAL_TOTAL_BLOCKS")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
}

// HISTORICALTOTALBLOCKS is a free data retrieval call binding the contract method 0xa4771aad.
//
// Solidity: function HISTORICAL_TOTAL_BLOCKS() view returns(uint256)
func (_L2OutputOracle *L2OutputOracleCallerSession) HISTORICALTOTALBLOCKS() (*big.Int, error) {
	return _L2OutputOracle.Contract.HISTORICALTOTALBLOCKS(&_L2OutputOracle.CallOpts)
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

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Output(opts *bind.TransactOpts, _proposal TypesOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Output", _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Output(_proposal TypesOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Output(_proposal TypesOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) Initialize(opts *bind.TransactOpts, _genesisL2Output [32]byte, _startingBlockNumber *big.Int, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "initialize", _genesisL2Output, _startingBlockNumber, _proposer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleSession) Initialize(_genesisL2Output [32]byte, _startingBlockNumber *big.Int, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _startingBlockNumber, _proposer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _proposer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) Initialize(_genesisL2Output [32]byte, _startingBlockNumber *big.Int, _proposer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _startingBlockNumber, _proposer, _owner)
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
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.TransferOwnership(&_L2OutputOracle.TransactOpts, newOwner)
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

// L2OutputOracleOutputDeletedIterator is returned from FilterOutputDeleted and is used to iterate over the raw logs and unpacked data for OutputDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleOutputDeletedIterator struct {
	Event *L2OutputOracleOutputDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleOutputDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleOutputDeleted)
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
		it.Event = new(L2OutputOracleOutputDeleted)
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
func (it *L2OutputOracleOutputDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleOutputDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleOutputDeleted represents a OutputDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleOutputDeleted struct {
	OutputRoot    [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOutputDeleted is a free log retrieval operation binding the contract event 0x11e942315215fbc11bf574b22ca610d001e704d870a2307833c188d31600b5c6.
//
// Solidity: event OutputDeleted(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterOutputDeleted(opts *bind.FilterOpts, outputRoot [][32]byte, l1Timestamp []*big.Int, l2BlockNumber []*big.Int) (*L2OutputOracleOutputDeletedIterator, error) {

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

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "OutputDeleted", outputRootRule, l1TimestampRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleOutputDeletedIterator{contract: _L2OutputOracle.contract, event: "OutputDeleted", logs: logs, sub: sub}, nil
}

// WatchOutputDeleted is a free log subscription operation binding the contract event 0x11e942315215fbc11bf574b22ca610d001e704d870a2307833c188d31600b5c6.
//
// Solidity: event OutputDeleted(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchOutputDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleOutputDeleted, outputRoot [][32]byte, l1Timestamp []*big.Int, l2BlockNumber []*big.Int) (event.Subscription, error) {

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

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "OutputDeleted", outputRootRule, l1TimestampRule, l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleOutputDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "OutputDeleted", log); err != nil {
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

// ParseOutputDeleted is a log parse operation binding the contract event 0x11e942315215fbc11bf574b22ca610d001e704d870a2307833c188d31600b5c6.
//
// Solidity: event OutputDeleted(bytes32 indexed outputRoot, uint256 indexed l1Timestamp, uint256 indexed l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseOutputDeleted(log types.Log) (*L2OutputOracleOutputDeleted, error) {
	event := new(L2OutputOracleOutputDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "OutputDeleted", log); err != nil {
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

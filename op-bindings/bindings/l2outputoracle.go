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

// L2OutputOracleOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type L2OutputOracleOutputProposal struct {
	OutputRoot [32]byte
	Timestamp  *big.Int
}

// L2OutputOracleMetaData contains all meta data concerning the L2OutputOracle contract.
var L2OutputOracleMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_submissionInterval\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_historicalTotalBlocks\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_startingTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sequencer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"L2OutputAppended\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l1Timestamp\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"L2OutputDeleted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousSequencer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newSequencer\",\"type\":\"address\"}],\"name\":\"SequencerChanged\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"HISTORICAL_TOTAL_BLOCKS\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_BLOCK_TIME\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_OUTPUT_ORACLE_VERSION\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_BLOCK_NUMBER\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"STARTING_TIMESTAMP\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SUBMISSION_INTERVAL\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_l2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_l1Blockhash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_l1BlockNumber\",\"type\":\"uint256\"}],\"name\":\"appendL2Output\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newSequencer\",\"type\":\"address\"}],\"name\":\"changeSequencer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"computeL2Timestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"_proposal\",\"type\":\"tuple\"}],\"name\":\"deleteL2Output\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_l2BlockNumber\",\"type\":\"uint256\"}],\"name\":\"getL2Output\",\"outputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"outputRoot\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"internalType\":\"structL2OutputOracle.OutputProposal\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_genesisL2Output\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_startingBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_sequencer\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"latestBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sequencer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6101206040523480156200001257600080fd5b5060405162001d7f38038062001d7f8339810160408190526200003591620005c8565b428310620000bc5760405162461bcd60e51b815260206004820152604360248201527f4f7574707574204f7261636c653a20496e697469616c204c3220626c6f636b2060448201527f74696d65206d757374206265206c657373207468616e2063757272656e742074606482015262696d6560e81b608482015260a4015b60405180910390fd5b608088905260a086905260c085905260e0849052610100839052620000e487868484620000f2565b50505050505050506200063a565b600160006200010182620001bf565b905080156200011a576000805461ff0019166101001790555b6040805180820182528781524260208083019182526000898152606790915292909220905181559051600190910155606685905562000158620002d2565b62000163846200033a565b6200016e83620004e8565b8015620001b7576000805461ff001916905560405160ff831681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050505050565b60008054610100900460ff161562000254578160ff166001148015620001f85750620001f6306200053a60201b620011491760201c565b155b6200024c5760405162461bcd60e51b815260206004820152602e602482015260008051602062001d3f83398151915260448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000b3565b506000919050565b60005460ff808416911610620002b35760405162461bcd60e51b815260206004820152602e602482015260008051602062001d3f83398151915260448201526d191e481a5b9a5d1a585b1a5e995960921b6064820152608401620000b3565b506000805460ff191660ff92909216919091179055600190565b919050565b600054610100900460ff166200032e5760405162461bcd60e51b815260206004820152602b602482015260008051602062001d5f83398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000b3565b6200033862000549565b565b6033546001600160a01b03163314620003965760405162461bcd60e51b815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65726044820152606401620000b3565b6001600160a01b038116620004065760405162461bcd60e51b815260206004820152602f60248201527f4f75747075744f7261636c653a206e65772073657175656e636572206973207460448201526e6865207a65726f206164647265737360881b6064820152608401620000b3565b6033546001600160a01b03828116911614156200048c5760405162461bcd60e51b815260206004820152603360248201527f4f75747075744f7261636c653a2073657175656e6365722063616e6e6f74206260448201527f652073616d6520617320746865206f776e6572000000000000000000000000006064820152608401620000b3565b6065546040516001600160a01b038084169216907f6ec88bae255aa7e73521c3beb17e9bc7940169e669440c5531733c0d2e91110d90600090a3606580546001600160a01b0319166001600160a01b0392909216919091179055565b603380546001600160a01b038381166001600160a01b0319831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b6001600160a01b03163b151590565b600054610100900460ff16620005a55760405162461bcd60e51b815260206004820152602b602482015260008051602062001d5f83398151915260448201526a6e697469616c697a696e6760a81b6064820152608401620000b3565b6200033833620004e8565b80516001600160a01b0381168114620002cd57600080fd5b600080600080600080600080610100898b031215620005e657600080fd5b885197506020890151965060408901519550606089015194506080890151935060a089015192506200061b60c08a01620005b0565b91506200062b60e08a01620005b0565b90509295985092959890939650565b60805160a05160c05160e05161010051611696620006a9600039600061013f0152600081816101a80152610ebd01526000818161022501528181610d8e0152610e88015260006103ba015260008181610259015281816106cb01528181610e640152610fe801526116966000f3fe6080604052600436106101285760003560e01c8063715018a6116100a5578063a4771aad11610074578063d20b1a5111610059578063d20b1a51146103fc578063dcec33481461041c578063f2fde38b1461043157600080fd5b8063a4771aad146103a8578063d1de856c146103dc57600080fd5b8063715018a6146102cd5780637cbb46e3146102e25780638da5cb5b14610309578063a25ae5571461033457600080fd5b80632af8ded8116100fc5780634ab65d73116100e15780634ab65d7314610213578063529933df146102475780635c1bba381461027b57600080fd5b80632af8ded8146101dd5780634599c788146101fd57600080fd5b80622134cc1461012d578063093b3d901461017457806320e9fcd41461019657806325188104146101ca575b600080fd5b34801561013957600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020015b60405180910390f35b34801561018057600080fd5b5061019461018f3660046114a1565b610451565b005b3480156101a257600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b6101946101d8366004611517565b6106f7565b3480156101e957600080fd5b506101946101f836600461156d565b610aa0565b34801561020957600080fd5b5061016160665481565b34801561021f57600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b34801561025357600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b34801561028757600080fd5b506065546102a89073ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff909116815260200161016b565b3480156102d957600080fd5b50610194610cfd565b3480156102ee57600080fd5b506102f7600181565b60405160ff909116815260200161016b565b34801561031557600080fd5b5060335473ffffffffffffffffffffffffffffffffffffffff166102a8565b34801561034057600080fd5b5061038d61034f36600461158f565b604080518082019091526000808252602082015250600090815260676020908152604091829020825180840190935280548352600101549082015290565b6040805182518152602092830151928101929092520161016b565b3480156103b457600080fd5b506101617f000000000000000000000000000000000000000000000000000000000000000081565b3480156103e857600080fd5b506101616103f736600461158f565b610d8a565b34801561040857600080fd5b506101946104173660046115a8565b610ee7565b34801561042857600080fd5b50610161610fe4565b34801561043d57600080fd5b5061019461044c36600461156d565b611019565b60335473ffffffffffffffffffffffffffffffffffffffff1633146104d7576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064015b60405180910390fd5b60665460009081526067602090815260409182902082518084019093528054808452600190910154918301919091528251146105bb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605260248201527f4f75747075744f7261636c653a20546865206f757470757420726f6f7420746f60448201527f2064656c65746520646f6573206e6f74206d6174636820746865206c6174657360648201527f74206f75747075742070726f706f73616c2e0000000000000000000000000000608482015260a4016104ce565b8060200151826020015114610678576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605060248201527f4f75747075744f7261636c653a205468652074696d657374616d7020746f206460448201527f656c65746520646f6573206e6f74206d6174636820746865206c61746573742060648201527f6f75747075742070726f706f73616c2e00000000000000000000000000000000608482015260a4016104ce565b606654602082015182516040517f7320566fd5256cf8923648a5d9f560f1e92f1435a1bb32ddd1fe107f224ad35990600090a460668054600090815260676020526040812081815560010155546106f0907f00000000000000000000000000000000000000000000000000000000000000009061161d565b6066555050565b60655473ffffffffffffffffffffffffffffffffffffffff16331461079e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602960248201527f4f75747075744f7261636c653a2063616c6c6572206973206e6f74207468652060448201527f73657175656e636572000000000000000000000000000000000000000000000060648201526084016104ce565b6107a6610fe4565b831461085a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604760248201527f4f75747075744f7261636c653a20426c6f636b206e756d626572206d7573742060448201527f626520657175616c20746f206e65787420657870656374656420626c6f636b2060648201527f6e756d6265722e00000000000000000000000000000000000000000000000000608482015260a4016104ce565b4261086484610d8a565b106108f1576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4f75747075744f7261636c653a2043616e6e6f7420617070656e64204c32206f60448201527f757470757420696e206675747572652e0000000000000000000000000000000060648201526084016104ce565b8361097e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602c60248201527f4f75747075744f7261636c653a2043616e6e6f74207375626d697420656d707460448201527f79204c32206f75747075742e000000000000000000000000000000000000000060648201526084016104ce565b8115610a3a5781814014610a3a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604760248201527f4f75747075744f7261636c653a20426c6f636b6861736820646f6573206e6f7460448201527f206d61746368207468652068617368206174207468652065787065637465642060648201527f6865696768742e00000000000000000000000000000000000000000000000000608482015260a4016104ce565b6040805180820182528581524260208083018281526000888152606790925284822093518455516001909301929092556066869055915185929187917fd6703ded1701060d9ae1793db76d594790a4e775781225f79b5aa8a77987c0809190a450505050565b60335473ffffffffffffffffffffffffffffffffffffffff163314610b21576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016104ce565b73ffffffffffffffffffffffffffffffffffffffff8116610bc4576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f4f75747075744f7261636c653a206e65772073657175656e636572206973207460448201527f6865207a65726f2061646472657373000000000000000000000000000000000060648201526084016104ce565b60335473ffffffffffffffffffffffffffffffffffffffff82811691161415610c6f576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603360248201527f4f75747075744f7261636c653a2073657175656e6365722063616e6e6f74206260448201527f652073616d6520617320746865206f776e65720000000000000000000000000060648201526084016104ce565b60655460405173ffffffffffffffffffffffffffffffffffffffff8084169216907f6ec88bae255aa7e73521c3beb17e9bc7940169e669440c5531733c0d2e91110d90600090a3606580547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff92909216919091179055565b60335473ffffffffffffffffffffffffffffffffffffffff163314610d7e576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016104ce565b610d886000611165565b565b60007f0000000000000000000000000000000000000000000000000000000000000000821015610e62576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152605660248201527f4f75747075744f7261636c653a20426c6f636b206e756d626572206d7573742060448201527f62652067726561746572207468616e206f7220657175616c20746f207468652060648201527f7374617274696e6720626c6f636b206e756d6265722e00000000000000000000608482015260a4016104ce565b7f0000000000000000000000000000000000000000000000000000000000000000610ead7f00000000000000000000000000000000000000000000000000000000000000008461161d565b610eb79190611634565b610ee1907f0000000000000000000000000000000000000000000000000000000000000000611671565b92915050565b60016000610ef4826111dc565b90508015610f2957600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff166101001790555b60408051808201825287815242602080830191825260008981526067909152929092209051815590516001909101556066859055610f65611367565b610f6e84610aa0565b610f7783611165565b8015610fdc57600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00ff16905560405160ff831681527f7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb38474024989060200160405180910390a15b505050505050565b60007f00000000000000000000000000000000000000000000000000000000000000006066546110149190611671565b905090565b60335473ffffffffffffffffffffffffffffffffffffffff16331461109a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016104ce565b73ffffffffffffffffffffffffffffffffffffffff811661113d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016104ce565b61114681611165565b50565b73ffffffffffffffffffffffffffffffffffffffff163b151590565b6033805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681179093556040519116919082907f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e090600090a35050565b60008054610100900460ff1615611293578160ff1660011480156111ff5750303b155b61128b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016104ce565b506000919050565b60005460ff80841691161061132a576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f496e697469616c697a61626c653a20636f6e747261637420697320616c72656160448201527f647920696e697469616c697a656400000000000000000000000000000000000060648201526084016104ce565b50600080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660ff92909216919091179055600190565b919050565b600054610100900460ff166113fe576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016104ce565b610d88600054610100900460ff16611498576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f496e697469616c697a61626c653a20636f6e7472616374206973206e6f74206960448201527f6e697469616c697a696e6700000000000000000000000000000000000000000060648201526084016104ce565b610d8833611165565b6000604082840312156114b357600080fd5b6040516040810181811067ffffffffffffffff821117156114fd577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052823581526020928301359281019290925250919050565b6000806000806080858703121561152d57600080fd5b5050823594602084013594506040840135936060013592509050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461136257600080fd5b60006020828403121561157f57600080fd5b61158882611549565b9392505050565b6000602082840312156115a157600080fd5b5035919050565b600080600080608085870312156115be57600080fd5b84359350602085013592506115d560408601611549565b91506115e360608601611549565b905092959194509250565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008282101561162f5761162f6115ee565b500390565b6000817fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff048311821515161561166c5761166c6115ee565b500290565b60008219821115611684576116846115ee565b50019056fea164736f6c634300080a000a496e697469616c697a61626c653a20636f6e747261637420697320616c726561496e697469616c697a61626c653a20636f6e7472616374206973206e6f742069",
}

// L2OutputOracleABI is the input ABI used to generate the binding from.
// Deprecated: Use L2OutputOracleMetaData.ABI instead.
var L2OutputOracleABI = L2OutputOracleMetaData.ABI

// L2OutputOracleBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use L2OutputOracleMetaData.Bin instead.
var L2OutputOracleBin = L2OutputOracleMetaData.Bin

// DeployL2OutputOracle deploys a new Ethereum contract, binding an instance of L2OutputOracle to it.
func DeployL2OutputOracle(auth *bind.TransactOpts, backend bind.ContractBackend, _submissionInterval *big.Int, _genesisL2Output [32]byte, _historicalTotalBlocks *big.Int, _startingBlockNumber *big.Int, _startingTimestamp *big.Int, _l2BlockTime *big.Int, _sequencer common.Address, _owner common.Address) (common.Address, *types.Transaction, *L2OutputOracle, error) {
	parsed, err := L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(L2OutputOracleBin), backend, _submissionInterval, _genesisL2Output, _historicalTotalBlocks, _startingBlockNumber, _startingTimestamp, _l2BlockTime, _sequencer, _owner)
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

// L2OUTPUTORACLEVERSION is a free data retrieval call binding the contract method 0x7cbb46e3.
//
// Solidity: function L2_OUTPUT_ORACLE_VERSION() view returns(uint8)
func (_L2OutputOracle *L2OutputOracleCaller) L2OUTPUTORACLEVERSION(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "L2_OUTPUT_ORACLE_VERSION")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// L2OUTPUTORACLEVERSION is a free data retrieval call binding the contract method 0x7cbb46e3.
//
// Solidity: function L2_OUTPUT_ORACLE_VERSION() view returns(uint8)
func (_L2OutputOracle *L2OutputOracleSession) L2OUTPUTORACLEVERSION() (uint8, error) {
	return _L2OutputOracle.Contract.L2OUTPUTORACLEVERSION(&_L2OutputOracle.CallOpts)
}

// L2OUTPUTORACLEVERSION is a free data retrieval call binding the contract method 0x7cbb46e3.
//
// Solidity: function L2_OUTPUT_ORACLE_VERSION() view returns(uint8)
func (_L2OutputOracle *L2OutputOracleCallerSession) L2OUTPUTORACLEVERSION() (uint8, error) {
	return _L2OutputOracle.Contract.L2OUTPUTORACLEVERSION(&_L2OutputOracle.CallOpts)
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
func (_L2OutputOracle *L2OutputOracleCaller) GetL2Output(opts *bind.CallOpts, _l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "getL2Output", _l2BlockNumber)

	if err != nil {
		return *new(L2OutputOracleOutputProposal), err
	}

	out0 := *abi.ConvertType(out[0], new(L2OutputOracleOutputProposal)).(*L2OutputOracleOutputProposal)

	return out0, err

}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleSession) GetL2Output(_l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
	return _L2OutputOracle.Contract.GetL2Output(&_L2OutputOracle.CallOpts, _l2BlockNumber)
}

// GetL2Output is a free data retrieval call binding the contract method 0xa25ae557.
//
// Solidity: function getL2Output(uint256 _l2BlockNumber) view returns((bytes32,uint256))
func (_L2OutputOracle *L2OutputOracleCallerSession) GetL2Output(_l2BlockNumber *big.Int) (L2OutputOracleOutputProposal, error) {
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

// Sequencer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function sequencer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCaller) Sequencer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _L2OutputOracle.contract.Call(opts, &out, "sequencer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Sequencer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function sequencer() view returns(address)
func (_L2OutputOracle *L2OutputOracleSession) Sequencer() (common.Address, error) {
	return _L2OutputOracle.Contract.Sequencer(&_L2OutputOracle.CallOpts)
}

// Sequencer is a free data retrieval call binding the contract method 0x5c1bba38.
//
// Solidity: function sequencer() view returns(address)
func (_L2OutputOracle *L2OutputOracleCallerSession) Sequencer() (common.Address, error) {
	return _L2OutputOracle.Contract.Sequencer(&_L2OutputOracle.CallOpts)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactor) AppendL2Output(opts *bind.TransactOpts, _l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "appendL2Output", _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleSession) AppendL2Output(_l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// AppendL2Output is a paid mutator transaction binding the contract method 0x25188104.
//
// Solidity: function appendL2Output(bytes32 _l2Output, uint256 _l2BlockNumber, bytes32 _l1Blockhash, uint256 _l1BlockNumber) payable returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) AppendL2Output(_l2Output [32]byte, _l2BlockNumber *big.Int, _l1Blockhash [32]byte, _l1BlockNumber *big.Int) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.AppendL2Output(&_L2OutputOracle.TransactOpts, _l2Output, _l2BlockNumber, _l1Blockhash, _l1BlockNumber)
}

// ChangeSequencer is a paid mutator transaction binding the contract method 0x2af8ded8.
//
// Solidity: function changeSequencer(address _newSequencer) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) ChangeSequencer(opts *bind.TransactOpts, _newSequencer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "changeSequencer", _newSequencer)
}

// ChangeSequencer is a paid mutator transaction binding the contract method 0x2af8ded8.
//
// Solidity: function changeSequencer(address _newSequencer) returns()
func (_L2OutputOracle *L2OutputOracleSession) ChangeSequencer(_newSequencer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ChangeSequencer(&_L2OutputOracle.TransactOpts, _newSequencer)
}

// ChangeSequencer is a paid mutator transaction binding the contract method 0x2af8ded8.
//
// Solidity: function changeSequencer(address _newSequencer) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) ChangeSequencer(_newSequencer common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.ChangeSequencer(&_L2OutputOracle.TransactOpts, _newSequencer)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) DeleteL2Output(opts *bind.TransactOpts, _proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "deleteL2Output", _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// DeleteL2Output is a paid mutator transaction binding the contract method 0x093b3d90.
//
// Solidity: function deleteL2Output((bytes32,uint256) _proposal) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) DeleteL2Output(_proposal L2OutputOracleOutputProposal) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.DeleteL2Output(&_L2OutputOracle.TransactOpts, _proposal)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _sequencer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactor) Initialize(opts *bind.TransactOpts, _genesisL2Output [32]byte, _startingBlockNumber *big.Int, _sequencer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.contract.Transact(opts, "initialize", _genesisL2Output, _startingBlockNumber, _sequencer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _sequencer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleSession) Initialize(_genesisL2Output [32]byte, _startingBlockNumber *big.Int, _sequencer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _startingBlockNumber, _sequencer, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xd20b1a51.
//
// Solidity: function initialize(bytes32 _genesisL2Output, uint256 _startingBlockNumber, address _sequencer, address _owner) returns()
func (_L2OutputOracle *L2OutputOracleTransactorSession) Initialize(_genesisL2Output [32]byte, _startingBlockNumber *big.Int, _sequencer common.Address, _owner common.Address) (*types.Transaction, error) {
	return _L2OutputOracle.Contract.Initialize(&_L2OutputOracle.TransactOpts, _genesisL2Output, _startingBlockNumber, _sequencer, _owner)
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

// L2OutputOracleL2OutputAppendedIterator is returned from FilterL2OutputAppended and is used to iterate over the raw logs and unpacked data for L2OutputAppended events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppendedIterator struct {
	Event *L2OutputOracleL2OutputAppended // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputAppendedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputAppended)
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
		it.Event = new(L2OutputOracleL2OutputAppended)
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
func (it *L2OutputOracleL2OutputAppendedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputAppendedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputAppended represents a L2OutputAppended event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputAppended struct {
	L2Output      [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterL2OutputAppended is a free log retrieval operation binding the contract event 0xd6703ded1701060d9ae1793db76d594790a4e775781225f79b5aa8a77987c080.
//
// Solidity: event L2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputAppended(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (*L2OutputOracleL2OutputAppendedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "L2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputAppendedIterator{contract: _L2OutputOracle.contract, event: "L2OutputAppended", logs: logs, sub: sub}, nil
}

// WatchL2OutputAppended is a free log subscription operation binding the contract event 0xd6703ded1701060d9ae1793db76d594790a4e775781225f79b5aa8a77987c080.
//
// Solidity: event L2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputAppended(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputAppended, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "L2OutputAppended", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputAppended)
				if err := _L2OutputOracle.contract.UnpackLog(event, "L2OutputAppended", log); err != nil {
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

// ParseL2OutputAppended is a log parse operation binding the contract event 0xd6703ded1701060d9ae1793db76d594790a4e775781225f79b5aa8a77987c080.
//
// Solidity: event L2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputAppended(log types.Log) (*L2OutputOracleL2OutputAppended, error) {
	event := new(L2OutputOracleL2OutputAppended)
	if err := _L2OutputOracle.contract.UnpackLog(event, "L2OutputAppended", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// L2OutputOracleL2OutputDeletedIterator is returned from FilterL2OutputDeleted and is used to iterate over the raw logs and unpacked data for L2OutputDeleted events raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeletedIterator struct {
	Event *L2OutputOracleL2OutputDeleted // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleL2OutputDeletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleL2OutputDeleted)
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
		it.Event = new(L2OutputOracleL2OutputDeleted)
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
func (it *L2OutputOracleL2OutputDeletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleL2OutputDeletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleL2OutputDeleted represents a L2OutputDeleted event raised by the L2OutputOracle contract.
type L2OutputOracleL2OutputDeleted struct {
	L2Output      [32]byte
	L1Timestamp   *big.Int
	L2BlockNumber *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterL2OutputDeleted is a free log retrieval operation binding the contract event 0x7320566fd5256cf8923648a5d9f560f1e92f1435a1bb32ddd1fe107f224ad359.
//
// Solidity: event L2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterL2OutputDeleted(opts *bind.FilterOpts, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (*L2OutputOracleL2OutputDeletedIterator, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "L2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleL2OutputDeletedIterator{contract: _L2OutputOracle.contract, event: "L2OutputDeleted", logs: logs, sub: sub}, nil
}

// WatchL2OutputDeleted is a free log subscription operation binding the contract event 0x7320566fd5256cf8923648a5d9f560f1e92f1435a1bb32ddd1fe107f224ad359.
//
// Solidity: event L2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchL2OutputDeleted(opts *bind.WatchOpts, sink chan<- *L2OutputOracleL2OutputDeleted, _l2Output [][32]byte, _l1Timestamp []*big.Int, _l2BlockNumber []*big.Int) (event.Subscription, error) {

	var _l2OutputRule []interface{}
	for _, _l2OutputItem := range _l2Output {
		_l2OutputRule = append(_l2OutputRule, _l2OutputItem)
	}
	var _l1TimestampRule []interface{}
	for _, _l1TimestampItem := range _l1Timestamp {
		_l1TimestampRule = append(_l1TimestampRule, _l1TimestampItem)
	}
	var _l2BlockNumberRule []interface{}
	for _, _l2BlockNumberItem := range _l2BlockNumber {
		_l2BlockNumberRule = append(_l2BlockNumberRule, _l2BlockNumberItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "L2OutputDeleted", _l2OutputRule, _l1TimestampRule, _l2BlockNumberRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleL2OutputDeleted)
				if err := _L2OutputOracle.contract.UnpackLog(event, "L2OutputDeleted", log); err != nil {
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

// ParseL2OutputDeleted is a log parse operation binding the contract event 0x7320566fd5256cf8923648a5d9f560f1e92f1435a1bb32ddd1fe107f224ad359.
//
// Solidity: event L2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l1Timestamp, uint256 indexed _l2BlockNumber)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseL2OutputDeleted(log types.Log) (*L2OutputOracleL2OutputDeleted, error) {
	event := new(L2OutputOracleL2OutputDeleted)
	if err := _L2OutputOracle.contract.UnpackLog(event, "L2OutputDeleted", log); err != nil {
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

// L2OutputOracleSequencerChangedIterator is returned from FilterSequencerChanged and is used to iterate over the raw logs and unpacked data for SequencerChanged events raised by the L2OutputOracle contract.
type L2OutputOracleSequencerChangedIterator struct {
	Event *L2OutputOracleSequencerChanged // Event containing the contract specifics and raw log

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
func (it *L2OutputOracleSequencerChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(L2OutputOracleSequencerChanged)
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
		it.Event = new(L2OutputOracleSequencerChanged)
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
func (it *L2OutputOracleSequencerChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *L2OutputOracleSequencerChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// L2OutputOracleSequencerChanged represents a SequencerChanged event raised by the L2OutputOracle contract.
type L2OutputOracleSequencerChanged struct {
	PreviousSequencer common.Address
	NewSequencer      common.Address
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterSequencerChanged is a free log retrieval operation binding the contract event 0x6ec88bae255aa7e73521c3beb17e9bc7940169e669440c5531733c0d2e91110d.
//
// Solidity: event SequencerChanged(address indexed previousSequencer, address indexed newSequencer)
func (_L2OutputOracle *L2OutputOracleFilterer) FilterSequencerChanged(opts *bind.FilterOpts, previousSequencer []common.Address, newSequencer []common.Address) (*L2OutputOracleSequencerChangedIterator, error) {

	var previousSequencerRule []interface{}
	for _, previousSequencerItem := range previousSequencer {
		previousSequencerRule = append(previousSequencerRule, previousSequencerItem)
	}
	var newSequencerRule []interface{}
	for _, newSequencerItem := range newSequencer {
		newSequencerRule = append(newSequencerRule, newSequencerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.FilterLogs(opts, "SequencerChanged", previousSequencerRule, newSequencerRule)
	if err != nil {
		return nil, err
	}
	return &L2OutputOracleSequencerChangedIterator{contract: _L2OutputOracle.contract, event: "SequencerChanged", logs: logs, sub: sub}, nil
}

// WatchSequencerChanged is a free log subscription operation binding the contract event 0x6ec88bae255aa7e73521c3beb17e9bc7940169e669440c5531733c0d2e91110d.
//
// Solidity: event SequencerChanged(address indexed previousSequencer, address indexed newSequencer)
func (_L2OutputOracle *L2OutputOracleFilterer) WatchSequencerChanged(opts *bind.WatchOpts, sink chan<- *L2OutputOracleSequencerChanged, previousSequencer []common.Address, newSequencer []common.Address) (event.Subscription, error) {

	var previousSequencerRule []interface{}
	for _, previousSequencerItem := range previousSequencer {
		previousSequencerRule = append(previousSequencerRule, previousSequencerItem)
	}
	var newSequencerRule []interface{}
	for _, newSequencerItem := range newSequencer {
		newSequencerRule = append(newSequencerRule, newSequencerItem)
	}

	logs, sub, err := _L2OutputOracle.contract.WatchLogs(opts, "SequencerChanged", previousSequencerRule, newSequencerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(L2OutputOracleSequencerChanged)
				if err := _L2OutputOracle.contract.UnpackLog(event, "SequencerChanged", log); err != nil {
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

// ParseSequencerChanged is a log parse operation binding the contract event 0x6ec88bae255aa7e73521c3beb17e9bc7940169e669440c5531733c0d2e91110d.
//
// Solidity: event SequencerChanged(address indexed previousSequencer, address indexed newSequencer)
func (_L2OutputOracle *L2OutputOracleFilterer) ParseSequencerChanged(log types.Log) (*L2OutputOracleSequencerChanged, error) {
	event := new(L2OutputOracleSequencerChanged)
	if err := _L2OutputOracle.contract.UnpackLog(event, "SequencerChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

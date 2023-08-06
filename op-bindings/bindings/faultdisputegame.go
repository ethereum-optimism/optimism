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

// IFaultDisputeGameOutputProposal is an auto generated low-level Go binding around an user-defined struct.
type IFaultDisputeGameOutputProposal struct {
	Index         *big.Int
	L2BlockNumber *big.Int
	OutputRoot    [32]byte
}

// FaultDisputeGameMetaData contains all meta data concerning the FaultDisputeGame contract.
var FaultDisputeGameMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"GameType\",\"name\":\"_gameType\",\"type\":\"uint8\"},{\"internalType\":\"Claim\",\"name\":\"_absolutePrestate\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_maxGameDepth\",\"type\":\"uint256\"},{\"internalType\":\"Duration\",\"name\":\"_gameDuration\",\"type\":\"uint64\"},{\"internalType\":\"contractIBigStepper\",\"name\":\"_vm\",\"type\":\"address\"},{\"internalType\":\"contractL2OutputOracle\",\"name\":\"_l2oo\",\"type\":\"address\"},{\"internalType\":\"contractBlockOracle\",\"name\":\"_blockOracle\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"CannotDefendRootClaim\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClaimAlreadyExists\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClockNotExpired\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ClockTimeExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GameDepthExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GameNotInProgress\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidParent\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidPrestate\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"L1HeadTooOld\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValidStep\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"parentIndex\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"Claim\",\"name\":\"claim\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"claimant\",\"type\":\"address\"}],\"name\":\"Move\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"enumGameStatus\",\"name\":\"status\",\"type\":\"uint8\"}],\"name\":\"Resolved\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"ABSOLUTE_PRESTATE\",\"outputs\":[{\"internalType\":\"Claim\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"BLOCK_ORACLE\",\"outputs\":[{\"internalType\":\"contractBlockOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GAME_DURATION\",\"outputs\":[{\"internalType\":\"Duration\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_OUTPUT_ORACLE\",\"outputs\":[{\"internalType\":\"contractL2OutputOracle\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MAX_GAME_DEPTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VM\",\"outputs\":[{\"internalType\":\"contractIBigStepper\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_ident\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_partOffset\",\"type\":\"uint256\"}],\"name\":\"addLocalData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_parentIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"}],\"name\":\"attack\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"bondManager\",\"outputs\":[{\"internalType\":\"contractIBondManager\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"claimData\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"parentIndex\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"countered\",\"type\":\"bool\"},{\"internalType\":\"Claim\",\"name\":\"claim\",\"type\":\"bytes32\"},{\"internalType\":\"Position\",\"name\":\"position\",\"type\":\"uint128\"},{\"internalType\":\"Clock\",\"name\":\"clock\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"claimDataLen\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"len_\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"createdAt\",\"outputs\":[{\"internalType\":\"Timestamp\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_parentIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"}],\"name\":\"defend\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"extraData\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"extraData_\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gameData\",\"outputs\":[{\"internalType\":\"GameType\",\"name\":\"gameType_\",\"type\":\"uint8\"},{\"internalType\":\"Claim\",\"name\":\"rootClaim_\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"extraData_\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"gameType\",\"outputs\":[{\"internalType\":\"GameType\",\"name\":\"gameType_\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"l1BlockNumber_\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Head\",\"outputs\":[{\"internalType\":\"Hash\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"l2BlockNumber_\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_challengeIndex\",\"type\":\"uint256\"},{\"internalType\":\"Claim\",\"name\":\"_claim\",\"type\":\"bytes32\"},{\"internalType\":\"bool\",\"name\":\"_isAttack\",\"type\":\"bool\"}],\"name\":\"move\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proposals\",\"outputs\":[{\"components\":[{\"internalType\":\"uint128\",\"name\":\"index\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"l2BlockNumber\",\"type\":\"uint128\"},{\"internalType\":\"Hash\",\"name\":\"outputRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structIFaultDisputeGame.OutputProposal\",\"name\":\"starting\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint128\",\"name\":\"index\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"l2BlockNumber\",\"type\":\"uint128\"},{\"internalType\":\"Hash\",\"name\":\"outputRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structIFaultDisputeGame.OutputProposal\",\"name\":\"disputed\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resolve\",\"outputs\":[{\"internalType\":\"enumGameStatus\",\"name\":\"status_\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"rootClaim\",\"outputs\":[{\"internalType\":\"Claim\",\"name\":\"rootClaim_\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"status\",\"outputs\":[{\"internalType\":\"enumGameStatus\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_claimIndex\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"_isAttack\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_stateData\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_proof\",\"type\":\"bytes\"}],\"name\":\"step\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101c06040523480156200001257600080fd5b5060405162002cb338038062002cb38339810160408190526200003591620000a1565b6000608081905260a052600760c05260ff9096166101a05260e094909452610100929092526001600160401b0316610120526001600160a01b039081166101405290811661016052166101805262000145565b6001600160a01b03811681146200009e57600080fd5b50565b600080600080600080600060e0888a031215620000bd57600080fd5b875160ff81168114620000cf57600080fd5b602089015160408a015160608b015192995090975095506001600160401b0381168114620000fc57600080fd5b60808901519094506200010f8162000088565b60a0890151909350620001228162000088565b60c0890151909250620001358162000088565b8091505092959891949750929550565b60805160a05160c05160e05161010051610120516101405161016051610180516101a051612a976200021c600039600081816105220152611e5d01526000818161035e01526116e701526000818161059b015281816114b40152818161158801526116610152600081816104ec015281816107450152611bdc0152600081816105cf01528181610ab7015261109801526000818161032a015281816109bf01528181610ed701526119e30152600081816102210152611b3f01526000610d3401526000610d0b01526000610ce20152612a976000f3fe6080604052600436106101ac5760003560e01c80636361506d116100ec578063c0c3a0921161008a578063c6f0308c11610064578063c6f0308c1461061d578063cf09e0d014610681578063d8cc1a3c146106a2578063fa24f743146106c257600080fd5b8063c0c3a09214610589578063c31b29ce146105bd578063c55cd0c71461060a57600080fd5b80638b85902b116100c65780638b85902b1461049a57806392931298146104da578063bbdc02db1461050e578063bcef3b551461054c57600080fd5b80636361506d1461045a5780638129fc1c146104705780638980e0cc1461048557600080fd5b8063363cc4271161015957806354fd4d501161013357806354fd4d501461038057806355ef20e6146103a2578063609d333414610432578063632247ea1461044757600080fd5b8063363cc427146102b95780634778efe814610318578063529184c91461034c57600080fd5b80632810e1d61161018a5780632810e1d614610251578063298c90051461026657806335fef567146102a657600080fd5b80631e27052a146101b1578063200d2ed2146101d3578063266198f91461020f575b600080fd5b3480156101bd57600080fd5b506101d16101cc366004612338565b6106e6565b005b3480156101df57600080fd5b506000546101f99068010000000000000000900460ff1681565b6040516102069190612389565b60405180910390f35b34801561021b57600080fd5b506102437f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610206565b34801561025d57600080fd5b506101f96108a5565b34801561027257600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c900360400135610243565b6101d16102b4366004612338565b610ccb565b3480156102c557600080fd5b506000546102f3906901000000000000000000900473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610206565b34801561032457600080fd5b506102437f000000000000000000000000000000000000000000000000000000000000000081565b34801561035857600080fd5b506102f37f000000000000000000000000000000000000000000000000000000000000000081565b34801561038c57600080fd5b50610395610cdb565b6040516102069190612440565b3480156103ae57600080fd5b5060408051606080820183526003546fffffffffffffffffffffffffffffffff808216845270010000000000000000000000000000000091829004811660208086019190915260045485870152855193840186526005548083168552929092041690820152600654928101929092526104249182565b60405161020692919061245a565b34801561043e57600080fd5b50610395610d7e565b6101d16104553660046124c3565b610d8c565b34801561046657600080fd5b5061024360015481565b34801561047c57600080fd5b506101d1611360565b34801561049157600080fd5b50600254610243565b3480156104a657600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c900360200135610243565b3480156104e657600080fd5b506102f37f000000000000000000000000000000000000000000000000000000000000000081565b34801561051a57600080fd5b5060405160ff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610206565b34801561055857600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c900335610243565b34801561059557600080fd5b506102f37f000000000000000000000000000000000000000000000000000000000000000081565b3480156105c957600080fd5b506105f17f000000000000000000000000000000000000000000000000000000000000000081565b60405167ffffffffffffffff9091168152602001610206565b6101d1610618366004612338565b6118bd565b34801561062957600080fd5b5061063d6106383660046124f8565b6118c9565b6040805163ffffffff90961686529315156020860152928401919091526fffffffffffffffffffffffffffffffff908116606084015216608082015260a001610206565b34801561068d57600080fd5b506000546105f19067ffffffffffffffff1681565b3480156106ae57600080fd5b506101d16106bd36600461255a565b61193a565b3480156106ce57600080fd5b506106d7611e5b565b604051610206939291906125e4565b6000805468010000000000000000900460ff16600281111561070a5761070a61235a565b14610741576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16637dc0d1d06040518163ffffffff1660e01b8152600401602060405180830381865afa1580156107ae573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906107d2919061260f565b7f9a1f5e7f00000000000000000000000000000000000000000000000000000000601c8190526020859052909150600084600181146108395760028114610843576003811461084d576004811461085757600581146108675763ff137e656000526004601cfd5b600154915061086e565b600454915061086e565b600654915061086e565b60035460801c60c01b915061086e565b4660c01b91505b50604052600160038511811b6005031b60605260808390526000806084601c82865af161089f573d6000803e3d6000fd5b50505050565b60008060005468010000000000000000900460ff1660028111156108cb576108cb61235a565b14610902576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60025460009061091490600190612674565b90506fffffffffffffffffffffffffffffffff815b67ffffffffffffffff8110156109fe5760006002828154811061094e5761094e61268b565b6000918252602090912060039091020180547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff9093019290915060ff640100000000909104161561099f5750610929565b60028101546000906109e3906fffffffffffffffffffffffffffffffff167f0000000000000000000000000000000000000000000000000000000000000000611eb8565b9050838110156109f7578093508260010194505b5050610929565b50600060028381548110610a1457610a1461268b565b600091825260208220600390910201805490925063ffffffff90811691908214610a7e5760028281548110610a4b57610a4b61268b565b906000526020600020906003020160020160109054906101000a90046fffffffffffffffffffffffffffffffff16610aaa565b600283015470010000000000000000000000000000000090046fffffffffffffffffffffffffffffffff165b9050677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c16610aee67ffffffffffffffff831642612674565b610b0a836fffffffffffffffffffffffffffffffff1660401c90565b67ffffffffffffffff16610b1e91906126ba565b11610b55576040517ff2440b5300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600283810154610bf7906fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b610c019190612701565b67ffffffffffffffff16158015610c2857506fffffffffffffffffffffffffffffffff8414155b15610c365760029550610c3b565b600195505b600080548791907fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff1668010000000000000000836002811115610c8057610c8061235a565b021790556002811115610c9557610c9561235a565b6040517f5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da6090600090a2505050505090565b905090565b610cd782826000610d8c565b5050565b6060610d067f0000000000000000000000000000000000000000000000000000000000000000611f6d565b610d2f7f0000000000000000000000000000000000000000000000000000000000000000611f6d565b610d587f0000000000000000000000000000000000000000000000000000000000000000611f6d565b604051602001610d6a93929190612728565b604051602081830303815290604052905090565b6060610cc6602060406120aa565b6000805468010000000000000000900460ff166002811115610db057610db061235a565b14610de7576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82158015610df3575080155b15610e2a576040517fa42637bc00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028481548110610e3f57610e3f61268b565b600091825260208083206040805160a081018252600394909402909101805463ffffffff808216865264010000000090910460ff16151593850193909352600181015491840191909152600201546fffffffffffffffffffffffffffffffff80821660608501819052700100000000000000000000000000000000909204166080840152919350610ed39190859061214116565b90507f0000000000000000000000000000000000000000000000000000000000000000610f92826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff161115610fd4576040517f56f57b2b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b815160009063ffffffff90811614611034576002836000015163ffffffff16815481106110035761100361268b565b906000526020600020906003020160020160109054906101000a90046fffffffffffffffffffffffffffffffff1690505b608083015160009067ffffffffffffffff1667ffffffffffffffff164261106d846fffffffffffffffffffffffffffffffff1660401c90565b67ffffffffffffffff1661108191906126ba565b61108b9190612674565b9050677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c1667ffffffffffffffff821611156110fe576040517f3381d11400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000604082901b42179050600061111f888660009182526020526040902090565b60008181526007602052604090205490915060ff161561116b576040517f80497e3b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081815260076020908152604080832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001908117909155815160a08101835263ffffffff808f1682529381018581529281018d81526fffffffffffffffffffffffffffffffff808c16606084019081528982166080850190815260028054808801825599819052945160039099027f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace8101805498511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009099169a909916999099179690961790965590517f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5acf8701559351925184167001000000000000000000000000000000000292909316919091177f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ad09093019290925580548b9081106112e3576112e361268b565b6000918252602082206003909102018054921515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff9093169290921790915560405133918a918c917f9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be91a4505050505050505050565b600080547fffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000164267ffffffffffffffff161781556040805160a08101825263ffffffff815260208101929092526002919081016113e57ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c90033590565b815260016020820152604001426fffffffffffffffffffffffffffffffff9081169091528254600181810185556000948552602080862085516003909402018054918601511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff000000000090921663ffffffff909416939093171782556040840151908201556060830151608090930151821670010000000000000000000000000000000002929091169190911760029091015573ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016637f00642061150e60207ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c9003013590565b6040518263ffffffff1660e01b815260040161152c91815260200190565b602060405180830381865afa158015611549573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061156d919061279e565b9050600073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001663a25ae5576115b8600185612674565b6040518263ffffffff1660e01b81526004016115d691815260200190565b606060405180830381865afa1580156115f3573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116179190612806565b6040517fa25ae5570000000000000000000000000000000000000000000000000000000081526004810184905290915060009073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063a25ae55790602401606060405180830381865afa1580156116a8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906116cc9190612806565b9050600073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000166399d548aa367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003604001356040518263ffffffff1660e01b815260040161175891815260200190565b6040805180830381865afa158015611774573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906117989190612892565b905081602001516fffffffffffffffffffffffffffffffff16816020015167ffffffffffffffff16116117f7576040517f13809ba500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040805160a081018252908190810180611812600189612674565b6fffffffffffffffffffffffffffffffff9081168252604088810151821660208085019190915298519281019290925291835280516060810182529782168852858101518216888801529451878601529085019590955280518051818601519087167001000000000000000000000000000000009188168202176003559084015160045590840151805194810151948616949095160292909217600555919091015160065551600155565b610cd782826001610d8c565b600281815481106118d957600080fd5b600091825260209091206003909102018054600182015460029092015463ffffffff8216935064010000000090910460ff1691906fffffffffffffffffffffffffffffffff8082169170010000000000000000000000000000000090041685565b6000805468010000000000000000900460ff16600281111561195e5761195e61235a565b14611995576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000600287815481106119aa576119aa61268b565b6000918252602082206003919091020160028101549092506fffffffffffffffffffffffffffffffff16908715821760011b9050611a097f000000000000000000000000000000000000000000000000000000000000000060016126ba565b611aa5826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff1614611ae6576040517f5f53dd9800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000808915611b6957611b0a836fffffffffffffffffffffffffffffffff16612149565b67ffffffffffffffff1615611b3d57611b34611b27600186612919565b865463ffffffff166121ef565b60010154611b5f565b7f00000000000000000000000000000000000000000000000000000000000000005b9150849050611b83565b84600101549150611b80846001611b27919061294a565b90505b818989604051611b9492919061297e565b604051809103902014611bd3576040517f696550ff00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081600101547f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663f8e0cb968c8c8c8c6040518563ffffffff1660e01b8152600401611c3994939291906129d7565b6020604051808303816000875af1158015611c58573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611c7c919061279e565b600284810154929091149250600091611d27906fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b611dc3886fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b611dcd9190612a09565b611dd79190612701565b67ffffffffffffffff161590508115158103611e1f576040517ffb4e40dd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b505084547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff166401000000001790945550505050505050505050565b7f0000000000000000000000000000000000000000000000000000000000000000367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003356060611eb1610d7e565b9050909192565b600080611f45847e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff1690508083036001841b600180831b0386831b17039250505092915050565b606081600003611fb057505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115611fda5780611fc481612a2a565b9150611fd39050600a83612a62565b9150611fb4565b60008167ffffffffffffffff811115611ff557611ff56127b7565b6040519080825280601f01601f19166020018201604052801561201f576020820181803683370190505b5090505b84156120a257612034600183612674565b9150612041600a86612a76565b61204c9060306126ba565b60f81b8183815181106120615761206161268b565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535061209b600a86612a62565b9450612023565b949350505050565b606060006120e184367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036126ba565b90508267ffffffffffffffff1667ffffffffffffffff811115612106576121066127b7565b6040519080825280601f01601f191660200182016040528015612130576020820181803683370190505b509150828160208401375092915050565b151760011b90565b6000806121d6837e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b600167ffffffffffffffff919091161b90920392915050565b60008061220d846fffffffffffffffffffffffffffffffff1661228c565b9050600283815481106122225761222261268b565b906000526020600020906003020191505b60028201546fffffffffffffffffffffffffffffffff82811691161461228557815460028054909163ffffffff169081106122705761227061268b565b90600052602060002090600302019150612233565b5092915050565b60008119600183011681612320827e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff169390931c8015179392505050565b6000806040838503121561234b57600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b60208101600383106123c4577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b60005b838110156123e55781810151838201526020016123cd565b8381111561089f5750506000910152565b6000815180845261240e8160208601602086016123ca565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061245360208301846123f6565b9392505050565b82516fffffffffffffffffffffffffffffffff90811682526020808501518216818401526040808601518185015284518316606085015290840151909116608083015282015160a082015260c08101612453565b803580151581146124be57600080fd5b919050565b6000806000606084860312156124d857600080fd5b83359250602084013591506124ef604085016124ae565b90509250925092565b60006020828403121561250a57600080fd5b5035919050565b60008083601f84011261252357600080fd5b50813567ffffffffffffffff81111561253b57600080fd5b60208301915083602082850101111561255357600080fd5b9250929050565b6000806000806000806080878903121561257357600080fd5b86359550612583602088016124ae565b9450604087013567ffffffffffffffff808211156125a057600080fd5b6125ac8a838b01612511565b909650945060608901359150808211156125c557600080fd5b506125d289828a01612511565b979a9699509497509295939492505050565b60ff8416815282602082015260606040820152600061260660608301846123f6565b95945050505050565b60006020828403121561262157600080fd5b815173ffffffffffffffffffffffffffffffffffffffff8116811461245357600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008282101561268657612686612645565b500390565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082198211156126cd576126cd612645565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b600067ffffffffffffffff8084168061271c5761271c6126d2565b92169190910692915050565b6000845161273a8184602089016123ca565b80830190507f2e000000000000000000000000000000000000000000000000000000000000008082528551612776816001850160208a016123ca565b600192019182015283516127918160028401602088016123ca565b0160020195945050505050565b6000602082840312156127b057600080fd5b5051919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b80516fffffffffffffffffffffffffffffffff811681146124be57600080fd5b60006060828403121561281857600080fd5b6040516060810181811067ffffffffffffffff82111715612862577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60405282518152612875602084016127e6565b6020820152612886604084016127e6565b60408201529392505050565b6000604082840312156128a457600080fd5b6040516040810167ffffffffffffffff82821081831117156128ef577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b816040528451835260208501519150808216821461290c57600080fd5b5060208201529392505050565b60006fffffffffffffffffffffffffffffffff8381169083168181101561294257612942612645565b039392505050565b60006fffffffffffffffffffffffffffffffff80831681851680830382111561297557612975612645565b01949350505050565b8183823760009101908152919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b6040815260006129eb60408301868861298e565b82810360208401526129fe81858761298e565b979650505050505050565b600067ffffffffffffffff8381169083168181101561294257612942612645565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203612a5b57612a5b612645565b5060010190565b600082612a7157612a716126d2565b500490565b600082612a8557612a856126d2565b50069056fea164736f6c634300080f000a",
}

// FaultDisputeGameABI is the input ABI used to generate the binding from.
// Deprecated: Use FaultDisputeGameMetaData.ABI instead.
var FaultDisputeGameABI = FaultDisputeGameMetaData.ABI

// FaultDisputeGameBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use FaultDisputeGameMetaData.Bin instead.
var FaultDisputeGameBin = FaultDisputeGameMetaData.Bin

// DeployFaultDisputeGame deploys a new Ethereum contract, binding an instance of FaultDisputeGame to it.
func DeployFaultDisputeGame(auth *bind.TransactOpts, backend bind.ContractBackend, _gameType uint8, _absolutePrestate [32]byte, _maxGameDepth *big.Int, _gameDuration uint64, _vm common.Address, _l2oo common.Address, _blockOracle common.Address) (common.Address, *types.Transaction, *FaultDisputeGame, error) {
	parsed, err := FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(FaultDisputeGameBin), backend, _gameType, _absolutePrestate, _maxGameDepth, _gameDuration, _vm, _l2oo, _blockOracle)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &FaultDisputeGame{FaultDisputeGameCaller: FaultDisputeGameCaller{contract: contract}, FaultDisputeGameTransactor: FaultDisputeGameTransactor{contract: contract}, FaultDisputeGameFilterer: FaultDisputeGameFilterer{contract: contract}}, nil
}

// FaultDisputeGame is an auto generated Go binding around an Ethereum contract.
type FaultDisputeGame struct {
	FaultDisputeGameCaller     // Read-only binding to the contract
	FaultDisputeGameTransactor // Write-only binding to the contract
	FaultDisputeGameFilterer   // Log filterer for contract events
}

// FaultDisputeGameCaller is an auto generated read-only Go binding around an Ethereum contract.
type FaultDisputeGameCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FaultDisputeGameTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FaultDisputeGameFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FaultDisputeGameSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FaultDisputeGameSession struct {
	Contract     *FaultDisputeGame // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// FaultDisputeGameCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FaultDisputeGameCallerSession struct {
	Contract *FaultDisputeGameCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// FaultDisputeGameTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FaultDisputeGameTransactorSession struct {
	Contract     *FaultDisputeGameTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// FaultDisputeGameRaw is an auto generated low-level Go binding around an Ethereum contract.
type FaultDisputeGameRaw struct {
	Contract *FaultDisputeGame // Generic contract binding to access the raw methods on
}

// FaultDisputeGameCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FaultDisputeGameCallerRaw struct {
	Contract *FaultDisputeGameCaller // Generic read-only contract binding to access the raw methods on
}

// FaultDisputeGameTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FaultDisputeGameTransactorRaw struct {
	Contract *FaultDisputeGameTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFaultDisputeGame creates a new instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGame(address common.Address, backend bind.ContractBackend) (*FaultDisputeGame, error) {
	contract, err := bindFaultDisputeGame(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGame{FaultDisputeGameCaller: FaultDisputeGameCaller{contract: contract}, FaultDisputeGameTransactor: FaultDisputeGameTransactor{contract: contract}, FaultDisputeGameFilterer: FaultDisputeGameFilterer{contract: contract}}, nil
}

// NewFaultDisputeGameCaller creates a new read-only instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameCaller(address common.Address, caller bind.ContractCaller) (*FaultDisputeGameCaller, error) {
	contract, err := bindFaultDisputeGame(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameCaller{contract: contract}, nil
}

// NewFaultDisputeGameTransactor creates a new write-only instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameTransactor(address common.Address, transactor bind.ContractTransactor) (*FaultDisputeGameTransactor, error) {
	contract, err := bindFaultDisputeGame(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameTransactor{contract: contract}, nil
}

// NewFaultDisputeGameFilterer creates a new log filterer instance of FaultDisputeGame, bound to a specific deployed contract.
func NewFaultDisputeGameFilterer(address common.Address, filterer bind.ContractFilterer) (*FaultDisputeGameFilterer, error) {
	contract, err := bindFaultDisputeGame(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameFilterer{contract: contract}, nil
}

// bindFaultDisputeGame binds a generic wrapper to an already deployed contract.
func bindFaultDisputeGame(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(FaultDisputeGameABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FaultDisputeGame *FaultDisputeGameRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FaultDisputeGame.Contract.FaultDisputeGameCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FaultDisputeGame *FaultDisputeGameRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.FaultDisputeGameTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FaultDisputeGame *FaultDisputeGameRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.FaultDisputeGameTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FaultDisputeGame *FaultDisputeGameCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FaultDisputeGame.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FaultDisputeGame *FaultDisputeGameTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FaultDisputeGame *FaultDisputeGameTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.contract.Transact(opts, method, params...)
}

// ABSOLUTEPRESTATE is a free data retrieval call binding the contract method 0x266198f9.
//
// Solidity: function ABSOLUTE_PRESTATE() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameCaller) ABSOLUTEPRESTATE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "ABSOLUTE_PRESTATE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// ABSOLUTEPRESTATE is a free data retrieval call binding the contract method 0x266198f9.
//
// Solidity: function ABSOLUTE_PRESTATE() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameSession) ABSOLUTEPRESTATE() ([32]byte, error) {
	return _FaultDisputeGame.Contract.ABSOLUTEPRESTATE(&_FaultDisputeGame.CallOpts)
}

// ABSOLUTEPRESTATE is a free data retrieval call binding the contract method 0x266198f9.
//
// Solidity: function ABSOLUTE_PRESTATE() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ABSOLUTEPRESTATE() ([32]byte, error) {
	return _FaultDisputeGame.Contract.ABSOLUTEPRESTATE(&_FaultDisputeGame.CallOpts)
}

// BLOCKORACLE is a free data retrieval call binding the contract method 0x529184c9.
//
// Solidity: function BLOCK_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCaller) BLOCKORACLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "BLOCK_ORACLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BLOCKORACLE is a free data retrieval call binding the contract method 0x529184c9.
//
// Solidity: function BLOCK_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameSession) BLOCKORACLE() (common.Address, error) {
	return _FaultDisputeGame.Contract.BLOCKORACLE(&_FaultDisputeGame.CallOpts)
}

// BLOCKORACLE is a free data retrieval call binding the contract method 0x529184c9.
//
// Solidity: function BLOCK_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) BLOCKORACLE() (common.Address, error) {
	return _FaultDisputeGame.Contract.BLOCKORACLE(&_FaultDisputeGame.CallOpts)
}

// GAMEDURATION is a free data retrieval call binding the contract method 0xc31b29ce.
//
// Solidity: function GAME_DURATION() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCaller) GAMEDURATION(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "GAME_DURATION")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GAMEDURATION is a free data retrieval call binding the contract method 0xc31b29ce.
//
// Solidity: function GAME_DURATION() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameSession) GAMEDURATION() (uint64, error) {
	return _FaultDisputeGame.Contract.GAMEDURATION(&_FaultDisputeGame.CallOpts)
}

// GAMEDURATION is a free data retrieval call binding the contract method 0xc31b29ce.
//
// Solidity: function GAME_DURATION() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GAMEDURATION() (uint64, error) {
	return _FaultDisputeGame.Contract.GAMEDURATION(&_FaultDisputeGame.CallOpts)
}

// L2OUTPUTORACLE is a free data retrieval call binding the contract method 0xc0c3a092.
//
// Solidity: function L2_OUTPUT_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2OUTPUTORACLE(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "L2_OUTPUT_ORACLE")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2OUTPUTORACLE is a free data retrieval call binding the contract method 0xc0c3a092.
//
// Solidity: function L2_OUTPUT_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameSession) L2OUTPUTORACLE() (common.Address, error) {
	return _FaultDisputeGame.Contract.L2OUTPUTORACLE(&_FaultDisputeGame.CallOpts)
}

// L2OUTPUTORACLE is a free data retrieval call binding the contract method 0xc0c3a092.
//
// Solidity: function L2_OUTPUT_ORACLE() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2OUTPUTORACLE() (common.Address, error) {
	return _FaultDisputeGame.Contract.L2OUTPUTORACLE(&_FaultDisputeGame.CallOpts)
}

// MAXGAMEDEPTH is a free data retrieval call binding the contract method 0x4778efe8.
//
// Solidity: function MAX_GAME_DEPTH() view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCaller) MAXGAMEDEPTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "MAX_GAME_DEPTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXGAMEDEPTH is a free data retrieval call binding the contract method 0x4778efe8.
//
// Solidity: function MAX_GAME_DEPTH() view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameSession) MAXGAMEDEPTH() (*big.Int, error) {
	return _FaultDisputeGame.Contract.MAXGAMEDEPTH(&_FaultDisputeGame.CallOpts)
}

// MAXGAMEDEPTH is a free data retrieval call binding the contract method 0x4778efe8.
//
// Solidity: function MAX_GAME_DEPTH() view returns(uint256)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) MAXGAMEDEPTH() (*big.Int, error) {
	return _FaultDisputeGame.Contract.MAXGAMEDEPTH(&_FaultDisputeGame.CallOpts)
}

// VM is a free data retrieval call binding the contract method 0x92931298.
//
// Solidity: function VM() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCaller) VM(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "VM")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// VM is a free data retrieval call binding the contract method 0x92931298.
//
// Solidity: function VM() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameSession) VM() (common.Address, error) {
	return _FaultDisputeGame.Contract.VM(&_FaultDisputeGame.CallOpts)
}

// VM is a free data retrieval call binding the contract method 0x92931298.
//
// Solidity: function VM() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) VM() (common.Address, error) {
	return _FaultDisputeGame.Contract.VM(&_FaultDisputeGame.CallOpts)
}

// BondManager is a free data retrieval call binding the contract method 0x363cc427.
//
// Solidity: function bondManager() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCaller) BondManager(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "bondManager")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// BondManager is a free data retrieval call binding the contract method 0x363cc427.
//
// Solidity: function bondManager() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameSession) BondManager() (common.Address, error) {
	return _FaultDisputeGame.Contract.BondManager(&_FaultDisputeGame.CallOpts)
}

// BondManager is a free data retrieval call binding the contract method 0x363cc427.
//
// Solidity: function bondManager() view returns(address)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) BondManager() (common.Address, error) {
	return _FaultDisputeGame.Contract.BondManager(&_FaultDisputeGame.CallOpts)
}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, bool countered, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameCaller) ClaimData(opts *bind.CallOpts, arg0 *big.Int) (struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "claimData", arg0)

	outstruct := new(struct {
		ParentIndex uint32
		Countered   bool
		Claim       [32]byte
		Position    *big.Int
		Clock       *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.ParentIndex = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.Countered = *abi.ConvertType(out[1], new(bool)).(*bool)
	outstruct.Claim = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)
	outstruct.Position = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Clock = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, bool countered, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameSession) ClaimData(arg0 *big.Int) (struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	return _FaultDisputeGame.Contract.ClaimData(&_FaultDisputeGame.CallOpts, arg0)
}

// ClaimData is a free data retrieval call binding the contract method 0xc6f0308c.
//
// Solidity: function claimData(uint256 ) view returns(uint32 parentIndex, bool countered, bytes32 claim, uint128 position, uint128 clock)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ClaimData(arg0 *big.Int) (struct {
	ParentIndex uint32
	Countered   bool
	Claim       [32]byte
	Position    *big.Int
	Clock       *big.Int
}, error) {
	return _FaultDisputeGame.Contract.ClaimData(&_FaultDisputeGame.CallOpts, arg0)
}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameCaller) ClaimDataLen(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "claimDataLen")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameSession) ClaimDataLen() (*big.Int, error) {
	return _FaultDisputeGame.Contract.ClaimDataLen(&_FaultDisputeGame.CallOpts)
}

// ClaimDataLen is a free data retrieval call binding the contract method 0x8980e0cc.
//
// Solidity: function claimDataLen() view returns(uint256 len_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ClaimDataLen() (*big.Int, error) {
	return _FaultDisputeGame.Contract.ClaimDataLen(&_FaultDisputeGame.CallOpts)
}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCaller) CreatedAt(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "createdAt")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameSession) CreatedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.CreatedAt(&_FaultDisputeGame.CallOpts)
}

// CreatedAt is a free data retrieval call binding the contract method 0xcf09e0d0.
//
// Solidity: function createdAt() view returns(uint64)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) CreatedAt() (uint64, error) {
	return _FaultDisputeGame.Contract.CreatedAt(&_FaultDisputeGame.CallOpts)
}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCaller) ExtraData(opts *bind.CallOpts) ([]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "extraData")

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameSession) ExtraData() ([]byte, error) {
	return _FaultDisputeGame.Contract.ExtraData(&_FaultDisputeGame.CallOpts)
}

// ExtraData is a free data retrieval call binding the contract method 0x609d3334.
//
// Solidity: function extraData() pure returns(bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) ExtraData() ([]byte, error) {
	return _FaultDisputeGame.Contract.ExtraData(&_FaultDisputeGame.CallOpts)
}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint8 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GameData(opts *bind.CallOpts) (struct {
	GameType  uint8
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "gameData")

	outstruct := new(struct {
		GameType  uint8
		RootClaim [32]byte
		ExtraData []byte
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.GameType = *abi.ConvertType(out[0], new(uint8)).(*uint8)
	outstruct.RootClaim = *abi.ConvertType(out[1], new([32]byte)).(*[32]byte)
	outstruct.ExtraData = *abi.ConvertType(out[2], new([]byte)).(*[]byte)

	return *outstruct, err

}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint8 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameSession) GameData() (struct {
	GameType  uint8
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	return _FaultDisputeGame.Contract.GameData(&_FaultDisputeGame.CallOpts)
}

// GameData is a free data retrieval call binding the contract method 0xfa24f743.
//
// Solidity: function gameData() view returns(uint8 gameType_, bytes32 rootClaim_, bytes extraData_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GameData() (struct {
	GameType  uint8
	RootClaim [32]byte
	ExtraData []byte
}, error) {
	return _FaultDisputeGame.Contract.GameData(&_FaultDisputeGame.CallOpts)
}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint8 gameType_)
func (_FaultDisputeGame *FaultDisputeGameCaller) GameType(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "gameType")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint8 gameType_)
func (_FaultDisputeGame *FaultDisputeGameSession) GameType() (uint8, error) {
	return _FaultDisputeGame.Contract.GameType(&_FaultDisputeGame.CallOpts)
}

// GameType is a free data retrieval call binding the contract method 0xbbdc02db.
//
// Solidity: function gameType() view returns(uint8 gameType_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) GameType() (uint8, error) {
	return _FaultDisputeGame.Contract.GameType(&_FaultDisputeGame.CallOpts)
}

// L1BlockNumber is a free data retrieval call binding the contract method 0x298c9005.
//
// Solidity: function l1BlockNumber() pure returns(uint256 l1BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCaller) L1BlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l1BlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L1BlockNumber is a free data retrieval call binding the contract method 0x298c9005.
//
// Solidity: function l1BlockNumber() pure returns(uint256 l1BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameSession) L1BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L1BlockNumber(&_FaultDisputeGame.CallOpts)
}

// L1BlockNumber is a free data retrieval call binding the contract method 0x298c9005.
//
// Solidity: function l1BlockNumber() pure returns(uint256 l1BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L1BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L1BlockNumber(&_FaultDisputeGame.CallOpts)
}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameCaller) L1Head(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l1Head")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameSession) L1Head() ([32]byte, error) {
	return _FaultDisputeGame.Contract.L1Head(&_FaultDisputeGame.CallOpts)
}

// L1Head is a free data retrieval call binding the contract method 0x6361506d.
//
// Solidity: function l1Head() view returns(bytes32)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L1Head() ([32]byte, error) {
	return _FaultDisputeGame.Contract.L1Head(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCaller) L2BlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "l2BlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameSession) L2BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2BlockNumber(&_FaultDisputeGame.CallOpts)
}

// L2BlockNumber is a free data retrieval call binding the contract method 0x8b85902b.
//
// Solidity: function l2BlockNumber() pure returns(uint256 l2BlockNumber_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) L2BlockNumber() (*big.Int, error) {
	return _FaultDisputeGame.Contract.L2BlockNumber(&_FaultDisputeGame.CallOpts)
}

// Proposals is a free data retrieval call binding the contract method 0x55ef20e6.
//
// Solidity: function proposals() view returns((uint128,uint128,bytes32) starting, (uint128,uint128,bytes32) disputed)
func (_FaultDisputeGame *FaultDisputeGameCaller) Proposals(opts *bind.CallOpts) (struct {
	Starting IFaultDisputeGameOutputProposal
	Disputed IFaultDisputeGameOutputProposal
}, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "proposals")

	outstruct := new(struct {
		Starting IFaultDisputeGameOutputProposal
		Disputed IFaultDisputeGameOutputProposal
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Starting = *abi.ConvertType(out[0], new(IFaultDisputeGameOutputProposal)).(*IFaultDisputeGameOutputProposal)
	outstruct.Disputed = *abi.ConvertType(out[1], new(IFaultDisputeGameOutputProposal)).(*IFaultDisputeGameOutputProposal)

	return *outstruct, err

}

// Proposals is a free data retrieval call binding the contract method 0x55ef20e6.
//
// Solidity: function proposals() view returns((uint128,uint128,bytes32) starting, (uint128,uint128,bytes32) disputed)
func (_FaultDisputeGame *FaultDisputeGameSession) Proposals() (struct {
	Starting IFaultDisputeGameOutputProposal
	Disputed IFaultDisputeGameOutputProposal
}, error) {
	return _FaultDisputeGame.Contract.Proposals(&_FaultDisputeGame.CallOpts)
}

// Proposals is a free data retrieval call binding the contract method 0x55ef20e6.
//
// Solidity: function proposals() view returns((uint128,uint128,bytes32) starting, (uint128,uint128,bytes32) disputed)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Proposals() (struct {
	Starting IFaultDisputeGameOutputProposal
	Disputed IFaultDisputeGameOutputProposal
}, error) {
	return _FaultDisputeGame.Contract.Proposals(&_FaultDisputeGame.CallOpts)
}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameCaller) RootClaim(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "rootClaim")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameSession) RootClaim() ([32]byte, error) {
	return _FaultDisputeGame.Contract.RootClaim(&_FaultDisputeGame.CallOpts)
}

// RootClaim is a free data retrieval call binding the contract method 0xbcef3b55.
//
// Solidity: function rootClaim() pure returns(bytes32 rootClaim_)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) RootClaim() ([32]byte, error) {
	return _FaultDisputeGame.Contract.RootClaim(&_FaultDisputeGame.CallOpts)
}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameCaller) Status(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "status")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameSession) Status() (uint8, error) {
	return _FaultDisputeGame.Contract.Status(&_FaultDisputeGame.CallOpts)
}

// Status is a free data retrieval call binding the contract method 0x200d2ed2.
//
// Solidity: function status() view returns(uint8)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Status() (uint8, error) {
	return _FaultDisputeGame.Contract.Status(&_FaultDisputeGame.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FaultDisputeGame.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameSession) Version() (string, error) {
	return _FaultDisputeGame.Contract.Version(&_FaultDisputeGame.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_FaultDisputeGame *FaultDisputeGameCallerSession) Version() (string, error) {
	return _FaultDisputeGame.Contract.Version(&_FaultDisputeGame.CallOpts)
}

// AddLocalData is a paid mutator transaction binding the contract method 0x1e27052a.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) AddLocalData(opts *bind.TransactOpts, _ident *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "addLocalData", _ident, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0x1e27052a.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) AddLocalData(_ident *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0x1e27052a.
//
// Solidity: function addLocalData(uint256 _ident, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) AddLocalData(_ident *big.Int, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _partOffset)
}

// Attack is a paid mutator transaction binding the contract method 0xc55cd0c7.
//
// Solidity: function attack(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Attack(opts *bind.TransactOpts, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "attack", _parentIndex, _claim)
}

// Attack is a paid mutator transaction binding the contract method 0xc55cd0c7.
//
// Solidity: function attack(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Attack(_parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Attack(&_FaultDisputeGame.TransactOpts, _parentIndex, _claim)
}

// Attack is a paid mutator transaction binding the contract method 0xc55cd0c7.
//
// Solidity: function attack(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Attack(_parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Attack(&_FaultDisputeGame.TransactOpts, _parentIndex, _claim)
}

// Defend is a paid mutator transaction binding the contract method 0x35fef567.
//
// Solidity: function defend(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Defend(opts *bind.TransactOpts, _parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "defend", _parentIndex, _claim)
}

// Defend is a paid mutator transaction binding the contract method 0x35fef567.
//
// Solidity: function defend(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Defend(_parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Defend(&_FaultDisputeGame.TransactOpts, _parentIndex, _claim)
}

// Defend is a paid mutator transaction binding the contract method 0x35fef567.
//
// Solidity: function defend(uint256 _parentIndex, bytes32 _claim) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Defend(_parentIndex *big.Int, _claim [32]byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Defend(&_FaultDisputeGame.TransactOpts, _parentIndex, _claim)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Initialize(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "initialize")
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Initialize() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Initialize(&_FaultDisputeGame.TransactOpts)
}

// Initialize is a paid mutator transaction binding the contract method 0x8129fc1c.
//
// Solidity: function initialize() returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Initialize() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Initialize(&_FaultDisputeGame.TransactOpts)
}

// Move is a paid mutator transaction binding the contract method 0x632247ea.
//
// Solidity: function move(uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Move(opts *bind.TransactOpts, _challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "move", _challengeIndex, _claim, _isAttack)
}

// Move is a paid mutator transaction binding the contract method 0x632247ea.
//
// Solidity: function move(uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Move(_challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Move(&_FaultDisputeGame.TransactOpts, _challengeIndex, _claim, _isAttack)
}

// Move is a paid mutator transaction binding the contract method 0x632247ea.
//
// Solidity: function move(uint256 _challengeIndex, bytes32 _claim, bool _isAttack) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Move(_challengeIndex *big.Int, _claim [32]byte, _isAttack bool) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Move(&_FaultDisputeGame.TransactOpts, _challengeIndex, _claim, _isAttack)
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameTransactor) Resolve(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "resolve")
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameSession) Resolve() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Resolve(&_FaultDisputeGame.TransactOpts)
}

// Resolve is a paid mutator transaction binding the contract method 0x2810e1d6.
//
// Solidity: function resolve() returns(uint8 status_)
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Resolve() (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Resolve(&_FaultDisputeGame.TransactOpts)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) Step(opts *bind.TransactOpts, _claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "step", _claimIndex, _isAttack, _stateData, _proof)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) Step(_claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Step(&_FaultDisputeGame.TransactOpts, _claimIndex, _isAttack, _stateData, _proof)
}

// Step is a paid mutator transaction binding the contract method 0xd8cc1a3c.
//
// Solidity: function step(uint256 _claimIndex, bool _isAttack, bytes _stateData, bytes _proof) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) Step(_claimIndex *big.Int, _isAttack bool, _stateData []byte, _proof []byte) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.Step(&_FaultDisputeGame.TransactOpts, _claimIndex, _isAttack, _stateData, _proof)
}

// FaultDisputeGameMoveIterator is returned from FilterMove and is used to iterate over the raw logs and unpacked data for Move events raised by the FaultDisputeGame contract.
type FaultDisputeGameMoveIterator struct {
	Event *FaultDisputeGameMove // Event containing the contract specifics and raw log

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
func (it *FaultDisputeGameMoveIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaultDisputeGameMove)
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
		it.Event = new(FaultDisputeGameMove)
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
func (it *FaultDisputeGameMoveIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaultDisputeGameMoveIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaultDisputeGameMove represents a Move event raised by the FaultDisputeGame contract.
type FaultDisputeGameMove struct {
	ParentIndex *big.Int
	Claim       [32]byte
	Claimant    common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMove is a free log retrieval operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) FilterMove(opts *bind.FilterOpts, parentIndex []*big.Int, claim [][32]byte, claimant []common.Address) (*FaultDisputeGameMoveIterator, error) {

	var parentIndexRule []interface{}
	for _, parentIndexItem := range parentIndex {
		parentIndexRule = append(parentIndexRule, parentIndexItem)
	}
	var claimRule []interface{}
	for _, claimItem := range claim {
		claimRule = append(claimRule, claimItem)
	}
	var claimantRule []interface{}
	for _, claimantItem := range claimant {
		claimantRule = append(claimantRule, claimantItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.FilterLogs(opts, "Move", parentIndexRule, claimRule, claimantRule)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameMoveIterator{contract: _FaultDisputeGame.contract, event: "Move", logs: logs, sub: sub}, nil
}

// WatchMove is a free log subscription operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) WatchMove(opts *bind.WatchOpts, sink chan<- *FaultDisputeGameMove, parentIndex []*big.Int, claim [][32]byte, claimant []common.Address) (event.Subscription, error) {

	var parentIndexRule []interface{}
	for _, parentIndexItem := range parentIndex {
		parentIndexRule = append(parentIndexRule, parentIndexItem)
	}
	var claimRule []interface{}
	for _, claimItem := range claim {
		claimRule = append(claimRule, claimItem)
	}
	var claimantRule []interface{}
	for _, claimantItem := range claimant {
		claimantRule = append(claimantRule, claimantItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.WatchLogs(opts, "Move", parentIndexRule, claimRule, claimantRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaultDisputeGameMove)
				if err := _FaultDisputeGame.contract.UnpackLog(event, "Move", log); err != nil {
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

// ParseMove is a log parse operation binding the contract event 0x9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be.
//
// Solidity: event Move(uint256 indexed parentIndex, bytes32 indexed claim, address indexed claimant)
func (_FaultDisputeGame *FaultDisputeGameFilterer) ParseMove(log types.Log) (*FaultDisputeGameMove, error) {
	event := new(FaultDisputeGameMove)
	if err := _FaultDisputeGame.contract.UnpackLog(event, "Move", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FaultDisputeGameResolvedIterator is returned from FilterResolved and is used to iterate over the raw logs and unpacked data for Resolved events raised by the FaultDisputeGame contract.
type FaultDisputeGameResolvedIterator struct {
	Event *FaultDisputeGameResolved // Event containing the contract specifics and raw log

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
func (it *FaultDisputeGameResolvedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FaultDisputeGameResolved)
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
		it.Event = new(FaultDisputeGameResolved)
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
func (it *FaultDisputeGameResolvedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FaultDisputeGameResolvedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FaultDisputeGameResolved represents a Resolved event raised by the FaultDisputeGame contract.
type FaultDisputeGameResolved struct {
	Status uint8
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterResolved is a free log retrieval operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) FilterResolved(opts *bind.FilterOpts, status []uint8) (*FaultDisputeGameResolvedIterator, error) {

	var statusRule []interface{}
	for _, statusItem := range status {
		statusRule = append(statusRule, statusItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.FilterLogs(opts, "Resolved", statusRule)
	if err != nil {
		return nil, err
	}
	return &FaultDisputeGameResolvedIterator{contract: _FaultDisputeGame.contract, event: "Resolved", logs: logs, sub: sub}, nil
}

// WatchResolved is a free log subscription operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) WatchResolved(opts *bind.WatchOpts, sink chan<- *FaultDisputeGameResolved, status []uint8) (event.Subscription, error) {

	var statusRule []interface{}
	for _, statusItem := range status {
		statusRule = append(statusRule, statusItem)
	}

	logs, sub, err := _FaultDisputeGame.contract.WatchLogs(opts, "Resolved", statusRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FaultDisputeGameResolved)
				if err := _FaultDisputeGame.contract.UnpackLog(event, "Resolved", log); err != nil {
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

// ParseResolved is a log parse operation binding the contract event 0x5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da60.
//
// Solidity: event Resolved(uint8 indexed status)
func (_FaultDisputeGame *FaultDisputeGameFilterer) ParseResolved(log types.Log) (*FaultDisputeGameResolved, error) {
	event := new(FaultDisputeGameResolved)
	if err := _FaultDisputeGame.contract.UnpackLog(event, "Resolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

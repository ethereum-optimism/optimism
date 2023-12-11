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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_gameType\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"_absolutePrestate\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_maxGameDepth\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_gameDuration\",\"type\":\"uint64\",\"internalType\":\"Duration\"},{\"name\":\"_vm\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"},{\"name\":\"_l2oo\",\"type\":\"address\",\"internalType\":\"contractL2OutputOracle\"},{\"name\":\"_blockOracle\",\"type\":\"address\",\"internalType\":\"contractBlockOracle\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"ABSOLUTE_PRESTATE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"BLOCK_ORACLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractBlockOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"GAME_DURATION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"Duration\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"L2_OUTPUT_ORACLE\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractL2OutputOracle\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"MAX_GAME_DEPTH\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"VM\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIBigStepper\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"addLocalData\",\"inputs\":[{\"name\":\"_ident\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_localContext\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_partOffset\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"attack\",\"inputs\":[{\"name\":\"_parentIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_claim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"bondManager\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIBondManager\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"claimData\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"parentIndex\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"countered\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"claim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"position\",\"type\":\"uint128\",\"internalType\":\"Position\"},{\"name\":\"clock\",\"type\":\"uint128\",\"internalType\":\"Clock\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"claimDataLen\",\"inputs\":[],\"outputs\":[{\"name\":\"len_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"createdAt\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"Timestamp\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"defend\",\"inputs\":[{\"name\":\"_parentIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_claim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"extraData\",\"inputs\":[],\"outputs\":[{\"name\":\"extraData_\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"gameData\",\"inputs\":[],\"outputs\":[{\"name\":\"gameType_\",\"type\":\"uint8\",\"internalType\":\"GameType\"},{\"name\":\"rootClaim_\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"extraData_\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"gameType\",\"inputs\":[],\"outputs\":[{\"name\":\"gameType_\",\"type\":\"uint8\",\"internalType\":\"GameType\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"l1BlockNumber\",\"inputs\":[],\"outputs\":[{\"name\":\"l1BlockNumber_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"l1Head\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"l2BlockNumber\",\"inputs\":[],\"outputs\":[{\"name\":\"l2BlockNumber_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"move\",\"inputs\":[{\"name\":\"_challengeIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_claim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"},{\"name\":\"_isAttack\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"proposals\",\"inputs\":[],\"outputs\":[{\"name\":\"starting\",\"type\":\"tuple\",\"internalType\":\"structIFaultDisputeGame.OutputProposal\",\"components\":[{\"name\":\"index\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"l2BlockNumber\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"outputRoot\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}]},{\"name\":\"disputed\",\"type\":\"tuple\",\"internalType\":\"structIFaultDisputeGame.OutputProposal\",\"components\":[{\"name\":\"index\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"l2BlockNumber\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"outputRoot\",\"type\":\"bytes32\",\"internalType\":\"Hash\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"resolve\",\"inputs\":[],\"outputs\":[{\"name\":\"status_\",\"type\":\"uint8\",\"internalType\":\"enumGameStatus\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"resolveClaim\",\"inputs\":[{\"name\":\"_claimIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"rootClaim\",\"inputs\":[],\"outputs\":[{\"name\":\"rootClaim_\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"status\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumGameStatus\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"step\",\"inputs\":[{\"name\":\"_claimIndex\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_isAttack\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"_stateData\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_proof\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"Move\",\"inputs\":[{\"name\":\"parentIndex\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"},{\"name\":\"claim\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"Claim\"},{\"name\":\"claimant\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Resolved\",\"inputs\":[{\"name\":\"status\",\"type\":\"uint8\",\"indexed\":true,\"internalType\":\"enumGameStatus\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"CannotDefendRootClaim\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ClaimAlreadyExists\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ClaimAlreadyResolved\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ClockNotExpired\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ClockTimeExceeded\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"GameDepthExceeded\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"GameNotInProgress\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidParent\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"InvalidPrestate\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"L1HeadTooOld\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OutOfOrderResolution\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"UnexpectedRootClaim\",\"inputs\":[{\"name\":\"rootClaim\",\"type\":\"bytes32\",\"internalType\":\"Claim\"}]},{\"type\":\"error\",\"name\":\"ValidStep\",\"inputs\":[]}]",
	Bin: "0x6101606040523480156200001257600080fd5b5060405162002b5d38038062002b5d83398101604081905262000035916200008f565b60ff9096166101405260809490945260a0929092526001600160401b031660c0526001600160a01b0390811660e05290811661010052166101205262000133565b6001600160a01b03811681146200008c57600080fd5b50565b600080600080600080600060e0888a031215620000ab57600080fd5b875160ff81168114620000bd57600080fd5b602089015160408a015160608b015192995090975095506001600160401b0381168114620000ea57600080fd5b6080890151909450620000fd8162000076565b60a0890151909350620001108162000076565b60c0890151909250620001238162000076565b8091505092959891949750929550565b60805160a05160c05160e05161010051610120516101405161297b620001e2600039600081816105410152611c4401526000818161034901526112fd0152600081816105ba015281816110ca0152818161119e015261127701526000818161050b015281816117fb0152611adc0152600081816105ee01528181610bc60152611d3301526000818161031501528181610a0501526115fa01526000818161020a0152611756015261297b6000f3fe6080604052600436106101b75760003560e01c80638980e0cc116100ec578063c55cd0c71161008a578063d8cc1a3c11610064578063d8cc1a3c146106c1578063e66825b2146106e1578063fa24f74314610701578063fdffbb281461072557600080fd5b8063c55cd0c714610629578063c6f0308c1461063c578063cf09e0d0146106a057600080fd5b8063bbdc02db116100c6578063bbdc02db1461052d578063bcef3b551461056b578063c0c3a092146105a8578063c31b29ce146105dc57600080fd5b80638980e0cc146104a45780638b85902b146104b957806392931298146104f957600080fd5b8063529184c911610159578063609d333411610133578063609d333414610451578063632247ea146104665780636361506d146104795780638129fc1c1461048f57600080fd5b8063529184c91461033757806354fd4d501461036b57806355ef20e6146103c157600080fd5b8063298c900511610195578063298c90051461024f57806335fef5671461028f578063363cc427146102a45780634778efe81461030357600080fd5b8063200d2ed2146101bc578063266198f9146101f85780632810e1d61461023a575b600080fd5b3480156101c857600080fd5b506000546101e29068010000000000000000900460ff1681565b6040516101ef91906122c8565b60405180910390f35b34801561020457600080fd5b5061022c7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020016101ef565b34801561024657600080fd5b506101e2610738565b34801561025b57600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036040013561022c565b6102a261029d366004612309565b61089c565b005b3480156102b057600080fd5b506000546102de906901000000000000000000900473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020016101ef565b34801561030f57600080fd5b5061022c7f000000000000000000000000000000000000000000000000000000000000000081565b34801561034357600080fd5b506102de7f000000000000000000000000000000000000000000000000000000000000000081565b34801561037757600080fd5b506103b46040518060400160405280600681526020017f302e302e3133000000000000000000000000000000000000000000000000000081525081565b6040516101ef9190612396565b3480156103cd57600080fd5b5060408051606080820183526003546fffffffffffffffffffffffffffffffff808216845270010000000000000000000000000000000091829004811660208086019190915260045485870152855193840186526005548083168552929092041690820152600654928101929092526104439182565b6040516101ef9291906123b0565b34801561045d57600080fd5b506103b46108ac565b6102a2610474366004612419565b6108ba565b34801561048557600080fd5b5061022c60015481565b34801561049b57600080fd5b506102a2610ed0565b3480156104b057600080fd5b5060025461022c565b3480156104c557600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036020013561022c565b34801561050557600080fd5b506102de7f000000000000000000000000000000000000000000000000000000000000000081565b34801561053957600080fd5b5060405160ff7f00000000000000000000000000000000000000000000000000000000000000001681526020016101ef565b34801561057757600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90033561022c565b3480156105b457600080fd5b506102de7f000000000000000000000000000000000000000000000000000000000000000081565b3480156105e857600080fd5b506106107f000000000000000000000000000000000000000000000000000000000000000081565b60405167ffffffffffffffff90911681526020016101ef565b6102a2610637366004612309565b6114d4565b34801561064857600080fd5b5061065c61065736600461244e565b6114e0565b6040805163ffffffff90961686529315156020860152928401919091526fffffffffffffffffffffffffffffffff908116606084015216608082015260a0016101ef565b3480156106ac57600080fd5b506000546106109067ffffffffffffffff1681565b3480156106cd57600080fd5b506102a26106dc3660046124b0565b611551565b3480156106ed57600080fd5b506102a26106fc36600461253a565b611a7d565b34801561070d57600080fd5b50610716611c42565b6040516101ef93929190612566565b6102a261073336600461244e565b611c9f565b60008060005468010000000000000000900460ff16600281111561075e5761075e612299565b14610795576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60095460ff166107d1576040517f9a07664600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60026000815481106107e5576107e5612591565b6000918252602090912060039091020154640100000000900460ff1661080c57600261080f565b60015b6000805491925082917fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff166801000000000000000083600281111561085657610856612299565b02179055600281111561086b5761086b612299565b6040517f5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da6090600090a290565b905090565b6108a8828260006108ba565b5050565b606061089760206040611fd1565b6000805468010000000000000000900460ff1660028111156108de576108de612299565b14610915576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82158015610921575080155b15610958576040517fa42637bc00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006002848154811061096d5761096d612591565b600091825260208083206040805160a081018252600394909402909101805463ffffffff808216865264010000000090910460ff16151593850193909352600181015491840191909152600201546fffffffffffffffffffffffffffffffff80821660608501819052700100000000000000000000000000000000909204166080840152919350610a019190859061206816565b90507f0000000000000000000000000000000000000000000000000000000000000000610ac0826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff161115610b02576040517f56f57b2b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b815160009063ffffffff90811614610b62576002836000015163ffffffff1681548110610b3157610b31612591565b906000526020600020906003020160020160109054906101000a90046fffffffffffffffffffffffffffffffff1690505b608083015160009067ffffffffffffffff1667ffffffffffffffff1642610b9b846fffffffffffffffffffffffffffffffff1660401c90565b67ffffffffffffffff16610baf91906125ef565b610bb99190612607565b9050677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c1667ffffffffffffffff82161115610c2c576040517f3381d11400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000604082901b42176000888152608086901b6fffffffffffffffffffffffffffffffff8b1617602052604081209192509060008181526007602052604090205490915060ff1615610caa576040517f80497e3b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081815260076020908152604080832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001908117909155815160a08101835263ffffffff808f1682529381018581529281018d81526fffffffffffffffffffffffffffffffff808c16606084019081528982166080850190815260028054808801825599819052945160039099027f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace8101805498511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009099169a909916999099179690961790965590517f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5acf8701559351925184167001000000000000000000000000000000000292909316919091177f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ad09093019290925580548b908110610e2257610e22612591565b6000918252602080832060039092029091018054931515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff909416939093179092558a8152600890915260409020600254610e8690600190612607565b8154600181018355600092835260208320015560405133918a918c917f9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be91a4505050505050505050565b367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90033560001a6001811480610f10575060ff81166002145b610f76576040517ff40239db000000000000000000000000000000000000000000000000000000008152367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c900335600482015260240160405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000164267ffffffffffffffff161781556040805160a08101825263ffffffff81526020810192909252600291908101610ffb7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c90033590565b815260016020820152604001426fffffffffffffffffffffffffffffffff9081169091528254600181810185556000948552602080862085516003909402018054918601511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff000000000090921663ffffffff909416939093171782556040840151908201556060830151608090930151821670010000000000000000000000000000000002929091169190911760029091015573ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016637f00642061112460207ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c9003013590565b6040518263ffffffff1660e01b815260040161114291815260200190565b602060405180830381865afa15801561115f573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611183919061261e565b9050600073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001663a25ae5576111ce600185612607565b6040518263ffffffff1660e01b81526004016111ec91815260200190565b606060405180830381865afa158015611209573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061122d9190612686565b6040517fa25ae5570000000000000000000000000000000000000000000000000000000081526004810184905290915060009073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063a25ae55790602401606060405180830381865afa1580156112be573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906112e29190612686565b9050600073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000166399d548aa367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003604001356040518263ffffffff1660e01b815260040161136e91815260200190565b6040805180830381865afa15801561138a573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113ae9190612712565b905081602001516fffffffffffffffffffffffffffffffff16816020015167ffffffffffffffff161161140d576040517f13809ba500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040805160a081018252908190810180611428600189612607565b6fffffffffffffffffffffffffffffffff908116825260408881015182166020808501919091529851928101929092529183528051606081018252978216885285810151821688880152945187860152908501959095528051805181860151908716700100000000000000000000000000000000918816820217600355908401516004559084015180519481015194861694909516029290921760055591909101516006555160015550565b6108a8828260016108ba565b600281815481106114f057600080fd5b600091825260209091206003909102018054600182015460029092015463ffffffff8216935064010000000090910460ff1691906fffffffffffffffffffffffffffffffff8082169170010000000000000000000000000000000090041685565b6000805468010000000000000000900460ff16600281111561157557611575612299565b146115ac576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000600287815481106115c1576115c1612591565b6000918252602082206003919091020160028101549092506fffffffffffffffffffffffffffffffff16908715821760011b90506116207f000000000000000000000000000000000000000000000000000000000000000060016125ef565b6116bc826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff16146116fd576040517f5f53dd9800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080891561178057611721836fffffffffffffffffffffffffffffffff16612070565b67ffffffffffffffff16156117545761174b61173e600186612799565b865463ffffffff16612116565b60010154611776565b7f00000000000000000000000000000000000000000000000000000000000000005b915084905061179a565b8460010154915061179784600161173e91906127ca565b90505b600882901b60088a8a6040516117b19291906127fe565b6040518091039020901b146117f2576040517f696550ff00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081600101547f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663e14ced328c8c8c8c60006040518663ffffffff1660e01b815260040161185b959493929190612857565b6020604051808303816000875af115801561187a573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061189e919061261e565b600284810154929091149250600091611949906fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b6119e5886fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b6119ef9190612891565b6119f991906128b2565b67ffffffffffffffff161590508115158103611a41576040517ffb4e40dd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b505084547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff166401000000001790945550505050505050505050565b6000805468010000000000000000900460ff166002811115611aa157611aa1612299565b14611ad8576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16637dc0d1d06040518163ffffffff1660e01b8152600401602060405180830381865afa158015611b45573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611b699190612900565b7f52f0f3ad00000000000000000000000000000000000000000000000000000000601c8190526020869052604085905290915060008560018114611bd55760028114611bdf5760038114611be95760048114611bf35760058114611c035763ff137e656000526004601cfd5b6001549150611c0a565b6004549150611c0a565b6006549150611c0a565b60035460801c60c01b9150611c0a565b4660c01b91505b50606052600160038611811b6005031b60805260a083905260008060a4601c82865af1611c3b573d6000803e3d6000fd5b5050505050565b7f0000000000000000000000000000000000000000000000000000000000000000367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003356060611c986108ac565b9050909192565b6000805468010000000000000000900460ff166002811115611cc357611cc3612299565b14611cfa576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028281548110611d0f57611d0f612591565b60009182526020909120600260039092020190810154909150677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c1690611d7f90700100000000000000000000000000000000900467ffffffffffffffff1642612607565b6002830154611daf9190700100000000000000000000000000000000900460401c67ffffffffffffffff166125ef565b11611de6576040517ff2440b5300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600082815260086020526040902082158015611e04575060095460ff165b15611e3b576040517ff1a9458100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8054158015611e4957508215155b15611e80576040517ff1a9458100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000805b8254811015611f4e576000838281548110611ea157611ea1612591565b6000918252602080832090910154808352600890915260409091205490915015611ef7576040517f9a07664600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028281548110611f0c57611f0c612591565b600091825260209091206003909102018054909150640100000000900460ff16611f3b57600193505050611f4e565b505080611f4790612936565b9050611e84565b5082547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff16640100000000821515021783556000848152600860205260408120611f979161225f565b83600003611fcb57600980547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff001660011790555b50505050565b6060600061200884367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036125ef565b90508267ffffffffffffffff1667ffffffffffffffff81111561202d5761202d612637565b6040519080825280601f01601f191660200182016040528015612057576020820181803683370190505b509150828160208401375092915050565b151760011b90565b6000806120fd837e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b600167ffffffffffffffff919091161b90920392915050565b600080612134846fffffffffffffffffffffffffffffffff166121b3565b90506002838154811061214957612149612591565b906000526020600020906003020191505b60028201546fffffffffffffffffffffffffffffffff8281169116146121ac57815460028054909163ffffffff1690811061219757612197612591565b9060005260206000209060030201915061215a565b5092915050565b60008119600183011681612247827e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff169390931c8015179392505050565b508054600082559060005260206000209081019061227d9190612280565b50565b5b808211156122955760008155600101612281565b5090565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b6020810160038310612303577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b6000806040838503121561231c57600080fd5b50508035926020909101359150565b6000815180845260005b8181101561235157602081850181015186830182015201612335565b81811115612363576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b6020815260006123a9602083018461232b565b9392505050565b82516fffffffffffffffffffffffffffffffff90811682526020808501518216818401526040808601518185015284518316606085015290840151909116608083015282015160a082015260c081016123a9565b8035801515811461241457600080fd5b919050565b60008060006060848603121561242e57600080fd5b833592506020840135915061244560408501612404565b90509250925092565b60006020828403121561246057600080fd5b5035919050565b60008083601f84011261247957600080fd5b50813567ffffffffffffffff81111561249157600080fd5b6020830191508360208285010111156124a957600080fd5b9250929050565b600080600080600080608087890312156124c957600080fd5b863595506124d960208801612404565b9450604087013567ffffffffffffffff808211156124f657600080fd5b6125028a838b01612467565b9096509450606089013591508082111561251b57600080fd5b5061252889828a01612467565b979a9699509497509295939492505050565b60008060006060848603121561254f57600080fd5b505081359360208301359350604090920135919050565b60ff84168152826020820152606060408201526000612588606083018461232b565b95945050505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115612602576126026125c0565b500190565b600082821015612619576126196125c0565b500390565b60006020828403121561263057600080fd5b5051919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b80516fffffffffffffffffffffffffffffffff8116811461241457600080fd5b60006060828403121561269857600080fd5b6040516060810181811067ffffffffffffffff821117156126e2577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052825181526126f560208401612666565b602082015261270660408401612666565b60408201529392505050565b60006040828403121561272457600080fd5b6040516040810167ffffffffffffffff828210818311171561276f577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b816040528451835260208501519150808216821461278c57600080fd5b5060208201529392505050565b60006fffffffffffffffffffffffffffffffff838116908316818110156127c2576127c26125c0565b039392505050565b60006fffffffffffffffffffffffffffffffff8083168185168083038211156127f5576127f56125c0565b01949350505050565b8183823760009101908152919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b60608152600061286b60608301878961280e565b828103602084015261287e81868861280e565b9150508260408301529695505050505050565b600067ffffffffffffffff838116908316818110156127c2576127c26125c0565b600067ffffffffffffffff808416806128f4577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910692915050565b60006020828403121561291257600080fd5b815173ffffffffffffffffffffffffffffffffffffffff811681146123a957600080fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203612967576129676125c0565b506001019056fea164736f6c634300080f000a",
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

// AddLocalData is a paid mutator transaction binding the contract method 0xe66825b2.
//
// Solidity: function addLocalData(uint256 _ident, bytes32 _localContext, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) AddLocalData(opts *bind.TransactOpts, _ident *big.Int, _localContext [32]byte, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "addLocalData", _ident, _localContext, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0xe66825b2.
//
// Solidity: function addLocalData(uint256 _ident, bytes32 _localContext, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameSession) AddLocalData(_ident *big.Int, _localContext [32]byte, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _localContext, _partOffset)
}

// AddLocalData is a paid mutator transaction binding the contract method 0xe66825b2.
//
// Solidity: function addLocalData(uint256 _ident, bytes32 _localContext, uint256 _partOffset) returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) AddLocalData(_ident *big.Int, _localContext [32]byte, _partOffset *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.AddLocalData(&_FaultDisputeGame.TransactOpts, _ident, _localContext, _partOffset)
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

// ResolveClaim is a paid mutator transaction binding the contract method 0xfdffbb28.
//
// Solidity: function resolveClaim(uint256 _claimIndex) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactor) ResolveClaim(opts *bind.TransactOpts, _claimIndex *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.contract.Transact(opts, "resolveClaim", _claimIndex)
}

// ResolveClaim is a paid mutator transaction binding the contract method 0xfdffbb28.
//
// Solidity: function resolveClaim(uint256 _claimIndex) payable returns()
func (_FaultDisputeGame *FaultDisputeGameSession) ResolveClaim(_claimIndex *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ResolveClaim(&_FaultDisputeGame.TransactOpts, _claimIndex)
}

// ResolveClaim is a paid mutator transaction binding the contract method 0xfdffbb28.
//
// Solidity: function resolveClaim(uint256 _claimIndex) payable returns()
func (_FaultDisputeGame *FaultDisputeGameTransactorSession) ResolveClaim(_claimIndex *big.Int) (*types.Transaction, error) {
	return _FaultDisputeGame.Contract.ResolveClaim(&_FaultDisputeGame.TransactOpts, _claimIndex)
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

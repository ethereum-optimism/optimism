// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/solc"
)

const FaultDisputeGameStorageLayoutJSON = "{\"storage\":[{\"astId\":1000,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"createdAt\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_userDefinedValueType(Timestamp)1019\"},{\"astId\":1001,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"status\",\"offset\":8,\"slot\":\"0\",\"type\":\"t_enum(GameStatus)1010\"},{\"astId\":1002,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"bondManager\",\"offset\":9,\"slot\":\"0\",\"type\":\"t_contract(IBondManager)1009\"},{\"astId\":1003,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"l1Head\",\"offset\":0,\"slot\":\"1\",\"type\":\"t_userDefinedValueType(Hash)1017\"},{\"astId\":1004,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"claimData\",\"offset\":0,\"slot\":\"2\",\"type\":\"t_array(t_struct(ClaimData)1011_storage)dyn_storage\"},{\"astId\":1005,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"proposals\",\"offset\":0,\"slot\":\"3\",\"type\":\"t_struct(OutputProposals)1013_storage\"},{\"astId\":1006,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"claims\",\"offset\":0,\"slot\":\"7\",\"type\":\"t_mapping(t_userDefinedValueType(ClaimHash)1015,t_bool)\"},{\"astId\":1007,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"subgames\",\"offset\":0,\"slot\":\"8\",\"type\":\"t_mapping(t_uint256,t_array(t_uint256)dyn_storage)\"},{\"astId\":1008,\"contract\":\"src/dispute/FaultDisputeGame.sol:FaultDisputeGame\",\"label\":\"subgameAtRootResolved\",\"offset\":0,\"slot\":\"9\",\"type\":\"t_bool\"}],\"types\":{\"t_array(t_struct(ClaimData)1011_storage)dyn_storage\":{\"encoding\":\"dynamic_array\",\"label\":\"struct IFaultDisputeGame.ClaimData[]\",\"numberOfBytes\":\"32\",\"base\":\"t_struct(ClaimData)1011_storage\"},\"t_array(t_uint256)dyn_storage\":{\"encoding\":\"dynamic_array\",\"label\":\"uint256[]\",\"numberOfBytes\":\"32\",\"base\":\"t_uint256\"},\"t_bool\":{\"encoding\":\"inplace\",\"label\":\"bool\",\"numberOfBytes\":\"1\"},\"t_contract(IBondManager)1009\":{\"encoding\":\"inplace\",\"label\":\"contract IBondManager\",\"numberOfBytes\":\"20\"},\"t_enum(GameStatus)1010\":{\"encoding\":\"inplace\",\"label\":\"enum GameStatus\",\"numberOfBytes\":\"1\"},\"t_mapping(t_uint256,t_array(t_uint256)dyn_storage)\":{\"encoding\":\"mapping\",\"label\":\"mapping(uint256 =\u003e uint256[])\",\"numberOfBytes\":\"32\",\"key\":\"t_uint256\",\"value\":\"t_array(t_uint256)dyn_storage\"},\"t_mapping(t_userDefinedValueType(ClaimHash)1015,t_bool)\":{\"encoding\":\"mapping\",\"label\":\"mapping(ClaimHash =\u003e bool)\",\"numberOfBytes\":\"32\",\"key\":\"t_userDefinedValueType(ClaimHash)1015\",\"value\":\"t_bool\"},\"t_struct(ClaimData)1011_storage\":{\"encoding\":\"inplace\",\"label\":\"struct IFaultDisputeGame.ClaimData\",\"numberOfBytes\":\"96\"},\"t_struct(OutputProposal)1012_storage\":{\"encoding\":\"inplace\",\"label\":\"struct IFaultDisputeGame.OutputProposal\",\"numberOfBytes\":\"64\"},\"t_struct(OutputProposals)1013_storage\":{\"encoding\":\"inplace\",\"label\":\"struct IFaultDisputeGame.OutputProposals\",\"numberOfBytes\":\"128\"},\"t_uint128\":{\"encoding\":\"inplace\",\"label\":\"uint128\",\"numberOfBytes\":\"16\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"},\"t_uint32\":{\"encoding\":\"inplace\",\"label\":\"uint32\",\"numberOfBytes\":\"4\"},\"t_userDefinedValueType(Claim)1014\":{\"encoding\":\"inplace\",\"label\":\"Claim\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(ClaimHash)1015\":{\"encoding\":\"inplace\",\"label\":\"ClaimHash\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Clock)1016\":{\"encoding\":\"inplace\",\"label\":\"Clock\",\"numberOfBytes\":\"16\"},\"t_userDefinedValueType(Hash)1017\":{\"encoding\":\"inplace\",\"label\":\"Hash\",\"numberOfBytes\":\"32\"},\"t_userDefinedValueType(Position)1018\":{\"encoding\":\"inplace\",\"label\":\"Position\",\"numberOfBytes\":\"16\"},\"t_userDefinedValueType(Timestamp)1019\":{\"encoding\":\"inplace\",\"label\":\"Timestamp\",\"numberOfBytes\":\"8\"}}}"

var FaultDisputeGameStorageLayout = new(solc.StorageLayout)

var FaultDisputeGameDeployedBin = "0x6080604052600436106101b75760003560e01c80638129fc1c116100ec578063c31b29ce1161008a578063cf09e0d011610064578063cf09e0d0146106c0578063d8cc1a3c146106e1578063fa24f74314610701578063fdffbb281461072557600080fd5b8063c31b29ce146105fc578063c55cd0c714610649578063c6f0308c1461065c57600080fd5b806392931298116100c65780639293129814610519578063bbdc02db1461054d578063bcef3b551461058b578063c0c3a092146105c857600080fd5b80638129fc1c146104af5780638980e0cc146104c45780638b85902b146104d957600080fd5b80634778efe81161015957806355ef20e61161013357806355ef20e6146103e1578063609d333414610471578063632247ea146104865780636361506d1461049957600080fd5b80634778efe814610323578063529184c91461035757806354fd4d501461038b57600080fd5b80632810e1d6116101955780632810e1d61461025c578063298c90051461027157806335fef567146102b1578063363cc427146102c457600080fd5b80631e27052a146101bc578063200d2ed2146101de578063266198f91461021a575b600080fd5b3480156101c857600080fd5b506101dc6101d736600461228f565b610738565b005b3480156101ea57600080fd5b506000546102049068010000000000000000900460ff1681565b60405161021191906122e0565b60405180910390f35b34801561022657600080fd5b5061024e7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610211565b34801561026857600080fd5b506102046108f7565b34801561027d57600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036040013561024e565b6101dc6102bf36600461228f565b610a5b565b3480156102d057600080fd5b506000546102fe906901000000000000000000900473ffffffffffffffffffffffffffffffffffffffff1681565b60405173ffffffffffffffffffffffffffffffffffffffff9091168152602001610211565b34801561032f57600080fd5b5061024e7f000000000000000000000000000000000000000000000000000000000000000081565b34801561036357600080fd5b506102fe7f000000000000000000000000000000000000000000000000000000000000000081565b34801561039757600080fd5b506103d46040518060400160405280600681526020017f302e302e3130000000000000000000000000000000000000000000000000000081525081565b604051610211919061238c565b3480156103ed57600080fd5b5060408051606080820183526003546fffffffffffffffffffffffffffffffff808216845270010000000000000000000000000000000091829004811660208086019190915260045485870152855193840186526005548083168552929092041690820152600654928101929092526104639182565b6040516102119291906123a6565b34801561047d57600080fd5b506103d4610a6b565b6101dc61049436600461240f565b610a79565b3480156104a557600080fd5b5061024e60015481565b3480156104bb57600080fd5b506101dc61108f565b3480156104d057600080fd5b5060025461024e565b3480156104e557600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036020013561024e565b34801561052557600080fd5b506102fe7f000000000000000000000000000000000000000000000000000000000000000081565b34801561055957600080fd5b5060405160ff7f0000000000000000000000000000000000000000000000000000000000000000168152602001610211565b34801561059757600080fd5b50367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90033561024e565b3480156105d457600080fd5b506102fe7f000000000000000000000000000000000000000000000000000000000000000081565b34801561060857600080fd5b506106307f000000000000000000000000000000000000000000000000000000000000000081565b60405167ffffffffffffffff9091168152602001610211565b6101dc61065736600461228f565b611693565b34801561066857600080fd5b5061067c610677366004612444565b61169f565b6040805163ffffffff90961686529315156020860152928401919091526fffffffffffffffffffffffffffffffff908116606084015216608082015260a001610211565b3480156106cc57600080fd5b506000546106309067ffffffffffffffff1681565b3480156106ed57600080fd5b506101dc6106fc3660046124a6565b611710565b34801561070d57600080fd5b50610716611c39565b60405161021193929190612530565b6101dc610733366004612444565b611c96565b6000805468010000000000000000900460ff16600281111561075c5761075c6122b1565b14610793576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60007f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16637dc0d1d06040518163ffffffff1660e01b8152600401602060405180830381865afa158015610800573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610824919061255b565b7f9a1f5e7f00000000000000000000000000000000000000000000000000000000601c81905260208590529091506000846001811461088b5760028114610895576003811461089f57600481146108a957600581146108b95763ff137e656000526004601cfd5b60015491506108c0565b60045491506108c0565b60065491506108c0565b60035460801c60c01b91506108c0565b4660c01b91505b50604052600160038511811b6005031b60605260808390526000806084601c82865af16108f1573d6000803e3d6000fd5b50505050565b60008060005468010000000000000000900460ff16600281111561091d5761091d6122b1565b14610954576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60095460ff16610990576040517f9a07664600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60026000815481106109a4576109a4612591565b6000918252602090912060039091020154640100000000900460ff166109cb5760026109ce565b60015b6000805491925082917fffffffffffffffffffffffffffffffffffffffffffffff00ffffffffffffffff1668010000000000000000836002811115610a1557610a156122b1565b021790556002811115610a2a57610a2a6122b1565b6040517f5e186f09b9c93491f14e277eea7faa5de6a2d4bda75a79af7a3684fbfb42da6090600090a290565b905090565b610a6782826000610a79565b5050565b6060610a5660206040611fc7565b6000805468010000000000000000900460ff166002811115610a9d57610a9d6122b1565b14610ad4576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b82158015610ae0575080155b15610b17576040517fa42637bc00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028481548110610b2c57610b2c612591565b600091825260208083206040805160a081018252600394909402909101805463ffffffff808216865264010000000090910460ff16151593850193909352600181015491840191909152600201546fffffffffffffffffffffffffffffffff80821660608501819052700100000000000000000000000000000000909204166080840152919350610bc09190859061205e16565b90507f0000000000000000000000000000000000000000000000000000000000000000610c7f826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff161115610cc1576040517f56f57b2b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b815160009063ffffffff90811614610d21576002836000015163ffffffff1681548110610cf057610cf0612591565b906000526020600020906003020160020160109054906101000a90046fffffffffffffffffffffffffffffffff1690505b608083015160009067ffffffffffffffff1667ffffffffffffffff1642610d5a846fffffffffffffffffffffffffffffffff1660401c90565b67ffffffffffffffff16610d6e91906125ef565b610d789190612607565b9050677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c1667ffffffffffffffff82161115610deb576040517f3381d11400000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000604082901b42176000888152608086901b6fffffffffffffffffffffffffffffffff8b1617602052604081209192509060008181526007602052604090205490915060ff1615610e69576040517f80497e3b00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081815260076020908152604080832080547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00166001908117909155815160a08101835263ffffffff808f1682529381018581529281018d81526fffffffffffffffffffffffffffffffff808c16606084019081528982166080850190815260028054808801825599819052945160039099027f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace8101805498511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000009099169a909916999099179690961790965590517f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5acf8701559351925184167001000000000000000000000000000000000292909316919091177f405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ad09093019290925580548b908110610fe157610fe1612591565b6000918252602080832060039092029091018054931515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff909416939093179092558a815260089091526040902060025461104590600190612607565b8154600181018355600092835260208320015560405133918a918c917f9b3245740ec3b155098a55be84957a4da13eaf7f14a8bc6f53126c0b9350f2be91a4505050505050505050565b367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90033560001a60018114806110cf575060ff81166002145b611135576040517ff40239db000000000000000000000000000000000000000000000000000000008152367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c900335600482015260240160405180910390fd5b600080547fffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000164267ffffffffffffffff161781556040805160a08101825263ffffffff815260208101929092526002919081016111ba7ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c90033590565b815260016020820152604001426fffffffffffffffffffffffffffffffff9081169091528254600181810185556000948552602080862085516003909402018054918601511515640100000000027fffffffffffffffffffffffffffffffffffffffffffffffffffffff000000000090921663ffffffff909416939093171782556040840151908201556060830151608090930151821670010000000000000000000000000000000002929091169190911760029091015573ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016637f0064206112e360207ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe369081013560f01c9003013590565b6040518263ffffffff1660e01b815260040161130191815260200190565b602060405180830381865afa15801561131e573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611342919061261e565b9050600073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001663a25ae55761138d600185612607565b6040518263ffffffff1660e01b81526004016113ab91815260200190565b606060405180830381865afa1580156113c8573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906113ec9190612686565b6040517fa25ae5570000000000000000000000000000000000000000000000000000000081526004810184905290915060009073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000169063a25ae55790602401606060405180830381865afa15801561147d573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906114a19190612686565b9050600073ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000166399d548aa367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003604001356040518263ffffffff1660e01b815260040161152d91815260200190565b6040805180830381865afa158015611549573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061156d9190612712565b905081602001516fffffffffffffffffffffffffffffffff16816020015167ffffffffffffffff16116115cc576040517f13809ba500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6040805160a0810182529081908101806115e7600189612607565b6fffffffffffffffffffffffffffffffff908116825260408881015182166020808501919091529851928101929092529183528051606081018252978216885285810151821688880152945187860152908501959095528051805181860151908716700100000000000000000000000000000000918816820217600355908401516004559084015180519481015194861694909516029290921760055591909101516006555160015550565b610a6782826001610a79565b600281815481106116af57600080fd5b600091825260209091206003909102018054600182015460029092015463ffffffff8216935064010000000090910460ff1691906fffffffffffffffffffffffffffffffff8082169170010000000000000000000000000000000090041685565b6000805468010000000000000000900460ff166002811115611734576117346122b1565b1461176b576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60006002878154811061178057611780612591565b6000918252602082206003919091020160028101549092506fffffffffffffffffffffffffffffffff16908715821760011b90506117df7f000000000000000000000000000000000000000000000000000000000000000060016125ef565b61187b826fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff16146118bc576040517f5f53dd9800000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600080891561193f576118e0836fffffffffffffffffffffffffffffffff16612066565b67ffffffffffffffff16156119135761190a6118fd600186612799565b865463ffffffff1661210c565b60010154611935565b7f00000000000000000000000000000000000000000000000000000000000000005b9150849050611959565b846001015491506119568460016118fd91906127ca565b90505b600882901b60088a8a6040516119709291906127fe565b6040518091039020901b146119b1576040517f696550ff00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600081600101547f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663f8e0cb968c8c8c8c6040518563ffffffff1660e01b8152600401611a179493929190612857565b6020604051808303816000875af1158015611a36573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190611a5a919061261e565b600284810154929091149250600091611b05906fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b611ba1886fffffffffffffffffffffffffffffffff167e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b611bab9190612889565b611bb591906128aa565b67ffffffffffffffff161590508115158103611bfd576040517ffb4e40dd00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b505084547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff166401000000001790945550505050505050505050565b7f0000000000000000000000000000000000000000000000000000000000000000367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c9003356060611c8f610a6b565b9050909192565b6000805468010000000000000000900460ff166002811115611cba57611cba6122b1565b14611cf1576040517f67fe195000000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028281548110611d0657611d06612591565b60009182526020909120600260039092020190810154909150677fffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000060011c1690611d7690700100000000000000000000000000000000900467ffffffffffffffff1642612607565b6002830154611da69190700100000000000000000000000000000000900460401c67ffffffffffffffff166125ef565b11611ddd576040517ff2440b5300000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600082815260086020526040902082158015611dfb575060095460ff165b15611e32576040517ff1a9458100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b8054158015611e4057508215155b15611e77576040517ff1a9458100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6000805b8254811015611f45576000838281548110611e9857611e98612591565b6000918252602080832090910154808352600890915260409091205490915015611eee576040517f9a07664600000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b600060028281548110611f0357611f03612591565b600091825260209091206003909102018054909150640100000000900460ff16611f3257600193505050611f45565b505080611f3e906128f8565b9050611e7b565b5082547fffffffffffffffffffffffffffffffffffffffffffffffffffffff00ffffffff16640100000000821515021783556000848152600860205260408120611f8e91612255565b836000036108f157600980547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0016600117905550505050565b60606000611ffe84367ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe81013560f01c90036125ef565b90508267ffffffffffffffff1667ffffffffffffffff81111561202357612023612637565b6040519080825280601f01601f19166020018201604052801561204d576020820181803683370190505b509150828160208401375092915050565b151760011b90565b6000806120f3837e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b600167ffffffffffffffff919091161b90920392915050565b60008061212a846fffffffffffffffffffffffffffffffff166121a9565b90506002838154811061213f5761213f612591565b906000526020600020906003020191505b60028201546fffffffffffffffffffffffffffffffff8281169116146121a257815460028054909163ffffffff1690811061218d5761218d612591565b90600052602060002090600302019150612150565b5092915050565b6000811960018301168161223d827e09010a0d15021d0b0e10121619031e080c141c0f111807131b17061a05041f7f07c4acdd0000000000000000000000000000000000000000000000000000000067ffffffffffffffff831160061b83811c63ffffffff1060051b1792831c600181901c17600281901c17600481901c17600881901c17601081901c170260fb1c1a1790565b67ffffffffffffffff169390931c8015179392505050565b50805460008255906000526020600020908101906122739190612276565b50565b5b8082111561228b5760008155600101612277565b5090565b600080604083850312156122a257600080fd5b50508035926020909101359150565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b602081016003831061231b577f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b91905290565b6000815180845260005b818110156123475760208185018101518683018201520161232b565b81811115612359576000602083870101525b50601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b60208152600061239f6020830184612321565b9392505050565b82516fffffffffffffffffffffffffffffffff90811682526020808501518216818401526040808601518185015284518316606085015290840151909116608083015282015160a082015260c0810161239f565b8035801515811461240a57600080fd5b919050565b60008060006060848603121561242457600080fd5b833592506020840135915061243b604085016123fa565b90509250925092565b60006020828403121561245657600080fd5b5035919050565b60008083601f84011261246f57600080fd5b50813567ffffffffffffffff81111561248757600080fd5b60208301915083602082850101111561249f57600080fd5b9250929050565b600080600080600080608087890312156124bf57600080fd5b863595506124cf602088016123fa565b9450604087013567ffffffffffffffff808211156124ec57600080fd5b6124f88a838b0161245d565b9096509450606089013591508082111561251157600080fd5b5061251e89828a0161245d565b979a9699509497509295939492505050565b60ff841681528260208201526060604082015260006125526060830184612321565b95945050505050565b60006020828403121561256d57600080fd5b815173ffffffffffffffffffffffffffffffffffffffff8116811461239f57600080fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60008219821115612602576126026125c0565b500190565b600082821015612619576126196125c0565b500390565b60006020828403121561263057600080fd5b5051919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b80516fffffffffffffffffffffffffffffffff8116811461240a57600080fd5b60006060828403121561269857600080fd5b6040516060810181811067ffffffffffffffff821117156126e2577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604052825181526126f560208401612666565b602082015261270660408401612666565b60408201529392505050565b60006040828403121561272457600080fd5b6040516040810167ffffffffffffffff828210818311171561276f577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b816040528451835260208501519150808216821461278c57600080fd5b5060208201529392505050565b60006fffffffffffffffffffffffffffffffff838116908316818110156127c2576127c26125c0565b039392505050565b60006fffffffffffffffffffffffffffffffff8083168185168083038211156127f5576127f56125c0565b01949350505050565b8183823760009101908152919050565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b60408152600061286b60408301868861280e565b828103602084015261287e81858761280e565b979650505050505050565b600067ffffffffffffffff838116908316818110156127c2576127c26125c0565b600067ffffffffffffffff808416806128ec577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b92169190910692915050565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203612929576129296125c0565b506001019056fea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(FaultDisputeGameStorageLayoutJSON), FaultDisputeGameStorageLayout); err != nil {
		panic(err)
	}

	layouts["FaultDisputeGame"] = FaultDisputeGameStorageLayout
	deployedBytecodes["FaultDisputeGame"] = FaultDisputeGameDeployedBin
}

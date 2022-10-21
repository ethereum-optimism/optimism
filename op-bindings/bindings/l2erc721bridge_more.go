// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"encoding/json"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

const L2ERC721BridgeStorageLayoutJSON = "{\"storage\":[{\"astId\":26242,\"contract\":\"contracts/L2/L2ERC721Bridge.sol:L2ERC721Bridge\",\"label\":\"__gap\",\"offset\":0,\"slot\":\"0\",\"type\":\"t_array(t_uint256)49_storage\"}],\"types\":{\"t_array(t_uint256)49_storage\":{\"encoding\":\"inplace\",\"label\":\"uint256[49]\",\"numberOfBytes\":\"1568\"},\"t_uint256\":{\"encoding\":\"inplace\",\"label\":\"uint256\",\"numberOfBytes\":\"32\"}}}"

var L2ERC721BridgeStorageLayout = new(solc.StorageLayout)

var L2ERC721BridgeDeployedBin = "0x608060405234801561001057600080fd5b50600436106100725760003560e01c8063761f449311610050578063761f4493146100f2578063aa55745214610105578063c89701a21461011857600080fd5b80633687011a146100775780633cb747bf1461008c57806354fd4d50146100dd575b600080fd5b61008a61008536600461116a565b61013f565b005b6100b37f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b6100e56101eb565b6040516100d49190611267565b61008a61010036600461127a565b61028e565b61008a610113366004611312565b6107f5565b6100b37f000000000000000000000000000000000000000000000000000000000000000081565b333b156101d3576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602d60248201527f4552433732314272696467653a206163636f756e74206973206e6f742065787460448201527f65726e616c6c79206f776e65640000000000000000000000000000000000000060648201526084015b60405180910390fd5b6101e386863333888888886108b1565b505050505050565b60606102167f0000000000000000000000000000000000000000000000000000000000000000610e4f565b61023f7f0000000000000000000000000000000000000000000000000000000000000000610e4f565b6102687f0000000000000000000000000000000000000000000000000000000000000000610e4f565b60405160200161027a93929190611389565b604051602081830303815290604052905090565b3373ffffffffffffffffffffffffffffffffffffffff7f0000000000000000000000000000000000000000000000000000000000000000161480156103ac57507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff167f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff16636e296e456040518163ffffffff1660e01b8152600401602060405180830381865afa158015610370573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061039491906113ff565b73ffffffffffffffffffffffffffffffffffffffff16145b610438576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603f60248201527f4552433732314272696467653a2066756e6374696f6e2063616e206f6e6c792060448201527f62652063616c6c65642066726f6d20746865206f74686572206272696467650060648201526084016101ca565b3073ffffffffffffffffffffffffffffffffffffffff8816036104dd576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602a60248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e2063616e6e60448201527f6f742062652073656c660000000000000000000000000000000000000000000060648201526084016101ca565b610507877fe49bc7f800000000000000000000000000000000000000000000000000000000610f8c565b610593576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603660248201527f4c324552433732314272696467653a206c6f63616c20746f6b656e20696e746560448201527f7266616365206973206e6f7420636f6d706c69616e740000000000000000000060648201526084016101ca565b8673ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa1580156105de573d6000803e3d6000fd5b505050506040513d601f19601f8201168201806040525081019061060291906113ff565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff16146106e2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152604b60248201527f4c324552433732314272696467653a2077726f6e672072656d6f746520746f6b60448201527f656e20666f72204f7074696d69736d204d696e7461626c65204552433732312060648201527f6c6f63616c20746f6b656e000000000000000000000000000000000000000000608482015260a4016101ca565b6040517fa144819400000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff85811660048301526024820185905288169063a144819490604401600060405180830381600087803b15801561075257600080fd5b505af1158015610766573d6000803e3d6000fd5b505050508473ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff168873ffffffffffffffffffffffffffffffffffffffff167f1f39bf6707b5d608453e0ae4c067b562bcc4c85c0f562ef5d2c774d2e7f131ac878787876040516107e49493929190611465565b60405180910390a450505050505050565b73ffffffffffffffffffffffffffffffffffffffff8516610898576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f4552433732314272696467653a206e667420726563697069656e742063616e6e60448201527f6f7420626520616464726573732830290000000000000000000000000000000060648201526084016101ca565b6108a887873388888888886108b1565b50505050505050565b73ffffffffffffffffffffffffffffffffffffffff8716610954576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602f60248201527f4552433732314272696467653a2072656d6f746520746f6b656e2063616e6e6f60448201527f742062652061646472657373283029000000000000000000000000000000000060648201526084016101ca565b6040517f6352211e0000000000000000000000000000000000000000000000000000000081526004810185905273ffffffffffffffffffffffffffffffffffffffff891690636352211e90602401602060405180830381865afa1580156109bf573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906109e391906113ff565b73ffffffffffffffffffffffffffffffffffffffff168673ffffffffffffffffffffffffffffffffffffffff1614610a9d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602e60248201527f5769746864726177616c206973206e6f74206265696e6720696e69746961746560448201527f64206279204e4654206f776e657200000000000000000000000000000000000060648201526084016101ca565b60008873ffffffffffffffffffffffffffffffffffffffff1663d6c0b2c46040518163ffffffff1660e01b8152600401602060405180830381865afa158015610aea573d6000803e3d6000fd5b505050506040513d601f19601f82011682018060405250810190610b0e91906113ff565b90508773ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1614610bcb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f4c324552433732314272696467653a2072656d6f746520746f6b656e20646f6560448201527f73206e6f74206d6174636820676976656e2076616c756500000000000000000060648201526084016101ca565b6040517f9dc29fac00000000000000000000000000000000000000000000000000000000815273ffffffffffffffffffffffffffffffffffffffff8881166004830152602482018790528a1690639dc29fac90604401600060405180830381600087803b158015610c3b57600080fd5b505af1158015610c4f573d6000803e3d6000fd5b50505050600063761f449360e01b828b8a8a8a8989604051602401610c7a97969594939291906114a5565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181529181526020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167fffffffff000000000000000000000000000000000000000000000000000000009094169390931790925290517f3dbb202b00000000000000000000000000000000000000000000000000000000815290915073ffffffffffffffffffffffffffffffffffffffff7f00000000000000000000000000000000000000000000000000000000000000001690633dbb202b90610d8f907f00000000000000000000000000000000000000000000000000000000000000009085908a90600401611502565b600060405180830381600087803b158015610da957600080fd5b505af1158015610dbd573d6000803e3d6000fd5b505050508773ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff168b73ffffffffffffffffffffffffffffffffffffffff167fb7460e2a880f256ebef3406116ff3eee0cee51ebccdc2a40698f87ebb2e9c1a58a8a8989604051610e3b9493929190611465565b60405180910390a450505050505050505050565b606081600003610e9257505060408051808201909152600181527f3000000000000000000000000000000000000000000000000000000000000000602082015290565b8160005b8115610ebc5780610ea681611576565b9150610eb59050600a836115dd565b9150610e96565b60008167ffffffffffffffff811115610ed757610ed76115f1565b6040519080825280601f01601f191660200182016040528015610f01576020820181803683370190505b5090505b8415610f8457610f16600183611620565b9150610f23600a86611637565b610f2e90603061164b565b60f81b818381518110610f4357610f43611663565b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350610f7d600a866115dd565b9450610f05565b949350505050565b6000610f9783610faf565b8015610fa85750610fa88383611014565b9392505050565b6000610fdb827f01ffc9a700000000000000000000000000000000000000000000000000000000611014565b801561100e575061100c827fffffffff00000000000000000000000000000000000000000000000000000000611014565b155b92915050565b604080517fffffffff000000000000000000000000000000000000000000000000000000008316602480830191909152825180830390910181526044909101909152602080820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff167f01ffc9a700000000000000000000000000000000000000000000000000000000178152825160009392849283928392918391908a617530fa92503d915060005190508280156110cc575060208210155b80156110d85750600081115b979650505050505050565b73ffffffffffffffffffffffffffffffffffffffff8116811461110557600080fd5b50565b803563ffffffff8116811461111c57600080fd5b919050565b60008083601f84011261113357600080fd5b50813567ffffffffffffffff81111561114b57600080fd5b60208301915083602082850101111561116357600080fd5b9250929050565b60008060008060008060a0878903121561118357600080fd5b863561118e816110e3565b9550602087013561119e816110e3565b9450604087013593506111b360608801611108565b9250608087013567ffffffffffffffff8111156111cf57600080fd5b6111db89828a01611121565b979a9699509497509295939492505050565b60005b838110156112085781810151838201526020016111f0565b83811115611217576000848401525b50505050565b600081518084526112358160208601602086016111ed565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169290920160200192915050565b602081526000610fa8602083018461121d565b600080600080600080600060c0888a03121561129557600080fd5b87356112a0816110e3565b965060208801356112b0816110e3565b955060408801356112c0816110e3565b945060608801356112d0816110e3565b93506080880135925060a088013567ffffffffffffffff8111156112f357600080fd5b6112ff8a828b01611121565b989b979a50959850939692959293505050565b600080600080600080600060c0888a03121561132d57600080fd5b8735611338816110e3565b96506020880135611348816110e3565b95506040880135611358816110e3565b94506060880135935061136d60808901611108565b925060a088013567ffffffffffffffff8111156112f357600080fd5b6000845161139b8184602089016111ed565b80830190507f2e0000000000000000000000000000000000000000000000000000000000000080825285516113d7816001850160208a016111ed565b600192019182015283516113f28160028401602088016111ed565b0160020195945050505050565b60006020828403121561141157600080fd5b8151610fa8816110e3565b8183528181602085013750600060208284010152600060207fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f840116840101905092915050565b73ffffffffffffffffffffffffffffffffffffffff8516815283602082015260606040820152600061149b60608301848661141c565b9695505050505050565b600073ffffffffffffffffffffffffffffffffffffffff808a1683528089166020840152808816604084015280871660608401525084608083015260c060a08301526114f560c08301848661141c565b9998505050505050505050565b73ffffffffffffffffffffffffffffffffffffffff84168152606060208201526000611531606083018561121d565b905063ffffffff83166040830152949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036115a7576115a7611547565b5060010190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b6000826115ec576115ec6115ae565b500490565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008282101561163257611632611547565b500390565b600082611646576116466115ae565b500690565b6000821982111561165e5761165e611547565b500190565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fdfea164736f6c634300080f000a"

func init() {
	if err := json.Unmarshal([]byte(L2ERC721BridgeStorageLayoutJSON), L2ERC721BridgeStorageLayout); err != nil {
		panic(err)
	}

	layouts["L2ERC721Bridge"] = L2ERC721BridgeStorageLayout
	deployedBytecodes["L2ERC721Bridge"] = L2ERC721BridgeDeployedBin
}

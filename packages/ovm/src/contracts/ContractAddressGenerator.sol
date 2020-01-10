pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {RLPEncode} from "./RLPEncode.sol";

/**
 * @title ContractAddressGenerator
 * @notice Libary contract which generates CREATE & CREATE2 addresses.
 *         This is used in Rollup to make sure we have address parity with Ethereum mainchain.
 */
contract ContractAddressGenerator { // TODO: Make this a library
    // RLP encoding library
    RLPEncode rlp;
    /***************
    * Constructor *
    **************/
    constructor(address _rlpEncodeAddress) public {
        rlp = RLPEncode(_rlpEncodeAddress);
    }
    /**
     * @notice Generate a contract address using CREATE.
     * @param _origin The address of the contract which is calling CREATE.
     * @param _nonce The contract nonce of the origin contract (incremented each time CREATE is called).
     */
    function getAddressFromCREATE(address _origin, uint _nonce) public view returns(address) { //TODO add view/pure back in
        // RLP encode the origin address
        bytes memory encodedOrigin = rlp.encodeAddress(_origin);
        // RLP encode the contract nonce
        bytes memory encodedNonce = rlp.encodeUint(_nonce);
        //create a list consisting of the address and nonce
        bytes[] memory list = new bytes[](2);
        list[0] = encodedOrigin;
        list[1] = encodedNonce;
        // RLP encode the list
        bytes memory encodedList = rlp.encodeList(list);
        // hash the encoded list
        bytes32 encodedListHash = keccak256(encodedList);
        // return an address from the last 20 bytes of the hash
        return address(bytes20(uint160(uint256(encodedListHash))));
    }

    /**
     * @notice Generate a contract address using CREATE2.
     * @param _origin The address of the contract which is calling CREATE2.
     * @param _salt A salt which can be any 32 byte value -- this allows you to deploy
     *              the same initcode twice with different addresses.
     * @param _ovmInitcode The initcode for the contract we are CREATE2ing.
     */
    function getAddressFromCREATE2(address _origin, bytes32 _salt, bytes memory _ovmInitcode) public pure returns(address) {
        return address(bytes20(keccak256(abi.encodePacked(byte(0xff), _origin, _salt, keccak256(_ovmInitcode)))));
    }

}

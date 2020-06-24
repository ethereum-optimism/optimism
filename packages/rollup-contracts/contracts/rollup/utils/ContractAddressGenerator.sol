pragma solidity ^0.5.0;

/* Internal Imports */
import { RLPWriter } from "./RLPWriter.sol";

/**
 * @title ContractAddressGenerator
 * @notice Libary contract which generates CREATE & CREATE2 addresses.
 *         This is used in Rollup to make sure we have address parity with the
 *         Ethereum mainchain.
 */
contract ContractAddressGenerator {
    RLPWriter rlp;

    constructor() public {
        rlp = new RLPWriter();
    }

    /**
     * @notice Generate a contract address using CREATE.
     * @param _origin The address of the contract which is calling CREATE.
     * @param _nonce The contract nonce of the origin contract (incremented
     *               each time CREATE is called).
     * @return Address of the contract to be created.
     */
    function getAddressFromCREATE(address _origin, uint _nonce) public view returns (address) {
        // Create a list of RLP encoded parameters.
        bytes[] memory list = new bytes[](2);
        list[0] = rlp.encodeAddress(_origin);
        list[1] = rlp.encodeUint(_nonce);

        // RLP encode the list itself.
        bytes memory encodedList = rlp.encodeList(list);

        // Return an address from the hash of the encoded list.
        return getAddressFromHash(keccak256(encodedList));
    }

    /**
     * @notice Generate a contract address using CREATE2.
     * @param _origin The address of the contract which is calling CREATE2.
     * @param _salt A salt which can be any 32 byte value -- this allows you to deploy
     *              the same initcode twice with different addresses.
     * @param _ovmInitcode The initcode for the contract we are CREATE2ing.
     * @return Address of the contract to be created.
     */
    function getAddressFromCREATE2(
        address _origin,
        bytes32 _salt,
        bytes memory _ovmInitcode
    ) public pure returns (address) {
        // Hash all of the parameters together.
        bytes32 hashedData = keccak256(abi.encodePacked(
            byte(0xff),
            _origin,
            _salt,
            keccak256(_ovmInitcode)
        ));

        return getAddressFromHash(hashedData);
    }

    /**
     * @dev Determines an address from a 32 byte hash. Since addresses are only
     *      20 bytes, we need to retrieve the last 20 bytes from the original
     *      hash. Converting to uint256 and then uint160 gives us these bytes.
     * @param _hash Hash to convert to an address.
     * @return Hash converted to an address.
     */
    function getAddressFromHash(bytes32 _hash) internal pure returns (address) {
        return address(bytes20(uint160(uint256(_hash))));
    }
}

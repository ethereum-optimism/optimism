pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title ContractAddressGenerator
 * @notice Libary contract which generates CREATE & CREATE2 addresses.
 *         This is used in Rollup to make sure we have address parity with Ethereum mainchain.
 */
contract ContractAddressGenerator { // TODO: Make this a library

    /**
     * @notice Generate a contract address using CREATE.
     *         This code was found here: https://ethereum.stackexchange.com/a/47083
     * @param _origin The address of the contract which is calling CREATE.
     * @param _nonce The contract nonce of the origin contract (incremented each time CREATE is called).
     */
    function getAddressFromCREATE(address _origin, uint _nonce) public pure returns(address) {
        // TODO: Replace with either an RLP encoding library, or at the very least more bytes for the nonce.
        //       In its current form we will overflow after 4 bytes.
        bytes memory data;
        if(_nonce == 0x00)          data = abi.encodePacked(byte(0xd6), byte(0x94), _origin, byte(0x80));
        else if(_nonce <= 0x7f)     data = abi.encodePacked(byte(0xd6), byte(0x94), _origin, byte(uint8(_nonce)));
        else if(_nonce <= 0xff)     data = abi.encodePacked(byte(0xd7), byte(0x94), _origin, byte(0x81), uint8(_nonce));
        else if(_nonce <= 0xffff)   data = abi.encodePacked(byte(0xd8), byte(0x94), _origin, byte(0x82), uint16(_nonce));
        else if(_nonce <= 0xffffff) data = abi.encodePacked(byte(0xd9), byte(0x94), _origin, byte(0x83), uint24(_nonce));
        else                        data = abi.encodePacked(byte(0xda), byte(0x94), _origin, byte(0x84), uint32(_nonce));
        return address(bytes20(keccak256(data)));
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

pragma solidity ^0.5.16;

contract TestConstants {
    bytes32 public hash;

    function storeConstant() external {
        hash = keccak256(
            abi.encode(
                keccak256('EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)'),
                1
            )
        );
    }

    function getConstant() external returns(bytes32) {
        return hash;
    }
}
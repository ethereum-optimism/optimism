pragma solidity ^0.5.0;

contract SimpleCreate2 {
    address public contractAddress;
    
    function create2(bytes memory bytecode, bytes32 salt) public {
    	address create2Address;
        assembly {
            create2Address := create2(0, add(bytecode, 0x20), mload(bytecode), salt)
        }
        contractAddress = create2Address;
    }
}

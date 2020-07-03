pragma solidity ^0.5.0;

contract Precompiles {
	bytes public copiedData;

    function recoverAddr(
    	bytes32 msgHash,
    	uint8 v,
    	bytes32 r,
    	bytes32 s
    ) public view returns (address) {
      // return ecrecover(msgHash, v, r, s);

      assembly {
        let pointer := mload(0x40)

        mstore(pointer, msgHash)
        mstore(add(pointer, 0x20), v)
        mstore(add(pointer, 0x40), r)
        mstore(add(pointer, 0x60), s)

        if iszero(staticcall(not(0), 0x01, pointer, 0x80, pointer, 0x20)) {
            revert(0, 0)
        }

        let size := returndatasize
        returndatacopy(pointer, 0, size)
        return(pointer,size)
      }
    }

    function calculateSHA256(bytes memory input) public view returns (bytes32){
    	return sha256(input);
    }

    function calldataCopy(bytes memory data) public returns (bytes memory) {
	    bytes memory ret = new bytes(data.length);
	    assembly {
	        let len := mload(data)
	        if iszero(call(gas, 0x04, 0, add(data, 0x20), len, add(ret,0x20), len)) {
	            invalid()
	        }
	    }
	    copiedData = ret;
	}

    function expmod(uint base, uint e, uint m) public view returns (uint o) {
        assembly {
            // define pointer
            let p := mload(0x40)
            // store data assembly-favouring ways
            mstore(p, 0x20)             // Length of Base
            mstore(add(p, 0x20), 0x20)  // Length of Exponent
            mstore(add(p, 0x40), 0x20)  // Length of Modulus
            mstore(add(p, 0x60), base)  // Base
            mstore(add(p, 0x80), e)     // Exponent
            mstore(add(p, 0xa0), m)     // Modulus
            if iszero(staticcall(sub(gas, 2000), 0x05, p, 0xc0, p, 0x20)) {
             revert(0, 0)
            }
            // data
            o := mload(p)
        }
    }
}

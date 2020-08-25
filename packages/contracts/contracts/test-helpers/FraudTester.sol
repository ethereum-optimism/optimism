pragma solidity ^0.5.0;

contract BaseFraudTester {
    mapping (bytes32 => bytes32) public builtInStorage;

    function setStorage(bytes32 key, bytes32 value) public {
        builtInStorage[key] = value;
    }

    function setStorageMultiple(bytes32 key, bytes32 value, uint256 count) public {
        for (uint256 i = 0; i < count; i++) {
            setStorage(keccak256(abi.encodePacked(key, i)), value);
        }
    }

    function setStorageMultipleSameKey(bytes32 key, bytes32 value, uint256 count) public {
        for (uint256 i = 0; i < count; i++) {
            setStorage(
                keccak256(abi.encodePacked(key)),
                keccak256(abi.encodePacked(value, i))
            );
        }
    }

    function getStorage(bytes32 key) public view returns (bytes32) {
        return builtInStorage[key];
    }
}

contract FraudTester is BaseFraudTester {
    function createContract(bytes memory _initcode) public {
        assembly {
            let newContractAddress := create(0, add(_initcode, 0x20), mload(_initcode))

            // TODO: add back this check
            // if iszero(extcodesize(newContractAddress)) {
            //     revert(0, 0)
            // }
        }
    }

    function createContractMultiple(bytes memory _initcode, uint256 _count) public {
        for (uint256 i = 0; i < _count; i++) {
            createContract(_initcode);
        }
    }
}

contract MicroFraudTester {
    uint256 _testValue = 123;

    function test() public view returns (uint256) {
        return _testValue;
    }
}
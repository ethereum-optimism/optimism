pragma solidity ^0.5.0;

import {SafetyChecker} from "../SafetyChecker.sol";

/**
 * @title SafeContractRegistry
 * @notice The registry of all OVM-safe smart contracts!
 */
contract SafeContractRegistry {
    // Codehash => deployed address
    mapping(bytes32=>address) public safeContracts;
    SafetyChecker safetyChecker;
    
    constructor(address _safetyCheckerAddress) public {
        safetyChecker = SafetyChecker(_safetyCheckerAddress);
    }

    function registerNewContract(bytes memory _initcode) public {
        address codeContractAddress;
        // Deploy a new contract with this _ovmContractInitCode
        assembly {
            // Set our codeContractAddress to the address returned by our CREATE operation
            codeContractAddress := create(0, add(_initcode, 0x20), mload(_initcode))
            // Make sure that the CREATE was successful (actually deployed something)
            if iszero(extcodesize(codeContractAddress)) {
                revert(0, 0)
            }
        }

        // Safety check the deployed bytecode!
        bytes memory deployedBytecode = getDeployedBytecode(codeContractAddress);
        require(safetyChecker.isBytecodeSafe(deployedBytecode));

        // And finally add it to our registry of safe contracts :)
        bytes32 codeHash;
        assembly {
            codeHash := extcodehash(codeContractAddress)
        }
        safeContracts[codeHash] = codeContractAddress;
    }

    function getDeployedBytecode(address _contractAddress) public view returns (bytes memory codeContractBytecode) {
        assembly {
            // retrieve the size of the code
            let size := extcodesize(_contractAddress)
            // allocate output byte array - this could also be done without assembly
            // by using codeContractBytecode = new bytes(size)
            codeContractBytecode := mload(0x40)
            // new "memory end" including padding
            mstore(0x40, add(codeContractBytecode, and(add(add(size, 0x20), 0x1f), not(0x1f))))
            // store length in memory
            mstore(codeContractBytecode, size)
            // actually retrieve the code, this needs assembly
            extcodecopy(_contractAddress, add(codeContractBytecode, 0x20), 0, size)
        }
    }
}

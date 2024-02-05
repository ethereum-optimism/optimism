// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Vm } from "forge-std/Vm.sol";
import { KontrolCheats } from "kontrol-cheatcodes/KontrolCheats.sol";

// The GhostBytes contracts are a workaround to create a symbolic bytes array. This is slow, but
// required until symbolic bytes are supported in Kontrol: https://github.com/runtimeverification/kontrol/issues/272
contract GhostBytes {
    bytes public ghostBytes;
}

contract GhostBytes10 {
    bytes public ghostBytes0;
    bytes public ghostBytes1;
    bytes public ghostBytes2;
    bytes public ghostBytes3;
    bytes public ghostBytes4;
    bytes public ghostBytes5;
    bytes public ghostBytes6;
    bytes public ghostBytes7;
    bytes public ghostBytes8;
    bytes public ghostBytes9;

    function getGhostBytesArray() public view returns (bytes[] memory _arr) {
        _arr = new bytes[](10);
        _arr[0] = ghostBytes0;
        _arr[1] = ghostBytes1;
        _arr[2] = ghostBytes2;
        _arr[3] = ghostBytes3;
        _arr[4] = ghostBytes4;
        _arr[5] = ghostBytes5;
        _arr[6] = ghostBytes6;
        _arr[7] = ghostBytes7;
        _arr[8] = ghostBytes8;
        _arr[9] = ghostBytes9;
    }
}

/// @notice Tests inheriting this contract cannot be run with forge
abstract contract KontrolUtils is KontrolCheats {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @dev Creates a fresh bytes with length greater than 31
    /// @param bytesLength: Length of the fresh bytes. Should be concrete
    function freshBigBytes(uint256 bytesLength) internal returns (bytes memory sBytes) {
        require(bytesLength >= 32, "Small bytes");

        uint256 bytesSlotValue;
        unchecked {
            bytesSlotValue = bytesLength * 2 + 1;
        }

        // Deploy ghost contract
        GhostBytes ghostBytes = new GhostBytes();

        // Make the storage of the ghost contract symbolic
        kevm.symbolicStorage(address(ghostBytes));

        // Load the size encoding into the first slot of ghostBytes
        vm.store(address(ghostBytes), bytes32(uint256(0)), bytes32(bytesSlotValue));

        sBytes = ghostBytes.ghostBytes();
    }

    /// @dev Creates a bounded symbolic bytes[] memory representing a withdrawal proof.
    function freshWithdrawalProof() public returns (bytes[] memory withdrawalProof) {
        // ASSUME: Withdrawal proofs do not currently exceed 6 elements in length. This can be
        // shrank to 2 for faster proof speeds during testing and development.
        // TODO: Allow the array length range between 0 and 10 elements. This can be done once
        // symbolic bytes are supported in Kontrol: https://github.com/runtimeverification/kontrol/issues/272
        uint256 arrayLength = 6;
        withdrawalProof = new bytes[](arrayLength);

        for (uint256 i = 0; i < withdrawalProof.length; ++i) {
            // ASSUME: Each element is 600 bytes. Proof elements are 17 * 32 = 544 bytes long, plus
            // ~10% margin for RLP encoding:, giving us the 600 byte assumption.
            withdrawalProof[i] = freshBigBytes(600);
        }
    }
}

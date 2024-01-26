// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Vm } from "forge-std/Vm.sol";
import { Types } from "src/libraries/Types.sol";
import { KontrolCheats } from "kontrol-cheatcodes/KontrolCheats.sol";

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

/// @notice tests inheriting this contract cannot be run with forge
abstract contract KontrolUtils is KontrolCheats {
    /// @dev we only care about the vm signature
    // Cheat code address, 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D.
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

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

    /// @dev Creates a bounded symbolic bytes[] memory representing a withdrawal proof
    /// Each element is 17 * 32 = 544 bytes long, plus ~10% margin for RLP encoding: each element is 600 bytes
    /// The length of the array to 10 or fewer elements
    function freshWithdrawalProof() public returns (bytes[] memory withdrawalProof) {
        // Assume arrayLength = 2 for faster proof speeds
        // TODO: have the array length range between 0 and 10 elements
        uint256 arrayLength = 6;

        withdrawalProof = new bytes[](arrayLength);

        // Deploy ghost contract
        // GhostBytes10 ghostBytes10 = new GhostBytes10();

        // Make the storage of the ghost contract symbolic
        // kevm.symbolicStorage(address(ghostBytes10));

        // Each bytes element will have a length of 600
        // uint256 bytesSlotValue = 600 * 2 + 1;

        // Load the size encoding into the first slot of ghostBytes
        // vm.store(address(ghostBytes10), bytes32(uint256(0)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(1)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(2)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(3)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(4)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(5)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(6)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(7)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(8)), bytes32(bytesSlotValue));
        // vm.store(address(ghostBytes10), bytes32(uint256(9)), bytes32(bytesSlotValue));

        // withdrawalProof = ghostBytes10.getGhostBytesArray();

        // Second approach

        // withdrawalProof[0] = ghostBytes10.ghostBytes0();
        // withdrawalProof[1] = ghostBytes10.ghostBytes1();
        // withdrawalProof[2] = ghostBytes10.ghostBytes2();
        // withdrawalProof[3] = ghostBytes10.ghostBytes3();
        // withdrawalProof[4] = ghostBytes10.ghostBytes4();
        // withdrawalProof[5] = ghostBytes10.ghostBytes5();
        // withdrawalProof[6] = ghostBytes10.ghostBytes6();
        // withdrawalProof[7] = ghostBytes10.ghostBytes7();
        // withdrawalProof[8] = ghostBytes10.ghostBytes8();
        // withdrawalProof[9] = ghostBytes10.ghostBytes9();

        // First approach

        for (uint256 i = 0; i < withdrawalProof.length; ++i) {
            withdrawalProof[i] = freshBigBytes(600);
        }
    }
}

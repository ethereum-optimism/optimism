// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Vm } from "forge-std/Vm.sol";
import { Types } from "src/libraries/Types.sol";
import { KontrolCheats } from "kontrol-cheatcodes/KontrolCheats.sol";

contract GhostBytes {
    bytes public ghostBytes;
}

/// @notice tests inheriting this contract cannot be run with forge
abstract contract KontrolUtils is KontrolCheats {

    /// @dev we only care about the vm signature
    // Cheat code address, 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D.
    address private constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm private constant vm = Vm(VM_ADDRESS);


    function createWithdrawalTransaction(
      uint256 _tx0,
      address _tx1,
      address _tx2,
      uint256 _tx3,
      uint256 _tx4,
      bytes   memory _tx5
    ) internal pure returns (Types.WithdrawalTransaction memory _tx) {
        _tx = Types.WithdrawalTransaction (
                                           _tx0,
                                           _tx1,
                                           _tx2,
                                           _tx3,
                                           _tx4,
                                           _tx5
        );
    }

    function freshBytesArray(uint256 symbolicArrayLength) public returns (bytes[] memory symbolicArray) {
        symbolicArray = new bytes[](symbolicArrayLength);

        for (uint256 i = 0; i < symbolicArray.length; ++i) {
            symbolicArray[i] = abi.encodePacked(kevm.freshUInt(32));
        }
    }

    /// @dev Returns a symbolic bytes32
    function freshBytes32() public returns (bytes32) {
        return bytes32(kevm.freshUInt(32));
    }

    /// @dev Returns a symbolic adress
    function freshAdress() public returns (address) {
        return address(uint160(kevm.freshUInt(20)));
    }

    /// @dev Creates a fresh bytes with length greater than 31
    /// @param bytesLength: Length of the fresh bytes. Should be concrete
    function freshBigBytes(uint256 bytesLength) internal returns (bytes memory sBytes) {
        require(bytesLength >= 32, "Small bytes");

        uint256 bytesSlotValue;
        unchecked {
            bytesSlotValue = bytesLength * 2 + 1;
        }

        /* Deploy ghost contract */
        GhostBytes ghostBytes = new GhostBytes();

        /* Make the storage of the ghost contract symbolic */
        kevm.symbolicStorage(address(ghostBytes));

        /* Load the size encoding into the first slot of ghostBytes*/
        vm.store(address(ghostBytes), bytes32(uint256(0)), bytes32(bytesSlotValue));

        /* vm.assume(ghostBytes.bytesLength() == bytesLength); */
        sBytes = ghostBytes.ghostBytes();
    }

    /// @dev Creates a bounded symbolic bytes[] memory representing a withdrawal proof
    /// Each element is 17 * 32 = 544 bytes long, plus ~10% margin for RLP encoding: each element is 600 bytes
    /// The length of the array to 10 or fewer elements
    function freshWithdrawalProof() public returns (bytes[] memory withdrawalProof) {
        /* Assuming arrayLength = 2 for faster proof speeds. For full generality replace with the code below */
        uint256 arrayLength = 2;
        /* uint256 arrayLength = kevm.freshUInt(32); */
        /* vm.assume(arrayLength <= 10); */

        withdrawalProof = new bytes[](arrayLength);

        for (uint256 i = 0; i < withdrawalProof.length; ++i) {
            withdrawalProof[i] = freshBigBytes(60); // abi.encodePacked(freshBytes32());  // abi.encodePacked(kevm.freshUInt(32));
        }
    }
}

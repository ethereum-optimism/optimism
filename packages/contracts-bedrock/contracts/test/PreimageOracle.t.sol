// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";

import { PreimageOracle } from "../cannon/PreimageOracle.sol";
import {
    PreimageKey,
    PreimageOffset,
    PreimagePart,
    PreimageLength
} from "../cannon/lib/CannonTypes.sol";
import { MissingPreimage, UnauthorizedCaller } from "../cannon/lib/CannonErrors.sol";

contract PreimageOracle_Test is Test {
    PreimageOracle oracle;

    /// @notice Sets up the testing suite.
    function setUp() public {
        oracle = new PreimageOracle(address(this));
        vm.label(address(oracle), "PreimageOracle");
    }

    /// @notice Test the pre-image key computation with a known pre-image.
    function test_computePreimageKey_succeeds() public {
        bytes memory preimage = hex"deadbeef";
        PreimageKey key = oracle.computePreimageKey(preimage);
        bytes32 unwrappedKey = PreimageKey.unwrap(key);
        bytes32 known = 0x02fd4e189132273036449fc9e11198c739161b4c0116a9a2dccdfa1c492006f1;
        assertEq(unwrappedKey, known, "computePreimageKey");
    }

    /// @notice Tests that a pre-image is correctly set.
    function test_loadKeccak256PreimagePart_succeeds() public {
        // Set the pre-image
        bytes memory preimage = hex"deadbeef";
        PreimageKey key = oracle.computePreimageKey(preimage);
        PreimageOffset offset = PreimageOffset.wrap(0);
        oracle.loadKeccak256PreimagePart(offset, preimage);

        // Validate the pre-image part
        PreimagePart part = oracle.preimageParts(key, offset);
        bytes32 expectedPart = 0x0000000000000004deadbeef0000000000000000000000000000000000000000;
        assertEq(PreimagePart.unwrap(part), expectedPart);

        // Validate the pre-image length
        PreimageLength length = oracle.preimageLengths(key);
        assertEq(PreimageLength.unwrap(length), preimage.length);

        // Validate that the pre-image part is set
        bool ok = oracle.preimagePartOk(key, offset);
        assertTrue(ok);
    }

    /// @notice Tests that a pre-image cannot be set with an out-of-bounds offset.
    function test_loadKeccak256PreimagePart_outOfBoundsOffset_reverts() public {
        bytes memory preimage = hex"deadbeef";
        PreimageOffset offset = PreimageOffset.wrap(preimage.length + 8);

        vm.expectRevert();
        oracle.loadKeccak256PreimagePart(offset, preimage);
    }

    /// @notice Reading a pre-image part that has not been set should revert.
    function testFuzz_readPreimage_missingPreimage_reverts(bytes32 rawKey, uint256 rawOffset)
        public
    {
        PreimageKey key = PreimageKey.wrap(rawKey);
        PreimageOffset offset = PreimageOffset.wrap(rawOffset);

        vm.expectRevert(
            abi.encodeWithSignature("MissingPreimage(bytes32,uint256)", rawKey, rawOffset)
        );
        oracle.readPreimage(key, offset);
    }
}

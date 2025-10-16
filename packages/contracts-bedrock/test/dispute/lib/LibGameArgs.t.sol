// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

contract LibGameArgs_Harness {
    function decode(bytes memory _buf) public pure returns (LibGameArgs.GameArgs memory) {
        return LibGameArgs.decode(_buf);
    }
}

/// @title LibGameArgs_Decode_Test
/// @notice Test contract for the LibGameArgs library's decode function.
contract LibGameArgs_Decode_Test is Test {
    LibGameArgs_Harness internal harness;

    function setUp() public {
        harness = new LibGameArgs_Harness();
    }

    /// @notice Struct to hold game arguments for testing purposes.
    ///   Avoids "stack too deep" errors in the test functions.
    struct GameArgs {
        bytes32 absolutePrestate;
        address vm;
        address asr;
        address weth;
        uint256 l2ChainId;
        address proposer;
        address challenger;
    }

    function test_decodeFull_succeeds() public {
        GameArgs memory args = GameArgs({
            absolutePrestate: keccak256(abi.encodePacked("absolutePrestate")),
            vm: vm.randomAddress(),
            asr: address(0x2),
            weth: address(0x3),
            l2ChainId: 42,
            proposer: address(0x123),
            challenger: address(0x456)
        });
        bytes memory buf = abi.encodePacked(
            args.absolutePrestate, args.vm, args.asr, args.weth, args.l2ChainId, args.proposer, args.challenger
        );

        LibGameArgs.GameArgs memory decoded = harness.decode(buf);
        assertEq(decoded.absolutePrestate, args.absolutePrestate);
        assertEq(decoded.vm, args.vm);
        assertEq(decoded.anchorStateRegistry, args.asr);
        assertEq(decoded.weth, args.weth);
        assertEq(decoded.l2ChainId, args.l2ChainId);
        assertEq(decoded.proposer, args.proposer);
        assertEq(decoded.challenger, args.challenger);
    }

    function test_decodeShort_succeeds() public {
        GameArgs memory args = GameArgs({
            absolutePrestate: keccak256(abi.encodePacked("absolutePrestate")),
            vm: vm.randomAddress(),
            asr: address(0x2),
            weth: address(0x3),
            l2ChainId: 42,
            proposer: address(0x123),
            challenger: address(0x456)
        });
        bytes memory buf = abi.encodePacked(args.absolutePrestate, args.vm, args.asr, args.weth, args.l2ChainId);

        LibGameArgs.GameArgs memory decoded = harness.decode(buf);
        assertEq(decoded.absolutePrestate, args.absolutePrestate);
        assertEq(decoded.vm, args.vm);
        assertEq(decoded.anchorStateRegistry, args.asr);
        assertEq(decoded.weth, args.weth);
        assertEq(decoded.l2ChainId, args.l2ChainId);
        assertEq(decoded.proposer, address(0));
        assertEq(decoded.challenger, address(0));
    }

    function test_decode_invalidLengthOverfull_reverts() public {
        GameArgs memory args = GameArgs({
            absolutePrestate: keccak256(abi.encodePacked("absolutePrestate")),
            vm: vm.randomAddress(),
            asr: address(0x2),
            weth: address(0x3),
            l2ChainId: 42,
            proposer: address(0x123),
            challenger: address(0x456)
        });
        bytes memory buf = abi.encodePacked(
            args.absolutePrestate,
            args.vm,
            args.asr,
            args.weth,
            args.l2ChainId,
            args.proposer,
            args.challenger,
            uint256(999)
        );

        vm.expectRevert("GameArgs: decode with invalid length");
        harness.decode(buf);
    }

    function test_decode_invalidLength_reverts(bytes memory _buf) public {
        bool ok = (_buf.length == 124 || _buf.length == 164);
        vm.assume(!ok);
        vm.expectRevert("GameArgs: decode with invalid length");
        harness.decode(_buf);
    }

    function test_isValidPermissionlessArgs_works() public pure {
        bytes memory validBuf = new bytes(124);
        assertTrue(LibGameArgs.isValidPermissionlessArgs(validBuf));
        validBuf = new bytes(164);
        assertFalse(LibGameArgs.isValidPermissionlessArgs(validBuf));
    }

    function test_isValidPermissionedArgs_works() public pure {
        bytes memory validBuf = new bytes(164);
        assertTrue(LibGameArgs.isValidPermissionedArgs(validBuf));
        validBuf = new bytes(124);
        assertFalse(LibGameArgs.isValidPermissionedArgs(validBuf));
    }
}

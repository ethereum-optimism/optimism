// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { PrestateRegistry } from "src/dispute/PrestateRegistry.sol";
import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

contract PrestateRegistry_Test is CommonTest {
    PrestateRegistry registry;

    function setUp() public override {
        super.enableFaultProofs();
        super.setUp();

        registry = new PrestateRegistry({
            _superchainConfig: superchainConfig,
            _l2GenesisBlockTimestamp: Timestamp.wrap(0),
            _l2GenesisBlock: 0,
            _l2ChainId: 0xfacade,
            _l2BlockTime: 2
        });
    }

    /// @dev Tests that a hardfork may always be registered by the guardian.
    function testFuzz_registerHardfork_guardian_succeeds(Timestamp _activationTime) public {
        // Manually construct some prestate information for several VMs and programs.
        PrestateRegistry.PrestateInformation[] memory info = new PrestateRegistry.PrestateInformation[](3);
        info[0] = PrestateRegistry.PrestateInformation({
            vmID: 0,
            programID: 0,
            prestateHash: Hash.wrap(bytes32(uint256(0xbeef)))
        });
        info[1] = PrestateRegistry.PrestateInformation({
            vmID: 0,
            programID: 1,
            prestateHash: Hash.wrap(bytes32(uint256(0xbabe)))
        });
        info[2] = PrestateRegistry.PrestateInformation({
            vmID: 1,
            programID: 0,
            prestateHash: Hash.wrap(bytes32(uint256(0xdead)))
        });

        // Register the hardfork.
        vm.prank(superchainConfig.guardian());
        registry.registerHardfork(_activationTime, info);

        // Grab the hardfork from the contract.
        uint256 hardforkActivation = registry.hardforks(0);
        uint256 l2Block = registry.l2TimestampToBlock(_activationTime);
        assertEq(hardforkActivation, l2Block);

        // Loop through the prestate information and check that it was registered.
        for (uint256 i = 0; i < info.length; i++) {
            PrestateRegistry.PrestateInformation memory localInfo = info[i];
            assertEq(
                registry.activePrestate(l2Block, localInfo.vmID, localInfo.programID).raw(),
                localInfo.prestateHash.raw()
            );
        }
    }

    /// @dev Tests that a hardfork may not be registered by any other address than the guardian.
    function testFuzz_registerHardfork_notGuardian_reverts(address _a) public {
        vm.assume(_a != superchainConfig.guardian());

        vm.expectRevert(BadAuth.selector);
        registry.registerHardfork(Timestamp.wrap(0), new PrestateRegistry.PrestateInformation[](0));
    }

    /// @dev Tests that a hardfork may not be registered by any other address than the guardian.
    function test_registerHardfork_outOfOrder_reverts() public {
        vm.prank(superchainConfig.guardian());
        registry.registerHardfork(Timestamp.wrap(10), new PrestateRegistry.PrestateInformation[](0));
        vm.prank(superchainConfig.guardian());
        vm.expectRevert(OutOfOrderHardfork.selector);
        registry.registerHardfork(Timestamp.wrap(10), new PrestateRegistry.PrestateInformation[](0));
    }

    /// @dev Tests that a hardfork may be revoked by the guardian if it has not yet activated.
    function testFuzz_revokePendingFork_guardian_succeeds() public {
        // Register a hardfork.
        vm.prank(superchainConfig.guardian());
        registry.registerHardfork(Timestamp.wrap(10), new PrestateRegistry.PrestateInformation[](0));

        // Ensure it was registered.
        assertEq(registry.hardforks(0), registry.l2TimestampToBlock(Timestamp.wrap(10)));

        // Revoke the hardfork.
        vm.prank(superchainConfig.guardian());
        registry.revokePendingFork();

        // Ensure that the hardfork was removed.
        vm.expectRevert();
        registry.hardforks(0);
    }

    /// @dev Tests that a hardfork may not be revoked by any other address than the guardian.
    function testFuzz_revokePendingFork_notGuardian_reverts(address _a) public {
        vm.assume(_a != superchainConfig.guardian());

        // Register a hardfork.
        vm.prank(superchainConfig.guardian());
        registry.registerHardfork(Timestamp.wrap(10), new PrestateRegistry.PrestateInformation[](0));

        // Attempt to revoke the hardfork.
        vm.expectRevert(BadAuth.selector);
        registry.revokePendingFork();
    }

    /// @dev Tests that a hardfork may not be revoked by the guardian if it has activated.
    function testFuzz_revokePendingFork_alreadyActivated_reverts() public {
        // Register a hardfork.
        vm.prank(superchainConfig.guardian());
        registry.registerHardfork(Timestamp.wrap(1), new PrestateRegistry.PrestateInformation[](0));

        // Ensure it was registered.
        assertEq(registry.hardforks(0), registry.l2TimestampToBlock(Timestamp.wrap(1)));

        // Revoke the hardfork.
        vm.prank(superchainConfig.guardian());
        vm.expectRevert(ForkAlreadyActivated.selector);
        registry.revokePendingFork();

        // Ensure that the hardfork still exists.
        assertEq(registry.hardforks(0), registry.l2TimestampToBlock(Timestamp.wrap(1)));
    }

    /// @dev Static test to check if the absolute prestate in the latest active hardfork for a given VM and program
    ///      is returned.
    function test_activePrestate_succeeds() public {
        // Generate some dummy hardfork information.
        PrestateRegistry.PrestateInformation[] memory info = new PrestateRegistry.PrestateInformation[](1);

        // Register a few hardforks.
        vm.startPrank(superchainConfig.guardian());
        info[0] = PrestateRegistry.PrestateInformation({
            vmID: 0,
            programID: 0,
            prestateHash: Hash.wrap(bytes32(uint256(0xbeef)))
        });
        registry.registerHardfork(Timestamp.wrap(10), info);
        info[0] = PrestateRegistry.PrestateInformation({
            vmID: 0,
            programID: 0,
            prestateHash: Hash.wrap(bytes32(uint256(0xbabe)))
        });
        registry.registerHardfork(Timestamp.wrap(20), info);
        info[0] = PrestateRegistry.PrestateInformation({
            vmID: 0,
            programID: 0,
            prestateHash: Hash.wrap(bytes32(uint256(0xdead)))
        });
        registry.registerHardfork(Timestamp.wrap(30), info);
        vm.stopPrank();

        registry.l2TimestampToBlock(Timestamp.wrap(10));
        registry.l2TimestampToBlock(Timestamp.wrap(20));
        registry.l2TimestampToBlock(Timestamp.wrap(30));

        // Ensure that the active prestate is returned correctly for each hardfork
        vm.expectRevert(NoRegisteredForks.selector);
        registry.activePrestate(0, 0, 0).raw();
        vm.expectRevert(NoRegisteredForks.selector);
        registry.activePrestate(4, 0, 0).raw();

        assertEq(registry.activePrestate(5, 0, 0).raw(), bytes32(uint256(0xbeef)));
        assertEq(registry.activePrestate(9, 0, 0).raw(), bytes32(uint256(0xbeef)));

        assertEq(registry.activePrestate(10, 0, 0).raw(), bytes32(uint256(0xbabe)));
        assertEq(registry.activePrestate(14, 0, 0).raw(), bytes32(uint256(0xbabe)));

        assertEq(registry.activePrestate(15, 0, 0).raw(), bytes32(uint256(0xdead)));
        assertEq(registry.activePrestate(20, 0, 0).raw(), bytes32(uint256(0xdead)));
    }
}

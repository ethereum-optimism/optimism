// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";

import { AttestationDisputeGame } from "../dispute/AttestationDisputeGame.sol";
import { IDisputeGameFactory } from "../dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "../dispute/IDisputeGame.sol";
import { IBondManager } from "../dispute/IBondManager.sol";
import { BondManager } from "../dispute/BondManager.sol";
import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";
import { Portal_Initializer } from "./CommonTest.t.sol";

/**
 * @title AttestationDisputeGame_Test
 */
contract AttestationDisputeGame_Test is Portal_Initializer {
    bytes32 constant TYPE_HASH = 0x2676994b0652bcdf7968635d15b78aac9aaf797cc94c5adeb94376cc28f987d6;

    DisputeGameFactory factory;
    BondManager bm;
    AttestationDisputeGame disputeGameImplementation;
    AttestationDisputeGame disputeGameProxy;

    // SystemConfig `attestorSet` keys
    uint256[] attestorKeys;

    // Emitted when a new dispute game is created by the [DisputeGameFactory]
    event DisputeGameCreated(
        address indexed disputeProxy,
        GameType indexed gameType,
        Claim indexed rootClaim
    );

    function setUp() public override {
        super.setUp();

        factory = new DisputeGameFactory(address(this));
        vm.label(address(factory), "DisputeGameFactory");
        bm = new BondManager(factory);
        vm.label(address(bm), "BondManager");

        vm.label(address(systemConfig), "SystemConfig");
        // Transfer ownership of the system config to this contract
        vm.prank(address(1));
        systemConfig.transferOwnership(address(this));

        // Add 5 signers to the attestor set
        for (uint256 i = 1; i < 6; i++) {
            attestorKeys.push(i);
            systemConfig.setAttestor(vm.addr(i), true);
        }
        systemConfig.setAttestationThreshold(5);

        // Create the dispute game implementation
        disputeGameImplementation = new AttestationDisputeGame(
            IBondManager(address(bm)),
            systemConfig,
            oracle
        );
        vm.label(address(disputeGameImplementation), "AttestationDisputeGame_Implementation");

        // Set the implementation in the factory
        GameType gt = GameType.ATTESTATION;
        factory.setImplementation(gt, IDisputeGame(address(disputeGameImplementation)));

        // Create the attestation dispute game in the factory
        bytes memory extraData = hex"";
        Claim rootClaim = Claim.wrap(bytes32(0));
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        disputeGameProxy = AttestationDisputeGame(
            address(factory.create(gt, rootClaim, extraData))
        );
        assertEq(address(factory.games(gt, rootClaim, extraData)), address(disputeGameProxy));
        vm.label(address(disputeGameProxy), "AttestationDisputeGame_Proxy");
    }

    /********************
     * INIT STATE TESTS *
     ********************/

    /**
     * @dev Tests that, upon initialization, the game type is set to `IN_PROGRESS`
     */
    function test_initGameStatus_succeeds() public {
        assertEq(uint8(disputeGameProxy.status()), uint8(GameStatus.IN_PROGRESS));
    }

    /**
     * @dev Tests that, upon initialization, the game type is set to `ATTESTATION`
     */
    function test_gameType_succeeds() public {
        assertEq(uint8(disputeGameProxy.gameType()), uint8(GameType.ATTESTATION));
    }

    /**
     * @dev Tests that, upon initialization, the version is set to "0.0.1"
     */
    function test_version_succeeds() public {
        assertEq(disputeGameProxy.version(), "0.0.1");
    }

    /**
     * @dev Tests that the dispute type hash was properly configured.
     *      The intended preimage is "Dispute(bytes32 outputRoot,uint256 l2BlockNumber)"
     */
    function test_disputeTypeHash_succeeds() public {
        assertEq(Hash.unwrap(disputeGameProxy.DISPUTE_TYPE_HASH()), TYPE_HASH);
    }

    /**
     * @dev Tests that the dispute game implementation properly forwards the
     *      signature threshold call to the `SystemConfig`
     */
    function test_attestationThreshold_succeeds() public {
        assertEq(disputeGameProxy.frozenSignatureThreshold(), systemConfig.attestationThreshold());
    }

    /**
     * @dev Tests that the default initialization set the proper values.
     */
    function test_defaultInitialization_succeeds() public {
        // Assert that the oracle was properly set
        assertEq(address(disputeGameProxy.L2_OUTPUT_ORACLE()), address(oracle));

        // Assert that the system config was properly set
        assertEq(address(disputeGameProxy.SYSTEM_CONFIG()), address(systemConfig));

        // Assert that the bond manager was properly set
        IBondManager _bondManager = disputeGameProxy.BOND_MANAGER();
        assertEq(address(_bondManager), address(bm));

        // Assert that the attestor set was copied over from the `SystemConfig`
        uint256 frozenSetLength = uint256(vm.load(address(disputeGameProxy), bytes32(uint256(2))));
        address[] memory frozenSet = new address[](frozenSetLength);
        for (uint256 i = 0; i < frozenSetLength; i++) {
            frozenSet[i] = disputeGameProxy.frozenAttestorSet(i);
        }
        assertEq(frozenSet, systemConfig.attestorSet());

        // Assert that the signature threshold was copied over from the `SystemConfig`
        assertEq(disputeGameProxy.frozenSignatureThreshold(), systemConfig.attestationThreshold());
    }

    /********************
     * ATTESTOR SET TESTS *
     ********************/

    /**
     * @dev Tests that changing the `SystemConfig`'s attestor set does not change the
     *      frozen attestor set of the `AttestationDisputeGame`.
     */
    function test_changeAttestorSet_staysFrozen_succeeds() public {
        // Add 5 more signers to the attestor set
        for (uint256 i = 6; i < 11; i++) {
            systemConfig.setAttestor(vm.addr(i), true);
        }

        // Grab the length of the frozen attestor set from the `AttestationDisputeGame`
        // The frozen attestor set is stored at slot 2 in the `AttestationDisputeGame` storage
        uint256 frozenLength = uint256(vm.load(address(disputeGameProxy), bytes32(uint256(2))));
        // Grab the length of the canonical attestor set from the `SystemConfig`
        uint256 sysConfigLength = systemConfig.attestorSet().length;

        // Assert that the frozen attestor set length is 5 and the attestor set length is 10
        assertEq(frozenLength, 5);
        assertEq(sysConfigLength, 10);
    }

    /**
     * @dev Tests that after changing the `SystemConfig`'s attestor set, the attestor
     *      set of all new `AttestationDisputeGame`s reflect these changes.
     */
    function test_changeAttestorSet_newGame_succeeds() public {
        // Add 5 more signers to the attestor set and ensure that the current game's
        // attestor set remains static.
        test_changeAttestorSet_staysFrozen_succeeds();

        // Create a new attestation dispute game.
        AttestationDisputeGame newGame = AttestationDisputeGame(
            address(factory.create(GameType.ATTESTATION, Claim.wrap(bytes32(0)), abi.encode(20)))
        );

        // Grab the length of the frozen attestor set from the new `AttestationDisputeGame`
        uint256 frozenLength = uint256(vm.load(address(newGame), bytes32(uint256(2))));
        // Grab the length of the canonical attestor set from the `SystemConfig`.
        uint256 sysConfigLength = systemConfig.attestorSet().length;

        // Assert that the frozen attestor set length for the new is 10 and the attestor set length is 10
        assertEq(frozenLength, sysConfigLength);
    }

    /**
     * @dev Tests that changing the `SystemConfig`'s signature threshold does not change the
     *      frozen signature threshold of the `AttestationDisputeGame`.
     */
    function test_changeAttestationThreshold_staysFrozen_succeeds() public {
        // Update the signature threshold in the system configuration
        systemConfig.setAttestationThreshold(6);

        // Assert that the frozen signature threshold is still 5
        assertEq(disputeGameProxy.frozenSignatureThreshold(), 5);
        // Assert that the canonical signature threshold is 6 after the update
        assertEq(systemConfig.attestationThreshold(), 6);
    }

    /**
     * @dev Tests that after changing the `SystemConfig`'s signature threshold, the signature
     *      threshold of all new `AttestationDisputeGame`s reflect these changes.
     */
    function test_changeAttestationThreshold_newGame_succeeds() public {
        // Change the signature threshold in the `SystemConfig`
        test_changeAttestationThreshold_staysFrozen_succeeds();

        // Create a new attestation dispute game.
        AttestationDisputeGame newGame = AttestationDisputeGame(
            address(factory.create(GameType.ATTESTATION, Claim.wrap(bytes32(0)), abi.encode(20)))
        );

        // Assert that the canonical signature threshold is 6 after the update
        assertEq(systemConfig.attestationThreshold(), 6);
        // Assert that the new game's signature threshold is 6.
        assertEq(newGame.frozenSignatureThreshold(), systemConfig.attestationThreshold());
    }
}

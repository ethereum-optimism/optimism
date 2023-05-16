// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { ECDSA } from "@solady/utils/ECDSA.sol";

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
    using stdStorage for StdStorage;

    bytes32 constant TYPE_HASH = 0x2676994b0652bcdf7968635d15b78aac9aaf797cc94c5adeb94376cc28f987d6;

    // Deployed contracts on setUp()
    DisputeGameFactory factory;
    BondManager bm;
    AttestationDisputeGame disputeGameImplementation;
    AttestationDisputeGame disputeGameProxy;

    // Clones with immutable args parameters
    bytes public extraData;
    Claim public rootClaim;
    uint256 public l2BlockNumber;

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

        // Create the dispute game factory
        factory = new DisputeGameFactory(address(this));
        vm.label(address(factory), "DisputeGameFactory");

        // Create the bond manager
        bm = new BondManager(factory);
        vm.label(address(bm), "BondManager");

        // Transfer ownership of the system to this contract
        vm.label(address(systemConfig), "SystemConfig");
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
        disputeGameImplementation.initialize();
        vm.label(address(disputeGameImplementation), "AttestationDisputeGame_Implementation");

        // Set the implementation in the factory
        GameType gt = GameType.ATTESTATION;
        factory.setImplementation(gt, IDisputeGame(address(disputeGameImplementation)));

        // Create the attestation dispute game in the factory
        l2BlockNumber = 100;
        extraData = abi.encode(l2BlockNumber);
        rootClaim = Claim.wrap(bytes32(0));
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
     * @dev Test the EIP712 domain separator
     */
    function test_eip712Domain_succeeds() public {
        (
            bytes1 fields,
            string memory name,
            string memory version,
            uint256 chainId,
            address verifyingContract,
            bytes32 salt,
            uint256[] memory extensions
        ) = disputeGameProxy.eip712Domain();
        assertEq(fields, bytes1(0x0f));
        assertEq(name, "AttestationDisputeGame");
        assertEq(version, "0.0.1");
        assertEq(chainId, block.chainid);
        assertEq(verifyingContract, address(disputeGameProxy));
        assertEq(salt, bytes32(0));
        assertEq(extensions.length, 0);
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

    /**
     * @dev Tests that the bond manager is set correctly.
     */
    function test_bondManager_succeeds() public {
        assertEq(address(disputeGameProxy.BOND_MANAGER()), address(bm));
    }

    /***********************
     * CWIA ARGUMENT TESTS *
     ***********************/

    /**
     * @dev Test that the extraData is correctly appended to the dispute game clone.
     */
    function test_extraData_succeeds() public {
        assertEq(disputeGameProxy.extraData(), extraData);
    }

    /**
     * @dev Test that the root claim is correctly appended to the dispute game clone.
     */
    function test_rootClaim_succeeds() public {
        assertEq(Claim.unwrap(disputeGameProxy.rootClaim()), Claim.unwrap(rootClaim));
    }

    /**
     * @dev Test that the L2 block number is correctly appended to the dispute game clone.
     */
    function test_l2BlockNumber_succeeds() public {
        assertEq(disputeGameProxy.l2BlockNumber(), l2BlockNumber);
    }

    /**********************
     * ATTESTOR SET TESTS *
     **********************/

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

    /******************************
     * DISPUTE TESTS - HAPPY PATH *
     ******************************/

    /**
     * @dev Challenge the attestation dispute game with a valid claim.
     */
    function test_challenge_succeeds() public {
        bytes32 msgHash = Hash.unwrap(disputeGameImplementation.getTypedDataHash());
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(1, msgHash);
        bytes memory signature = abi.encodePacked(r, s, v);
        assertEq(signature.length, 65);
        disputeGameImplementation.challenge(signature);
        assertTrue(disputeGameImplementation.challenges(vm.addr(1)));
        assertEq(disputeGameImplementation.attestationSubmitters(0), address(this));
    }

    /****************************
     * DISPUTE TESTS - SAD PATH *
     ****************************/

    /**
     * @dev Duplicate challenges should revert.
     */
    function test_challenge_duplicate_reverts() public {
        test_challenge_succeeds();

        bytes32 msgHash = Hash.unwrap(disputeGameImplementation.getTypedDataHash());
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(1, msgHash);
        bytes memory signature = abi.encodePacked(r, s, v);
        assertEq(signature.length, 65);
        vm.expectRevert(AlreadyChallenged.selector);
        disputeGameImplementation.challenge(signature);
    }

    /**
     * @dev Challenging the attestation dispute game should revert
     *      when not in progress.
     */
    function test_challenge_notInProgress_reverts() public {
        address target = address(disputeGameImplementation);
        // stdstore.target(target).sig("status()").checked_write(bytes32(uint256(1)));

        vm.store(
            target,
            bytes32(0x0000000000000000000000000000000000000000000000000000000003e90001),
            bytes32(uint256(1))
        );
        // GameStatus status = disputeGameImplementation.status();
        // assertTrue(uint256(status) != uint256(GameStatus.IN_PROGRESS));
        // vm.expectRevert(GameNotInProgress.selector);
        vm.expectRevert();
        disputeGameImplementation.challenge(bytes(""));
    }

    /**
     * @dev Challenging the attestation dispute game should revert
     *      when the signature is invalid.
     */
    function testFuzz_challenge_invalidSignature_reverts(bytes calldata signature) public {
        // The game should be in progress
        GameStatus status = disputeGameProxy.status();
        assertEq(uint256(status), uint256(GameStatus.IN_PROGRESS));

        // Make sure we didn't accidentally generate a valid signature
        Hash atstHash = disputeGameProxy.getTypedDataHash();
        vm.expectRevert(InvalidSignature.selector);
        address recovered = ECDSA.recoverCalldata(Hash.unwrap(atstHash), signature);
        address[] memory attestorSet = systemConfig.attestorSet();
        for (uint256 i = 0; i < attestorSet.length; i++) {
            vm.assume(recovered != attestorSet[i]);
        }

        vm.expectRevert(InvalidSignature.selector);
        disputeGameProxy.challenge(signature);
    }
}

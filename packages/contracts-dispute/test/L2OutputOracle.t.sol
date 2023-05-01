// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import "src/types/Errors.sol";
import "src/types/Types.sol";

import { LibClock } from "src/lib/LibClock.sol";
import { LibHashing } from "src/lib/LibHashing.sol";
import { LibPosition } from "src/lib/LibPosition.sol";

import { ResourceMetering } from "contracts-bedrock/L1/ResourceMetering.sol";
import { SystemConfig } from "contracts-bedrock/L1/SystemConfig.sol";
import { L2OutputOracle } from "contracts-bedrock/L1/L2OutputOracle.sol";

import { AttestationDisputeGame } from "src/AttestationDisputeGame.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";
import { BondManager } from "src/BondManager.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";

/// @title
contract AttestationDisputeGame_Test is Test {
    bytes32 constant TYPE_HASH = 0x2676994b0652bcdf7968635d15b78aac9aaf797cc94c5adeb94376cc28f987d6;

    DisputeGameFactory factory;
    BondManager bm;
    AttestationDisputeGame disputeGameImplementation;
    SystemConfig systemConfig;
    L2OutputOracle l2oo;
    AttestationDisputeGame disputeGameProxy;

    // L2OutputOracle Constructor arguments
    address internal proposer = 0x000000000000000000000000000000000000AbBa;
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal submissionInterval = 1800;
    uint256 internal l2BlockTime = 1;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 2;

    // SystemConfig `signerSet` keys
    uint256[] signerKeys;

    /// @notice Emitted when a new dispute game is created by the [DisputeGameFactory]
    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    function setUp() public {
        // vm.warp(startingTimestamp);

        factory = new DisputeGameFactory(address(this));
        vm.label(address(factory), "DisputeGameFactory");
        bm = new BondManager(factory);
        vm.label(address(bm), "BondManager");

        ResourceMetering.ResourceConfig memory _config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 1000000000,
            elasticityMultiplier: 2,
            baseFeeMaxChangeDenominator: 2,
            minimumBaseFee: 10,
            systemTxMaxGas: 100000000,
            maximumBaseFee: 1000
        });

        systemConfig = new SystemConfig(
            address(this), // _owner,
            100, // _overhead,
            100, // _scalar,
            keccak256("BATCHER.HASH"), // _batcherHash,
            type(uint64).max, // _gasLimit,
            address(0), // _unsafeBlockSigner,
            _config
        );
        vm.label(address(systemConfig), "SystemConfig");

        // Add 5 signers to the signer set
        for (uint256 i = 1; i < 6; i++) {
            signerKeys.push(i);
            systemConfig.authenticateSigner(vm.addr(i), true);
        }
        systemConfig.setSignatureThreshold(5);

        l2oo = new L2OutputOracle({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _finalizationPeriodSeconds: 7 days,
            _bondManager: IBondManager(address(bm)),
            _disputeGameFactory: IDisputeGameFactory(address(factory))
        });
        vm.label(address(l2oo), "L2OutputOracle");

        // Create the dispute game implementation
        disputeGameImplementation = new AttestationDisputeGame(IBondManager(address(bm)), systemConfig, l2oo);
        vm.label(address(disputeGameImplementation), "AttestationDisputeGame_Implementation");

        // Set the implementation in the factory
        GameType gt = GameType.ATTESTATION;
        factory.setImplementation(gt, IDisputeGame(address(disputeGameImplementation)));

        // Create the attestation dispute game in the factory
        bytes memory extraData = hex"";
        Claim rootClaim = Claim.wrap(bytes32(0));
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        disputeGameProxy = AttestationDisputeGame(address(factory.create(gt, rootClaim, extraData)));
        assertEq(address(factory.games(gt, rootClaim, extraData)), address(disputeGameProxy));
        vm.label(address(disputeGameProxy), "AttestationDisputeGame_Proxy");
    }

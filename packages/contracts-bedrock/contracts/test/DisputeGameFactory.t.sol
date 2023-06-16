// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { Test } from "forge-std/Test.sol";
import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";
import { IDisputeGame } from "../dispute/interfaces/IDisputeGame.sol";
import { Proxy } from "../universal/Proxy.sol";

contract DisputeGameFactory_Init is Test {
    DisputeGameFactory factory;
    FakeClone fakeClone;

    event DisputeGameCreated(
        address indexed disputeProxy,
        GameType indexed gameType,
        Claim indexed rootClaim
    );

    event ImplementationSet(address indexed impl, GameType indexed gameType);

    function setUp() public virtual {
        Proxy proxy = new Proxy(address(this));
        DisputeGameFactory impl = new DisputeGameFactory();

        proxy.upgradeToAndCall({
            _implementation: address(impl),
            _data: abi.encodeCall(impl.initialize, (address(this)))
        });
        factory = DisputeGameFactory(address(proxy));
        vm.label(address(factory), "DisputeGameFactoryProxy");

        fakeClone = new FakeClone();
    }
}

contract DisputeGameFactory_Create_Test is DisputeGameFactory_Init {
    /**
     * @dev Tests that the `create` function succeeds when creating a new dispute game
     *      with a `GameType` that has an implementation set.
     */
    function testFuzz_create_succeeds(
        uint8 gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));

        // Set all three implementations to the same `FakeClone` contract.
        for (uint8 i; i < 3; i++) {
            factory.setImplementation(GameType.wrap(i), IDisputeGame(address(fakeClone)));
        }

        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        IDisputeGame proxy = factory.create(gt, rootClaim, extraData);

        (IDisputeGame game, uint256 timestamp) = factory.games(gt, rootClaim, extraData);

        // Ensure that the dispute game was assigned to the `disputeGames` mapping.
        assertEq(address(game), address(proxy));
        assertEq(timestamp, block.timestamp);
        assertEq(factory.gameCount(), 1);

        (IDisputeGame game2, uint256 timestamp2) = factory.gameAtIndex(0);
        assertEq(address(game2), address(proxy));
        assertEq(timestamp2, block.timestamp);
    }

    /**
     * @dev Tests that the `create` function reverts when there is no implementation
     *      set for the given `GameType`.
     */
    function testFuzz_create_noImpl_reverts(
        uint8 gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));

        vm.expectRevert(abi.encodeWithSelector(NoImplementation.selector, gt));
        factory.create(gt, rootClaim, extraData);
    }

    /**
     * @dev Tests that the `create` function reverts when there exists a dispute game with the same UUID.
     */
    function testFuzz_create_sameUUID_reverts(
        uint8 gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));

        // Set all three implementations to the same `FakeClone` contract.
        for (uint8 i; i < 3; i++) {
            factory.setImplementation(GameType.wrap(i), IDisputeGame(address(fakeClone)));
        }

        // Create our first dispute game - this should succeed.
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        IDisputeGame proxy = factory.create(gt, rootClaim, extraData);

        (IDisputeGame game, uint256 timestamp) = factory.games(gt, rootClaim, extraData);
        // Ensure that the dispute game was assigned to the `disputeGames` mapping.
        assertEq(address(game), address(proxy));
        assertEq(timestamp, block.timestamp);

        // Ensure that the `create` function reverts when called with parameters that would result in the same UUID.
        vm.expectRevert(
            abi.encodeWithSelector(
                GameAlreadyExists.selector,
                factory.getGameUUID(gt, rootClaim, extraData)
            )
        );
        factory.create(gt, rootClaim, extraData);
    }
}

contract DisputeGameFactory_SetImplementation_Test is DisputeGameFactory_Init {
    /**
     * @dev Tests that the `setImplementation` function properly sets the implementation for a given `GameType`.
     */
    function test_setImplementation_succeeds() public {
        // There should be no implementation for the `GameTypes.FAULT` enum value, it has not been set.
        assertEq(address(factory.gameImpls(GameTypes.FAULT)), address(0));

        vm.expectEmit(true, true, true, true, address(factory));
        emit ImplementationSet(address(1), GameTypes.FAULT);

        // Set the implementation for the `GameTypes.FAULT` enum value.
        factory.setImplementation(GameTypes.FAULT, IDisputeGame(address(1)));

        // Ensure that the implementation for the `GameTypes.FAULT` enum value is set.
        assertEq(address(factory.gameImpls(GameTypes.FAULT)), address(1));
    }

    /**
     * @dev Tests that the `setImplementation` function reverts when called by a non-owner.
     */
    function test_setImplementation_notOwner_reverts() public {
        // Ensure that the `setImplementation` function reverts when called by a non-owner.
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        factory.setImplementation(GameTypes.FAULT, IDisputeGame(address(1)));
    }
}

contract DisputeGameFactory_GetGameUUID_Test is DisputeGameFactory_Init {
    /**
     * @dev Tests that the `getGameUUID` function returns the correct hash when comparing
     *      against the keccak256 hash of the abi-encoded parameters.
     */
    function testDiff_getGameUUID_succeeds(
        uint8 gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));

        assertEq(
            Hash.unwrap(factory.getGameUUID(gt, rootClaim, extraData)),
            keccak256(abi.encode(gt, rootClaim, extraData))
        );
    }
}

contract DisputeGameFactory_Owner_Test is DisputeGameFactory_Init {
    /**
     * @dev Tests that the `owner` function returns the correct address after deployment.
     */
    function test_owner_succeeds() public {
        assertEq(factory.owner(), address(this));
    }
}

contract DisputeGameFactory_TransferOwnership_Test is DisputeGameFactory_Init {
    /**
     * @dev Tests that the `transferOwnership` function succeeds when called by the owner.
     */
    function test_transferOwnership_succeeds() public {
        factory.transferOwnership(address(1));
        assertEq(factory.owner(), address(1));
    }

    /**
     * @dev Tests that the `transferOwnership` function reverts when called by a non-owner.
     */
    function test_transferOwnership_notOwner_reverts() public {
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        factory.transferOwnership(address(1));
    }
}

/**
 * @title PackingTester
 * @notice Exposes the internal packing functions so that they can be fuzzed
 *         in a roundtrip manner.
 */
contract PackingTester is DisputeGameFactory {
    function packSlot(address _addr, uint256 _num) external pure returns (GameId) {
        return _packSlot(_addr, _num);
    }

    function unpackSlot(GameId _slot) external pure returns (address, uint256) {
        return _unpackSlot(_slot);
    }
}

/**
 * @title DisputeGameFactory_PackSlot_Test
 * @notice Fuzzes the PackingTester contract
 */
contract DisputeGameFactory_PackSlot_Test is Test {
    PackingTester tester;

    function setUp() public {
        tester = new PackingTester();
    }

    /**
     * @dev Tests that the `packSlot` and `unpackSlot` functions roundtrip correctly.
     */
    function testFuzz_packSlot_succeeds(address _addr, uint96 _num) public {
        GameId slot = tester.packSlot(_addr, uint256(_num));
        (address addr, uint256 num) = tester.unpackSlot(slot);
        assertEq(addr, _addr);
        assertEq(num, _num);
    }
}

/**
 * @dev A fake clone used for testing the `DisputeGameFactory` contract's `create` function.
 */
contract FakeClone {
    function initialize() external {
        // noop
    }
}

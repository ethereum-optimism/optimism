// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { SetDisputeGameImpl, SetDisputeGameImplInput } from "scripts/deploy/SetDisputeGameImpl.s.sol";

// Contracts
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { ETHLockbox } from "src/L1/ETHLockbox.sol";

// Libraries
import { GameType, Proposal, Hash } from "src/dispute/lib/Types.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IPauseSource } from "interfaces/L1/IPauseSource.sol";

contract SetDisputeGameImplInput_Test is Test {
    SetDisputeGameImplInput input;

    function setUp() public {
        input = new SetDisputeGameImplInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("SetDisputeGameImplInput: not set");
        input.factory();

        vm.expectRevert("SetDisputeGameImplInput: not set");
        input.impl();

        // gameType doesn't revert when not set, returns 0
        assertEq(input.gameType(), 0);
    }

    function test_set_succeeds() public {
        address factory = makeAddr("factory");
        address impl = makeAddr("impl");
        uint32 gameType = 1;

        vm.etch(factory, hex"01");
        vm.etch(impl, hex"01");

        input.set(input.factory.selector, factory);
        input.set(input.impl.selector, impl);
        input.set(input.gameType.selector, gameType);

        assertEq(address(input.factory()), factory);
        assertEq(address(input.impl()), impl);
        assertEq(input.gameType(), gameType);
    }

    function test_set_withZeroAddress_reverts() public {
        vm.expectRevert("SetDisputeGameImplInput: cannot set zero address");
        input.set(input.factory.selector, address(0));

        vm.expectRevert("SetDisputeGameImplInput: cannot set zero address");
        input.set(input.impl.selector, address(0));
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("SetDisputeGameImplInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));

        vm.expectRevert("SetDisputeGameImplInput: unknown selector");
        input.set(bytes4(0xdeadbeef), uint32(1));
    }
}

contract SetDisputeGameImpl_Test is Test {
    SetDisputeGameImpl script;
    SetDisputeGameImplInput input;
    IDisputeGameFactory factory;
    IAnchorStateRegistry anchorStateRegistry;
    address mockImpl;
    uint32 gameType;

    function setUp() public {
        script = new SetDisputeGameImpl();
        input = new SetDisputeGameImplInput();
        DisputeGameFactory dgfImpl = new DisputeGameFactory();
        SuperchainConfig supConfigImpl = new SuperchainConfig();
        AnchorStateRegistry anchorStateRegistryImpl = new AnchorStateRegistry(0);
        ETHLockbox ethLockboxImpl = new ETHLockbox();

        Proxy supConfigProxy = new Proxy(address(1));
        vm.prank(address(1));
        supConfigProxy.upgradeToAndCall(
            address(supConfigImpl), abi.encodeCall(supConfigImpl.initialize, (address(this)))
        );

        Proxy ethLockboxProxy = new Proxy(address(1));
        IOptimismPortal2[] memory portals = new IOptimismPortal2[](0);
        vm.prank(address(1));
        ethLockboxProxy.upgradeToAndCall(
            address(ethLockboxImpl),
            abi.encodeCall(IETHLockbox.initialize, (ISuperchainConfig(address(supConfigProxy)), portals))
        );

        Proxy factoryProxy = new Proxy(address(1));
        vm.prank(address(1));
        factoryProxy.upgradeToAndCall(address(dgfImpl), abi.encodeCall(dgfImpl.initialize, (address(this))));
        factory = IDisputeGameFactory(address(factoryProxy));

        Proxy anchorStateRegistryProxy = new Proxy(address(1));
        vm.prank(address(1));
        anchorStateRegistryProxy.upgradeToAndCall(
            address(anchorStateRegistryImpl),
            abi.encodeCall(
                IAnchorStateRegistry.initialize,
                (
                    IPauseSource(address(ethLockboxProxy)),
                    factory,
                    Proposal({ root: Hash.wrap(0), l2SequenceNumber: 0 }),
                    GameType.wrap(100)
                )
            )
        );
        anchorStateRegistry = IAnchorStateRegistry(address(anchorStateRegistryProxy));

        mockImpl = makeAddr("impl");
        gameType = 999;
    }

    function test_run_succeeds() public {
        input.set(input.factory.selector, address(factory));
        input.set(input.impl.selector, mockImpl);
        input.set(input.anchorStateRegistry.selector, address(anchorStateRegistry));
        input.set(input.gameType.selector, gameType);

        script.run(input);
    }

    function test_run_whenImplAlreadySet_reverts() public {
        input.set(input.factory.selector, address(factory));
        input.set(input.impl.selector, mockImpl);
        input.set(input.anchorStateRegistry.selector, address(anchorStateRegistry));
        input.set(input.gameType.selector, gameType);

        // First run should succeed
        script.run(input);

        // Subsequent runs should revert
        vm.expectRevert("SDGI-10");
        script.run(input);
    }

    function test_assertValid_whenNotValid_reverts() public {
        input.set(input.factory.selector, address(factory));
        input.set(input.impl.selector, mockImpl);
        input.set(input.gameType.selector, gameType);

        // First run should succeed
        script.run(input);

        vm.broadcast(address(this));
        factory.setImplementation(GameType.wrap(gameType), IDisputeGame(address(0)));

        vm.expectRevert("SDGI-30");
        script.assertValid(input);
    }
}

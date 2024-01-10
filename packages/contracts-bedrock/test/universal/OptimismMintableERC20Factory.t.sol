// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { NextImpl } from "test/mocks/NextImpl.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Target contract dependencies
import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Target contract
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

contract OptimismMintableTokenFactory_Test is Bridge_Initializer {
    event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);
    event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer);

    /// @dev Tests that the constructor is initialized correctly.
    function test_constructor_succeeds() external {
        OptimismMintableERC20Factory impl = new OptimismMintableERC20Factory();
        assertEq(address(impl.BRIDGE()), address(0));
        assertEq(address(impl.bridge()), address(0));
    }

    /// @dev Tests that the proxy is initialized correctly.
    function test_initialize_succeeds() external {
        assertEq(address(l1OptimismMintableERC20Factory.BRIDGE()), address(l1StandardBridge));
        assertEq(address(l1OptimismMintableERC20Factory.bridge()), address(l1StandardBridge));
    }

    function test_upgrading_succeeds() external {
        Proxy proxy = Proxy(deploy.mustGetAddress("OptimismMintableERC20FactoryProxy"));
        // Check an unused slot before upgrading.
        bytes32 slot21Before = vm.load(address(l1OptimismMintableERC20Factory), bytes32(uint256(21)));
        assertEq(bytes32(0), slot21Before);

        NextImpl nextImpl = new NextImpl();
        vm.startPrank(EIP1967Helper.getAdmin(address(proxy)));
        // Reviewer note: the NextImpl() still uses reinitializer. If we want to remove that, we'll need to use a
        //   two step upgrade with the Storage lib.
        proxy.upgradeToAndCall(address(nextImpl), abi.encodeWithSelector(NextImpl.initialize.selector, 2));
        assertEq(proxy.implementation(), address(nextImpl));

        // Verify that the NextImpl contract initialized its values according as expected
        bytes32 slot21After = vm.load(address(l1OptimismMintableERC20Factory), bytes32(uint256(21)));
        bytes32 slot21Expected = NextImpl(address(l1OptimismMintableERC20Factory)).slot21Init();
        assertEq(slot21Expected, slot21After);
    }

    function test_createStandardL2Token_succeeds() external {
        address remote = address(4);

        // Defaults to 18 decimals
        address local = calculateTokenAddress(remote, "Beep", "BOOP", 18);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        address addr = l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
        assertTrue(addr == local);
        assertTrue(OptimismMintableERC20(local).decimals() == 18);
    }

    function test_createStandardL2TokenWithDecimals_succeeds() external {
        address remote = address(4);
        address local = calculateTokenAddress(remote, "Beep", "BOOP", 6);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        address addr = l2OptimismMintableERC20Factory.createOptimismMintableERC20WithDecimals(remote, "Beep", "BOOP", 6);
        assertTrue(addr == local);

        assertTrue(OptimismMintableERC20(local).decimals() == 6);
    }

    function test_createStandardL2Token_sameTwice_reverts() external {
        address remote = address(4);

        vm.prank(alice);
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");

        vm.expectRevert();

        vm.prank(alice);
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2Token_remoteIsZero_reverts() external {
        address remote = address(0);
        vm.expectRevert("OptimismMintableERC20Factory: must provide remote token address");
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function calculateTokenAddress(
        address _remote,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        internal
        view
        returns (address)
    {
        bytes memory constructorArgs = abi.encode(address(l2StandardBridge), _remote, _name, _symbol, _decimals);
        bytes memory bytecode = abi.encodePacked(type(OptimismMintableERC20).creationCode, constructorArgs);
        bytes32 salt = keccak256(abi.encode(_remote, _name, _symbol, _decimals));
        bytes32 hash = keccak256(
            abi.encodePacked(bytes1(0xff), address(l2OptimismMintableERC20Factory), salt, keccak256(bytecode))
        );
        return address(uint160(uint256(hash)));
    }
}

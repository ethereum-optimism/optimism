// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract dependencies
import { L1BlockInterop, ConfigType } from "src/L2/L1BlockInterop.sol";

contract L1BlockInteropTest is CommonTest {
    error NotDepositor();
    error NotDependency();
    error DependencySetSizeTooLarge();
    error CantRemovedChainId();

    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);
    event DependencyAdded(uint256 indexed chainId);
    event DependencyRemoved(uint256 indexed chainId);

    modifier prankDepositor() {
        vm.startPrank(l1Block.DEPOSITOR_ACCOUNT());
        _;
    }

    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @dev Tests that an arbitrary chain ID can be added to the dependency set.
    function testFuzz_isInDependencySet_succeeds(uint256 _chainId) public prankDepositor {
        // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf);

        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));

        assertTrue(l1Block.isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns true when the chain's chain ID is passed as the input.
    function test_isInDependencySet_chainChainId_succeeds() public view {
        assertTrue(l1Block.isInDependencySet(block.chainid));
    }

    /// @dev Tests that `isInDependencySet` reverts when the input chain ID is not in the dependency set
    ///      and is not the chain's chain ID.
    function testFuzz_isInDependencySet_notDependency_reverts(uint256 _chainId) public view {
        vm.assume(_chainId != block.chainid);

        // Check that the chain ID is not in the dependency set
        assertFalse(l1Block.isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns false when the dependency set is empty.
    function testFuzz_isInDependencySet_dependencySetEmpty_succeeds(uint256 _chainId) public view {
        vm.assume(_chainId != block.chainid);

        assertEq(l1Block.dependencySetSize(), 0);

        assertFalse(l1Block.isInDependencySet(_chainId));
    }

    /// @dev Tests that the dependency set size is correct when adding an arbitrary number of chain IDs.
    function testFuzz_dependencySetSize_succeeds(uint256[] calldata _dependencySet) public prankDepositor {
        for (uint256 i = 0; i < _dependencySet.length; i++) {
            // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
            vm.assume(_dependencySet[i] != 0xfbb67fda52d4bfb8bf);
            l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_dependencySet[i]));
        }

        // Count the number of unique items in _dependencySet to compare with the dependency set size,
        // since the dependency set is a set and should not contain duplicates
        uint256 uniqueCount = 0;
        bool found;
        for (uint256 i = 0; i < _dependencySet.length; i++) {
            found = false;
            for (uint256 j = 0; j < i; j++) {
                if (_dependencySet[i] == _dependencySet[j]) {
                    found = true;
                    break;
                }
            }
            if (!found) {
                uniqueCount++;
            }
        }

        assertEq(l1Block.dependencySetSize(), uniqueCount);
    }

    /// @dev Tests that the dependency set size is correct when the dependency set is empty.
    function test_dependencySetSize_dependencySetEmpty_succeeds() public view {
        assertEq(l1Block.dependencySetSize(), 0);
    }

    /// @dev Tests that the config for the gas paying token can be set.
    function testFuzz_setConfig_gasPayingToken_succeeds(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        public
        prankDepositor
    {
        vm.expectEmit(address(l1Block));
        emit GasPayingTokenSet({ token: _token, decimals: _decimals, name: _name, symbol: _symbol });

        l1Block.setConfig(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, _decimals, _name, _symbol));
    }

    /// @dev Tests that setting the gas paying token config as not the depositor reverts.
    function testFuzz_setConfig_gasPayingToken_notDepositor_reverts(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        public
    {
        vm.expectRevert(NotDepositor.selector);
        l1Block.setConfig(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, _decimals, _name, _symbol));
    }

    /// @dev Tests that the config for adding a dependency can be set.
    function testFuzz_setConfig_addDependency_succeeds(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL

        vm.expectEmit(address(l1Block));
        emit DependencyAdded(_chainId);

        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config as not the depositor reverts.
    function testFuzz_setConfig_addDependency_notDepositor_reverts(uint256 _chainId) public {
        vm.expectRevert(NotDepositor.selector);
        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config when the dependency set size is too large reverts.
    function test_setConfig_addDependency_DependencySetSizeTooLarge_reverts() public prankDepositor {
        for (uint256 i = 0; i < type(uint8).max; i++) {
            l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(i));
        }

        assertEq(l1Block.dependencySetSize(), type(uint8).max);

        vm.expectRevert(DependencySetSizeTooLarge.selector);
        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(1));
    }

    /// @dev Tests that the config for removing a dependency can be set.
    function testFuzz_setConfig_removeDependency_succeeds(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL

        // Add the chain ID to the dependency set before removing it
        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));

        vm.expectEmit(address(l1Block));
        emit DependencyRemoved(_chainId);

        l1Block.setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the remove dependency config as not the depositor reverts.
    function testFuzz_setConfig_removeDependency_notDepositor_reverts(uint256 _chainId) public {
        vm.expectRevert(NotDepositor.selector);
        l1Block.setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the remove dependency config for the chain's chain ID reverts.
    function test_setConfig_removeDependency_chainChainId_reverts() public prankDepositor {
        vm.expectRevert(CantRemovedChainId.selector);
        l1Block.setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(block.chainid));
    }

    /// @dev Tests that setting the remove dependency config for a chain ID that is not in the dependency set reverts.
    function testFuzz_setConfig_removeDependency_notDependency_reverts(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
        vm.assume(_chainId != block.chainid);

        vm.expectRevert(NotDependency.selector);
        l1Block.setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }
}

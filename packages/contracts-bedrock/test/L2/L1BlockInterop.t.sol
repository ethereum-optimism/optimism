// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract dependencies
import { L1BlockInterop, ConfigType } from "src/L2/L1BlockInterop.sol";

contract L1BlockInteropTest is CommonTest {
    /// @notice Thrown when a non-depositor account attempts to set L1 block values.
    error NotDepositor();

    /// @notice Error when a chain ID is not in the interop dependency set.
    error NotDependency();

    /// @notice Error when the interop dependency set size is too large.
    error DependencySetSizeTooLarge();

    /// @notice Error when the chain's chain ID is attempted to be removed from the interop dependency set.
    error CantRemovedChainId();

    /// @notice Event emitted when the gas paying token is set.
    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);

    /// @notice Event emitted when a new dependency is added to the interop dependency set.
    event DependencyAdded(uint256 indexed chainId);

    /// @notice Event emitted when a dependency is removed from the interop dependency set.
    event DependencyRemoved(uint256 indexed chainId);

    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function setUp() public virtual override {
        super.setUp();
        vm.etch(address(l1Block), address(new L1BlockInterop()).code);
    }

    /// @dev Tests that an arbitrary chain ID can be added to the dependency set.
    function testFuzz_isInDependencySet_succeeds(uint256 _chainId) public {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf);

        vm.prank(_l1BlockInterop().DEPOSITOR_ACCOUNT());
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));

        assertTrue(_l1BlockInterop().isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns true when the chain's chain ID is passed as the input.
    function test_isInDependencySet_chainChainId_succeeds() public view {
        assertTrue(_l1BlockInterop().isInDependencySet(block.chainid));
    }

    /// @dev Tests that `isInDependencySet` reverts when the input chain ID is not in the dependency set
    ///      and is not the chain's chain ID.
    function testFuzz_isInDependencySet_notDependency_reverts(uint256 _chainId) public view {
        vm.assume(_chainId != block.chainid);

        // Check that the chain ID is not in the dependency set
        assertFalse(_l1BlockInterop().isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns false when the dependency set is empty.
    function testFuzz_isInDependencySet_dependencySetEmpty_succeeds(uint256 _chainId) public view {
        vm.assume(_chainId != block.chainid);

        assertTrue(_l1BlockInterop().dependencySetSize() == 0);

        assertFalse(_l1BlockInterop().isInDependencySet(_chainId));
    }

    /// @dev Tests that the dependency set size is correct when adding an arbitrary number of chain IDs.
    function testFuzz_dependencySetSize_succeeds(uint256[] calldata _dependencySet) public {
        vm.startPrank(_l1BlockInterop().DEPOSITOR_ACCOUNT());

        for (uint256 i = 0; i < _dependencySet.length; i++) {
            // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
            vm.assume(_dependencySet[i] != 0xfbb67fda52d4bfb8bf);
            _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_dependencySet[i]));
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

        assertEq(_l1BlockInterop().dependencySetSize(), uniqueCount);
    }

    /// @dev Tests that the dependency set size is correct when the dependency set is empty.
    function test_dependencySetSize_dependencySetEmpty_succeeds() public view {
        assertTrue(_l1BlockInterop().dependencySetSize() == 0);
    }

    /// @dev Tests that the config for the gas paying token can be set.
    function testFuzz_setConfig_gasPayingToken_succeeds(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        public
    {
        vm.expectEmit(address(l1Block));
        emit GasPayingTokenSet({ token: _token, decimals: _decimals, name: _name, symbol: _symbol });

        vm.prank(_l1BlockInterop().DEPOSITOR_ACCOUNT());
        _l1BlockInterop().setConfig(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, _decimals, _name, _symbol));
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
        _l1BlockInterop().setConfig(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, _decimals, _name, _symbol));
    }

    /// @dev Tests that the config for adding a dependency can be set.
    function testFuzz_setConfig_addDependency_succeeds(uint256 _chainId) public {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL

        vm.expectEmit(address(l1Block));
        emit DependencyAdded(_chainId);

        vm.prank(_l1BlockInterop().DEPOSITOR_ACCOUNT());
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config as not the depositor reverts.
    function testFuzz_setConfig_addDependency_notDepositor_reverts(uint256 _chainId) public {
        vm.expectRevert(NotDepositor.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config when the dependency set size is too large reverts.
    function test_setConfig_addDependency_DependencySetSizeTooLarge_reverts() public {
        vm.startPrank(_l1BlockInterop().DEPOSITOR_ACCOUNT());

        for (uint256 i = 0; i < type(uint8).max; i++) {
            _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(i));
        }

        assertEq(_l1BlockInterop().dependencySetSize(), type(uint8).max);

        vm.expectRevert(DependencySetSizeTooLarge.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(1));
    }

    /// @dev Tests that the config for removing a dependency can be set.
    function testFuzz_setConfig_removeDependency_succeeds(uint256 _chainId) public {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL

        vm.startPrank(_l1BlockInterop().DEPOSITOR_ACCOUNT());

        // Add the chain ID to the dependency set before removing it
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));

        vm.expectEmit(address(l1Block));
        emit DependencyRemoved(_chainId);

        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the remove dependency config as not the depositor reverts.
    function testFuzz_setConfig_removeDependency_notDepositor_reverts(uint256 _chainId) public {
        vm.expectRevert(NotDepositor.selector);
        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the remove dependency config for the chain's chain ID reverts.
    function test_setConfig_removeDependency_chainChainId_reverts() public {
        vm.startPrank(_l1BlockInterop().DEPOSITOR_ACCOUNT());
        vm.expectRevert(CantRemovedChainId.selector);
        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(block.chainid));
    }

    /// @dev Tests that setting the remove dependency config for a chain ID that is not in the dependency set reverts.
    function testFuzz_setConfig_removeDependency_notDependency_reverts(uint256 _chainId) public {
        vm.assume(_chainId != 0xfbb67fda52d4bfb8bf); // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
        vm.assume(_chainId != block.chainid);

        vm.startPrank(_l1BlockInterop().DEPOSITOR_ACCOUNT());
        vm.expectRevert(NotDependency.selector);
        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Returns the L1BlockInterop instance.
    function _l1BlockInterop() internal view returns (L1BlockInterop) {
        return L1BlockInterop(address(l1Block));
    }
}

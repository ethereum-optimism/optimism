// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract dependencies
import { L1BlockInterop, ConfigType } from "src/L2/L1BlockInterop.sol";
import "src/libraries/L1BlockErrors.sol";

contract L1BlockInteropTest is CommonTest {
    event GasPayingTokenSet(address indexed token, uint8 indexed decimals, bytes32 name, bytes32 symbol);
    event DependencyAdded(uint256 indexed chainId);
    event DependencyRemoved(uint256 indexed chainId);

    modifier prankDepositor() {
        vm.startPrank(l1Block.DEPOSITOR_ACCOUNT());
        _;
        vm.stopPrank();
    }

    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();
    }

    /// @dev Tests that an arbitrary chain ID can be added to the dependency set.
    function testFuzz_isInDependencySet_succeeds(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != block.chainid);

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

        assertEq(_l1BlockInterop().dependencySetSize(), 0);

        assertFalse(_l1BlockInterop().isInDependencySet(_chainId));
    }

    /// @dev Tests that the dependency set size is correct when adding an arbitrary number of chain IDs.
    function testFuzz_dependencySetSize_succeeds(uint8 _dependencySetSize) public prankDepositor {
        vm.assume(_dependencySetSize <= type(uint8).max);

        uint256 uniqueCount = 0;

        for (uint256 i = 0; i < _dependencySetSize; i++) {
            if (i == block.chainid) continue;
            _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(i));
            uniqueCount++;
        }

        assertEq(_l1BlockInterop().dependencySetSize(), uniqueCount);
    }

    /// @dev Tests that the dependency set size is correct when the dependency set is empty.
    function test_dependencySetSize_dependencySetEmpty_succeeds() public view {
        assertEq(_l1BlockInterop().dependencySetSize(), 0);
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
    function testFuzz_setConfig_addDependency_succeeds(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != block.chainid);

        vm.expectEmit(address(l1Block));
        emit DependencyAdded(_chainId);

        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    function test_setConfig_addDependency_chainChainId_reverts() public prankDepositor {
        vm.expectRevert(AlreadyDependency.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(block.chainid));
    }

    function test_setConfig_addDependency_AlreadyDependency_reverts(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != block.chainid);

        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));

        vm.expectRevert(AlreadyDependency.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config as not the depositor reverts.
    function testFuzz_setConfig_addDependency_notDepositor_reverts(uint256 _chainId) public {
        vm.expectRevert(NotDepositor.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Tests that setting the add dependency config when the dependency set size is too large reverts.
    function test_setConfig_addDependency_DependencySetSizeTooLarge_reverts() public prankDepositor {
        for (uint256 i = 0; i < type(uint8).max; i++) {
            _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(i));
        }

        assertEq(_l1BlockInterop().dependencySetSize(), type(uint8).max);

        vm.expectRevert(DependencySetSizeTooLarge.selector);
        _l1BlockInterop().setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(1));
    }

    /// @dev Tests that the config for removing a dependency can be set.
    function testFuzz_setConfig_removeDependency_succeeds(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != block.chainid);

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
    function test_setConfig_removeDependency_chainChainId_reverts() public prankDepositor {
        vm.expectRevert(CantRemovedDependency.selector);
        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(block.chainid));
    }

    /// @dev Tests that setting the remove dependency config for a chain ID that is not in the dependency set reverts.
    function testFuzz_setConfig_removeDependency_notDependency_reverts(uint256 _chainId) public prankDepositor {
        vm.assume(_chainId != block.chainid);

        vm.expectRevert(NotDependency.selector);
        _l1BlockInterop().setConfig(ConfigType.REMOVE_DEPENDENCY, abi.encode(_chainId));
    }

    /// @dev Returns the L1BlockInterop instance.
    function _l1BlockInterop() internal view returns (L1BlockInterop) {
        return L1BlockInterop(address(l1Block));
    }
}

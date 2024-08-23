// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { CREATE3, Bytes32AddressLib } from "@rari-capital/solmate/src/utils/CREATE3.sol";
import { IBeacon } from "@openzeppelin/contracts-v5/proxy/beacon/IBeacon.sol";

// Target contract
import { OptimismSuperchainERC20Factory, OptimismSuperchainERC20 } from "src/L2/OptimismSuperchainERC20Factory.sol";

/// @title OptimismSuperchainERC20FactoryTest
/// @notice Contract for testing the OptimismSuperchainERC20Factory contract.
contract OptimismSuperchainERC20FactoryTest is Test {
    using Bytes32AddressLib for bytes32;

    OptimismSuperchainERC20 public superchainERC20Impl;
    OptimismSuperchainERC20Factory public superchainERC20Factory;

    /// @notice Sets up the test suite.
    function setUp() public {
        superchainERC20Impl = new OptimismSuperchainERC20();

        // Deploy the OptimismSuperchainERC20Beacon contract
        _deployBeacon();

        superchainERC20Factory = new OptimismSuperchainERC20Factory();
    }

    /// @notice Deploy the OptimismSuperchainERC20Beacon predeploy contract
    function _deployBeacon() internal {
        // Deploy the OptimismSuperchainERC20Beacon implementation
        address _addr = Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON;
        address _impl = Predeploys.predeployToCodeNamespace(_addr);
        vm.etch(_impl, vm.getDeployedCode("OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon"));

        // Deploy the ERC1967Proxy contract at the Predeploy
        bytes memory code = vm.getDeployedCode("universal/Proxy.sol:Proxy");
        vm.etch(_addr, code);
        EIP1967Helper.setAdmin(_addr, Predeploys.PROXY_ADMIN);
        EIP1967Helper.setImplementation(_addr, _impl);

        // Mock implementation address
        vm.mockCall(
            _impl, abi.encodeWithSelector(IBeacon.implementation.selector), abi.encode(address(superchainERC20Impl))
        );
    }

    /// @notice Test that calling `deploy` with valid parameters succeeds.
    function test_deploy_succeeds(
        address _caller,
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        public
    {
        // Arrange
        bytes32 salt = keccak256(abi.encode(_remoteToken, _name, _symbol, _decimals));
        address deployment = _calculateTokenAddress(salt, address(superchainERC20Factory));

        vm.expectEmit(address(superchainERC20Factory));
        emit OptimismSuperchainERC20Factory.OptimismSuperchainERC20Created(deployment, _remoteToken, _caller);

        // Act
        vm.prank(_caller);
        address addr = superchainERC20Factory.deploy(_remoteToken, _name, _symbol, _decimals);

        // Assert
        assertTrue(addr == deployment);
        assertTrue(OptimismSuperchainERC20(deployment).decimals() == _decimals);
        assertTrue(OptimismSuperchainERC20(deployment).remoteToken() == _remoteToken);
        assertEq(OptimismSuperchainERC20(deployment).name(), _name);
        assertEq(OptimismSuperchainERC20(deployment).symbol(), _symbol);
        assertEq(superchainERC20Factory.deployments(deployment), _remoteToken);
    }

    /// @notice Test that calling `deploy` with the same parameters twice reverts.
    function test_deploy_sameTwice_reverts(
        address _caller,
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        external
    {
        // Arrange
        vm.prank(_caller);
        superchainERC20Factory.deploy(_remoteToken, _name, _symbol, _decimals);

        vm.expectRevert(bytes("DEPLOYMENT_FAILED"));

        // Act
        vm.prank(_caller);
        superchainERC20Factory.deploy(_remoteToken, _name, _symbol, _decimals);
    }

    /// @notice Precalculates the address of the token contract using CREATE3.
    function _calculateTokenAddress(bytes32 _salt, address _deployer) internal pure returns (address) {
        address proxy =
            keccak256(abi.encodePacked(bytes1(0xFF), _deployer, _salt, CREATE3.PROXY_BYTECODE_HASH)).fromLast20Bytes();

        return keccak256(abi.encodePacked(hex"d694", proxy, hex"01")).fromLast20Bytes();
    }
}

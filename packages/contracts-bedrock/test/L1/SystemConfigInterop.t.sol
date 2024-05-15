// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

// Target contract dependencies
import { SystemConfigInterop, NoCode } from "src/L1/SystemConfigInterop.sol";
import { ConfigType } from "src/L2/L1BlockInterop.sol";
import { OptimismPortalInterop } from "src/L1/OptimismPortalInterop.sol";

contract SystemConfigWithSetGasPayingToken is SystemConfigInterop {
    /// @notice External method to set the gas paying token.
    /// @param _token Address of the token to set as the gas paying token.
    function setGasPayingToken(address _token) external {
        _setGasPayingToken(_token);
    }
}

contract SystemConfigInterop_Test is CommonTest {
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function setUp() public virtual override {
        super.setUp();
        vm.etch(address(systemConfig), address(new SystemConfigWithSetGasPayingToken()).code);
        vm.etch(address(optimismPortal), address(new OptimismPortalInterop()).code);

        // Set OptimismPortal address in SystemConfig
        _setOptimismPortalAddress(address(optimismPortal));
    }

    /// @dev Tests that the gas paying token can be set.
    function testFuzz_setGasPayingToken_succeeds(
        address _token,
        string calldata _name,
        string calldata _symbol
    )
        public
    {
        vm.assume(_token != address(0));
        vm.assume(_token != Constants.ETHER);
        vm.assume(bytes(_name).length <= 32);
        vm.assume(bytes(_symbol).length <= 32);

        vm.mockCall(_token, abi.encodeWithSelector(ERC20.decimals.selector), abi.encode(18));
        vm.mockCall(_token, abi.encodeWithSelector(ERC20.name.selector), abi.encode(_name));
        vm.mockCall(_token, abi.encodeWithSelector(ERC20.symbol.selector), abi.encode(_symbol));

        vm.mockCall(
            address(optimismPortal),
            abi.encodeWithSelector(OptimismPortalInterop.setConfig.selector),
            abi.encode(ConfigType.GAS_PAYING_TOKEN, abi.encode(_token, 18, _name, _symbol))
        );

        _systemConfigInterop().setGasPayingToken(_token);
    }

    /// @dev Tests that setting the gas paying token with no code at the OptimismPortal address reverts.
    function testFuzz_setGasPayingToken_optimismPortalNoCode_reverts(address _noCodeAddress, address _token) public {
        vm.assume(_noCodeAddress.code.length == 0);
        vm.assume(_token != address(0));
        vm.assume(_token != Constants.ETHER);

        // Set OptimismPortal address in SystemConfig to an address with no code
        _setOptimismPortalAddress(_noCodeAddress);

        vm.expectRevert(abi.encodeWithSelector(NoCode.selector, _noCodeAddress));
        _systemConfigInterop().setGasPayingToken(_token);
    }

    /// @dev Tests that a dependency can be added.
    function testFuzz_addDependency_succeeds(uint256 _chainId) public {
        vm.mockCall(
            address(optimismPortal),
            abi.encodeWithSelector(OptimismPortalInterop.setConfig.selector),
            abi.encode(ConfigType.GAS_PAYING_TOKEN, abi.encode(_chainId))
        );

        vm.prank(systemConfig.owner());
        _systemConfigInterop().addDependency(_chainId);
    }

    /// @dev Tests that adding a dependency as not the owner reverts.
    function testFuzz_addDependency_notOwner_reverts(uint256 _chainId) public {
        vm.expectRevert("Ownable: caller is not the owner");
        _systemConfigInterop().addDependency(_chainId);
    }

    /// @dev Tests that adding a dependency with no code at the OptimismPortal address reverts.
    function testFuzz_addDependency_optimismPortalNoCode_reverts(address _noCodeAddress, uint256 _chainId) public {
        vm.assume(_noCodeAddress.code.length == 0);

        // Set OptimismPortal address in SystemConfig to an address with no code
        _setOptimismPortalAddress(_noCodeAddress);

        vm.prank(systemConfig.owner());
        vm.expectRevert(abi.encodeWithSelector(NoCode.selector, _noCodeAddress));
        _systemConfigInterop().addDependency(_chainId);
    }

    /// @dev Tests that a dependency can be removed.
    function testFuzz_removeDependency_succeeds(uint256 _chainId) public {
        vm.mockCall(
            address(optimismPortal),
            abi.encodeWithSelector(OptimismPortalInterop.setConfig.selector),
            abi.encode(ConfigType.GAS_PAYING_TOKEN, abi.encode(_chainId))
        );

        vm.prank(systemConfig.owner());
        _systemConfigInterop().removeDependency(_chainId);
    }

    /// @dev Tests that removing a dependency as not the owner reverts.
    function testFuzz_removeDependency_notOwner_reverts(uint256 _chainId) public {
        vm.expectRevert("Ownable: caller is not the owner");
        _systemConfigInterop().removeDependency(_chainId);
    }

    /// @dev Tests that removing a dependency with no code at the OptimismPortal address reverts.
    function testFuzz_removeDependency_optimismPortalNoCode_reverts(address _noCodeAddress, uint256 _chainId) public {
        vm.assume(_noCodeAddress.code.length == 0);

        // Set OptimismPortal address in SystemConfig to an address with no code
        _setOptimismPortalAddress(_noCodeAddress);

        vm.prank(systemConfig.owner());
        vm.expectRevert(abi.encodeWithSelector(NoCode.selector, _noCodeAddress));
        _systemConfigInterop().removeDependency(_chainId);
    }

    /// @dev Returns the SystemConfigWithSetGasPayingToken instance.
    function _systemConfigInterop() internal view returns (SystemConfigWithSetGasPayingToken) {
        return SystemConfigWithSetGasPayingToken(address(systemConfig));
    }

    /// @dev Sets the OptimismPortal address in the SystemConfig contract.
    function _setOptimismPortalAddress(address _address) internal {
        vm.store(
            address(systemConfig), bytes32(systemConfig.OPTIMISM_PORTAL_SLOT()), bytes32(uint256(uint160(_address)))
        );
    }
}

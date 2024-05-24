// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { StaticConfig } from "src/libraries/StaticConfig.sol";
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";

// Target contract dependencies
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";
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
        super.enableInterop();
        super.setUp();
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
        // don't use vm's address
        vm.assume(_token != address(vm));
        // don't use console's address
        vm.assume(_token != CONSOLE);
        // don't use create2 deployer's address
        vm.assume(_token != CREATE2_FACTORY);
        // don't use default test's address
        vm.assume(_token != DEFAULT_TEST_CONTRACT);
        // don't use multicall3's address
        vm.assume(_token != MULTICALL3_ADDRESS);

        vm.assume(bytes(_name).length <= 32);
        vm.assume(bytes(_symbol).length <= 32);

        vm.mockCall(_token, abi.encodeWithSelector(ERC20.decimals.selector), abi.encode(18));
        vm.mockCall(_token, abi.encodeWithSelector(ERC20.name.selector), abi.encode(_name));
        vm.mockCall(_token, abi.encodeWithSelector(ERC20.symbol.selector), abi.encode(_symbol));

        vm.expectCall(
            address(optimismPortal),
            abi.encodeCall(
                OptimismPortalInterop.setConfig,
                (
                    ConfigType.GAS_PAYING_TOKEN,
                    StaticConfig.encodeSetGasPayingToken({
                        _token: _token,
                        _decimals: 18,
                        _name: GasPayingToken.sanitize(_name),
                        _symbol: GasPayingToken.sanitize(_symbol)
                    })
                )
            )
        );

        _systemConfigWithSetGasPayingToken().setGasPayingToken(_token);
    }

    /// @dev Tests that a dependency can be added.
    function testFuzz_addDependency_succeeds(uint256 _chainId) public {
        vm.expectCall(
            address(optimismPortal),
            abi.encodeCall(
                OptimismPortalInterop.setConfig, (ConfigType.ADD_DEPENDENCY, StaticConfig.encodeAddDependency(_chainId))
            )
        );

        vm.prank(systemConfig.owner());
        _systemConfigInterop().addDependency(_chainId);
    }

    /// @dev Tests that adding a dependency as not the owner reverts.
    function testFuzz_addDependency_notOwner_reverts(uint256 _chainId) public {
        vm.expectRevert("Ownable: caller is not the owner");
        _systemConfigInterop().addDependency(_chainId);
    }

    /// @dev Tests that a dependency can be removed.
    function testFuzz_removeDependency_succeeds(uint256 _chainId) public {
        vm.expectCall(
            address(optimismPortal),
            abi.encodeCall(
                OptimismPortalInterop.setConfig,
                (ConfigType.REMOVE_DEPENDENCY, StaticConfig.encodeRemoveDependency(_chainId))
            )
        );

        vm.prank(systemConfig.owner());
        _systemConfigInterop().removeDependency(_chainId);
    }

    /// @dev Tests that removing a dependency as not the owner reverts.
    function testFuzz_removeDependency_notOwner_reverts(uint256 _chainId) public {
        vm.expectRevert("Ownable: caller is not the owner");
        _systemConfigInterop().removeDependency(_chainId);
    }

    /// @dev Returns the SystemConfigInterop instance.
    function _systemConfigInterop() internal view returns (SystemConfigInterop) {
        return SystemConfigInterop(address(systemConfig));
    }

    /// @dev Returns the SystemConfigWithSetGasPayingToken instance.
    function _systemConfigWithSetGasPayingToken() internal returns (SystemConfigWithSetGasPayingToken) {
        vm.etch(address(systemConfig), address(new SystemConfigWithSetGasPayingToken()).code);

        return SystemConfigWithSetGasPayingToken(address(systemConfig));
    }
}

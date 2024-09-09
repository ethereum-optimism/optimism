// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IERC20 } from "@openzeppelin/contracts-v5/token/ERC20/IERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts-v5/proxy/ERC1967/ERC1967Proxy.sol";
import { Initializable } from "@openzeppelin/contracts-v5/proxy/utils/Initializable.sol";
import { IERC165 } from "@openzeppelin/contracts-v5/utils/introspection/IERC165.sol";

// Target contract
import {
    OptimismSuperchainERC20,
    IOptimismSuperchainERC20Extension,
    CallerNotL2ToL2CrossDomainMessenger,
    InvalidCrossDomainSender,
    OnlyBridge,
    ZeroAddress
} from "src/L2/OptimismSuperchainERC20.sol";
import { ISuperchainERC20Extensions } from "src/L2/interfaces/ISuperchainERC20.sol";

/// @title OptimismSuperchainERC20Test
/// @notice Contract for testing the OptimismSuperchainERC20 contract.
contract OptimismSuperchainERC20Test is Test {
    address internal constant ZERO_ADDRESS = address(0);
    address internal constant REMOTE_TOKEN = address(0x123);
    string internal constant NAME = "OptimismSuperchainERC20";
    string internal constant SYMBOL = "SCE";
    uint8 internal constant DECIMALS = 18;
    address internal constant BRIDGE = Predeploys.L2_STANDARD_BRIDGE;
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    OptimismSuperchainERC20 public superchainERC20Impl;
    OptimismSuperchainERC20 public superchainERC20;

    /// @notice Sets up the test suite.
    function setUp() public {
        superchainERC20Impl = new OptimismSuperchainERC20();
        superchainERC20 = _deploySuperchainERC20Proxy(REMOTE_TOKEN, NAME, SYMBOL, DECIMALS);
    }

    /// @notice Helper function to deploy a proxy of the OptimismSuperchainERC20 contract.
    function _deploySuperchainERC20Proxy(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        internal
        returns (OptimismSuperchainERC20)
    {
        return OptimismSuperchainERC20(
            address(
                // TODO: Use the SuperchainERC20 Beacon Proxy
                new ERC1967Proxy(
                    address(superchainERC20Impl),
                    abi.encodeCall(OptimismSuperchainERC20.initialize, (_remoteToken, _name, _symbol, _decimals))
                )
            )
        );
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Test that the contract's `initializer` sets the correct values.
    function test_initializer_succeeds() public view {
        assertEq(superchainERC20.name(), NAME);
        assertEq(superchainERC20.symbol(), SYMBOL);
        assertEq(superchainERC20.decimals(), DECIMALS);
        assertEq(superchainERC20.remoteToken(), REMOTE_TOKEN);
    }

    /// @notice Tests the `initialize` function reverts when the contract is already initialized.
    function testFuzz_initializer_reverts(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        public
    {
        // Expect the revert with `InvalidInitialization` selector
        vm.expectRevert(Initializable.InvalidInitialization.selector);

        // Call the `initialize` function again
        superchainERC20.initialize(_remoteToken, _name, _symbol, _decimals);
    }

    /// @notice Tests the `mint` function reverts when the caller is not the bridge.
    function testFuzz_mint_callerNotBridge_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != BRIDGE);

        // Expect the revert with `OnlyBridge` selector
        vm.expectRevert(OnlyBridge.selector);

        // Call the `mint` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC20.mint(_to, _amount);
    }

    /// @notice Tests the `mint` function reverts when the amount is zero.
    function testFuzz_mint_zeroAddressTo_reverts(uint256 _amount) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        // Call the `mint` function with the zero address
        vm.prank(BRIDGE);
        superchainERC20.mint({ _to: ZERO_ADDRESS, _amount: _amount });
    }

    /// @notice Tests the `mint` succeeds and emits the `Mint` event.
    function testFuzz_mint_succeeds(address _to, uint256 _amount) public {
        // Ensure `_to` is not the zero address
        vm.assume(_to != ZERO_ADDRESS);

        // Get the total supply and balance of `_to` before the mint to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _toBalanceBefore = superchainERC20.balanceOf(_to);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IERC20.Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `Mint` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IOptimismSuperchainERC20Extension.Mint(_to, _amount);

        // Call the `mint` function with the bridge caller
        vm.prank(BRIDGE);
        superchainERC20.mint(_to, _amount);

        // Check the total supply and balance of `_to` after the mint were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore + _amount);
        assertEq(superchainERC20.balanceOf(_to), _toBalanceBefore + _amount);
    }

    /// @notice Tests the `burn` function reverts when the caller is not the bridge.
    function testFuzz_burn_callerNotBridge_reverts(address _caller, address _from, uint256 _amount) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != BRIDGE);

        // Expect the revert with `OnlyBridge` selector
        vm.expectRevert(OnlyBridge.selector);

        // Call the `burn` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC20.burn(_from, _amount);
    }

    /// @notice Tests the `burn` function reverts when the amount is zero.
    function testFuzz_burn_zeroAddressFrom_reverts(uint256 _amount) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        // Call the `burn` function with the zero address
        vm.prank(BRIDGE);
        superchainERC20.burn({ _from: ZERO_ADDRESS, _amount: _amount });
    }

    /// @notice Tests the `burn` burns the amount and emits the `Burn` event.
    function testFuzz_burn_succeeds(address _from, uint256 _amount) public {
        // Ensure `_from` is not the zero address
        vm.assume(_from != ZERO_ADDRESS);

        // Mint some tokens to `_from` so then they can be burned
        vm.prank(BRIDGE);
        superchainERC20.mint(_from, _amount);

        // Get the total supply and balance of `_from` before the burn to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _fromBalanceBefore = superchainERC20.balanceOf(_from);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IERC20.Transfer(_from, ZERO_ADDRESS, _amount);

        // Look for the emit of the `Burn` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IOptimismSuperchainERC20Extension.Burn(_from, _amount);

        // Call the `burn` function with the bridge caller
        vm.prank(BRIDGE);
        superchainERC20.burn(_from, _amount);

        // Check the total supply and balance of `_from` after the burn were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore - _amount);
        assertEq(superchainERC20.balanceOf(_from), _fromBalanceBefore - _amount);
    }

    /// @notice Tests the `sendERC20` function reverts when the `_to` address is the zero address.
    function testFuzz_sendERC20_zeroAddressTo_reverts(uint256 _amount, uint256 _chainId) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        // Call the `sendERC20` function with the zero address
        vm.prank(BRIDGE);
        superchainERC20.sendERC20({ _to: ZERO_ADDRESS, _amount: _amount, _chainId: _chainId });
    }

    /// @notice Tests the `sendERC20` function burns the sender tokens, sends the message, and emits the `SendERC20`
    /// event.
    function testFuzz_sendERC20_succeeds(address _sender, address _to, uint256 _amount, uint256 _chainId) external {
        // Ensure `_sender` is not the zero address
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);

        // Mint some tokens to the sender so then they can be sent
        vm.prank(BRIDGE);
        superchainERC20.mint(_sender, _amount);

        // Get the total supply and balance of `_sender` before the send to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _senderBalanceBefore = superchainERC20.balanceOf(_sender);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IERC20.Transfer(_sender, ZERO_ADDRESS, _amount);

        // Look for the emit of the `SendERC20` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit ISuperchainERC20Extensions.SendERC20(_sender, _to, _amount, _chainId);

        // Mock the call over the `sendMessage` function and expect it to be called properly
        bytes memory _message = abi.encodeCall(superchainERC20.relayERC20, (_sender, _to, _amount));
        _mockAndExpect(
            MESSENGER,
            abi.encodeWithSelector(
                IL2ToL2CrossDomainMessenger.sendMessage.selector, _chainId, address(superchainERC20), _message
            ),
            abi.encode("")
        );

        // Call the `sendERC20` function
        vm.prank(_sender);
        superchainERC20.sendERC20(_to, _amount, _chainId);

        // Check the total supply and balance of `_sender` after the send were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore - _amount);
        assertEq(superchainERC20.balanceOf(_sender), _senderBalanceBefore - _amount);
    }

    /// @notice Tests the `relayERC20` function reverts when the caller is not the L2ToL2CrossDomainMessenger.
    function testFuzz_relayERC20_notMessenger_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the messenger
        vm.assume(_caller != MESSENGER);
        vm.assume(_to != ZERO_ADDRESS);

        // Expect the revert with `CallerNotL2ToL2CrossDomainMessenger` selector
        vm.expectRevert(CallerNotL2ToL2CrossDomainMessenger.selector);

        // Call the `relayERC20` function with the non-messenger caller
        vm.prank(_caller);
        superchainERC20.relayERC20(_caller, _to, _amount);
    }

    /// @notice Tests the `relayERC20` function reverts when the `crossDomainMessageSender` that sent the message is not
    /// the same SuperchainERC20 address.
    function testFuzz_relayERC20_notCrossDomainSender_reverts(
        address _crossDomainMessageSender,
        address _to,
        uint256 _amount
    )
        public
    {
        vm.assume(_to != ZERO_ADDRESS);
        vm.assume(_crossDomainMessageSender != address(superchainERC20));

        // Mock the call over the `crossDomainMessageSender` function setting a wrong sender
        vm.mockCall(
            MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.crossDomainMessageSender.selector),
            abi.encode(_crossDomainMessageSender)
        );

        // Expect the revert with `InvalidCrossDomainSender` selector
        vm.expectRevert(InvalidCrossDomainSender.selector);

        // Call the `relayERC20` function with the sender caller
        vm.prank(MESSENGER);
        superchainERC20.relayERC20(_crossDomainMessageSender, _to, _amount);
    }

    /// @notice Tests the `relayERC20` function reverts when the `_to` address is the zero address.
    function testFuzz_relayERC20_zeroAddressTo_reverts(uint256 _amount) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        // Mock the call over the `crossDomainMessageSender` function setting the same address as value
        vm.mockCall(
            MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.crossDomainMessageSender.selector),
            abi.encode(address(superchainERC20))
        );

        // Call the `relayERC20` function with the zero address
        vm.prank(MESSENGER);
        superchainERC20.relayERC20({ _from: ZERO_ADDRESS, _to: ZERO_ADDRESS, _amount: _amount });
    }

    /// @notice Tests the `relayERC20` mints the proper amount and emits the `RelayERC20` event.
    function testFuzz_relayERC20_succeeds(address _from, address _to, uint256 _amount, uint256 _source) public {
        vm.assume(_from != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);

        // Mock the call over the `crossDomainMessageSender` function setting the same address as value
        _mockAndExpect(
            MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.crossDomainMessageSender.selector),
            abi.encode(address(superchainERC20))
        );

        // Mock the call over the `crossDomainMessageSource` function setting the source chain ID as value
        _mockAndExpect(
            MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.crossDomainMessageSource.selector),
            abi.encode(_source)
        );

        // Get the total supply and balance of `_to` before the relay to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _toBalanceBefore = superchainERC20.balanceOf(_to);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit IERC20.Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `RelayERC20` event
        vm.expectEmit(true, true, true, true, address(superchainERC20));
        emit ISuperchainERC20Extensions.RelayERC20(_from, _to, _amount, _source);

        // Call the `relayERC20` function with the messenger caller
        vm.prank(MESSENGER);
        superchainERC20.relayERC20(_from, _to, _amount);

        // Check the total supply and balance of `_to` after the relay were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore + _amount);
        assertEq(superchainERC20.balanceOf(_to), _toBalanceBefore + _amount);
    }

    /// @notice Tests the `decimals` function always returns the correct value.
    function testFuzz_decimals_succeeds(uint8 _decimals) public {
        OptimismSuperchainERC20 _newSuperchainERC20 = _deploySuperchainERC20Proxy(REMOTE_TOKEN, NAME, SYMBOL, _decimals);
        assertEq(_newSuperchainERC20.decimals(), _decimals);
    }

    /// @notice Tests the `REMOTE_TOKEN` function always returns the correct value.
    function testFuzz_remoteToken_succeeds(address _remoteToken) public {
        OptimismSuperchainERC20 _newSuperchainERC20 = _deploySuperchainERC20Proxy(_remoteToken, NAME, SYMBOL, DECIMALS);
        assertEq(_newSuperchainERC20.remoteToken(), _remoteToken);
    }

    /// @notice Tests the `name` function always returns the correct value.
    function testFuzz_name_succeeds(string memory _name) public {
        OptimismSuperchainERC20 _newSuperchainERC20 = _deploySuperchainERC20Proxy(REMOTE_TOKEN, _name, SYMBOL, DECIMALS);
        assertEq(_newSuperchainERC20.name(), _name);
    }

    /// @notice Tests the `symbol` function always returns the correct value.
    function testFuzz_symbol_succeeds(string memory _symbol) public {
        OptimismSuperchainERC20 _newSuperchainERC20 = _deploySuperchainERC20Proxy(REMOTE_TOKEN, NAME, _symbol, DECIMALS);
        assertEq(_newSuperchainERC20.symbol(), _symbol);
    }

    /// @notice Tests that the `supportsInterface` function returns true for the `IOptimismSuperchainERC20` interface.
    function test_supportInterface_succeeds() public view {
        assertTrue(superchainERC20.supportsInterface(type(IERC165).interfaceId));
        assertTrue(superchainERC20.supportsInterface(type(IOptimismSuperchainERC20Extension).interfaceId));
    }

    /// @notice Tests that the `supportsInterface` function returns false for any other interface than the
    /// `IOptimismSuperchainERC20` one.
    function testFuzz_supportInterface_returnFalse(bytes4 _interfaceId) public view {
        vm.assume(_interfaceId != type(IERC165).interfaceId);
        vm.assume(_interfaceId != type(IOptimismSuperchainERC20Extension).interfaceId);
        assertFalse(superchainERC20.supportsInterface(_interfaceId));
    }
}

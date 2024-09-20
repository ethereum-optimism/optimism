// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IERC20 } from "@openzeppelin/contracts-v5/token/ERC20/IERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";

// Target contract
import { SuperchainERC20 } from "src/L2/SuperchainERC20.sol";
import { ISuperchainERC20Extensions, ISuperchainERC20Errors } from "src/L2/interfaces/ISuperchainERC20.sol";

/// @notice Mock contract for the SuperchainERC20 contract so tests can mint tokens.
contract SuperchainERC20Mock is SuperchainERC20 {
    string private _name;
    string private _symbol;
    uint8 private _decimals;

    constructor(string memory __name, string memory __symbol, uint8 __decimals) {
        _name = __name;
        _symbol = __symbol;
        _decimals = __decimals;
    }

    function mint(address _account, uint256 _amount) public {
        _mint(_account, _amount);
    }

    function name() public view virtual override returns (string memory) {
        return _name;
    }

    function symbol() public view virtual override returns (string memory) {
        return _symbol;
    }

    function decimals() public view virtual override returns (uint8) {
        return _decimals;
    }
}
/// @title SuperchainERC20Test
/// @notice Contract for testing the SuperchainERC20 contract.

contract SuperchainERC20Test is Test {
    address internal constant ZERO_ADDRESS = address(0);
    string internal constant NAME = "SuperchainERC20";
    string internal constant SYMBOL = "SCE";
    uint8 internal constant DECIMALS = 18;
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    SuperchainERC20 public superchainERC20Impl;
    SuperchainERC20Mock public superchainERC20;

    /// @notice Sets up the test suite.
    function setUp() public {
        superchainERC20 = new SuperchainERC20Mock(NAME, SYMBOL, DECIMALS);
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Test that the contract's `constructor` sets the correct values.
    function test_constructor_succeeds() public view {
        assertEq(superchainERC20.name(), NAME);
        assertEq(superchainERC20.symbol(), SYMBOL);
        assertEq(superchainERC20.decimals(), DECIMALS);
    }

    /// @notice Tests the `sendERC20` function reverts when the `_to` address is the zero address.
    function testFuzz_sendERC20_zeroAddressTo_reverts(uint256 _amount, uint256 _chainId) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ISuperchainERC20Errors.ZeroAddress.selector);

        // Call the `sendERC20` function with the zero address
        superchainERC20.sendERC20({ _to: ZERO_ADDRESS, _amount: _amount, _chainId: _chainId });
    }

    /// @notice Tests the `sendERC20` function burns the sender tokens, sends the message, and emits the `SendERC20`
    /// event.
    function testFuzz_sendERC20_succeeds(address _sender, address _to, uint256 _amount, uint256 _chainId) external {
        // Ensure `_sender` is not the zero address
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);

        // Mint some tokens to the sender so then they can be sent
        superchainERC20.mint(_sender, _amount);

        // Get the total supply and balance of `_sender` before the send to compare later on the assertions
        uint256 _totalSupplyBefore = superchainERC20.totalSupply();
        uint256 _senderBalanceBefore = superchainERC20.balanceOf(_sender);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainERC20));
        emit IERC20.Transfer(_sender, ZERO_ADDRESS, _amount);

        // Look for the emit of the `SendERC20` event
        vm.expectEmit(address(superchainERC20));
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
        vm.expectRevert(ISuperchainERC20Errors.CallerNotL2ToL2CrossDomainMessenger.selector);

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
        vm.expectRevert(ISuperchainERC20Errors.InvalidCrossDomainSender.selector);

        // Call the `relayERC20` function with the sender caller
        vm.prank(MESSENGER);
        superchainERC20.relayERC20(_crossDomainMessageSender, _to, _amount);
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
        vm.expectEmit(address(superchainERC20));
        emit IERC20.Transfer(ZERO_ADDRESS, _to, _amount);

        // Look for the emit of the `RelayERC20` event
        vm.expectEmit(address(superchainERC20));
        emit ISuperchainERC20Extensions.RelayERC20(_from, _to, _amount, _source);

        // Call the `relayERC20` function with the messenger caller
        vm.prank(MESSENGER);
        superchainERC20.relayERC20(_from, _to, _amount);

        // Check the total supply and balance of `_to` after the relay were updated correctly
        assertEq(superchainERC20.totalSupply(), _totalSupplyBefore + _amount);
        assertEq(superchainERC20.balanceOf(_to), _toBalanceBefore + _amount);
    }

    /// @notice Tests the `name` function always returns the correct value.
    function testFuzz_name_succeeds(string memory _name) public {
        SuperchainERC20 _newSuperchainERC20 = new SuperchainERC20Mock(_name, SYMBOL, DECIMALS);
        assertEq(_newSuperchainERC20.name(), _name);
    }

    /// @notice Tests the `symbol` function always returns the correct value.
    function testFuzz_symbol_succeeds(string memory _symbol) public {
        SuperchainERC20 _newSuperchainERC20 = new SuperchainERC20Mock(NAME, _symbol, DECIMALS);
        assertEq(_newSuperchainERC20.symbol(), _symbol);
    }

    /// @notice Tests the `decimals` function always returns the correct value.
    function testFuzz_decimals_succeeds(uint8 _decimals) public {
        SuperchainERC20 _newSuperchainERC20 = new SuperchainERC20Mock(NAME, SYMBOL, _decimals);
        assertEq(_newSuperchainERC20.decimals(), _decimals);
    }
}

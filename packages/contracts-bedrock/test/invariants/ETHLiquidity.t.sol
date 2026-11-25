// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";

/// @title ETHLiquidity_User
/// @notice Actor contract that interacts with the ETHLiquidity contract. Always pretends to be the
///         SuperchainETHBridge contract since it's the only contract that can mint/burn ETH
///         liquidity. Tracks the ETH that flows into and out of the ETHLiquidity contract via ghost
///         variables so the invariant can assert exact conservation.
contract ETHLiquidity_User is StdUtils {
    /// @notice The Vm contract.
    Vm internal vm;

    /// @notice The ETHLiquidity contract.
    IETHLiquidity internal liquidity;

    /// @notice Total ETH released from the ETHLiquidity contract via mint (withdrawals).
    uint256 public totalMinted;

    /// @notice Total ETH locked into the ETHLiquidity contract via burn (deposits).
    uint256 public totalBurned;

    /// @notice Total ETH added to the ETHLiquidity contract via fund.
    uint256 public totalFunded;

    /// @param _vm The Vm contract.
    /// @param _liquidity The ETHLiquidity contract.
    /// @param _balance Starting balance for the bridge (source of burns) and this actor (source of funds).
    constructor(Vm _vm, IETHLiquidity _liquidity, uint256 _balance) {
        vm = _vm;
        liquidity = _liquidity;
        // The bridge is the source of burns; this actor is the source of funds.
        vm.deal(Predeploys.SUPERCHAIN_ETH_BRIDGE, _balance);
        vm.deal(address(this), _balance);
    }

    /// @notice Mint ETH liquidity (releases ETH from ETHLiquidity to the bridge).
    /// @param _amount The amount of ETH to mint.
    function mint(uint256 _amount) public {
        // Bound to the available liquidity so the withdrawal never reverts.
        _amount = bound(_amount, 0, address(liquidity).balance);
        vm.prank(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        liquidity.mint(_amount);
        totalMinted += _amount;
    }

    /// @notice Burn ETH liquidity (locks ETH from the bridge into ETHLiquidity).
    /// @param _amount The amount of ETH to burn.
    function burn(uint256 _amount) public {
        // burn pulls value from the (pranked) bridge, so bound to the bridge's balance.
        _amount = bound(_amount, 0, Predeploys.SUPERCHAIN_ETH_BRIDGE.balance);
        vm.prank(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        liquidity.burn{ value: _amount }();
        totalBurned += _amount;
    }

    /// @notice Fund ETH liquidity (adds external ETH to ETHLiquidity).
    /// @param _amount The amount of ETH to fund.
    function fund(uint256 _amount) public {
        // fund pulls value from this actor; bound to its balance and skip zero (which reverts).
        _amount = bound(_amount, 0, address(this).balance);
        if (_amount == 0) return;
        liquidity.fund{ value: _amount }();
        totalFunded += _amount;
    }
}

/// @title ETHLiquidity_MintBurn_Invariant
/// @notice Invariant that checks ETH is conserved across mint/burn/fund: the ETHLiquidity balance
///         always equals its starting balance plus deposits (burns + funds) minus withdrawals
///         (mints). mint/burn/fund only move ETH between accounts; they never create or destroy it.
contract ETHLiquidity_MintBurn_Invariant is CommonTest {
    /// @notice Starting balance handed to the bridge and the actor as spending money.
    uint256 internal constant STARTING_BALANCE = type(uint128).max;

    /// @notice The ETHLiquidity_User actor.
    ETHLiquidity_User internal actor;

    /// @notice ETHLiquidity balance captured at setup; conservation is measured against this.
    uint256 internal startingLiquidity;

    /// @notice Test setup.
    function setUp() public override {
        super.enableInterop();
        super.setUp();

        // Create a new ETHLiquidity_User actor.
        actor = new ETHLiquidity_User(vm, ethLiquidity, STARTING_BALANCE);

        // Capture the liquidity the contract was pre-loaded with at genesis.
        startingLiquidity = address(ethLiquidity).balance;

        // Set the target contract.
        targetContract(address(actor));

        // Set the target selectors.
        bytes4[] memory selectors = new bytes4[](3);
        selectors[0] = actor.mint.selector;
        selectors[1] = actor.burn.selector;
        selectors[2] = actor.fund.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(actor), selectors: selectors });
        targetSelector(selector);
    }

    /// @notice Invariant that checks ETH is conserved across mint/burn/fund.
    /// @custom:invariant The ETHLiquidity balance equals its starting balance plus all deposits
    ///                   (burns + funds) minus all withdrawals (mints).
    function invariant_mintBurnFund_conservesLiquidity() public view {
        assertEq(
            address(ethLiquidity).balance,
            startingLiquidity + actor.totalBurned() + actor.totalFunded() - actor.totalMinted()
        );
    }
}

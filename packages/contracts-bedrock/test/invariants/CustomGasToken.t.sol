// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { StdUtils } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Contracts
import { LiquidityController } from "src/L2/LiquidityController.sol";
import { NativeAssetLiquidity } from "src/L2/NativeAssetLiquidity.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { INativeAssetLiquidity } from "interfaces/L2/INativeAssetLiquidity.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @title CGT_Minter
/// @notice
contract LiquidityController_Minter is StdUtils {
    /// @notice Flag to indicate if the test has failed.
    bool public failed = false;

    /// @notice The Vm contract.
    Vm internal vm;

    /// @notice The LiquidityController contract.
    ILiquidityController internal liquidityController;

    /// @notice The RandomActor contract.
    RandomActor internal randomActor;

    /// @notice Ghost acconting
    uint256 public mintAmount;
    uint256 public burnAmount;

    bool public deltaBalanceMint;
    bool public deltaBalanceBurn;

    /// @param _vm The Vm contract.
    /// @param _liquidityController The LiquidityController contract.
    constructor(Vm _vm, ILiquidityController _liquidityController, RandomActor _randomActor) {
        vm = _vm;
        liquidityController = _liquidityController;
        randomActor = _randomActor;
    }

    function mint(uint256 _amount) public {
        mintAmount += _amount;

        uint256 _preBalance = payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance;

        liquidityController.mint(address(randomActor), _amount);

        deltaBalanceMint = _amount != (_preBalance - uint256(payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance));
    }

    function burn(uint256 _amount) public {
        _amount = bound(_amount, 0, address(this).balance);
        burnAmount += _amount;

        uint256 _preBalance = payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance;
        liquidityController.burn{ value: _amount }();
        deltaBalanceBurn = _preBalance + _amount != uint256(payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance);
    }

    receive() external payable { }
}

contract NativeAssetLiquidity_Fundooor is StdUtils {
    Vm internal vm;
    INativeAssetLiquidity internal nativeAssetLiquidity;

    /// @notice Flag to indicate if the test has failed.
    bool public failed = false;

    /// @notice Ghost accounting
    uint256 public fundAmount;

    constructor(Vm _vm) {
        vm = _vm;
        nativeAssetLiquidity = INativeAssetLiquidity(Predeploys.NATIVE_ASSET_LIQUIDITY);
    }

    function fund(uint256 _amount) public {
        _amount = bound(_amount, 0, address(this).balance);
        fundAmount += _amount;
        nativeAssetLiquidity.fund{ value: _amount }();
    }

    receive() external payable { }
}

/// @notice actor which receives fund and send them to either the minter or the funder actor,
///         keeping a closed loop (no vm.deal). It receive() function always revert, to insure mint()/safeSend is
///         always successfully sending the CGT.
contract RandomActor is StdUtils {
    Vm internal vm;
    address internal liquidityController_Minter;
    address internal nativeAssetLiquidity_Fundooor;

    /// @notice Flag to indicate if the actor has been called via receive()
    bool public hasBeenCalled = false;

    function initAddresses(address _liquidityController_Minter, address _nativeAssetLiquidity_Fundooor) public {
        liquidityController_Minter = _liquidityController_Minter;
        nativeAssetLiquidity_Fundooor = _nativeAssetLiquidity_Fundooor;
    }

    function sendCGTtoMinter(uint256 _amount) public {
        uint256 _amountToSend = bound(_amount, 0, address(this).balance);
        (bool success,) = payable(address(liquidityController_Minter)).call{ value: _amountToSend }("");
        require(success);
    }

    function sendCGTtoFunder(uint256 _amount) public {
        uint256 _amountToSend = bound(_amount, 0, address(this).balance);
        (bool success,) = payable(address(nativeAssetLiquidity_Fundooor)).call{ value: _amountToSend }("");
        require(success);
    }

    receive() external payable {
        hasBeenCalled = true;
    }
}

/// @title ETHLiquidity_MintBurn_Invariant
/// @notice Invariant that checks that the NativeAssetLiquidity contract's balance is always equal
///         to the sum of the initial supply, the deposits, the funds, and minus the withdrawals.
///         NAL Balance = Initial Supply + Deposits + Funds - Withdrawals
contract CustomGasToken_Invariants is CommonTest {
    /// @notice Starting balance of the contract - arbitrary value
    uint256 internal constant STARTING_BALANCE = type(uint248).max / 5;

    LiquidityController_Minter internal actor_minter;
    NativeAssetLiquidity_Fundooor internal actor_funder;
    RandomActor internal randomActor;

    /// @notice Test setup.
    function setUp() public override {
        enableCustomGasToken();
        super.setUp();

        randomActor = new RandomActor();
        actor_funder = new NativeAssetLiquidity_Fundooor(vm);
        actor_minter = new LiquidityController_Minter(vm, liquidityController, randomActor);

        randomActor.initAddresses(address(actor_minter), address(actor_funder));

        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        liquidityController.authorizeMinter(address(actor_minter));

        // Create the initial supply
        vm.deal(address(nativeAssetLiquidity), STARTING_BALANCE);

        // Set the target contract.
        targetContract(address(actor_minter));
        targetContract(address(actor_funder));

        // Set the target selectors.
        bytes4[] memory selectors = new bytes4[](2);
        selectors[0] = RandomActor.sendCGTtoMinter.selector;
        selectors[1] = RandomActor.sendCGTtoFunder.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(randomActor), selectors: selectors });
        targetSelector(selector);
    }

    /// @notice Invariant that checks that the NativeAssetLiquidity contract's balance is always equal
    ///         to the sum of the initial supply, the deposits, the funds, and minus the withdrawals.
    ///         NAL Balance = Initial Supply + Deposits + Funds - Withdrawals
    /// @custom:invariant No call sequence of mint, burn and fund should induce an accounting error.
    /// @dev liquidityController.burn() calls deposit, liquidityController.mint() calls withdraw
    function invariant_supplyConservation() public view {
        assertEq(
            address(nativeAssetLiquidity).balance,
            STARTING_BALANCE + actor_funder.fundAmount() + actor_minter.burnAmount() - actor_minter.mintAmount(),
            "NativeAssetLiquidity balance is not equal to the sum of the initial supply, the deposits, the funds, and minus the withdrawals"
        );
    }

    /// @notice Invariant that checks that the minted amount is equal to the withdrawn amount
    /// @dev Checks if the amount minted equals the amount transferred outside the NativeAssetLiquidity contract
    function invariant_mintedEqualsWithdrawn() public view {
        assertFalse(actor_minter.deltaBalanceMint(), "Minted amount is not equal to the withdrawn amount");
    }

    function invariant_burnedEqualsDeposited() public view {
        assertFalse(actor_minter.deltaBalanceBurn(), "Burned amount is not equal to the deposited amount");
    }

    function invariant_noDustLiquidityController() public view {
        assertEq(address(liquidityController).balance, 0, "LiquidityController balance is not 0");
    }

    function invariant_mintNeverCallsBack() public view {
        assertFalse(randomActor.hasBeenCalled(), "RandomActor receive() function has been triggered");
    }
}

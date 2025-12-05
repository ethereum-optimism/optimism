// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { StdUtils } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Features } from "src/libraries/Features.sol";
import { SafeSend } from "src/universal/SafeSend.sol";

// Contracts
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { INativeAssetLiquidity } from "interfaces/L2/INativeAssetLiquidity.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @title CGT_Minter
/// @notice An actor with the minter role (can mint and burn)
contract LiquidityController_Minter is StdUtils {
    /// @notice The Vm contract.
    Vm internal vm;

    /// @notice The LiquidityController contract.
    ILiquidityController internal liquidityController;

    /// @notice The RandomActor contract.
    RandomActor internal randomActor;

    /// @notice Ghost accounting
    uint256 public totalAmountMinted;
    uint256 public totalAmountBurned;
    bool public deltaBalanceAndMint; // NativeAssetLiquidity balance change != amount minted?
    bool public deltaBalanceAndBurn; // NativeAssetLiquidity balance change != amount burned?

    /// @param _vm The Vm contract.
    /// @param _liquidityController The LiquidityController contract.
    /// @param _randomActor The RandomActor contract.
    constructor(Vm _vm, ILiquidityController _liquidityController, RandomActor _randomActor) {
        vm = _vm;
        liquidityController = _liquidityController;
        randomActor = _randomActor;
    }

    /// @notice Mint custom gas token to the random actor.
    /// @param _amount The amount of CGT to mint.
    /// @dev Accounting invariants are leveraging the balance difference between pre and post-condition
    function mint(uint256 _amount) public {
        // precondition: nil - update ghost variables
        uint256 _preBalance = payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance;

        // action: mint to the random actor
        liquidityController.mint(address(randomActor), _amount);

        // postcondition: is the NativeAssetLiquidity contract's balance changed by an amount different than minted?
        deltaBalanceAndMint = _amount != (_preBalance - uint256(payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance));
        totalAmountMinted += _amount;
    }

    /// @notice Burn custom gas token.
    /// @param _amount The amount of CGT to burn, which is bounded to the actor's balance (avoid trivial revert)
    /// @dev Accounting invariant are leveraging the balance difference between pre and post-condition
    function burn(uint256 _amount) public {
        // precondition: amount to burn has an upper bound (this contract's balance)
        _amount = bound(_amount, 0, address(this).balance);
        uint256 _preBalance = payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance;

        // action: burn _amount
        liquidityController.burn{ value: _amount }();

        // postcondition: update ghost variables by tracking an accounting difference
        deltaBalanceAndBurn = _preBalance + _amount != uint256(payable(Predeploys.NATIVE_ASSET_LIQUIDITY).balance);
        totalAmountBurned += _amount;
    }

    /// @dev Receive needed to receive CGT from the random actor
    receive() external payable { }
}

/// @notice An actor which funds the NativeAssetLiquidity contract
/// @dev There is no underlying access control to this
contract NativeAssetLiquidity_Fundooor is StdUtils {
    /// @notice The Vm contract.
    Vm internal vm;

    /// @notice The NativeAssetLiquidity contract.
    INativeAssetLiquidity internal nativeAssetLiquidity;

    /// @notice Ghost accounting
    uint256 public totalAmountFunded;

    /// @param _vm The Vm contract.
    constructor(Vm _vm) {
        vm = _vm;
        nativeAssetLiquidity = INativeAssetLiquidity(Predeploys.NATIVE_ASSET_LIQUIDITY);
    }

    /// @notice Wrap fund() calls on the NativeAssetLiquidity contract.
    /// @param _amount The amount of CGT to fund.
    /// @dev The amount is bounded to the actor's balance (avoid trivial revert)
    function fund(uint256 _amount) public {
        // precondition: amount to fund has an upper bound (this contract's balance) + ghost accounting
        _amount = bound(_amount, 0, address(this).balance);

        // action: fund _amount
        new SafeSend{ value: _amount }(payable(address(nativeAssetLiquidity)));

        // postcondition: nil here (in the invariant tests)
        // update ghost variables
        totalAmountFunded += _amount;
    }

    receive() external payable { }
}

/// @notice actor which receives fund and send them to either the minter or the funder actor,
///         keeping a closed loop (no vm.deal). It receive() function always revert, to insure mint()/safeSend is
///         always successfully sending the CGT.
contract RandomActor is StdUtils {
    address internal liquidityController_Minter;
    address internal nativeAssetLiquidity_Fundooor;

    /// @notice Flag to indicate if the actor has been called via receive()
    bool public hasBeenCalled = false;

    /// @notice Error thrown when sending CGT to minter fails.
    error RandomActor_SendCGTToMinterFailed();

    /// @notice Error thrown when sending CGT to funder fails.
    error RandomActor_SendCGTtoFunderFailed();

    /// @notice Initialize the addresses of the minter and funder actors.
    /// @param _liquidityController_Minter The address of the minter actor.
    /// @param _nativeAssetLiquidity_Fundooor The address of the funder actor.
    /// @dev This function selector is excluded from the invariant tests
    function initAddresses(address _liquidityController_Minter, address _nativeAssetLiquidity_Fundooor) public {
        liquidityController_Minter = _liquidityController_Minter;
        nativeAssetLiquidity_Fundooor = _nativeAssetLiquidity_Fundooor;
    }

    /// @notice Send CGT to the minter actor.
    /// @param _amount The amount of CGT to send.
    /// @dev The amount is bounded to the actor's balance (avoid trivial revert)
    function sendCGTtoMinter(uint256 _amount) public {
        // precondition: amount to send has an upper bound (this contract's balance)
        uint256 _amountToSend = bound(_amount, 0, address(this).balance);

        // action: send _amountToSend to the minter actor
        (bool success,) = payable(address(liquidityController_Minter)).call{ value: _amountToSend }("");

        // postcondition: the call must succeed (test suite sanity check)
        if (!success) revert RandomActor_SendCGTToMinterFailed();
    }

    /// @notice Send CGT to the funder actor.
    /// @param _amount The amount of CGT to send.
    /// @dev The amount is bounded to the actor's balance (avoid trivial revert)
    function sendCGTtoFunder(uint256 _amount) public {
        // precondition: amount to send has an upper bound (this contract's balance)
        uint256 _amountToSend = bound(_amount, 0, address(this).balance);

        // action: send _amountToSend to the funder actor
        (bool success,) = payable(address(nativeAssetLiquidity_Fundooor)).call{ value: _amountToSend }("");

        // postcondition: the call must succeed (test suite sanity check)
        if (!success) revert RandomActor_SendCGTtoFunderFailed();
    }

    /// @dev We track if the SafeSend triggers a logic on the receiver via a ghost variable
    receive() external payable {
        hasBeenCalled = true;
    }

    fallback() external payable {
        hasBeenCalled = true;
    }
}

/// @title ETHLiquidity_MintBurn_Invariant
/// @notice Invariant that checks that the NativeAssetLiquidity contract's balance is always equal
///         to the sum of the initial supply, the deposits, the funds, and minus the withdrawals.
///         NAL Balance = Initial Supply + Deposits + Funds - Withdrawals
contract CustomGasToken_Invariants_Test is CommonTest {
    /// @notice Starting balance of the contract - arbitrary value (cf Config change)
    uint256 internal constant STARTING_BALANCE = type(uint248).max / 5;

    LiquidityController_Minter internal actor_minter;
    NativeAssetLiquidity_Fundooor internal actor_funder;
    RandomActor internal randomActor;

    /// @notice Test setup.
    function setUp() public override {
        super.setUp();
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        randomActor = new RandomActor();
        actor_funder = new NativeAssetLiquidity_Fundooor(vm);
        actor_minter = new LiquidityController_Minter(vm, liquidityController, randomActor);

        // Initialize the addresses of the minter and funder actors
        randomActor.initAddresses(address(actor_minter), address(actor_funder));

        // Authorize the minter actor (simple access control in unit tests)
        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        liquidityController.authorizeMinter(address(actor_minter));

        // Create the initial supply
        vm.deal(address(nativeAssetLiquidity), STARTING_BALANCE);

        // Set the target contract.
        targetContract(address(actor_minter));
        targetContract(address(actor_funder));

        // Set the target selectors (exclude the initAddresses function)
        bytes4[] memory selectors = new bytes4[](2);
        selectors[0] = RandomActor.sendCGTtoMinter.selector;
        selectors[1] = RandomActor.sendCGTtoFunder.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(randomActor), selectors: selectors });
        targetSelector(selector);
    }

    /// @notice Invariant that checks that the NativeAssetLiquidity contract's balance is always equal
    ///         to the sum of the initial supply, the deposits, the funds, and minus the withdrawals.
    ///         NAL Balance = Initial Supply + Deposits + Funds - Withdrawals
    /// @dev liquidityController.burn() calls deposit, liquidityController.mint() calls withdraw
    function invariant_supplyConservation() public view {
        assertEq(
            address(nativeAssetLiquidity).balance,
            STARTING_BALANCE + actor_funder.totalAmountFunded() + actor_minter.totalAmountBurned()
                - actor_minter.totalAmountMinted(),
            "NativeAssetLiquidity balance is not equal to the sum of the initial supply, the deposits, the funds, and minus the withdrawals"
        );
    }

    /// @notice Invariant that checks that the minted amount is equal to the withdrawn amount
    /// @dev Checks if the amount minted equals the amount transferred *outside* the NativeAssetLiquidity contract
    function invariant_mintedEqualsWithdrawn() public view {
        assertFalse(actor_minter.deltaBalanceAndMint(), "Minted amount is not equal to the withdrawn amount");
    }

    /// @notice Invariant that checks that the burned amount is equal to the deposited amount
    /// @dev Checks if the amount burned equals the amount transferred *to* the NativeAssetLiquidity contract
    function invariant_burnedEqualsDeposited() public view {
        assertFalse(actor_minter.deltaBalanceAndBurn(), "Burned amount is not equal to the deposited amount");
    }

    /// @notice Invariant that checks that the LiquidityController contract's balance is always 0
    /// @dev Checks if the LiquidityController there is no CGT being trapped in the LiquidityController contract
    function invariant_noDustLiquidityController() public view {
        assertEq(address(liquidityController).balance, 0, "LiquidityController balance is not 0");
    }

    /// @notice Invariant that checks that the mint function never calls back to the RandomActor contract
    /// @dev Checks if the mint function never calls back to the RandomActor contract (test SafeSend)
    function invariant_mintNeverCallsBack() public view {
        assertFalse(randomActor.hasBeenCalled(), "RandomActor receive() function has been triggered");
    }
}

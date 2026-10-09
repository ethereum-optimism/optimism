// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { WETH98 } from "src/universal/WETH98.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @custom:proxied true
/// @title DelayedWETH
/// @notice DelayedWETH is an extension to WETH9 that allows for delayed withdrawals. Accounts must trigger an unlock
///         function before they can withdraw WETH. Accounts must trigger unlock by specifying a sub-account and an
///         amount of WETH to unlock. Accounts can trigger the unlock function at any time, but must wait a delay
///         period before they can withdraw after the unlock function is triggered. DelayedWETH is designed to be used
///         by the DisputeGame contracts where unlock will only be triggered after a dispute is resolved. DelayedWETH
///         is meant to sit behind a proxy contract and has an owner address that can pull WETH from any account and
///         can recover ETH from the contract itself. The owner sets the delay at initialization and can change it
///         afterwards through `setDelay`, within the bounds fixed by the implementation. Variable and function
///         naming vaguely follows the vibe of WETH9. Not the prettiest contract in the world, but it gets the job
///         done.
contract DelayedWETH is Initializable, ProxyAdminOwnedBase, ReinitializableBase, WETH98, ISemver {
    /// @notice Represents a withdrawal request.
    struct WithdrawalRequest {
        uint256 amount;
        uint256 timestamp;
    }

    /// @notice Semantic version.
    /// @custom:semver 3.0.0
    string public constant version = "3.0.0";

    /// @notice Returns a withdrawal request for the given address.
    mapping(address => mapping(address => WithdrawalRequest)) public withdrawals;

    /// @notice The lowest value that `delay` may be set to.
    uint256 internal immutable MIN_DELAY;

    /// @notice The highest value that `delay` may be set to.
    uint256 internal immutable MAX_DELAY;

    /// @custom:legacy
    /// @custom:spacer systemConfig
    /// @notice Spacer taking up the legacy `systemConfig` address slot.
    address private spacer_4_0_20;

    /// @notice The ETHLockbox used as the pause identifier.
    IETHLockbox public ethLockbox;

    /// @notice Withdrawal delay in seconds. An unlocked withdrawal can only be executed once this much
    ///         time has passed since the unlock. Bounded by `MIN_DELAY` and `MAX_DELAY`.
    /// @custom:network-specific
    uint256 public delay;

    /// @notice Emitted when the withdrawal delay is set.
    /// @param delay The new withdrawal delay in seconds.
    event DelaySet(uint256 delay);

    /// @notice Thrown when the withdrawal delay bounds are zero or inverted.
    error DelayedWETH_InvalidDelayBounds();

    /// @notice Thrown when a withdrawal delay is outside the configured bounds.
    error DelayedWETH_InvalidDelay();

    /// @param _minDelay The lowest withdrawal delay a chain may use, in seconds.
    /// @param _maxDelay The highest withdrawal delay a chain may use, in seconds.
    constructor(uint256 _minDelay, uint256 _maxDelay) ReinitializableBase(2) {
        if (_minDelay == 0 || _minDelay > _maxDelay) {
            revert DelayedWETH_InvalidDelayBounds();
        }
        MIN_DELAY = _minDelay;
        MAX_DELAY = _maxDelay;
        _disableInitializers();
    }

    /// @notice Initializes the contract. Emits `DelaySet` on every call, including upgrades that
    ///         pass the current delay back in, so a re-initialization that changes nothing still
    ///         emits an event carrying the unchanged value.
    /// @param _ethLockbox The address of the ETHLockbox contract.
    /// @param _delay The withdrawal delay in seconds.
    function initialize(IETHLockbox _ethLockbox, uint256 _delay) external reinitializer(initVersion()) {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        // Now perform initialization logic.
        ethLockbox = _ethLockbox;

        // Set the withdrawal delay. Bounds-checked and emits the same event as the setter.
        _setDelay(_delay);
    }

    /// @notice Returns the lowest value that the withdrawal delay may be set to.
    /// @return The minimum withdrawal delay in seconds.
    function minDelay() external view returns (uint256) {
        return MIN_DELAY;
    }

    /// @notice Returns the highest value that the withdrawal delay may be set to.
    /// @return The maximum withdrawal delay in seconds.
    function maxDelay() external view returns (uint256) {
        return MAX_DELAY;
    }

    /// @notice Returns the SuperchainConfig contract.
    /// @return ISuperchainConfig The SuperchainConfig contract.
    function config() public view returns (ISuperchainConfig) {
        return ethLockbox.superchainConfig();
    }

    /// @notice Allows the ProxyAdmin owner to set the withdrawal delay. The new value applies to
    ///         every pending withdrawal request, including requests unlocked before the change.
    /// @param _delay The new withdrawal delay in seconds.
    function setDelay(uint256 _delay) external {
        // Only the ProxyAdmin owner can change the withdrawal delay.
        _assertOnlyProxyAdminOwner();
        _setDelay(_delay);
    }

    /// @notice Unlocks withdrawals for the sender's account, after a time delay.
    /// @param _guy Sub-account to unlock.
    /// @param _wad The amount of WETH to unlock.
    function unlock(address _guy, uint256 _wad) external {
        // Note that the unlock function can be called by any address, but the actual unlocking capability still only
        // gives the msg.sender the ability to withdraw from the account. As long as the unlock and withdraw functions
        // are called with the proper recipient addresses, this will be safe. Could be made safer by having external
        // accounts execute withdrawals themselves but that would have added extra complexity and made DelayedWETH a
        // leaky abstraction, so we chose this instead.
        WithdrawalRequest storage wd = withdrawals[msg.sender][_guy];
        wd.timestamp = block.timestamp;
        wd.amount += _wad;
    }

    /// @notice Withdraws an amount of ETH.
    /// @param _wad The amount of ETH to withdraw.
    function withdraw(uint256 _wad) public override {
        withdraw(msg.sender, _wad);
    }

    /// @notice Extension to withdrawal, must provide a sub-account to withdraw from.
    /// @param _guy Sub-account to withdraw from.
    /// @param _wad The amount of WETH to withdraw.
    function withdraw(address _guy, uint256 _wad) public {
        require(!ethLockbox.paused(), "DelayedWETH: contract is paused");
        WithdrawalRequest storage wd = withdrawals[msg.sender][_guy];
        require(wd.amount >= _wad, "DelayedWETH: insufficient unlocked withdrawal");
        require(wd.timestamp > 0, "DelayedWETH: withdrawal not unlocked");
        require(wd.timestamp + delay <= block.timestamp, "DelayedWETH: withdrawal delay not met");
        wd.amount -= _wad;
        super.withdraw(_wad);
    }

    /// @notice Allows the owner to recover from error cases by pulling ETH out of the contract.
    /// @param _wad The amount of WETH to recover.
    function recover(uint256 _wad) external {
        require(msg.sender == proxyAdminOwner(), "DelayedWETH: not owner");
        uint256 amount = _wad < address(this).balance ? _wad : address(this).balance;
        (bool success,) = payable(msg.sender).call{ value: amount }(hex"");
        require(success, "DelayedWETH: recover failed");
    }

    /// @notice Allows the owner to recover from error cases by pulling all WETH from a specific owner.
    /// @param _guy The address to recover the WETH from.
    function hold(address _guy) external {
        hold(_guy, balanceOf(_guy));
    }

    /// @notice Allows the owner to recover from error cases by pulling a specific amount of WETH from a specific owner.
    /// @param _guy The address to recover the WETH from.
    /// @param _wad The amount of WETH to recover.
    function hold(address _guy, uint256 _wad) public {
        require(msg.sender == proxyAdminOwner(), "DelayedWETH: not owner");
        _allowance[_guy][msg.sender] = _wad;
        emit Approval(_guy, msg.sender, _wad);
        transferFrom(_guy, msg.sender, _wad);
    }

    /// @notice Sets the withdrawal delay after checking it against the configured bounds.
    /// @param _delay The new withdrawal delay in seconds.
    function _setDelay(uint256 _delay) internal {
        if (_delay < MIN_DELAY || _delay > MAX_DELAY) {
            revert DelayedWETH_InvalidDelay();
        }
        delay = _delay;
        emit DelaySet(_delay);
    }
}

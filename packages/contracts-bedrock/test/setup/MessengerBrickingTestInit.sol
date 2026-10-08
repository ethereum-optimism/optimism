// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { OptimismPortal2_TestInit } from "test/L1/OptimismPortal2.t.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @title MessengerBrickingTestInit
/// @notice Shared base for tests of withdrawals relayed through the L1CrossDomainMessenger whose
///         outcome depends on the gas the Portal is finalized with. Measures the gas thresholds
///         instead of hard-coding them, because they depend on the compiler profile and EVM version.
abstract contract MessengerBrickingTestInit is OptimismPortal2_TestInit {
    /// @notice Range searched for the gas thresholds.
    uint256 internal constant GAS_SEARCH_LOWER_BOUND = 100_000;
    uint256 internal constant GAS_SEARCH_UPPER_BOUND = 3_000_000;

    /// @notice Transaction gas limit cap (EIP-7825).
    uint256 internal constant MAX_TX_GAS = 2 ** 24;

    /// @notice Returns the least gas with which the Portal forwards the withdrawal's gas limit and
    ///         the least gas with which the messenger relays the message successfully.
    function _gasThresholds(Types.WithdrawalTransaction memory _withdrawal)
        internal
        returns (uint256 minPortalGas_, uint256 minMessengerGas_)
    {
        minPortalGas_ = _leastGas(_withdrawal, _portalForwards);
        minMessengerGas_ = _leastGas(_withdrawal, _messengerRelays);
    }

    /// @notice Binary searches for the least gas with which finalizing `_withdrawal` satisfies
    ///         `_passes`, which must be monotonic in the gas.
    function _leastGas(
        Types.WithdrawalTransaction memory _withdrawal,
        function(Types.WithdrawalTransaction memory, bool, bytes memory) internal view returns (bool) _passes
    )
        internal
        returns (uint256)
    {
        uint256 low = GAS_SEARCH_LOWER_BOUND;
        uint256 high = GAS_SEARCH_UPPER_BOUND;
        assertFalse(_tryFinalize(_withdrawal, low, _passes), "gas search lower bound too high");
        assertTrue(_tryFinalize(_withdrawal, high, _passes), "gas search upper bound too low");
        while (high - low > 1) {
            uint256 mid = (low + high) / 2;
            if (_tryFinalize(_withdrawal, mid, _passes)) high = mid;
            else low = mid;
        }
        return high;
    }

    /// @notice Finalizes `_withdrawal` with `_gas`, evaluates `_passes` and reverts the state.
    function _tryFinalize(
        Types.WithdrawalTransaction memory _withdrawal,
        uint256 _gas,
        function(Types.WithdrawalTransaction memory, bool, bytes memory) internal view returns (bool) _passes
    )
        internal
        returns (bool passes_)
    {
        uint256 snapshot = vm.snapshotState();
        (bool success, bytes memory returnData) = address(optimismPortal2).call{ gas: _gas }(
            abi.encodeCall(optimismPortal2.finalizeWithdrawalTransaction, (_withdrawal))
        );
        passes_ = _passes(_withdrawal, success, returnData);
        vm.revertToState(snapshot);
    }

    /// @notice Whether the Portal forwarded the withdrawal's gas limit to the messenger.
    function _portalForwards(
        Types.WithdrawalTransaction memory,
        bool _success,
        bytes memory _returnData
    )
        internal
        pure
        returns (bool)
    {
        return _success
            || keccak256(_returnData)
                == keccak256(abi.encodePacked(IOptimismPortal2.OptimismPortal_L1CDMCallFailed.selector));
    }

    /// @notice Whether the messenger relayed the message successfully.
    function _messengerRelays(
        Types.WithdrawalTransaction memory _withdrawal,
        bool _success,
        bytes memory
    )
        internal
        view
        returns (bool)
    {
        return _success && l1CrossDomainMessenger.successfulMessages(keccak256(_withdrawal.data));
    }
}

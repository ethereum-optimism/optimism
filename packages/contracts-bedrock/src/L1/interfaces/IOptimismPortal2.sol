// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { Types } from "src/libraries/Types.sol";
import { DisputeGameFactory, IDisputeGame } from "src/dispute/DisputeGameFactory.sol";
import { ConfigType } from "src/L2/L1BlockInterop.sol";
import "src/dispute/lib/Types.sol";

/// @title IOptimismPortal2
/// @notice Interface for the OptimismPortal2 contract.
interface IOptimismPortal2 is ISemver {
    /// @notice Represents a proven withdrawal.
    /// @custom:field disputeGameProxy The address of the dispute game proxy that the withdrawal was proven against.
    /// @custom:field timestamp        Timestamp at whcih the withdrawal was proven.
    struct ProvenWithdrawal {
        IDisputeGame disputeGameProxy;
        uint64 timestamp;
    }

    /// @notice Emitted when a transaction is deposited from L1 to L2.
    ///         The parameters of this event are read by the rollup node and used to derive deposit
    ///         transactions on L2.
    /// @param from       Address that triggered the deposit transaction.
    /// @param to         Address that the deposit transaction is directed to.
    /// @param version    Version of this deposit transaction event.
    /// @param opaqueData ABI encoded deposit data to be parsed off-chain.
    event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);

    /// @notice Emitted when a withdrawal transaction is proven.
    /// @param withdrawalHash Hash of the withdrawal transaction.
    /// @param from           Address that triggered the withdrawal transaction.
    /// @param to             Address that the withdrawal transaction is directed to.
    event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);

    /// @notice Emitted when a withdrawal transaction is proven. Exists as a separate event to allow for backwards
    ///         compatibility for tooling that observes the `WithdrawalProven` event.
    /// @param withdrawalHash Hash of the withdrawal transaction.
    /// @param proofSubmitter Address of the proof submitter.
    event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter);

    /// @notice Emitted when a withdrawal transaction is finalized.
    /// @param withdrawalHash Hash of the withdrawal transaction.
    /// @param success        Whether the withdrawal transaction was successful.
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);

    /// @notice Emitted when a dispute game is blacklisted by the Guardian.
    /// @param disputeGame Address of the dispute game that was blacklisted.
    event DisputeGameBlacklisted(IDisputeGame indexed disputeGame);

    /// @notice Emitted when the Guardian changes the respected game type in the portal.
    /// @param newGameType The new respected game type.
    /// @param updatedAt   The timestamp at which the respected game type was updated.
    event RespectedGameTypeSet(GameType indexed newGameType, Timestamp indexed updatedAt);

    function donateETH() external payable;
    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256 _disputeGameIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external;
    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external;
    function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external;
    function blacklistDisputeGame(IDisputeGame _disputeGame) external;
    function setRespectedGameType(GameType _gameType) external;
    function numProofSubmitters(bytes32 _withdrawalHash) external view returns (uint256);
    receive() external payable;
}

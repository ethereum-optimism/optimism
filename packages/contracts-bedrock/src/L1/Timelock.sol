// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

/// @custom:proxied true
/// @title Timelock
/// @notice The Timelock contract is used as the owner for OP Stack contracts.
contract Timelock {
    using EnumerableSet for EnumerableSet.AddressSet;

    struct Call {
        bytes32 salt;
        address target;
        uint256 value;
        bytes data;
    }

    struct Proposal {
        bool executed;
        bool cancelled;
        uint64 eta;
        EnumerableSet.AddressSet approvals;
    }

    uint64 public longDelay;
    uint64 public shortDelay;

    EnumerableSet.AddressSet internal controllers; // TODO: Expose this data somehow
    mapping(bytes32 => Proposal) internal proposals; // TODO: Expose this data somehow

    error InvalidConstructor(string reason);
    error CallFailed(bytes revertData);

    event Approved(bytes32 indexed txHash, Call call, uint64 eta);
    event Executed(bytes32 indexed txHash, Call call);
    event Cancelled(bytes32 indexed txHash);

    constructor(address[] memory controllers_, uint64 longDelay_, uint64 shortDelay_) {
        if (controllers_.length == 0) revert InvalidConstructor("Controllers length must be greater than 0.");

        if (longDelay_ <= shortDelay_) revert InvalidConstructor("Long delay must be greater than short delay.");
        if (longDelay_ == 0) revert InvalidConstructor("Long delay must be greater than 0.");
        if (shortDelay_ == 0) revert InvalidConstructor("Short delay must be greater than 0.");

        if (longDelay_ > 180 days) revert InvalidConstructor("Long delay must be <= 6 months.");
        if (shortDelay_ > 30 days) revert InvalidConstructor("Short delay must be <= 1 year.");

        for (uint256 i = 0; i < controllers_.length; i++) {
            controllers.add(controllers_[i]);
        }
        longDelay = longDelay_;
        shortDelay = shortDelay_;
    }

    /// @dev Compute the hash for a proposal
    function hash(Call memory call_) public pure returns (bytes32 txHash) {
        txHash = keccak256(abi.encode(call_));
    }

    /// @dev Approve a proposal and set its eta
    function approve(Call calldata call_) external returns (uint64) {
        require(controllers.contains(msg.sender), "Not authorized.");

        bytes32 txHash = hash(call_);
        Proposal storage proposal = proposals[txHash];

        // Safety checks.
        require(!proposal.approvals.contains(msg.sender), "Already approved.");
        require(!proposal.executed, "Already executed.");
        require(!proposal.cancelled, "Already executed.");

        // If it's the first approval, set the eta to the long delay
        if (proposal.approvals.length() == 0) {
            proposal.eta = uint64(block.timestamp) + longDelay;
        }

        // Add the approval
        proposal.approvals.add(msg.sender);

        // If there are as many approvals as controllers, set the eta to the short delay
        if (proposal.approvals.length() == controllers.length()) {
            proposal.eta = uint64(block.timestamp) + shortDelay;
        }

        emit Approved(txHash, call_, proposal.eta);
        return proposal.eta;
    }

    /// @dev Any controller can veto a proposal and make it never executable
    function cancel(bytes32 txHash, address controller) public {
        require(controllers.contains(controller), "Invalid controller.");
        require(controller == msg.sender || isCallerGnosisSafeOwner(controller, msg.sender), "Not authorized.");

        Proposal storage proposal = proposals[txHash];
        require(!proposal.executed, "Already executed.");
        require(!proposal.cancelled, "Already cancelled.");

        proposal.cancelled = true;
        emit Cancelled(txHash);
    }

    /// @dev Individual Gnosis Safe owners can veto a proposal and make it never executable
    function cancel(Call memory call_, address controller) external {
        cancel(hash(call_), controller);
    }

    /// @dev Execute a proposal, if past the eta, not cancelled and not executed
    function execute(Call calldata call_) external returns (bytes memory) {
        bytes32 txHash = hash(call_);
        Proposal storage proposal = proposals[txHash];

        require(uint64(block.timestamp) >= proposal.eta, "ETA not reached.");
        require(!proposal.cancelled, "Proposal cancelled.");
        require(!proposal.executed, "Proposal already executed.");

        proposal.executed = true;
        emit Executed(txHash, call_);

        (bool success, bytes memory result) = call_.target.call{ value: call_.value }(call_.data);
        if (!success) revert CallFailed(result);
        return result;
    }

    /// @dev Returns true if the address is probably a Gnosis Safe
    function isGnosisSafe(address _who) internal view returns (bool) {
        bytes memory callData = abi.encodeWithSelector(bytes4(keccak256("getThreshold()")));
        (bool ok, bytes memory data) = _who.staticcall(callData);
        return ok && data.length == 32;
    }

    /// @dev Is the caller an owner of a Gnosis Safe?
    function isCallerGnosisSafeOwner(address controller, address who) internal view returns (bool) {
        if (!isGnosisSafe(who)) return false;
        GnosisSafe safe = GnosisSafe(payable(controller));
        return safe.isOwner(controller);
    }
}

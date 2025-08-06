// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";

/// @custom:proxied true
/// @title ProtocolVersions
/// @notice The ProtocolVersions contract is used to manage superchain protocol version information.
contract Timelock {

    event Approved(bytes32 indexed txHash, Call call, uint32 eta);
    event Cancelled(bytes32 indexed txHash, Call call);
    event Executed(bytes32 indexed txHash, Call call);
    error CallFailed(bytes revertData);

    struct Call {
        address target;
        uint256 value;
        bytes data;
        bytes32 salt;
    }

    struct Proposal {
        mapping (address => bool) approved;
        bool executed;
        bool cancelled;
        uint32 eta;
    }

    uint32 public longTermDelay;
    uint32 public shortTermDelay;

    mapping (address => bool) public controllers;
    mapping (bytes32 => Proposal) public proposals;


    constructor(address[] memory controllers_, uint32 longTermDelay_, uint32 shortTermDelay_) {
        for (uint256 i = 0; i < controllers_.length; i++) {
            controllers[controllers_[i]] = true;
        }
        longTermDelay = longTermDelay_;
        shortTermDelay = shortTermDelay_;
    }

    /// @dev Is the caller a Gnosis Safe?
    function isGnosisSafe(address _who) internal view returns (bool) {
        bytes memory callData = abi.encodeWithSelector(bytes4(keccak256("getThreshold()")));
        (bool ok, bytes memory data) = _who.staticcall(callData);
        return ok && data.length == 32;
    }

    /// @dev Is the caller an owner of a Gnosis Safe?
    function isCallerGnosisSafeOwner() internal view returns (bool) {
        if (!isGnosisSafe(msg.sender)) return false;
        GnosisSafe safe = GnosisSafe(msg.sender);
        return safe.isOwner(msg.sender);
    }

    /// @dev Compute the hash for a proposal
    function hash(Call calldata call_)
        public pure returns (bytes32 txHash)
    {
        txHash = keccak256(abi.encode(call_));
    }

    /// @dev Approve a proposal and set its eta
    function approve(Call calldata call_)
        external returns (uint32 eta)
    {
        require(controllers[msg.sender], "Not authorized.");

        bytes32 txHash = hash(call_);
        Proposal memory proposal = proposals[txHash];

        // Revert if already approved by this controller
        require(!proposal.approved[msg.sender], "Already approved.");

        // Set approval
        proposal.approved[msg.sender] = true;

        // If first approval, set eta to long term delay
        if (proposal.approved[msg.sender] == false) {
            proposal.eta = uint32(block.timestamp) + longTermDelay;
        }

        // If second approval, set eta to short term delay
        if (proposal.approved[msg.sender] == true) {
            proposal.eta = uint32(block.timestamp) + shortTermDelay;
        }

        proposals[txHash] = proposal;
        emit Approved(txHash, call_, proposal.eta);
    }

    /// @dev Any controller can veto a proposal and make it never executable
    function cancel(bytes32 txHash, address controller)
        public
    {
        require(controllers[controller], "Invalid controller.");

        require(
            controller == msg.sender ||
            isCallerGnosisSafeOwner(controller), "Not authorized.");

        Proposal storage proposal = proposals[txHash];
        proposal.cancelled = true;
        emit Cancelled(txHash, proposal.call);
    }

    /// @dev Individual Gnosis Safe owners can veto a proposal and make it never executable
    function cancel(Call memory call_, address controller)
        external
    {
        cancel(hash(call_), controller);
    }

    /// @dev Execute a proposal, if past the eta, not cancelled and not executed
    function execute(Call calldata call_)
        external returns (bytes memory)
    {
        bytes32 txHash = hash(call_);
        Proposal storage proposal = proposals[txHash];

        require(uint32(block.timestamp) >= proposal.eta, "ETA not reached.");
        require(!proposal.cancelled, "Proposal cancelled.");
        require(!proposal.executed, "Proposal already executed.");

        proposal.executed = true;
        emit Executed(txHash, call_);

        (bool success, bytes memory result) = call_.target.call{ value: call_.value }(call_.data);
        if (!success) CallFailed(result);
        return result;
    }
}

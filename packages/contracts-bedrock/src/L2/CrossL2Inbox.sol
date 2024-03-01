// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import { SafeCall } from "src/libraries/SafeCall.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";

/// @custom:upgradeable
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
contract CrossL2Inbox {
    struct Identifier {
        address origin;
        uint256 blocknumber;
        uint256 logIndex;
        uint256 timestamp;
        uint256 chainId;
    }

    // bytes32(uint256(keccak256("crossl2inbox.identifier.origin")) - 1)
    bytes32 public constant ORIGIN_SLOT = 0xd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c815;

    // bytes32(uint256(keccak256("crossl2inbox.identifier.blocknumber")) - 1)
    bytes32 public constant BLOCKNUMBER_SLOT = 0x5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc;

    // bytes32(uint256(keccak256("crossl2inbox.identifier.logindex")) - 1)
    bytes32 public constant LOG_INDEX_SLOT = 0xab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d;

    // bytes32(uint256(keccak256("crossl2inbox.identifier.timestamp")) - 1)
    bytes32 public constant TIMESTAMP_SLOT = 0x2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e02;

    // bytes32(uint256(keccak256("crossl2inbox.identifier.chainid")) - 1)
    bytes32 public constant CHAINID_SLOT = 0x6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d313;

    address public l1Block;

    function origin() public view returns (address _origin) {
        assembly {
            _origin := tload(ORIGIN_SLOT)
        }
    }

    function blocknumber() public view returns (uint256 _blocknumber) {
        assembly {
            _blocknumber := tload(BLOCKNUMBER_SLOT)
        }
    }

    function logIndex() public view returns (uint256 _logIndex) {
        assembly {
            _logIndex := tload(LOG_INDEX_SLOT)
        }
    }

    function timestamp() public view returns (uint256 _timestamp) {
        assembly {
            _timestamp := tload(TIMESTAMP_SLOT)
        }
    }

    function chainId() public view returns (uint256 _chainId) {
        assembly {
            _chainId := tload(CHAINID_SLOT)
        }
    }

    /// @notice Executes a cross chain message on the destination chain
    /// @param _msg The message payload, matching the initiating message.
    /// @param _id A Identifier pointing to the initiating message.
    /// @param _target Account that is called with _msg.
    function executeMessage(bytes calldata _msg, Identifier calldata _id, address _target) public payable {
        require(_id.timestamp <= block.timestamp, "CrossL2Inbox: invalid id timestamp"); // timestamp invariant
        uint256 chainId_ = _id.chainId;
        require(L1Block(l1Block).isInDependencySet(chainId_), "CrossL2Inbox: invalid id chainId"); // chainId invariant
        require(msg.sender == tx.origin, "CrossL2Inbox: Not EOA sender"); // only EOA invariant

        address origin_ = _id.origin;
        uint256 blocknumber = _id.blocknumber;
        uint256 logIndex = _id.logIndex;
        uint256 timestamp_ = _id.timestamp;

        assembly {
            tstore(ORIGIN_SLOT, origin_)
            tstore(BLOCKNUMBER_SLOT, blocknumber)
            tstore(LOG_INDEX_SLOT, logIndex)
            tstore(TIMESTAMP_SLOT, timestamp_)
            tstore(CHAINID_SLOT, chainId_)
        }

        bool success = SafeCall.call({ _target: _target, _gas: gasleft(), _value: msg.value, _calldata: _msg });

        require(success);
    }
}

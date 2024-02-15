// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

/// @custom:upgradeable
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
abstract contract CrossL2Inbox is Initializable {
    struct Identifier {
        address origin;
        uint256 blocknumber;
        uint256 logIndex;
        uint256 timestamp;
        uint256 chainid;
    }

    bytes32 public constant ORIGIN_SLOT = bytes32(uint256(keccak256("crossl2inbox.identifier.origin")) - 1);

    bytes32 public constant BLOCKNUMBER_SLOT = bytes32(uint256(keccak256("crossl2inbox.identifier.blocknumber")) - 1);

    bytes32 public constant LOG_INDEX_SLOT = bytes32(uint256(keccak256("crossl2inbox.identifier.logindex")) - 1);

    bytes32 public constant TIMESTAMP_SLOT = bytes32(uint256(keccak256("crossl2inbox.identifier.timestamp")) - 1);

    bytes32 public constant CHAINID_SLOT = bytes32(uint256(keccak256("crossl2inbox.identifier.chainid")) - 1);

    function getIdentifier() public view returns (Identifier memory identifier) {
        assembly {
            identifier.origin := TLOAD(ORIGIN_SLOT)
            identifier.blocknumber := tload(BLOCKNUMBER_SLOT)
            identifier.logIndex := tload(LOG_INDEX_SLOT)
            identifier.timestamp := tload(TIMESTAMP_SLOT)
            identifier.chainid := tload(CHAINID_SLOT)
        }
    }

    function executeMessage(address _target, bytes calldata _msg, Identifier calldata _id) public payable {
        require(msg.sender == tx.origin);
        require(_id.timestamp <= block.timestamp);
        //require(L1Block.isInDependencySet(_id.chainid));

        assembly {
            tstore(ORIGIN_SLOT, _id.origin)
            tstore(BLOCKNUMBER_SLOT, _id.blocknumber)
            tstore(LOG_INDEX_SLOT, _id.logIndex)
            tstore(TIMESTAMP_SLOT, _id.timestamp)
            tstore(CHAINID_SLOT, _id.chainid)
        }

        bool success = SafeCall.call({ _target: _target, _value: msg.value, _calldata: _msg });

        require(success);
    }
}

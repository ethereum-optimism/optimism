// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @custom:proxied
 * @title BatchInbox
 * @notice Calldata entries of version hashes which are checked against the precompile of blobs to verify they exist
 */
// slither-disable-next-line locked-ether
contract BatchInbox is OwnableUpgradeable, Semver {
    /**
     * @notice The address of the proposer;
     */
    address public proposer;
    /**
     * @notice Emitted when the proposer address is changed.
     *
     * @param previousProposer The previous proposer address.
     * @param newProposer      The new proposer address.
     */
    event ProposerChanged(address indexed previousProposer, address indexed newProposer);

    /**
     * @notice Reverts if called by any account other than the proposer.
     */
    modifier onlyProposer() {
        require(proposer == msg.sender, "BatchInbox: function can only be called by proposer");
        _;
    }
    /**
     * @custom:semver 0.0.1
     *
     * @param _owner                 The address of the owner.
     */
    constructor(
        address _proposer,
        address _owner
    ) Semver(0, 0, 1) {
        initialize(_proposer, _owner);
    }

    /**
     * @notice Initializer.
     *
     * @param _owner               The address of the owner.
     */
    function initialize(
        address _proposer,
        address _owner
    ) public initializer {
        require(_proposer != _owner, "BatchInbox: proposer cannot be the same as the owner");
        __Ownable_init();
        changeProposer(_proposer);
        _transferOwnership(_owner);
    }
    /**
     * @notice appends an array of valid version hashes to the chain through calldata, each VH is checked via the VH precompile.
     * the calldata should be contingious set of 32 byte version hashes to check via precompile. Will consume memory for 1 hash and check that the a hash value was parrtoed back to indicate validity.
     *
     */
    function appendSequencerBatch() external view onlyProposer {
        // Revert if the provided calldata does not consist of the 4 byte selector and segments of 32 bytes.
        require((msg.data.length - 4)%32 == 0);
        // Start reading calldata after the function selector.
        uint256 cursorPosition = 4;
        // Start loop. End once there is not sufficient remaining calldata to contain a 32 byte hash.
        while(cursorPosition <= (msg.data.length - 32)) {
            assembly{
                // Allocate memory for VH
                let memPtr := mload(0x40)
                // load 32 bytes from cursorPosition in calldata to memPtr location in memory
                calldatacopy(memPtr, cursorPosition, 0x20)
                // Set free pointer before function call.
                mstore(0x40, add(memPtr, 0x20))
                let result := staticcall(1500, 0x63, memPtr, 0x20, 0, 0)
                // check the RESULT does not indicate an error.
                switch result
                // Revert if precompile RESULT indicates an error.
                case 0 { revert(0, 0) }
                // Otherwise check the RETURNDATA
                default {
                    if eq(returndatasize(), 0) {
                        revert(0, 0)
                    }
                }
            }
            cursorPosition += 32;
        }
    }
    /**
     * @notice Transfers the proposer role to a new account (`newProposer`).
     *         Can only be called by the current owner.
     */
    function changeProposer(address _newProposer) public onlyOwner {
        require(
            _newProposer != address(0),
            "BatchInbox: new proposer cannot be the zero address"
        );

        require(
            _newProposer != owner(),
            "BatchInbox: proposer cannot be the same as the owner"
        );

        emit ProposerChanged(proposer, _newProposer);
        proposer = _newProposer;
    }
}

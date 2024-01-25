// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "./MultisigBase.sol";

import { console } from "forge-std/console.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";

import { IGnosisSafe, Enum } from "@eth-optimism-bedrock/scripts/interfaces/IGnosisSafe.sol";

/**
 * @title NestedMultisigBuilder
 * @notice Modeled from Optimism's SafeBuilder, but built for nested safes (Safes where the signers are other Safes).
 */
abstract contract NestedMultisigBuilder is MultisigBase {
    /**
     * -----------------------------------------------------------
     * Virtual Functions
     * -----------------------------------------------------------
     */

    /**
     * @notice Follow up assertions to ensure that the script ran to completion
     */
    function _postCheck() internal virtual view;

    /**
     * @notice Creates the calldata
     */
    function _buildCalls() internal virtual view returns (IMulticall3.Call3[] memory);

    /**
     * @notice Returns the nested safe address to execute the final transaction from
     */
    function _ownerSafe() internal virtual view returns (address);

    /**
     * -----------------------------------------------------------
     * Implemented Functions
     * -----------------------------------------------------------
     */

    /**
     * Step 1
     * ======
     * Generate a transaction approval data to sign. This method should be called by a threshold
     * of members of each of the multisigs involved in the nested multisig. Signers will pass
     * their signature to a facilitator, who will execute the approval transaction for each
     * multisig (see step 2).
     */
    function sign(address _signerSafe) public view returns (bool) {
        address nestedSafeAddress = _ownerSafe();
        IMulticall3.Call3[] memory nestedCalls = _buildCalls();
        IMulticall3.Call3 memory call = _generateApproveCall(nestedSafeAddress, nestedCalls);
        bytes32 hash = _getTransactionHash(_signerSafe, toArray(call));

        console.log("---\nIf submitting onchain, call Safe.approveHash on %s with the following hash:", _signerSafe);
        console.logBytes32(hash);
        _simulateForSigner(_signerSafe, nestedSafeAddress, nestedCalls);
        _printDataToSign(_signerSafe, toArray(call));

        return true;
    }

    /**
     * Step 2
     * ======
     * Execute an approval transaction. This method should be called by a facilitator
     * (non-signer), once for each of the multisigs involved in the nested multisig,
     * after collecting a threshold of signatures for each multisig (see step 1).
     */
    function approve(address _signerSafe, bytes memory _signatures) public returns (bool) {
        vm.startBroadcast();

        address nestedSafeAddress = _ownerSafe();
        IMulticall3.Call3[] memory nestedCalls = _buildCalls();
        IMulticall3.Call3 memory call = _generateApproveCall(nestedSafeAddress, nestedCalls);

        address[] memory approvers = _getApprovers(_signerSafe, toArray(call));
        _signatures = bytes.concat(_signatures, prevalidatedSignatures(approvers));

        return _executeTransaction(_signerSafe, toArray(call), _signatures);
    }

    /**
     * Step 3
     * ======
     * Execute the transaction. This method should be called by a facilitator (non-signer), after
     * all of the approval transactions have been submitted onchain (see step 2).
     */
    function run() public returns (bool) {
        vm.startBroadcast();

        address nestedSafeAddress = _ownerSafe();
        IMulticall3.Call3[] memory nestedCalls = _buildCalls();
        address[] memory approvers = _getApprovers(nestedSafeAddress, nestedCalls);
        bytes memory signatures = prevalidatedSignatures(approvers);

        bool success = _executeTransaction(nestedSafeAddress, nestedCalls, signatures);
        if (success) _postCheck();
        return success;
    }

    function _generateApproveCall(address _safe, IMulticall3.Call3[] memory _calls) internal view returns (IMulticall3.Call3 memory) {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        bytes32 hash = _getTransactionHash(_safe, _calls);

        console.log("---\nNested hash:");
        console.logBytes32(hash);

        return IMulticall3.Call3({
            target: _safe,
            allowFailure: false,
            callData: abi.encodeCall(safe.approveHash, (hash))
        });
    }

    function _getApprovers(address _safe, IMulticall3.Call3[] memory _calls) internal view returns (address[] memory) {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        bytes32 hash = _getTransactionHash(_safe, _calls);

        // get a list of owners that have approved this transaction
        uint256 threshold = safe.getThreshold();
        address[] memory owners = safe.getOwners();
        address[] memory approvers = new address[](threshold);
        uint256 approverIndex;
        for (uint256 i; i < owners.length; i++) {
            address owner = owners[i];
            uint256 approved = safe.approvedHashes(owner, hash);
            if (approved == 1) {
                approvers[approverIndex] = owner;
                approverIndex++;
                if (approverIndex == threshold) {
                    return approvers;
                }
            }
        }
        address[] memory subset = new address[](approverIndex);
        for (uint256 i; i < approverIndex; i++) {
            subset[i] = approvers[i];
        }
        return subset;
    }

    function _simulateForSigner(address _signerSafe, address _safe, IMulticall3.Call3[] memory _calls) internal view {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        IGnosisSafe signerSafe = IGnosisSafe(payable(_signerSafe));
        bytes memory data = abi.encodeCall(IMulticall3.aggregate3, (_calls));
        bytes32 hash = _getTransactionHash(_safe, data);
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](2);

        // simulate an approveHash, so that signer can verify the data they are signing
        bytes memory approveHashData = abi.encodeCall(IMulticall3.aggregate3, (toArray(
            IMulticall3.Call3({
                target: _safe,
                allowFailure: false,
                callData: abi.encodeCall(safe.approveHash, (hash))
            })
        )));
        bytes memory approveHashExec = abi.encodeCall(
            signerSafe.execTransaction,
            (
                address(multicall),
                0,
                approveHashData,
                Enum.Operation.DelegateCall,
                0,
                0,
                0,
                address(0),
                payable(address(0)),
                prevalidatedSignature(address(multicall))
            )
        );
        calls[0] = IMulticall3.Call3({
            target: _signerSafe,
            allowFailure: false,
            callData: approveHashExec
        });

        // simulate the final state changes tx, so that signer can verify the final results
        bytes memory finalExec = abi.encodeCall(
            safe.execTransaction,
            (
                address(multicall),
                0,
                data,
                Enum.Operation.DelegateCall,
                0,
                0,
                0,
                address(0),
                payable(address(0)),
                prevalidatedSignature(_signerSafe)
            )
        );
        calls[1] = IMulticall3.Call3({
            target: _safe,
            allowFailure: false,
            callData: finalExec
        });

        SimulationStateOverride[] memory overrides = new SimulationStateOverride[](2);
        // The state change simulation sets the multisig threshold to 1 in the
        // simulation to enable an approver to see what the final state change
        // will look like upon transaction execution. The multisig threshold
        // will not actually change in the transaction execution.
        overrides[0] = overrideSafeThreshold(_safe);
        // Set the signer safe threshold to 1, and set the owner to multicall.
        // This is a little hacky; reason is to simulate both the approve hash
        // and the final tx in a single Tenderly tx, using multicall. Given an
        // EOA cannot DELEGATECALL, multicall needs to own the signer safe.
        overrides[1] = overrideSafeThresholdAndOwner(_signerSafe, address(multicall));

        console.log("---\nSimulation link:");
        logSimulationLink({
            _to: address(multicall),
            _data: abi.encodeCall(IMulticall3.aggregate3, (calls)),
            _from: msg.sender,
            _overrides: overrides
        });
    }
}

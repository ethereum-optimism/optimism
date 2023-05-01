// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";

import { LibSort } from "../libraries/LibSort.sol";
import { IGnosisSafe, Enum } from "../interfaces/IGnosisSafe.sol";
import { EnhancedScript } from "../universal/EnhancedScript.sol";
import { GlobalConstants } from "../universal/GlobalConstants.sol";
import { ProxyAdmin } from "../../contracts/universal/ProxyAdmin.sol";

/**
 * @title SafeBuilder
 * @notice Builds SafeTransactions
 *         Assumes that a gnosis safe is used as the privileged account and the same
 *         gnosis safe is the owner the proxy admin.
 *         This could be optimized by checking for the number of approvals up front
 *         and not submitting the final approval as `execTransaction` can be called when
 *         there are `threshold - 1` approvals.
 *         Uses the "approved hashes" method of interacting with the gnosis safe. Allows
 *         for the most simple user experience when using automation and no indexer.
 *         Run the command without the `--broadcast` flag and it will print a tenderly URL.
 */
abstract contract SafeBuilder is EnhancedScript, GlobalConstants {
    /**
     * @notice Interface for multicall3.
     */
    IMulticall3 internal constant multicall = IMulticall3(MULTICALL3_ADDRESS);

    /**
     * @notice An array of approvals, used to generate the execution transaction.
     */
    address[] internal approvals;

    /**
     * @notice The entrypoint to this script.
     */
    function run(address _safe, address _proxyAdmin) public returns (bool) {
        vm.startBroadcast();
        bool success = _run(_safe, _proxyAdmin);
        if (success) _postCheck();
        return success;
    }

    /**
     * @notice Follow up assertions to ensure that the script ran to completion.
     */
    function _postCheck() internal virtual view;

    /**
     * @notice Creates the calldata
     */
    function buildCalldata(address _proxyAdmin) internal virtual view returns (bytes memory);

    /**
     * @notice Internal helper function to compute the safe transaction hash.
     */
    function _getTransactionHash(address _safe, address _proxyAdmin) internal returns (bytes32) {
        // Ensure that the required contracts exist
        require(address(multicall).code.length > 0, "multicall3 not deployed");
        require(_safe.code.length > 0, "no code at safe address");
        require(_proxyAdmin.code.length > 0, "no code at proxy admin address");

        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        uint256 nonce = safe.nonce();

        bytes memory data = buildCalldata(_proxyAdmin);

        // Compute the safe transaction hash
        bytes32 hash = safe.getTransactionHash({
            to: address(multicall),
            value: 0,
            data: data,
            operation: Enum.Operation.DelegateCall,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: address(0),
            _nonce: nonce
        });

        return hash;
    }


    /**
     * @notice The implementation of the upgrade. Split into its own function
     *         to allow for testability. This is subject to a race condition if
     *         the nonce changes by a different transaction finalizing while not
     *         all of the signers have used this script.
     */
    function _run(address _safe, address _proxyAdmin) public returns (bool) {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        bytes memory data = buildCalldata(_proxyAdmin);

        // Compute the safe transaction hash
        bytes32 hash = _getTransactionHash(_safe, _proxyAdmin);

        // Send a transaction to approve the hash
        safe.approveHash(hash);

        logSimulationLink({
            _to: address(safe),
            _from: msg.sender,
            _data: abi.encodeCall(safe.approveHash, (hash))
        });

        uint256 threshold = safe.getThreshold();
        address[] memory owners = safe.getOwners();

        for (uint256 i; i < owners.length; i++) {
            address owner = owners[i];
            uint256 approved = safe.approvedHashes(owner, hash);
            if (approved == 1) {
                approvals.push(owner);
            }
        }

        if (approvals.length >= threshold) {
            bytes memory signatures = buildSignatures();

            bool success = safe.execTransaction({
                to: address(multicall),
                value: 0,
                data: data,
                operation: Enum.Operation.DelegateCall,
                safeTxGas: 0,
                baseGas: 0,
                gasPrice: 0,
                gasToken: address(0),
                refundReceiver: payable(address(0)),
                signatures: signatures
            });

            logSimulationLink({
                _to: address(safe),
                _from: msg.sender,
                _data: abi.encodeCall(
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
                        signatures
                    )
                )
            });

            require(success, "call not successful");
            return true;
        } else {
            console.log("not enough approvals");
        }

        // Reset the approvals because they are only used transiently.
        assembly {
            sstore(approvals.slot, 0)
        }

        return false;
    }

    /**
     * @notice Builds the signatures by tightly packing them together.
     *         Ensures that they are sorted.
     */
    function buildSignatures() internal view returns (bytes memory) {
        address[] memory addrs = new address[](approvals.length);
        for (uint256 i; i < approvals.length; i++) {
            addrs[i] = approvals[i];
        }

        LibSort.sort(addrs);

        bytes memory signatures;
        uint8 v = 1;
        bytes32 s = bytes32(0);
        for (uint256 i; i < addrs.length; i++) {
            bytes32 r = bytes32(uint256(uint160(addrs[i])));
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
        return signatures;
    }

}


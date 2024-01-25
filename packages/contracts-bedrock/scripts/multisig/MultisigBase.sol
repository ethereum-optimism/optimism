// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { IMulticall3 } from "forge-std/interfaces/IMulticall3.sol";
import { IGnosisSafe, Enum } from "@eth-optimism-bedrock/scripts/interfaces/IGnosisSafe.sol";
import { LibSort } from "@eth-optimism-bedrock/scripts/libraries/LibSort.sol";
import "./Simulator.sol";

abstract contract MultisigBase is Simulator {
    IMulticall3 internal constant multicall = IMulticall3(MULTICALL3_ADDRESS);

    function _getTransactionHash(address _safe, IMulticall3.Call3[] memory calls) internal view returns (bytes32) {
        bytes memory data = abi.encodeCall(IMulticall3.aggregate3, (calls));
        return _getTransactionHash(_safe, data);
    }

    function _getTransactionHash(address _safe, bytes memory _data) internal view returns (bytes32) {
        return keccak256(_encodeTransactionData(_safe, _data));
    }

    function _encodeTransactionData(address _safe, bytes memory _data) internal view returns (bytes memory) {
        // Ensure that the required contracts exist
        require(address(multicall).code.length > 0, "multicall3 not deployed");
        require(_safe.code.length > 0, "no code at safe address");

        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        uint256 nonce = safe.nonce();
        console.log("Safe current nonce:", nonce);

        if (bytes(vm.envOr(string("SAFE_NONCE"), string(""))).length > 0) {
            nonce = vm.envUint("SAFE_NONCE");
            console.log("Creating transaction with nonce:", nonce);
        }

        return safe.encodeTransactionData({
            to: address(multicall),
            value: 0,
            data: _data,
            operation: Enum.Operation.DelegateCall,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: address(0),
            _nonce: nonce
        });
    }

    function _printDataToSign(address _safe, IMulticall3.Call3[] memory _calls) internal view {
        bytes memory data = abi.encodeCall(IMulticall3.aggregate3, (_calls));
        bytes memory txData = _encodeTransactionData(_safe, data);

        console.log("---\nData to sign:");
        console.log("vvvvvvvv");
        console.logBytes(txData);
        console.log("^^^^^^^^");
    }

    function _checkSignatures(address _safe, IMulticall3.Call3[] memory _calls, bytes memory _signatures) internal view {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        bytes memory data = abi.encodeCall(IMulticall3.aggregate3, (_calls));
        bytes32 hash = _getTransactionHash(_safe, data);

        uint256 signatureCount = uint256(_signatures.length / 0x41);
        uint256 threshold = safe.getThreshold();
        require(signatureCount >= threshold, "not enough signatures");

        // safe requires signatures to be sorted ascending by public key
        _signatures = sortSignatures(_signatures, hash);

        safe.checkSignatures({
            dataHash: hash,
            data: data,
            signatures: _signatures
        });
    }

    function _executeTransaction(address _safe, IMulticall3.Call3[] memory _calls, bytes memory _signatures) internal returns (bool) {
        IGnosisSafe safe = IGnosisSafe(payable(_safe));
        bytes memory data = abi.encodeCall(IMulticall3.aggregate3, (_calls));
        bytes32 hash = _getTransactionHash(_safe, data);

        uint256 signatureCount = uint256(_signatures.length / 0x41);
        uint256 threshold = safe.getThreshold();
        require(signatureCount >= threshold, "not enough signatures");

        // safe requires signatures to be sorted ascending by public key
        _signatures = sortSignatures(_signatures, hash);

        logSimulationLink({
            _to: _safe,
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
                    _signatures
                )
            )
        });

        return safe.execTransaction({
            to: address(multicall),
            value: 0,
            data: data,
            operation: Enum.Operation.DelegateCall,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: _signatures
        });
    }

    function toArray(IMulticall3.Call3 memory call) internal pure returns (IMulticall3.Call3[] memory) {
        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](1);
        calls[0] = call;
        return calls;
    }

    function prevalidatedSignatures(address[] memory _addresses) internal pure returns (bytes memory) {
        LibSort.sort(_addresses);
        bytes memory signatures;
        for (uint256 i; i < _addresses.length; i++) {
            signatures = bytes.concat(signatures, prevalidatedSignature(_addresses[i]));
        }
        return signatures;
    }

    function prevalidatedSignature(address _address) internal pure returns (bytes memory) {
        uint8 v = 1;
        bytes32 s = bytes32(0);
        bytes32 r = bytes32(uint256(uint160(_address)));
        return abi.encodePacked(r, s, v);
    }

    function sortSignatures(bytes memory _signatures, bytes32 dataHash) internal pure returns (bytes memory) {
        bytes memory sorted;
        uint256 count = uint256(_signatures.length / 0x41);
        uint256[] memory addressesAndIndexes = new uint256[](count);
        uint8 v;
        bytes32 r;
        bytes32 s;
        for (uint256 i; i < count; i++) {
            (v, r, s) = signatureSplit(_signatures, i);
            address owner;
            if (v <= 1) {
                owner = address(uint160(uint256(r)));
            } else if (v > 30) {
                owner = ecrecover(keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", dataHash)), v - 4, r, s);
            } else {
                owner = ecrecover(dataHash, v, r, s);
            }
            addressesAndIndexes[i] = uint256(uint256(uint160(owner)) << 0x60 | i); // address in first 160 bits, index in second 96 bits
        }
        LibSort.sort(addressesAndIndexes);
        for (uint256 i; i < count; i++) {
            uint256 index = addressesAndIndexes[i] & 0xffffffff;
            (v, r, s) = signatureSplit(_signatures, index);
            sorted = bytes.concat(sorted, abi.encodePacked(r, s, v));
        }
        return sorted;
    }

    // see https://github.com/safe-global/safe-contracts/blob/1ed486bb148fe40c26be58d1b517cec163980027/contracts/common/SignatureDecoder.sol
    function signatureSplit(bytes memory signatures, uint256 pos) internal pure returns (uint8 v, bytes32 r, bytes32 s) {
        assembly {
            let signaturePos := mul(0x41, pos)
            r := mload(add(signatures, add(signaturePos, 0x20)))
            s := mload(add(signatures, add(signaturePos, 0x40)))
            v := and(mload(add(signatures, add(signaturePos, 0x41))), 0xff)
        }
    }
}

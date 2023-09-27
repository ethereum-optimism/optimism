// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";

contract LivnessGuard_TestInit is Test {
    struct Signer {
        address owner;
        uint256 pk;
    }

    LivenessGuard livenessGuard;
    Safe safe;
    string mnemonic = "test test test test test test test test test test test junk";

    Signer[] signers;

    function newSigner(uint256 index) public returns (Signer memory signer_) {
        signer_.pk = vm.deriveKey(mnemonic, uint32(index));
        signer_.owner = vm.addr(signer_.pk);
    }

    function signTransaction(
        uint256 _pk,
        address _to,
        uint256 _value,
        bytes memory _data
    )
        public
        view
        returns (bytes memory sig_)
    {
        bytes32 txDataHash;
        {
            txDataHash = safe.getTransactionHash({
                to: _to,
                value: _value,
                data: _data,
                operation: Enum.Operation.Call,
                safeTxGas: 0,
                baseGas: 0,
                gasPrice: 0,
                gasToken: address(0),
                refundReceiver: address(0),
                _nonce: safe.nonce()
            });
        }

        (uint8 v, bytes32 r, bytes32 s) = vm.sign(_pk, txDataHash);
        sig_ = abi.encodePacked(v, r, s);
    }

    function exec(Signer[] memory _signers, address _to, bytes memory _data) internal {
        bytes memory sig;
        for (uint256 i; i < _signers.length; i++) {
            bytes.concat(sig, signTransaction(_signers[i].pk, address(safe), 0, _data));
        }
        safe.execTransaction({
            to: _to,
            value: 0,
            data: _data,
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(0),
            signatures: sig
        });
    }

    // @dev Create a new Safe instance with a minimimal proxy and implementation.
    function newSafe(Signer[] memory _signers) internal returns (Safe safe_) {
        SafeProxyFactory safeProxyFactory = new SafeProxyFactory();
        Safe safeSingleton = new Safe();

        bytes memory initData = abi.encodeWithSelector(
            Safe.setup.selector, _signers, 2, address(0), hex"", address(0), address(0), 0, address(0)
        );

        safe_ = Safe(payable(safeProxyFactory.createProxyWithNonce(address(safeSingleton), initData, block.timestamp)));
    }

    function setUp() public {
        // Create 3 signers
        for (uint256 i; i < 3; i++) {
            signers.push(newSigner(i));
        }

        Signer[] memory signers_ = signers;
        safe = newSafe(signers_);
        livenessGuard = new LivenessGuard(safe);

        // enable the module
        bytes memory data = abi.encodeCall(ModuleManager.enableModule, (address(livenessGuard)));
        bytes memory sig1 = signTransaction(signers[0].pk, address(safe), 0, data);
        bytes memory sig2 = signTransaction(signers[1].pk, address(safe), 0, data);
        bytes memory sigs = bytes.concat(sig1, sig2);
        safe.execTransaction({
            to: address(safe),
            value: 0,
            data: data,
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(0),
            signatures: sigs
        });
    }
}

contract LivnessGuard_TestCheckTx is LivnessGuard_TestInit {
    function test_checkTransaction_succeeds() external {
        Signer[] memory signers_ = signers;
        exec(signers, address(1111), hex"abba");

        for (uint256 i; i < signers.length; i++) {
            assertEq(livenessGuard.lastSigned(signers[i].owner), block.timestamp);
        }
    }
}

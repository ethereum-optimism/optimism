// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { LegacyCrossDomainUtils } from "src/libraries/LegacyCrossDomainUtils.sol";

// Target contract
import { Hashing } from "src/libraries/Hashing.sol";

contract Hashing_Harness {
    function hashSuperRootProof(Types.SuperRootProof memory _proof) external pure returns (bytes32) {
        return Hashing.hashSuperRootProof(_proof);
    }
}

contract Hashing_hashDepositSource_Test is CommonTest {
    /// @notice Tests that hashDepositSource returns the correct hash in a simple case.
    function test_hashDepositSource_succeeds() external pure {
        assertEq(
            Hashing.hashDepositSource(0xd25df7858efc1778118fb133ac561b138845361626dfb976699c5287ed0f4959, 0x1),
            0xf923fb07134d7d287cb52c770cc619e17e82606c21a875c92f4c63b65280a5cc
        );
    }
}

contract Hashing_hashCrossDomainMessage_Test is CommonTest {
    /// @notice Tests that hashCrossDomainMessage returns the correct hash in a simple case.
    function testDiff_hashCrossDomainMessage_succeeds(
        uint240 _nonce,
        uint16 _version,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        // Ensure the version is valid.
        uint16 version = uint16(bound(uint256(_version), 0, 1));
        uint256 nonce = Encoding.encodeVersionedNonce(_nonce, version);

        assertEq(
            Hashing.hashCrossDomainMessage(nonce, _sender, _target, _value, _gasLimit, _data),
            ffi.hashCrossDomainMessage(nonce, _sender, _target, _value, _gasLimit, _data)
        );
    }

    /// @notice Tests that hashCrossDomainMessageV0 matches the hash of the legacy encoding.
    function testFuzz_hashCrossDomainMessageV0_matchesLegacy_succeeds(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        external
        pure
    {
        assertEq(
            keccak256(LegacyCrossDomainUtils.encodeXDomainCalldata(_target, _sender, _message, _messageNonce)),
            Hashing.hashCrossDomainMessageV0(_target, _sender, _message, _messageNonce)
        );
    }
}

contract Hashing_hashWithdrawal_Test is CommonTest {
    /// @notice Tests that hashWithdrawal returns the correct hash in a simple case.
    function testDiff_hashWithdrawal_succeeds(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        assertEq(
            Hashing.hashWithdrawal(Types.WithdrawalTransaction(_nonce, _sender, _target, _value, _gasLimit, _data)),
            ffi.hashWithdrawal(_nonce, _sender, _target, _value, _gasLimit, _data)
        );
    }
}

contract Hashing_hashOutputRootProof_Test is CommonTest {
    /// @notice Tests that hashOutputRootProof returns the correct hash in a simple case.
    function testDiff_hashOutputRootProof_succeeds(
        bytes32 _stateRoot,
        bytes32 _messagePasserStorageRoot,
        bytes32 _latestBlockhash
    )
        external
    {
        bytes32 version = 0;
        assertEq(
            Hashing.hashOutputRootProof(
                Types.OutputRootProof({
                    version: version,
                    stateRoot: _stateRoot,
                    messagePasserStorageRoot: _messagePasserStorageRoot,
                    latestBlockhash: _latestBlockhash
                })
            ),
            ffi.hashOutputRootProof(version, _stateRoot, _messagePasserStorageRoot, _latestBlockhash)
        );
    }
}

contract Hashing_hashDepositTransaction_Test is CommonTest {
    /// @notice Tests that hashDepositTransaction returns the correct hash in a simple case.
    function testDiff_hashDepositTransaction_succeeds(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gas,
        bytes memory _data,
        uint64 _logIndex
    )
        external
    {
        assertEq(
            Hashing.hashDepositTransaction(
                Types.UserDepositTransaction(
                    _from,
                    _to,
                    false, // isCreate
                    _value,
                    _mint,
                    _gas,
                    _data,
                    bytes32(uint256(0)),
                    _logIndex
                )
            ),
            ffi.hashDepositTransaction(_from, _to, _mint, _value, _gas, _data, _logIndex)
        );
    }
}

contract Hashing_hashSuperRootProof_Test is CommonTest {
    Hashing_Harness internal harness;

    /// @notice Sets up the test.
    function setUp() public override {
        super.setUp();
        harness = new Hashing_Harness();
    }

    /// @notice Tests that the Solidity impl of hashSuperRootProof matches the FFI impl
    /// @param _proof The super root proof to test.
    function testDiff_hashSuperRootProof_succeeds(Types.SuperRootProof memory _proof) external {
        // Make sure the proof has the right version.
        _proof.version = 0x01;

        // Make sure the proof has at least one output root.
        if (_proof.outputRoots.length == 0) {
            _proof.outputRoots = new Types.OutputRootWithChainId[](1);
            _proof.outputRoots[0] = Types.OutputRootWithChainId({
                chainId: vm.randomUint(0, type(uint64).max),
                root: bytes32(vm.randomUint())
            });
        }

        // Encode using the Solidity implementation
        bytes32 hash1 = harness.hashSuperRootProof(_proof);

        // Encode using the FFI implementation
        bytes32 hash2 = ffi.hashSuperRootProof(_proof);

        // Compare the results
        assertEq(hash1, hash2, "Solidity and FFI implementations should match");
    }

    /// @notice Tests that hashSuperRootProof reverts when the version is incorrect.
    /// @param _proof The super root proof to test.
    function testFuzz_hashSuperRootProof_wrongVersion_reverts(Types.SuperRootProof memory _proof) external {
        // 0x01 is the correct version, so we need any other version.
        if (_proof.version == 0x01) {
            _proof.version = 0x00;
        }

        // Should always revert when the version is incorrect.
        vm.expectRevert(Encoding.Encoding_InvalidSuperRootVersion.selector);
        harness.hashSuperRootProof(_proof);
    }
}

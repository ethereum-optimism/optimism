pragma solidity 0.8.15;

import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

contract EchidnaFuzzHashing {
    bool internal failedCrossDomainHashHighVersion;
    bool internal failedCrossDomainHashV0;
    bool internal failedCrossDomainHashV1;

    /**
     * @notice Takes the necessary parameters to perform a cross domain hash with a randomly
     * generated version. Only schema versions 0 and 1 are supported and all others should revert.
     */
    function testHashCrossDomainMessageHighVersion(
        uint16 _version,
        uint240 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) public {
        // generate the versioned nonce
        uint256 encodedNonce = Encoding.encodeVersionedNonce(_nonce, _version);

        // hash the cross domain message. we don't need to store the result since the function
        // validates and should revert if an invalid version (>1) is encoded
        Hashing.hashCrossDomainMessage(encodedNonce, _sender, _target, _value, _gasLimit, _data);

        // check that execution never makes it this far for an invalid version
        if (_version > 1) {
            failedCrossDomainHashHighVersion = true;
        }
    }

    /**
     * @notice Takes the necessary parameters to perform a cross domain hash using the v0 schema
     * and compares the output of a call to the unversioned function to the v0 function directly
     */
    function testHashCrossDomainMessageV0(
        uint240 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) public {
        // generate the versioned nonce with the version set to 0
        uint256 encodedNonce = Encoding.encodeVersionedNonce(_nonce, 0);

        // hash the cross domain message using the unversioned and versioned functions for
        // comparison
        bytes32 sampleHash1 = Hashing.hashCrossDomainMessage(
            encodedNonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
        bytes32 sampleHash2 = Hashing.hashCrossDomainMessageV0(
            _target,
            _sender,
            _data,
            encodedNonce
        );

        // check that the output of both functions matches
        if (sampleHash1 != sampleHash2) {
            failedCrossDomainHashV0 = true;
        }
    }

    /**
     * @notice Takes the necessary parameters to perform a cross domain hash using the v1 schema
     * and compares the output of a call to the unversioned function to the v1 function directly
     */
    function testHashCrossDomainMessageV1(
        uint240 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) public {
        // generate the versioned nonce with the version set to 1
        uint256 encodedNonce = Encoding.encodeVersionedNonce(_nonce, 1);

        // hash the cross domain message using the unversioned and versioned functions for
        // comparison
        bytes32 sampleHash1 = Hashing.hashCrossDomainMessage(
            encodedNonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
        bytes32 sampleHash2 = Hashing.hashCrossDomainMessageV1(
            encodedNonce,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );

        // check that the output of both functions matches
        if (sampleHash1 != sampleHash2) {
            failedCrossDomainHashV1 = true;
        }
    }

    /**
     * @custom:invariant `hashCrossDomainMessage` reverts if `version` is > `1`.
     *
     * The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`.
     */
    function echidna_hash_xdomain_msg_high_version() public view returns (bool) {
        // ASSERTION: A call to hashCrossDomainMessage will never succeed for a version > 1
        return !failedCrossDomainHashHighVersion;
    }

    /**
     * @custom:invariant `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0`
     * are equivalent.
     *
     * If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be
     * equivalent.
     */
    function echidna_hash_xdomain_msg_0() public view returns (bool) {
        // ASSERTION: A call to hashCrossDomainMessage and hashCrossDomainMessageV0
        // should always match when the version passed is 0
        return !failedCrossDomainHashV0;
    }

    /**
     * @custom:invariant `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1`
     * are equivalent.
     *
     * If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be
     * equivalent.
     */
    function echidna_hash_xdomain_msg_1() public view returns (bool) {
        // ASSERTION: A call to hashCrossDomainMessage and hashCrossDomainMessageV1
        // should always match when the version passed is 1
        return !failedCrossDomainHashV1;
    }
}

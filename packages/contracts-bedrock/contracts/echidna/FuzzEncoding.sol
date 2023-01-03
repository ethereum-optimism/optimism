pragma solidity 0.8.15;

import { Encoding } from "../libraries/Encoding.sol";

contract EchidnaFuzzEncoding {
    bool internal failedRoundtripAToB;
    bool internal failedRoundtripBToA;

    /**
     * @notice Takes a pair of integers to be encoded into a versioned nonce with the
     *         Encoding library and then decoded and updates the test contract's state
     *         indicating if the round trip encoding failed.
     */
    function testRoundTripAToB(uint240 _nonce, uint16 _version) public {
        // Encode the nonce and version
        uint256 encodedVersionedNonce = Encoding.encodeVersionedNonce(_nonce, _version);

        // Decode the nonce and version
        uint240 decodedNonce;
        uint16 decodedVersion;

        (decodedNonce, decodedVersion) = Encoding.decodeVersionedNonce(encodedVersionedNonce);

        // If our round trip encoding did not return the original result, set our state.
        if ((decodedNonce != _nonce) || (decodedVersion != _version)) {
            failedRoundtripAToB = true;
        }
    }

    /**
     * @notice Takes an integer representing a packed version and nonce and attempts
     *         to decode them using the Encoding library before re-encoding and updates
     *         the test contract's state indicating if the round trip encoding failed.
     */
    function testRoundTripBToA(uint256 _versionedNonce) public {
        // Decode the nonce and version
        uint240 decodedNonce;
        uint16 decodedVersion;

        (decodedNonce, decodedVersion) = Encoding.decodeVersionedNonce(_versionedNonce);

        // Encode the nonce and version
        uint256 encodedVersionedNonce = Encoding.encodeVersionedNonce(decodedNonce, decodedVersion);

        // If our round trip encoding did not return the original result, set our state.
        if (encodedVersionedNonce != _versionedNonce) {
            failedRoundtripBToA = true;
        }
    }

    /**
     * @custom:invariant `testRoundTripAToB` never fails.
     *
     * Asserts that a raw versioned nonce can be encoded / decoded to reach the same raw value.
     */
    function echidna_round_trip_encoding_AToB() public view returns (bool) {
        // ASSERTION: The round trip encoding done in testRoundTripAToB(...)
        return !failedRoundtripAToB;
    }

    /**
     * @custom:invariant `testRoundTripBToA` never fails.
     *
     * Asserts that an encoded versioned nonce can always be decoded / re-encoded to reach
     * the same encoded value.
     */
    function echidna_round_trip_encoding_BToA() public view returns (bool) {
        // ASSERTION: The round trip encoding done in testRoundTripBToA should never
        // fail.
        return !failedRoundtripBToA;
    }
}

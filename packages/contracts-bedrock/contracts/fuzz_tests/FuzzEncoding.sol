import { Encoding } from "../libraries/Encoding.sol";

contract FuzzEncoding {
    bool failedRoundtripAToB;
    bool failedRoundtripBToA;

    /**
     * @notice Takes a pair of integers to be encoded into a versioned nonce with the
     *         Encoding library and then decoded and updates the test contract's state
     *         indicating if the round trip encoding failed.
     */
    function testRoundTripAToB(uint240 _nonce,  uint16 _version) public {
        // encode the address
        uint256 encodedVersionedNonce = Encoding.encodeVersionedNonce(_nonce, _version);

        // Unalias our address
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

        // Unalias our address
        uint240 decodedNonce;
        uint16 decodedVersion;

        (decodedNonce, decodedVersion) = Encoding.decodeVersionedNonce(_versionedNonce);

        // encode the address
        uint256 encodedVersionedNonce = Encoding.encodeVersionedNonce(decodedNonce, decodedVersion);

        // If our round trip encoding did not return the original result, set our state.
        if (encodedVersionedNonce != _versionedNonce) {
            failedRoundtripBToA = true;
        }
    }

    /**
     * @notice Verifies that testRoundTrip(...) did not ever fail.
     */
    function echidna_round_trip_encoding() public view returns(bool) {
        // ASSERTION: The round trip encoding done in testRoundTripAToB(...)/BToA(...) should never fail.
        return !failedRoundtripAToB && !failedRoundtripBToA;
    }
}
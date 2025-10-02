// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { DevFeatures } from "src/libraries/DevFeatures.sol";

contract DevFeatures_isEnabled_Test is Test {
    bytes32 internal constant FEATURE_A = bytes32(0x0000000000000000000000000000000000000000000000000000000000000001);
    bytes32 internal constant FEATURE_B = bytes32(0x0000000000000000000000000000000000000000000000000000000000000100);
    bytes32 internal constant FEATURE_C = bytes32(0x1000000000000000000000000000000000000000000000000000000000000000);

    bytes32 internal constant FEATURES_AB = FEATURE_A | FEATURE_B;
    bytes32 internal constant FEATURES_ABC = FEATURE_A | FEATURE_B | FEATURE_C;
    bytes32 internal constant FEATURES_AB_INVERTED = ~FEATURES_AB;
    bytes32 internal constant EMPTY_FEATURES =
        bytes32(0x0000000000000000000000000000000000000000000000000000000000000000);
    bytes32 internal constant ALL_FEATURES = bytes32(0x1111111111111111111111111111111111111111111111111111111111111111);

    function test_isDevFeatureEnabled_checkSingleFeatureExactMatch_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURE_B));
    }

    function test_isDevFeatureEnabled_checkSingleFeatureAgainstSuperset_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURE_B));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_ABC, FEATURE_A));
    }

    function test_isDevFeatureEnabled_checkSingleFeatureAgainstAll_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURE_B));
    }

    function test_isDevFeatureEnabled_checkSingleFeatureAgainstMismatchedBitmap_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURE_B));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_B));
    }

    function test_isDevFeatureEnabled_checkSingleFeatureAgainstEmptyBitmap_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURE_B));
    }

    function test_isDevFeatureEnabled_checkCombinedFeaturesAgainstExactMatch_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURES_AB));
    }

    function test_isDevFeatureEnabled_checkCombinedFeatureAgainstSuperset_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURES_AB));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_ABC, FEATURES_AB));
    }

    function test_isDevFeatureEnabled_checkCombinedFeaturesAgainstSubset_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURES_AB));
    }

    function test_isDevFeatureEnabled_checkCombinedFeaturesAgainstMismatchedBitmap_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_C, FEATURES_AB));
    }

    function test_isDevFeatureEnabled_checkEmptyVsEmpty_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, EMPTY_FEATURES));
    }

    function test_isDevFeatureEnabled_checkAllVsAll_works() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, ALL_FEATURES));
    }

    function test_isDevFeatureEnabled_checkEmptyAgainstAll_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, EMPTY_FEATURES));
    }

    function test_isDevFeatureEnabled_checkAllAgainstEmpty_works() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, ALL_FEATURES));
    }
}

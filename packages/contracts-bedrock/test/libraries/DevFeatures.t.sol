// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { DevFeatures } from "src/libraries/DevFeatures.sol";

contract DevFeatures_isDevFeatureEnabled_Test is Test {
    bytes32 internal constant FEATURE_A = bytes32(0x0000000000000000000000000000000000000000000000000000000000000001);
    bytes32 internal constant FEATURE_B = bytes32(0x0000000000000000000000000000000000000000000000000000000000000100);
    bytes32 internal constant FEATURE_C = bytes32(0x1000000000000000000000000000000000000000000000000000000000000000);

    bytes32 internal constant FEATURES_AB = FEATURE_A | FEATURE_B;
    bytes32 internal constant FEATURES_ABC = FEATURE_A | FEATURE_B | FEATURE_C;
    bytes32 internal constant FEATURES_AB_INVERTED = ~FEATURES_AB;
    bytes32 internal constant EMPTY_FEATURES =
        bytes32(0x0000000000000000000000000000000000000000000000000000000000000000);
    bytes32 internal constant ALL_FEATURES = bytes32(0x1111111111111111111111111111111111111111111111111111111111111111);

    /// @notice Tests that a single feature matches itself exactly.
    function test_isDevFeatureEnabled_singleFeatureExactMatch_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURE_B));
    }

    /// @notice Tests that a single feature is found within a superset bitmap.
    function test_isDevFeatureEnabled_singleFeatureInSuperset_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURE_B));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_ABC, FEATURE_A));
    }

    /// @notice Tests that a single feature is found within the ALL_FEATURES bitmap.
    function test_isDevFeatureEnabled_singleFeatureInAllFeatures_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURE_A));
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURE_B));
    }

    /// @notice Tests that a single feature is not found in a mismatched bitmap.
    function test_isDevFeatureEnabled_singleFeatureMismatchedBitmap_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURE_B));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURE_B));
    }

    /// @notice Tests that a single feature is not found in an empty bitmap.
    function test_isDevFeatureEnabled_singleFeatureEmptyBitmap_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURE_A));
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURE_B));
    }

    /// @notice Tests that combined features match exactly.
    function test_isDevFeatureEnabled_combinedFeaturesExactMatch_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_AB, FEATURES_AB));
    }

    /// @notice Tests that combined features are found within a superset bitmap.
    function test_isDevFeatureEnabled_combinedFeaturesInSuperset_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, FEATURES_AB));
        assertTrue(DevFeatures.isDevFeatureEnabled(FEATURES_ABC, FEATURES_AB));
    }

    /// @notice Tests that combined features are not found in a subset bitmap.
    function test_isDevFeatureEnabled_combinedFeaturesInSubset_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_A, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_B, FEATURES_AB));
    }

    /// @notice Tests that combined features are not found in a mismatched bitmap.
    function test_isDevFeatureEnabled_combinedFeaturesMismatchedBitmap_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURES_AB_INVERTED, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, FEATURES_AB));
        assertFalse(DevFeatures.isDevFeatureEnabled(FEATURE_C, FEATURES_AB));
    }

    /// @notice Tests that empty feature vs empty bitmap returns false.
    function test_isDevFeatureEnabled_emptyVsEmpty_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, EMPTY_FEATURES));
    }

    /// @notice Tests that ALL_FEATURES vs ALL_FEATURES returns true.
    function test_isDevFeatureEnabled_allVsAll_succeeds() public pure {
        assertTrue(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, ALL_FEATURES));
    }

    /// @notice Tests that empty feature against any bitmap returns false.
    function test_isDevFeatureEnabled_emptyFeatureAgainstAll_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(ALL_FEATURES, EMPTY_FEATURES));
    }

    /// @notice Tests that ALL_FEATURES against empty bitmap returns false.
    function test_isDevFeatureEnabled_allFeaturesAgainstEmpty_succeeds() public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(EMPTY_FEATURES, ALL_FEATURES));
    }

    /// @notice Fuzz test: any non-zero feature should match itself exactly.
    function testFuzz_isDevFeatureEnabled_featureMatchesSelf_succeeds(bytes32 _feature) public pure {
        vm.assume(_feature != bytes32(0));
        assertTrue(DevFeatures.isDevFeatureEnabled(_feature, _feature));
    }

    /// @notice Fuzz test: empty feature always returns false regardless of bitmap.
    function testFuzz_isDevFeatureEnabled_emptyFeatureAlwaysFalse_succeeds(bytes32 _bitmap) public pure {
        assertFalse(DevFeatures.isDevFeatureEnabled(_bitmap, EMPTY_FEATURES));
    }

    /// @notice Fuzz test: feature is found when bitmap is a superset containing the feature.
    function testFuzz_isDevFeatureEnabled_featureInSuperset_succeeds(bytes32 _bitmap, bytes32 _feature) public pure {
        vm.assume(_feature != bytes32(0));
        bytes32 superset = _bitmap | _feature;
        assertTrue(DevFeatures.isDevFeatureEnabled(superset, _feature));
    }

    /// @notice Fuzz test: feature not found when bitmap has none of the feature's bits.
    function testFuzz_isDevFeatureEnabled_featureNotInDisjointBitmap_succeeds(bytes32 _feature) public pure {
        vm.assume(_feature != bytes32(0));
        bytes32 disjointBitmap = ~_feature;
        assertFalse(DevFeatures.isDevFeatureEnabled(disjointBitmap, _feature));
    }
}

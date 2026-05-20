package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDevFeaturesSol_MultiLineConstant(t *testing.T) {
	// DevFeatures.sol constants may be split across lines.
	src := `
		library DevFeatures {
		    bytes32 public constant OPTIMISM_PORTAL_INTEROP =
		        bytes32(0x0000000000000000000000000000000000000000000000000000000000000001);
		    bytes32 public constant CANNON_KONA = bytes32(0x0000000000000000000000000000000000000000000000000000000000000010);
		}
	`
	got := parseDevFeaturesSol(src)
	require.Len(t, got, 2)
	require.Equal(t, "OPTIMISM_PORTAL_INTEROP", got[0].Name)
	require.Equal(t, strings.Repeat("0", 63)+"1", got[0].Hex)
	require.Equal(t, "CANNON_KONA", got[1].Name)
	require.Equal(t, strings.Repeat("0", 62)+"10", got[1].Hex)
}

func TestParseDevFeaturesGo(t *testing.T) {
	src := `
		package devfeatures

		import "github.com/ethereum/go-ethereum/common"

		var (
			OptimismPortalInteropFlag = common.HexToHash("0x01")
			CannonKonaFlag = common.HexToHash("0x02")
			L2CMFlag = common.HexToHash("0x10")
		)

		func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
			if hasFlag(flag, L2CMFlag) {
				return true
			}
			if flag == CannonKonaFlag {
				return true
			}
			return false
		}
	`
	consts, hardcoded, err := parseDevFeaturesGo(src)
	require.NoError(t, err)
	require.Equal(t, []GoDevFeatureConst{
		{Name: "OptimismPortalInteropFlag", Hex: strings.Repeat("0", 63) + "1"},
		{Name: "CannonKonaFlag", Hex: strings.Repeat("0", 63) + "2"},
		{Name: "L2CMFlag", Hex: strings.Repeat("0", 62) + "10"},
	}, consts)
	require.Equal(t, []string{"L2CMFlag", "CannonKonaFlag"}, hardcoded)
}

func TestParseHardcodedDevFeaturesSol(t *testing.T) {
	src := `
		library DevFeatures {
			function isDevFeatureEnabled(bytes32 _bitmap, bytes32 _feature) internal pure returns (bool) {
				if (hasFlag(_feature, L2CM)) return true;
				if (_feature == DevFeatures.CANNON_KONA) {
					return true;
				}
				if (hasFlag(_bitmap, _feature)) return true;
				return _feature != 0 && hasFlag(_bitmap, _feature);
			}
		}
	`
	got, err := parseHardcodedDevFeaturesSol(src)
	require.NoError(t, err)
	require.Equal(t, []string{"L2CM", "CANNON_KONA"}, got)
}

func TestParseHardcodedDevFeaturesSol_MissingFunctionFails(t *testing.T) {
	_, err := parseHardcodedDevFeaturesSol(`library DevFeatures {}`)
	require.ErrorContains(t, err, "missing isDevFeatureEnabled function")
}

func TestParseDevFeaturesGo_MissingFunctionFails(t *testing.T) {
	src := `
		package devfeatures

		import "github.com/ethereum/go-ethereum/common"

		var L2CMFlag = common.HexToHash("0x10")
	`
	_, _, err := parseDevFeaturesGo(src)
	require.ErrorContains(t, err, "missing IsDevFeatureEnabled function")
}

func TestParseDevFeaturesGo_InvalidHexLiteralFails(t *testing.T) {
	src := `
		package devfeatures

		import "github.com/ethereum/go-ethereum/common"

		var L2CMFlag = common.HexToHash("0xzz")

		func IsDevFeatureEnabled(bitmap, flag common.Hash) bool {
			return false
		}
	`
	_, _, err := parseDevFeaturesGo(src)
	require.ErrorContains(t, err, `common.HexToHash literal "0xzz" is not valid hex`)
}

func TestParseConfigSol_EnvVarStringIsCanonical(t *testing.T) {
	// The checker uses the environment variable string as canonical because
	// reader function names can differ from feature names.
	src := `
		library Config {
		    function devFeatureInterop() internal view returns (bool) {
		        return vm.envOr("DEV_FEATURE__OPTIMISM_PORTAL_INTEROP", false);
		    }
		    function l2ForkChain() internal view returns (string memory) {
		        return vm.envOr("L2_FORK_CHAIN", string("op"));
		    }
		}
	`
	got := parseConfigSol(src)
	require.Len(t, got, 1, "non-feature env readers should be filtered out")
	require.Equal(t, "devFeatureInterop", got[0].FuncName)
	require.Equal(t, "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP", got[0].EnvVar)
}

func TestParseFeatureFlagsSol(t *testing.T) {
	src := `
		function resolveFeaturesFromEnv() public {
		    if (Config.devFeatureInterop()) {
		        console.log("Setup: DEV_FEATURE__OPTIMISM_PORTAL_INTEROP is enabled");
		        devFeatureBitmap |= DevFeatures.OPTIMISM_PORTAL_INTEROP;
		    }
		}
		function getFeatureName(bytes32 _feature) public pure returns (string memory) {
		    if (_feature == DevFeatures.OPTIMISM_PORTAL_INTEROP) {
		        return "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP";
		    } else if (_feature == Features.ETH_LOCKBOX) {
		        return "SYS_FEATURE__ETH_LOCKBOX";
		    }
		}
	`
	got := parseFeatureFlagsSol(src)
	require.Equal(t, "OPTIMISM_PORTAL_INTEROP", got.ResolveDev["devFeatureInterop"])
	require.Equal(t, "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP", got.NameMap["DevFeatures.OPTIMISM_PORTAL_INTEROP"])
	require.Equal(t, "SYS_FEATURE__ETH_LOCKBOX", got.NameMap["Features.ETH_LOCKBOX"])
}

func TestScanSysFeatureSetupContent_CouplesReaderAndActivation(t *testing.T) {
	src := `
		function setUp() public {
		    if (Config.sysFeatureEthLockbox()) {
		        if (!systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX)) {
		            systemConfig.setFeature(Features.ETH_LOCKBOX, true);
		        }
		    }
		    if (Config.sysFeatureInterop()) {
		        console.log("read but not activated");
		    }
		    systemConfig.setFeature(Features.INTEROP, true);
		    if (Config.sysFeatureCustomGasToken()) {
		        cfg.setUseCustomGasToken(true);
		    }
		}
	`
	setup := newSysFeatureSetup()
	scanSysFeatureSetupContent(src, &setup)
	require.True(t, setup.Readers["sysFeatureEthLockbox"])
	require.True(t, setup.Readers["sysFeatureInterop"])
	require.True(t, setup.Readers["sysFeatureCustomGasToken"])
	require.True(t, setup.activates("sysFeatureEthLockbox", "ETH_LOCKBOX"))
	require.True(t, setup.activates("sysFeatureCustomGasToken", "CUSTOM_GAS_TOKEN"))
	require.False(t, setup.activates("sysFeatureInterop", "INTEROP"), "activation outside the Config branch must not count")
}

func TestExactlyOneBit(t *testing.T) {
	cases := []struct {
		hex  string
		want bool
	}{
		{strings.Repeat("0", 63) + "1", true},
		{strings.Repeat("0", 63) + "2", true},
		{strings.Repeat("0", 63) + "4", true},
		{strings.Repeat("0", 63) + "8", true},
		{strings.Repeat("0", 62) + "10", true},
		{strings.Repeat("0", 64), false},        // zero
		{strings.Repeat("0", 63) + "3", false},  // bitmap with two bits set
		{strings.Repeat("0", 62) + "11", false}, // two bits set
		{strings.Repeat("0", 63) + "f", false},  // single nibble, multiple bits
	}
	for _, c := range cases {
		require.Equal(t, c.want, exactlyOneBit(c.hex), "input=0x%s", c.hex)
	}
}

func goodRegistry() Registry {
	return Registry{
		Version: 1,
		Features: []Feature{
			{Name: "OPTIMISM_PORTAL_INTEROP", Type: "dev", Lifecycle: "active"},
			{Name: "L2CM", Type: "dev", Lifecycle: "hardcoded-on"},
			{Name: "DEPLOY_V2_DISPUTE_GAMES", Type: "dev", Lifecycle: "legacy"},
			{Name: "ETH_LOCKBOX", Type: "sys", Lifecycle: "active"},
			{Name: "CUSTOM_GAS_TOKEN", Type: "sys", Lifecycle: "active"},
			{Name: "INTEROP", Type: "sys", Lifecycle: "active"},
		},
		Combinations: Combinations{
			Matrix: [][]string{
				{},
				{"CUSTOM_GAS_TOKEN"},
				{"ETH_LOCKBOX"},
				{"OPTIMISM_PORTAL_INTEROP"},
				{"OPTIMISM_PORTAL_INTEROP", "ETH_LOCKBOX", "INTEROP"},
			},
			Requires: map[string][]string{
				"CUSTOM_GAS_TOKEN": {"L2CM"},
				"INTEROP":          {"ETH_LOCKBOX", "OPTIMISM_PORTAL_INTEROP"},
			},
			Excludes: [][]string{{"CUSTOM_GAS_TOKEN", "INTEROP"}},
		},
	}
}

func TestValidateRegistry_Happy(t *testing.T) {
	require.Empty(t, validateRegistry(goodRegistry()))
}

func TestValidateRegistry_WrongVersion(t *testing.T) {
	r := goodRegistry()
	r.Version = 2
	errs := validateRegistry(r)
	require.Contains(t, joinErrs(errs), "version must be 1")
}

func TestValidateRegistry_DuplicateFeature(t *testing.T) {
	r := goodRegistry()
	r.Features = append(r.Features, Feature{Name: "ETH_LOCKBOX", Type: "sys", Lifecycle: "active"})
	require.Contains(t, joinErrs(validateRegistry(r)), `duplicate feature "ETH_LOCKBOX"`)
}

func TestValidateRegistry_BadName(t *testing.T) {
	r := goodRegistry()
	r.Features[0].Name = "lowercase_name"
	require.Contains(t, joinErrs(validateRegistry(r)), "not UPPER_SNAKE_CASE")
}

func TestValidateRegistry_BadType(t *testing.T) {
	r := goodRegistry()
	r.Features[0].Type = "weird"
	require.Contains(t, joinErrs(validateRegistry(r)), "invalid type")
}

func TestValidateRegistry_BadLifecycle(t *testing.T) {
	r := goodRegistry()
	r.Features[0].Lifecycle = "deprecated"
	require.Contains(t, joinErrs(validateRegistry(r)), "invalid lifecycle")
}

func TestValidateRegistry_TwoBaselineRows(t *testing.T) {
	r := goodRegistry()
	r.Combinations.Matrix = append(r.Combinations.Matrix, []string{})
	require.Contains(t, joinErrs(validateRegistry(r)), "exactly one baseline")
}

func TestValidateRegistry_LegacyInMatrix(t *testing.T) {
	r := goodRegistry()
	r.Combinations.Matrix = append(r.Combinations.Matrix, []string{"DEPLOY_V2_DISPUTE_GAMES"})
	require.Contains(t, joinErrs(validateRegistry(r)), "legacy feature DEPLOY_V2_DISPUTE_GAMES must not appear")
}

func TestValidateRegistry_ActiveMissingFromMatrix(t *testing.T) {
	r := goodRegistry()
	// Remove the INTEROP row.
	r.Combinations.Matrix = r.Combinations.Matrix[:4]
	require.Contains(t, joinErrs(validateRegistry(r)), "active feature INTEROP must appear")
}

func TestValidateRegistry_RowMissingRequiredPrereq(t *testing.T) {
	r := goodRegistry()
	// Add an INTEROP row without its prerequisites.
	r.Combinations.Matrix = append(r.Combinations.Matrix, []string{"INTEROP"})
	require.Contains(t, joinErrs(validateRegistry(r)), "missing required prerequisite")
}

func TestValidateRegistry_HardcodedOnPrereqDoesNotNeedRow(t *testing.T) {
	r := goodRegistry()
	require.Empty(t, validateRegistry(r))
}

func TestValidateRegistry_RowViolatesExcludes(t *testing.T) {
	r := goodRegistry()
	r.Combinations.Matrix = append(r.Combinations.Matrix,
		[]string{"CUSTOM_GAS_TOKEN", "INTEROP"})
	require.Contains(t, joinErrs(validateRegistry(r)), "violates excludes")
}

func TestValidateRegistry_RowNotInDeclarationOrder(t *testing.T) {
	r := goodRegistry()
	r.Combinations.Matrix[4] = []string{"INTEROP", "OPTIMISM_PORTAL_INTEROP", "ETH_LOCKBOX"}
	require.Contains(t, joinErrs(validateRegistry(r)), "not in declaration order")
}

func TestValidateRegistry_RequiresUnknownFeature(t *testing.T) {
	r := goodRegistry()
	r.Combinations.Requires["INTEROP"] = append(r.Combinations.Requires["INTEROP"], "GHOST")
	require.Contains(t, joinErrs(validateRegistry(r)), `unknown feature "GHOST"`)
}

func TestValidateDefinitions_Happy(t *testing.T) {
	r := goodRegistry()
	devConsts := []DevFeatureConst{
		{Name: "OPTIMISM_PORTAL_INTEROP", Hex: strings.Repeat("0", 63) + "1"},
		{Name: "L2CM", Hex: strings.Repeat("0", 62) + "10"},
		{Name: "DEPLOY_V2_DISPUTE_GAMES", Hex: strings.Repeat("0", 61) + "100"},
	}
	sysConsts := []SysFeatureConst{
		{Name: "ETH_LOCKBOX", Literal: "ETH_LOCKBOX"},
		{Name: "CUSTOM_GAS_TOKEN", Literal: "CUSTOM_GAS_TOKEN"},
		{Name: "INTEROP", Literal: "INTEROP"},
	}
	require.Empty(t, validateDefinitions(r, devConsts, sysConsts))
}

func TestValidateDefinitions_MissingInSolidity(t *testing.T) {
	r := goodRegistry()
	errs := validateDefinitions(r, nil, nil)
	require.Contains(t, joinErrs(errs), "DevFeatures.sol missing constant for dev feature OPTIMISM_PORTAL_INTEROP")
	require.Contains(t, joinErrs(errs), "Features.sol missing constant for sys feature ETH_LOCKBOX")
}

func TestValidateDefinitions_ExtraInSolidity(t *testing.T) {
	r := goodRegistry()
	devConsts := []DevFeatureConst{
		{Name: "OPTIMISM_PORTAL_INTEROP", Hex: strings.Repeat("0", 63) + "1"},
		{Name: "L2CM", Hex: strings.Repeat("0", 62) + "10"},
		{Name: "DEPLOY_V2_DISPUTE_GAMES", Hex: strings.Repeat("0", 61) + "100"},
		{Name: "GHOST", Hex: strings.Repeat("0", 60) + "1000"},
	}
	sysConsts := []SysFeatureConst{
		{Name: "ETH_LOCKBOX", Literal: "ETH_LOCKBOX"},
		{Name: "CUSTOM_GAS_TOKEN", Literal: "CUSTOM_GAS_TOKEN"},
		{Name: "INTEROP", Literal: "INTEROP"},
	}
	errs := validateDefinitions(r, devConsts, sysConsts)
	require.Contains(t, joinErrs(errs), "DevFeatures.sol declares GHOST but feature-flags.yaml does not list it")
}

func TestValidateDefinitions_DevHexRules(t *testing.T) {
	r := Registry{Features: []Feature{{Name: "A", Type: "dev", Lifecycle: "active"}}}
	bad := []DevFeatureConst{{Name: "A", Hex: strings.Repeat("0", 63) + "3"}}
	errs := validateDefinitions(r, bad, nil)
	require.Contains(t, joinErrs(errs), "must have exactly one bit set")

	zero := []DevFeatureConst{{Name: "A", Hex: strings.Repeat("0", 64)}}
	errs = validateDefinitions(r, zero, nil)
	require.Contains(t, joinErrs(errs), "hex value is zero")

	long := []DevFeatureConst{{Name: "A", Hex: strings.Repeat("0", 64) + "1"}}
	errs = validateDefinitions(r, long, nil)
	require.Contains(t, joinErrs(errs), "must be 64 hex characters")

	dup := []DevFeatureConst{
		{Name: "A", Hex: strings.Repeat("0", 63) + "1"},
		{Name: "B", Hex: strings.Repeat("0", 63) + "1"},
	}
	r.Features = append(r.Features, Feature{Name: "B", Type: "dev", Lifecycle: "active"})
	errs = validateDefinitions(r, dup, nil)
	require.Contains(t, joinErrs(errs), "duplicates")
}

func TestValidateDefinitions_SysLiteralMustMatch(t *testing.T) {
	r := Registry{Features: []Feature{{Name: "FOO", Type: "sys", Lifecycle: "active"}}}
	sys := []SysFeatureConst{{Name: "FOO", Literal: "Foo"}}
	errs := validateDefinitions(r, nil, sys)
	require.Contains(t, joinErrs(errs), `string literal "Foo"`)
}

func TestValidateGoParity_HexMismatch(t *testing.T) {
	solConsts := []DevFeatureConst{
		{Name: "L2CM", Hex: strings.Repeat("0", 62) + "10"},
	}
	goConsts := []GoDevFeatureConst{
		{Name: "L2CMFlag", Hex: strings.Repeat("0", 61) + "100"},
	}
	errs := validateGoParity(Registry{}, solConsts, goConsts, nil, nil)
	joined := joinErrs(errs)
	require.Contains(t, joined, "devfeatures.go missing constant matching DevFeatures.L2CM")
	require.Contains(t, joined, "hex 0x"+strings.Repeat("0", 62)+"10")
	require.Contains(t, joined, "devfeatures.go has extra constant L2CMFlag")
	require.Contains(t, joined, "hex 0x"+strings.Repeat("0", 61)+"100")
}

func TestValidateGoParity_MissingInGo(t *testing.T) {
	solConsts := []DevFeatureConst{
		{Name: "L2CM", Hex: strings.Repeat("0", 62) + "10"},
	}
	errs := validateGoParity(Registry{}, solConsts, nil, nil, nil)
	require.Contains(t, joinErrs(errs), "devfeatures.go missing constant matching DevFeatures.L2CM (hex 0x"+strings.Repeat("0", 62)+"10)")
}

func TestValidateGoParity_ExtraInGo(t *testing.T) {
	goConsts := []GoDevFeatureConst{
		{Name: "L2CMFlag", Hex: strings.Repeat("0", 62) + "10"},
	}
	errs := validateGoParity(Registry{}, nil, goConsts, nil, nil)
	require.Contains(t, joinErrs(errs), "devfeatures.go has extra constant L2CMFlag (hex 0x"+strings.Repeat("0", 62)+"10) not in Solidity")
}

func TestValidateGoParity_HardcodedOnMustBeHardcodedOnBothSides(t *testing.T) {
	r := Registry{
		Features: []Feature{
			{Name: "L2CM", Type: "dev", Lifecycle: "hardcoded-on"},
		},
	}
	hex := strings.Repeat("0", 62) + "10"
	solConsts := []DevFeatureConst{{Name: "L2CM", Hex: hex}}
	goConsts := []GoDevFeatureConst{{Name: "L2CMFlag", Hex: hex}}
	errs := validateGoParity(r, solConsts, goConsts, nil, nil)
	joined := joinErrs(errs)
	require.Contains(t, joined, "DevFeatures.sol isDevFeatureEnabled missing hardcoded true branch for feature-flags.yaml hardcoded-on DevFeatures.L2CM")
	require.Contains(t, joined, "devfeatures.go IsDevFeatureEnabled missing hardcoded true branch for feature-flags.yaml hardcoded-on DevFeatures.L2CM")
}

func TestValidateGoParity_HardcodedOnMissingInGo(t *testing.T) {
	r := Registry{
		Features: []Feature{
			{Name: "L2CM", Type: "dev", Lifecycle: "hardcoded-on"},
		},
	}
	hex := strings.Repeat("0", 62) + "10"
	solConsts := []DevFeatureConst{{Name: "L2CM", Hex: hex}}
	goConsts := []GoDevFeatureConst{{Name: "L2CMFlag", Hex: hex}}
	errs := validateGoParity(r, solConsts, goConsts, []string{"L2CM"}, nil)
	require.Contains(t, joinErrs(errs), "devfeatures.go IsDevFeatureEnabled missing hardcoded true branch for feature-flags.yaml hardcoded-on DevFeatures.L2CM")
}

func TestValidateGoParity_HardcodedOnMissingInSolidity(t *testing.T) {
	r := Registry{
		Features: []Feature{
			{Name: "L2CM", Type: "dev", Lifecycle: "hardcoded-on"},
		},
	}
	hex := strings.Repeat("0", 62) + "10"
	solConsts := []DevFeatureConst{{Name: "L2CM", Hex: hex}}
	goConsts := []GoDevFeatureConst{{Name: "L2CMFlag", Hex: hex}}
	errs := validateGoParity(r, solConsts, goConsts, nil, []string{"L2CMFlag"})
	require.Contains(t, joinErrs(errs), "DevFeatures.sol isDevFeatureEnabled missing hardcoded true branch for feature-flags.yaml hardcoded-on DevFeatures.L2CM")
}

func TestValidateGoParity_SpuriousHardcodedFeature(t *testing.T) {
	r := Registry{
		Features: []Feature{
			{Name: "OPTIMISM_PORTAL_INTEROP", Type: "dev", Lifecycle: "active"},
		},
	}
	solConsts := []DevFeatureConst{
		{Name: "OPTIMISM_PORTAL_INTEROP", Hex: strings.Repeat("0", 63) + "1"},
	}
	goConsts := []GoDevFeatureConst{
		{Name: "OptimismPortalInteropFlag", Hex: strings.Repeat("0", 63) + "1"},
	}
	errs := validateGoParity(
		r,
		solConsts,
		goConsts,
		[]string{"OPTIMISM_PORTAL_INTEROP"},
		[]string{"OptimismPortalInteropFlag"},
	)
	joined := joinErrs(errs)
	require.Contains(t, joined, "DevFeatures.sol isDevFeatureEnabled hardcodes DevFeatures.OPTIMISM_PORTAL_INTEROP")
	require.Contains(t, joined, `feature-flags.yaml lifecycle is "active" (expected hardcoded-on)`)
	require.Contains(t, joined, "devfeatures.go IsDevFeatureEnabled hardcodes devfeatures.OptimismPortalInteropFlag")
}

func wiredFixture() (Registry, []ConfigReader, FeatureFlagsSol, SysFeatureSetup) {
	r := Registry{
		Features: []Feature{
			{Name: "OPTIMISM_PORTAL_INTEROP", Type: "dev", Lifecycle: "active"},
			{Name: "DEPLOY_V2_DISPUTE_GAMES", Type: "dev", Lifecycle: "legacy"},
			{Name: "ETH_LOCKBOX", Type: "sys", Lifecycle: "active"},
		},
	}
	readers := []ConfigReader{
		{FuncName: "devFeatureInterop", EnvVar: "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP"},
		{FuncName: "sysFeatureEthLockbox", EnvVar: "SYS_FEATURE__ETH_LOCKBOX"},
	}
	ff := FeatureFlagsSol{
		ResolveDev: map[string]string{"devFeatureInterop": "OPTIMISM_PORTAL_INTEROP"},
		NameMap: map[string]string{
			"DevFeatures.OPTIMISM_PORTAL_INTEROP": "DEV_FEATURE__OPTIMISM_PORTAL_INTEROP",
			"Features.ETH_LOCKBOX":                "SYS_FEATURE__ETH_LOCKBOX",
		},
	}
	setup := SysFeatureSetup{
		Readers: map[string]bool{"sysFeatureEthLockbox": true},
		Activations: map[string]map[string]bool{
			"sysFeatureEthLockbox": {"ETH_LOCKBOX": true},
		},
	}
	return r, readers, ff, setup
}

func TestValidateWiring_Happy(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	require.Empty(t, validateWiring(r, readers, ff, setup))
}

func TestValidateWiring_LegacyNotRequired(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	// Legacy features do not require readers, name mappings, or setFeature callsites.
	require.Empty(t, validateWiring(r, readers, ff, setup))
}

func TestValidateWiring_MissingConfigReader(t *testing.T) {
	r, _, ff, setup := wiredFixture()
	errs := validateWiring(r, nil, ff, setup)
	require.Contains(t, joinErrs(errs), `expected vm.envOr("DEV_FEATURE__OPTIMISM_PORTAL_INTEROP", false)`)
	require.Contains(t, joinErrs(errs), `expected vm.envOr("SYS_FEATURE__ETH_LOCKBOX", false)`)
}

func TestValidateWiring_MissingResolveBranch(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	ff.ResolveDev = map[string]string{}
	errs := validateWiring(r, readers, ff, setup)
	require.Contains(t, joinErrs(errs), "resolveFeaturesFromEnv missing branch")
}

func TestValidateWiring_MissingGetFeatureName(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	delete(ff.NameMap, "Features.ETH_LOCKBOX")
	errs := validateWiring(r, readers, ff, setup)
	require.Contains(t, joinErrs(errs), "getFeatureName missing branch for Features.ETH_LOCKBOX")
}

func TestValidateWiring_SysReaderNotConsumed(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	setup.Readers = map[string]bool{}
	errs := validateWiring(r, readers, ff, setup)
	require.Contains(t, joinErrs(errs), "is not consumed by any setup path")
}

func TestValidateWiring_SysReaderDoesNotActivateFeatureInSameBranch(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	setup.Activations = map[string]map[string]bool{
		"someOtherReader": {"ETH_LOCKBOX": true},
	}
	errs := validateWiring(r, readers, ff, setup)
	require.Contains(t, joinErrs(errs), "does not activate Features.ETH_LOCKBOX in the same setup branch")
}

func TestValidateWiring_ExtraReaderForUnknownFeature(t *testing.T) {
	r, readers, ff, setup := wiredFixture()
	readers = append(readers, ConfigReader{FuncName: "devFeatureGhost", EnvVar: "DEV_FEATURE__GHOST"})
	errs := validateWiring(r, readers, ff, setup)
	require.Contains(t, joinErrs(errs), `feature-flags.yaml does not list GHOST`)
}

func TestParseCircleCIConfig(t *testing.T) {
	// Anchor declared on one job, aliased by another; commented-out rows must not appear.
	src := `version: 2.1

commands:
  setup-features:
    parameters:
      features:
        default: ""
      system_features:
        default: "ALPHA BETA"

workflows:
  flow:
    jobs:
      - job1:
          matrix:
            parameters:
              features: &features_matrix
                - main
                - ALPHA
                - BETA
                - ALPHA,BETA
                # - GAMMA
                # - ALPHA,GAMMA
      - job2:
          matrix:
            parameters:
              features: *features_matrix
`
	got, err := parseCircleCIConfig([]byte(src))
	require.NoError(t, err)
	require.Equal(t, []string{"ALPHA", "BETA"}, got.SystemFeaturesDefault)
	require.Equal(t, []string{"main", "ALPHA", "BETA", "ALPHA,BETA"}, got.FeaturesMatrix)
}

func TestParseCircleCIConfig_MissingAnchorFails(t *testing.T) {
	src := `version: 2.1
commands:
  setup-features:
    parameters:
      system_features:
        default: "ALPHA"
`
	_, err := parseCircleCIConfig([]byte(src))
	require.ErrorContains(t, err, "could not find &features_matrix anchor")
}

func TestParseCircleCIConfig_MissingSystemFeaturesDefaultFails(t *testing.T) {
	src := `version: 2.1
workflows:
  flow:
    jobs:
      - job:
          matrix:
            parameters:
              features: &features_matrix
                - main
`
	_, err := parseCircleCIConfig([]byte(src))
	require.ErrorContains(t, err, "commands.setup-features.parameters.system_features.default missing")
}

func TestParseCIMatrixRow(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"main", nil},
		{"MAIN", nil},
		{"ALPHA", []string{"ALPHA"}},
		{"ALPHA,BETA", []string{"ALPHA", "BETA"}},
		{" ALPHA , BETA ", []string{"ALPHA", "BETA"}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, parseCIMatrixRow(c.in), "input=%q", c.in)
	}
}

func TestRenderCIMatrix_BaselineAndComma(t *testing.T) {
	r := Registry{
		Features: []Feature{
			{Name: "ALPHA"},
			{Name: "BETA"},
		},
		Combinations: Combinations{
			Matrix: [][]string{{}, {"ALPHA"}, {"ALPHA", "BETA"}},
		},
	}
	require.Equal(t, []string{"main", "ALPHA", "ALPHA,BETA"}, renderCIMatrix(r))
}

func TestRenderCIMatrix_CanonicalizesToDeclarationOrder(t *testing.T) {
	// validateRegistry would reject this row, but renderCIMatrix must still
	// produce a canonical string so CI parity errors are diff-stable.
	r := Registry{
		Features: []Feature{
			{Name: "OPTIMISM_PORTAL_INTEROP"},
			{Name: "ETH_LOCKBOX"},
			{Name: "INTEROP"},
		},
		Combinations: Combinations{
			Matrix: [][]string{{"INTEROP", "ETH_LOCKBOX", "OPTIMISM_PORTAL_INTEROP"}},
		},
	}
	require.Equal(t, []string{"OPTIMISM_PORTAL_INTEROP,ETH_LOCKBOX,INTEROP"}, renderCIMatrix(r))
}

func TestRenderCIMatrix_RoundTripsThroughParseCIRow(t *testing.T) {
	r := goodRegistry()
	rendered := renderCIMatrix(r)
	require.Equal(t, len(r.Combinations.Matrix), len(rendered))
	for i, raw := range rendered {
		parsed := parseCIMatrixRow(raw)
		// Compare to the registry row, canonicalized to declaration order (which it already is).
		expected := r.Combinations.Matrix[i]
		if len(expected) == 0 {
			require.Empty(t, parsed, "row %d", i)
		} else {
			require.Equal(t, expected, parsed, "row %d", i)
		}
	}
}

func ciFixture() (Registry, CircleCIConfig) {
	r := Registry{
		Features: []Feature{
			{Name: "OPTIMISM_PORTAL_INTEROP", Type: "dev", Lifecycle: "active"},
			{Name: "L2CM", Type: "dev", Lifecycle: "hardcoded-on"},
			{Name: "ETH_LOCKBOX", Type: "sys", Lifecycle: "active"},
			{Name: "CUSTOM_GAS_TOKEN", Type: "sys", Lifecycle: "active"},
			{Name: "INTEROP", Type: "sys", Lifecycle: "active"},
		},
		Combinations: Combinations{
			Matrix: [][]string{
				{},
				{"CUSTOM_GAS_TOKEN"},
				{"ETH_LOCKBOX"},
				{"OPTIMISM_PORTAL_INTEROP", "ETH_LOCKBOX", "INTEROP"},
			},
			Requires: map[string][]string{
				"CUSTOM_GAS_TOKEN": {"L2CM"},
				"INTEROP":          {"ETH_LOCKBOX", "OPTIMISM_PORTAL_INTEROP"},
			},
			Excludes: [][]string{{"CUSTOM_GAS_TOKEN", "INTEROP"}},
		},
	}
	ci := CircleCIConfig{
		SystemFeaturesDefault: []string{"CUSTOM_GAS_TOKEN", "ETH_LOCKBOX", "INTEROP"},
		FeaturesMatrix: []string{
			"main",
			"CUSTOM_GAS_TOKEN",
			"ETH_LOCKBOX",
			"OPTIMISM_PORTAL_INTEROP,ETH_LOCKBOX,INTEROP",
		},
	}
	return r, ci
}

func TestValidateCIParity_Happy(t *testing.T) {
	r, ci := ciFixture()
	require.Empty(t, validateCIParity(r, ci))
}

func TestValidateCIParity_SystemFeaturesDefaultMismatch(t *testing.T) {
	r, ci := ciFixture()
	ci.SystemFeaturesDefault = []string{"CUSTOM_GAS_TOKEN", "ETH_LOCKBOX"} // missing INTEROP
	errs := validateCIParity(r, ci)
	require.Contains(t, joinErrs(errs), "setup-features.parameters.system_features.default")
	require.Contains(t, joinErrs(errs), `"CUSTOM_GAS_TOKEN ETH_LOCKBOX INTEROP"`)
}

func TestValidateCIParity_MatrixMissingRow(t *testing.T) {
	r, ci := ciFixture()
	ci.FeaturesMatrix = ci.FeaturesMatrix[:3] // drop the interop combo row
	errs := validateCIParity(r, ci)
	require.Contains(t, joinErrs(errs), "features_matrix does not match")
}

func TestValidateCIParity_MatrixRowOutOfDeclarationOrder(t *testing.T) {
	r, ci := ciFixture()
	ci.FeaturesMatrix[3] = "INTEROP,ETH_LOCKBOX,OPTIMISM_PORTAL_INTEROP"
	errs := validateCIParity(r, ci)
	require.Contains(t, joinErrs(errs), "features_matrix does not match")
}

func TestValidateCIParity_RequiresViolationOnCIRow(t *testing.T) {
	r, ci := ciFixture()
	// Add an INTEROP row alone; this satisfies neither prereq.
	ci.FeaturesMatrix = append(ci.FeaturesMatrix, "INTEROP")
	errs := validateCIParity(r, ci)
	joined := joinErrs(errs)
	require.Contains(t, joined, `features_matrix[4] "INTEROP" enables INTEROP but is missing required prerequisite ETH_LOCKBOX`)
	require.Contains(t, joined, `features_matrix[4] "INTEROP" enables INTEROP but is missing required prerequisite OPTIMISM_PORTAL_INTEROP`)
}

func TestValidateCIParity_ExcludesViolationOnCIRow(t *testing.T) {
	r, ci := ciFixture()
	// Add a row that includes both members of the excluded set.
	ci.FeaturesMatrix = append(ci.FeaturesMatrix, "CUSTOM_GAS_TOKEN,INTEROP")
	errs := validateCIParity(r, ci)
	require.Contains(t, joinErrs(errs), `violates excludes[0]`)
}

func TestValidateCIParity_UnknownFeatureOnCIRow(t *testing.T) {
	r, ci := ciFixture()
	ci.FeaturesMatrix = append(ci.FeaturesMatrix, "GHOST")
	errs := validateCIParity(r, ci)
	require.Contains(t, joinErrs(errs), `features_matrix[4] "GHOST" references unknown feature "GHOST"`)
}

func TestValidateCIParity_HardcodedOnPrereqDoesNotNeedRow(t *testing.T) {
	// CUSTOM_GAS_TOKEN requires L2CM (hardcoded-on); the CI row does not need to name L2CM.
	r, ci := ciFixture()
	require.Empty(t, validateCIParity(r, ci))
}

func joinErrs(errs []string) string {
	return strings.Join(errs, "\n")
}

func TestParseChecksYaml_ExtractsFeatureFlagsCheck(t *testing.T) {
	src := `
phases:
  - name: setup
    checks:
      - name: lint-fix
        command: forge fmt
  - name: pre-build
    checks:
      - name: feature-flags
        description: Check feature flag registry
        command: go run ./scripts/checks/feature-flags
`
	got, err := parseChecksYaml([]byte(src))
	require.NoError(t, err)
	require.True(t, got.FeatureFlagsFound)
	require.Equal(t, 1, got.FeatureFlagsCount)
	require.Equal(t, "pre-build", got.FeatureFlagsPhase)
	require.Equal(t, "go run ./scripts/checks/feature-flags", got.FeatureFlagsCommand)
}

func TestParseChecksYaml_MissingFeatureFlagsCheck(t *testing.T) {
	src := `
phases:
  - name: setup
    checks:
      - name: lint-fix
        command: forge fmt
`
	got, err := parseChecksYaml([]byte(src))
	require.NoError(t, err)
	require.False(t, got.FeatureFlagsFound)
	require.Equal(t, 0, got.FeatureFlagsCount)
	require.Empty(t, got.FeatureFlagsCommand)
	require.Empty(t, got.FeatureFlagsPhase)
}

func TestParseChecksYaml_AlternatePhase(t *testing.T) {
	src := `
phases:
  - name: setup
    checks:
      - name: feature-flags
        command: go run ./scripts/checks/feature-flags
`
	got, err := parseChecksYaml([]byte(src))
	require.NoError(t, err)
	require.True(t, got.FeatureFlagsFound)
	require.Equal(t, 1, got.FeatureFlagsCount)
	require.Equal(t, "setup", got.FeatureFlagsPhase)
}

func TestParseChecksYaml_PhaseNameIsTrimmed(t *testing.T) {
	src := `
phases:
  - name: "  pre-build  "
    checks:
      - name: feature-flags
        command: go run ./scripts/checks/feature-flags
`
	got, err := parseChecksYaml([]byte(src))
	require.NoError(t, err)
	require.True(t, got.FeatureFlagsFound)
	require.Equal(t, "pre-build", got.FeatureFlagsPhase)
}

func TestParseChecksYaml_CountsDuplicateFeatureFlagsChecks(t *testing.T) {
	src := `
phases:
  - name: setup
    checks:
      - name: feature-flags
        command: go run ./scripts/checks/feature-flags
  - name: pre-build
    checks:
      - name: feature-flags
        command: go run ./scripts/checks/feature-flags
`
	got, err := parseChecksYaml([]byte(src))
	require.NoError(t, err)
	require.True(t, got.FeatureFlagsFound)
	require.Equal(t, 2, got.FeatureFlagsCount)
	require.Equal(t, "setup", got.FeatureFlagsPhase)
}

// happyCircleCIDoc is a minimal but complete CircleCI YAML fixture that
// satisfies every check in validateCIControlPlane.
const happyCircleCIDoc = `version: 2.1

commands:
  setup-features:
    parameters:
      features:
        type: string
        default: ""
      system_features:
        type: string
        default: "ALPHA BETA"

jobs:
  contracts-bedrock-tests:
    parameters:
      features:
        type: string
        default: ""
    steps:
      - setup-features:
          features: <<parameters.features>>

  contracts-bedrock-coverage:
    parameters:
      features:
        type: string
        default: ""
    steps:
      - setup-features:
          features: <<parameters.features>>

  contracts-bedrock-tests-upgrade:
    parameters:
      features:
        type: string
        default: ""
    steps:
      - setup-features:
          features: <<parameters.features>>

  contracts-bedrock-tests-l2-fork:
    parameters:
      features:
        type: string
        default: "L2CM"
    steps:
      - setup-features:
          features: <<parameters.features>>

  contracts-bedrock-checks-fast:
    steps:
      - run:
          name: Print forge version
          command: forge --version
      - run:
          name: Run checks
          command: just check-fast
          working_directory: packages/contracts-bedrock

workflows:
  contracts-feature-tests:
    jobs:
      - contracts-bedrock-tests:
          name: contracts-bedrock-tests-heavy-fuzz-modified <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: &features_matrix
                - main
                - ALPHA
      - contracts-bedrock-tests:
          name: contracts-bedrock-tests <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: *features_matrix
      - contracts-bedrock-tests:
          name: contracts-bedrock-tests-develop <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: *features_matrix
      - contracts-bedrock-coverage:
          name: contracts-bedrock-coverage <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: *features_matrix
      - contracts-bedrock-tests-upgrade:
          name: contracts-bedrock-tests-upgrade op-mainnet <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: *features_matrix
      - contracts-bedrock-tests-upgrade:
          name: contracts-bedrock-tests-upgrade-develop op-mainnet <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features: *features_matrix
      - contracts-bedrock-checks-fast:
          name: contracts-bedrock-checks-fast-feature-tests
      - required-contracts-ci:
          requires:
            - contracts-bedrock-checks-fast-feature-tests: terminal
`

func parseHappyCIControlPlane(t *testing.T) CIControlPlane {
	t.Helper()
	doc, err := parseCircleCIDoc([]byte(happyCircleCIDoc))
	require.NoError(t, err)
	return parseCIControlPlane(doc)
}

func TestParseCIControlPlane_HappyPath(t *testing.T) {
	cp := parseHappyCIControlPlane(t)

	require.Equal(t, "just check-fast", cp.ChecksFastCommand)
	require.Equal(t, "packages/contracts-bedrock", cp.ChecksFastWorkingDir)
	require.True(t, cp.HasCheckFastFeatureTests)
	require.True(t, cp.RequiredCIReqsCheckFast)
	require.True(t, cp.SetupFeaturesCommandExists)
	require.Equal(t, 1, cp.SetupFeaturesCommandCount)
	require.True(t, cp.SetupFeaturesHasFeaturesParam)
	require.True(t, cp.SetupFeaturesHasSystemFeaturesParam)
	require.Equal(t, 1, cp.FeaturesMatrixAnchorCount)

	require.Len(t, cp.SetupFeaturesCallers, 4)
	for _, c := range cp.SetupFeaturesCallers {
		require.True(t, setupFeaturesAllowedJobs[c.Job], "caller in unexpected job %q", c.Job)
		require.Equal(t, "<<parameters.features>>", c.Features)
		require.Empty(t, c.SystemFeatures)
		require.False(t, c.SystemFeaturesOverride)
		require.Greater(t, c.Line, 0, "caller in %s missing line number", c.Job)
	}

	require.Len(t, cp.MatrixConsumers, 6)
	gotConsumers := map[string]MatrixConsumer{}
	for _, mc := range cp.MatrixConsumers {
		gotConsumers[mc.InstanceName] = mc
		require.True(t, mc.UsesFeaturesAnchor, "consumer %q does not source from *features_matrix", mc.InstanceName)
		require.Equal(t, "contracts-feature-tests", mc.Workflow)
		require.Greater(t, mc.Line, 0, "consumer %q missing line number", mc.InstanceName)
	}
	for _, name := range contractsFeatureTestsMatrixConsumers {
		_, ok := gotConsumers[name]
		require.True(t, ok, "missing expected consumer %q", name)
	}

	require.Equal(t, "", cp.FeatureDefaults["contracts-bedrock-tests"])
	require.Equal(t, "", cp.FeatureDefaults["contracts-bedrock-coverage"])
	require.Equal(t, "", cp.FeatureDefaults["contracts-bedrock-tests-upgrade"])
	_, l2ForkRead := cp.FeatureDefaults["contracts-bedrock-tests-l2-fork"]
	require.False(t, l2ForkRead, "tests-l2-fork must be excluded from FeatureDefaults")
}

// happyChecksConfig returns a ChecksConfig that satisfies every check in
// validateChecksConfigControlPlane.
func happyChecksConfig() ChecksConfig {
	return ChecksConfig{
		FeatureFlagsCommand: featureFlagsCheckCommand,
		FeatureFlagsPhase:   "pre-build",
		FeatureFlagsFound:   true,
		FeatureFlagsCount:   1,
	}
}

func TestValidateChecksConfigControlPlane_Happy(t *testing.T) {
	require.Empty(t, validateChecksConfigControlPlane(happyChecksConfig()))
}

func TestValidateChecksConfigControlPlane_MissingCheck(t *testing.T) {
	cfg := happyChecksConfig()
	cfg.FeatureFlagsFound = false
	cfg.FeatureFlagsCount = 0
	errs := validateChecksConfigControlPlane(cfg)
	require.Contains(t, joinErrs(errs), "checks.yaml control-plane: feature-flags check is missing")
}

func TestValidateChecksConfigControlPlane_CommandMismatch(t *testing.T) {
	cfg := happyChecksConfig()
	cfg.FeatureFlagsCommand = "go run ./scripts/checks/feature-flags -strict"
	errs := validateChecksConfigControlPlane(cfg)
	require.Contains(t, joinErrs(errs), `checks.yaml control-plane: feature-flags check command is "go run ./scripts/checks/feature-flags -strict", expected "go run ./scripts/checks/feature-flags"`)
}

func TestValidateChecksConfigControlPlane_DuplicateCheck(t *testing.T) {
	cfg := happyChecksConfig()
	cfg.FeatureFlagsCount = 2
	errs := validateChecksConfigControlPlane(cfg)
	require.Contains(t, joinErrs(errs), "checks.yaml control-plane: feature-flags check appears 2 times, expected 1")
}

func TestValidateChecksConfigControlPlane_BuildGatedPhase(t *testing.T) {
	cfg := happyChecksConfig()
	cfg.FeatureFlagsPhase = "source"
	errs := validateChecksConfigControlPlane(cfg)
	require.Contains(t, joinErrs(errs), `checks.yaml control-plane: feature-flags check phase is "source", must be one of setup, pre-build`)
}

// happyControlPlane returns a CIControlPlane that satisfies every check in
// validateCIControlPlane. Negative tests mutate one field at a time.
func happyControlPlane() CIControlPlane {
	callers := make([]SetupFeaturesCaller, 0, 4)
	for _, job := range []string{
		jobContractsBedrockTests,
		jobContractsBedrockCoverage,
		jobContractsBedrockTestsUpgrade,
		jobContractsBedrockTestsL2Fork,
	} {
		callers = append(callers, SetupFeaturesCaller{
			Job:      job,
			Features: featuresParameterTemplate,
			Line:     1000,
		})
	}
	consumers := make([]MatrixConsumer, 0, len(contractsFeatureTestsMatrixConsumers))
	for i, name := range contractsFeatureTestsMatrixConsumers {
		consumers = append(consumers, MatrixConsumer{
			Workflow:           workflowContractsFeatureTests,
			JobType:            jobContractsBedrockTests,
			InstanceName:       name,
			UsesFeaturesAnchor: true,
			Line:               3000 + i,
		})
	}
	return CIControlPlane{
		ChecksFastCommand:                   checksFastCommand,
		ChecksFastWorkingDir:                contractsBedrockWorkingDir,
		HasCheckFastFeatureTests:            true,
		RequiredCIReqsCheckFast:             true,
		SetupFeaturesCommandExists:          true,
		SetupFeaturesCommandCount:           1,
		SetupFeaturesHasFeaturesParam:       true,
		SetupFeaturesHasSystemFeaturesParam: true,
		SetupFeaturesCallers:                callers,
		FeaturesMatrixAnchorCount:           1,
		MatrixConsumers:                     consumers,
		FeatureDefaults: map[string]string{
			jobContractsBedrockTests:        "",
			jobContractsBedrockCoverage:     "",
			jobContractsBedrockTestsUpgrade: "",
		},
	}
}

func TestValidateCIControlPlane_Happy(t *testing.T) {
	require.Empty(t, validateCIControlPlane(happyControlPlane()))
}

func TestValidateCIControlPlane_CheckFastCommandMutated(t *testing.T) {
	cp := happyControlPlane()
	cp.ChecksFastCommand = "just check-fast -run lint"
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), `CircleCI control-plane: contracts-bedrock-checks-fast command is "just check-fast -run lint", expected "just check-fast"`)
}

func TestValidateCIControlPlane_CheckFastMissingWorkingDir(t *testing.T) {
	cp := happyControlPlane()
	cp.ChecksFastWorkingDir = ""
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), `CircleCI control-plane: contracts-bedrock-checks-fast working_directory is "", expected "packages/contracts-bedrock"`)
}

func TestValidateCIControlPlane_MissingCheckFastFeatureTestsInvocation(t *testing.T) {
	cp := happyControlPlane()
	cp.HasCheckFastFeatureTests = false
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: workflow contracts-feature-tests does not invoke contracts-bedrock-checks-fast with name contracts-bedrock-checks-fast-feature-tests")
}

func TestValidateCIControlPlane_RequiredCIWithoutTerminal(t *testing.T) {
	cp := happyControlPlane()
	cp.RequiredCIReqsCheckFast = false
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: required-contracts-ci does not terminal-require contracts-bedrock-checks-fast-feature-tests")
}

func TestValidateCIControlPlane_RequiredCIOmitsCheckFast(t *testing.T) {
	// Omitting the requirement and requiring it without "terminal" both
	// collapse to RequiredCIReqsCheckFast == false, so they share one error.
	cp := happyControlPlane()
	cp.RequiredCIReqsCheckFast = false
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: required-contracts-ci does not terminal-require contracts-bedrock-checks-fast-feature-tests")
}

func TestValidateCIControlPlane_DuplicateSetupFeaturesCommand(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCommandCount = 2
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: commands.setup-features appears 2 times, expected 1")
}

func TestValidateCIControlPlane_SetupFeaturesCallerPassesSystemFeatures(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCallers[0].SystemFeatures = "ALPHA"
	cp.SetupFeaturesCallers[0].Line = 1234
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: setup-features caller in job contracts-bedrock-tests at line 1234 passes system_features override")
}

func TestValidateCIControlPlane_SetupFeaturesCallerPassesEmptySystemFeatures(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCallers[0].SystemFeaturesOverride = true
	cp.SetupFeaturesCallers[0].Line = 1234
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: setup-features caller in job contracts-bedrock-tests at line 1234 passes system_features override")
}

func TestValidateCIControlPlane_SetupFeaturesCallerInUnknownJob(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCallers = append(cp.SetupFeaturesCallers, SetupFeaturesCaller{
		Job:      "contracts-bedrock-tests-extra",
		Features: featuresParameterTemplate,
		Line:     1234,
	})
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: setup-features caller in job contracts-bedrock-tests-extra at line 1234 is not allowed")
}

func TestValidateCIControlPlane_SetupFeaturesCallerMissingFromExpectedJob(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCallers = cp.SetupFeaturesCallers[1:] // drop contracts-bedrock-tests
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: job contracts-bedrock-tests does not call setup-features")
}

func TestValidateCIControlPlane_SetupFeaturesCallerPassesLiteralFeatures(t *testing.T) {
	cp := happyControlPlane()
	cp.SetupFeaturesCallers[0].Features = "FOO"
	cp.SetupFeaturesCallers[0].Line = 1234
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), `CircleCI control-plane: setup-features caller in job contracts-bedrock-tests at line 1234 passes features="FOO", expected "<<parameters.features>>"`)
}

func TestValidateCIControlPlane_DuplicateFeaturesMatrixAnchor(t *testing.T) {
	cp := happyControlPlane()
	cp.FeaturesMatrixAnchorCount = 2
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: &features_matrix anchor count is 2, expected 1")
}

func TestValidateCIControlPlane_MatrixConsumerUsesInlineMatrix(t *testing.T) {
	cp := happyControlPlane()
	cp.MatrixConsumers[0].UsesFeaturesAnchor = false
	cp.MatrixConsumers[0].Line = 4242
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs),
		"CircleCI control-plane: matrix consumer "+contractsFeatureTestsMatrixConsumers[0]+
			" at line 4242 does not source matrix.parameters.features from *features_matrix")
}

func TestValidateCIControlPlane_UnexpectedMatrixConsumer(t *testing.T) {
	cp := happyControlPlane()
	cp.MatrixConsumers = append(cp.MatrixConsumers, MatrixConsumer{
		Workflow:           workflowContractsFeatureTests,
		JobType:            jobContractsBedrockTests,
		InstanceName:       "contracts-bedrock-tests-experimental",
		UsesFeaturesAnchor: true,
		Line:               5555,
	})
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: unexpected matrix consumer contracts-bedrock-tests-experimental at line 5555 in workflow contracts-feature-tests")
}

func TestValidateCIControlPlane_MissingExpectedMatrixConsumer(t *testing.T) {
	cp := happyControlPlane()
	cp.MatrixConsumers = cp.MatrixConsumers[1:] // drop contracts-bedrock-tests-heavy-fuzz-modified
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), "CircleCI control-plane: expected matrix consumer contracts-bedrock-tests-heavy-fuzz-modified is missing from workflow contracts-feature-tests")
}

func TestValidateCIControlPlane_NonEmptyFeatureDefault(t *testing.T) {
	cp := happyControlPlane()
	cp.FeatureDefaults[jobContractsBedrockTests] = "OPTIMISM_PORTAL_INTEROP"
	errs := validateCIControlPlane(cp)
	require.Contains(t, joinErrs(errs), `CircleCI control-plane: contracts-bedrock-tests parameters.features.default = "OPTIMISM_PORTAL_INTEROP", expected ""`)
}

// The following parser-driven tests feed YAML fixtures through parseCIControlPlane
// so the walker is exercised end-to-end, not just at the struct boundary.

func parseControlPlane(t *testing.T, src string) CIControlPlane {
	t.Helper()
	doc, err := parseCircleCIDoc([]byte(src))
	require.NoError(t, err)
	return parseCIControlPlane(doc)
}

func TestParseCIControlPlane_CountsDuplicateAnchor(t *testing.T) {
	src := `
workflows:
  contracts-feature-tests:
    jobs:
      - a:
          features: <<matrix.features>>
          matrix:
            parameters:
              features: &features_matrix
                - main
      - b:
          features: <<matrix.features>>
          matrix:
            parameters:
              features: &features_matrix
                - main
`
	cp := parseControlPlane(t, src)
	require.Equal(t, 2, cp.FeaturesMatrixAnchorCount)
}

func TestParseCIControlPlane_CheckFastCommandMutationIsCaptured(t *testing.T) {
	src := `
jobs:
  contracts-bedrock-checks-fast:
    steps:
      - run:
          name: Run checks
          command: just check-fast -run lint
          working_directory: packages/contracts-bedrock
`
	cp := parseControlPlane(t, src)
	require.Equal(t, "just check-fast -run lint", cp.ChecksFastCommand)
	require.Equal(t, "packages/contracts-bedrock", cp.ChecksFastWorkingDir)
}

func TestParseCIControlPlane_CheckFastCommandIsTrimmed(t *testing.T) {
	src := `
jobs:
  contracts-bedrock-checks-fast:
    steps:
      - run:
          name: Run checks
          command: "  just check-fast  "
          working_directory: packages/contracts-bedrock
`
	cp := parseControlPlane(t, src)
	require.Equal(t, "just check-fast", cp.ChecksFastCommand)
	require.Equal(t, "packages/contracts-bedrock", cp.ChecksFastWorkingDir)
}

func TestParseCIControlPlane_CheckFastFallbackPrefersRunChecksStep(t *testing.T) {
	src := `
jobs:
  contracts-bedrock-checks-fast:
    steps:
      - run:
          name: Prepare check-fast cache
          command: tools/check-fast-init
          working_directory: scripts
      - run:
          name: Run checks
          command: just check-fast -run lint
          working_directory: packages/contracts-bedrock
`
	cp := parseControlPlane(t, src)
	require.Equal(t, "just check-fast -run lint", cp.ChecksFastCommand)
	require.Equal(t, "packages/contracts-bedrock", cp.ChecksFastWorkingDir)
}

func TestParseCIControlPlane_SetupFeaturesScalarInvocationIsCaptured(t *testing.T) {
	src := `
jobs:
  contracts-bedrock-tests:
    steps:
      - setup-features
`
	cp := parseControlPlane(t, src)
	require.Len(t, cp.SetupFeaturesCallers, 1)
	require.Equal(t, "contracts-bedrock-tests", cp.SetupFeaturesCallers[0].Job)
	require.Empty(t, cp.SetupFeaturesCallers[0].Features)
	require.Greater(t, cp.SetupFeaturesCallers[0].Line, 0)
}

func TestParseCIControlPlane_SetupFeaturesCallerParamsAreNormalized(t *testing.T) {
	src := `
jobs:
  contracts-bedrock-tests:
    steps:
      - setup-features:
          features: "  <<parameters.features>>  "
          system_features: ""
`
	cp := parseControlPlane(t, src)
	require.Len(t, cp.SetupFeaturesCallers, 1)
	require.Equal(t, "<<parameters.features>>", cp.SetupFeaturesCallers[0].Features)
	require.Empty(t, cp.SetupFeaturesCallers[0].SystemFeatures)
	require.True(t, cp.SetupFeaturesCallers[0].SystemFeaturesOverride)
}

func TestParseCIControlPlane_InlineMatrixDoesNotCountAsAnchor(t *testing.T) {
	src := `
workflows:
  contracts-feature-tests:
    jobs:
      - contracts-bedrock-tests:
          name: contracts-bedrock-tests <<matrix.features>>
          features: <<matrix.features>>
          matrix:
            parameters:
              features:
                - main
                - ALPHA
`
	cp := parseControlPlane(t, src)
	require.Len(t, cp.MatrixConsumers, 1)
	require.False(t, cp.MatrixConsumers[0].UsesFeaturesAnchor)
}

func TestParseCIControlPlane_MatrixConsumerFeaturesIsTrimmed(t *testing.T) {
	src := `
workflows:
  contracts-feature-tests:
    jobs:
      - contracts-bedrock-tests:
          name: contracts-bedrock-tests <<matrix.features>>
          features: "  <<matrix.features>>  "
          matrix:
            parameters:
              features: &features_matrix
                - main
                - ALPHA
`
	cp := parseControlPlane(t, src)
	require.Len(t, cp.MatrixConsumers, 1)
	require.Equal(t, "contracts-bedrock-tests", cp.MatrixConsumers[0].InstanceName)
	require.True(t, cp.MatrixConsumers[0].UsesFeaturesAnchor)
}

// TestRealRepoControlPlane is an on-disk smoke test against the actual
// .circleci/continue/main.yml and packages/contracts-bedrock/checks.yaml.
// It catches drift between what this checker requires and the live tree.
func TestRealRepoControlPlane(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")
	pkgDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	checksPath := filepath.Join(pkgDir, "checks.yaml")
	circleciPath := filepath.Join(pkgDir, "..", "..", ".circleci", "continue", "main.yml")

	checksCfg, err := readChecksYaml(checksPath)
	require.NoError(t, err)
	require.Empty(t, validateChecksConfigControlPlane(checksCfg), "checks.yaml control-plane drift")

	_, cp, err := readCircleCIConfig(circleciPath)
	require.NoError(t, err)
	require.Empty(t, validateCIControlPlane(cp), "CircleCI control-plane drift")
}

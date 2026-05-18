package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDevFeaturesSol_MultiLineConstant(t *testing.T) {
	// OPTIMISM_PORTAL_INTEROP is intentionally split across two lines on develop.
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

func TestExactlyOneNibble(t *testing.T) {
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
		{strings.Repeat("0", 62) + "11", false}, // two nibbles set
		{strings.Repeat("0", 63) + "f", false},  // single nibble, multiple bits
	}
	for _, c := range cases {
		require.Equal(t, c.want, exactlyOneNibble(c.hex), "input=0x%s", c.hex)
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
	require.Contains(t, joinErrs(errs), "must have exactly one nibble set")

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

func joinErrs(errs []string) string {
	return strings.Join(errs, "\n")
}

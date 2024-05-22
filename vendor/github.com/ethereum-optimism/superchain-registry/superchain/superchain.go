package superchain

import (
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

//go:embed configs
var superchainFS embed.FS

//go:embed extra/addresses extra/bytecodes extra/genesis extra/genesis-system-configs
var extraFS embed.FS

//go:embed implementations
var implementationsFS embed.FS

//go:embed configs/**/semver.yaml
var semverFS embed.FS

type BlockID struct {
	Hash   Hash   `yaml:"hash"`
	Number uint64 `yaml:"number"`
}

type ChainGenesis struct {
	L1           BlockID      `yaml:"l1"`
	L2           BlockID      `yaml:"l2"`
	L2Time       uint64       `json:"l2_time" yaml:"l2_time"`
	ExtraData    *HexBytes    `yaml:"extra_data,omitempty"`
	SystemConfig SystemConfig `json:"system_config" yaml:"-"`
}

type SystemConfig struct {
	BatcherAddr       string `json:"batcherAddr"`
	Overhead          string `json:"overhead"`
	Scalar            string `json:"scalar"`
	GasLimit          uint64 `json:"gasLimit"`
	BaseFeeScalar     uint64 `json:"baseFeeScalar"`
	BlobBaseFeeScalar uint64 `json:"blobBaseFeeScalar"`
}

type GenesisData struct {
	L1     GenesisLayer `json:"l1" yaml:"l1"`
	L2     GenesisLayer `json:"l2" yaml:"l2"`
	L2Time int          `json:"l2_time" yaml:"l2_time"`
}

type GenesisLayer struct {
	Hash   string `json:"hash" yaml:"hash"`
	Number int    `json:"number" yaml:"number"`
}

type HardForkConfiguration struct {
	CanyonTime  *uint64 `json:"canyon_time,omitempty" yaml:"canyon_time,omitempty"`
	DeltaTime   *uint64 `json:"delta_time,omitempty" yaml:"delta_time,omitempty"`
	EcotoneTime *uint64 `json:"ecotone_time,omitempty" yaml:"ecotone_time,omitempty"`
	FjordTime   *uint64 `json:"fjord_time,omitempty" yaml:"fjord_time,omitempty"`
}

type SuperchainLevel uint

const (
	Standard SuperchainLevel = 2
	Frontier SuperchainLevel = 1
)

type ChainConfig struct {
	Name         string `yaml:"name"`
	ChainID      uint64 `yaml:"chain_id"`
	PublicRPC    string `yaml:"public_rpc"`
	SequencerRPC string `yaml:"sequencer_rpc"`
	Explorer     string `yaml:"explorer"`

	SuperchainLevel SuperchainLevel `yaml:"superchain_level"`
	// If SuperchainTime is set, hardforks times after SuperchainTime
	// will be inherited from the superchain-wide config.
	SuperchainTime *uint64 `yaml:"superchain_time"`

	BatchInboxAddr Address `yaml:"batch_inbox_addr"`

	Genesis ChainGenesis `yaml:"genesis"`

	// Superchain is a simple string to identify the superchain.
	// This is implied by directory structure, and not encoded in the config file itself.
	Superchain string `yaml:"-"`
	// Chain is a simple string to identify the chain, within its superchain context.
	// This matches the resource filename, it is not encoded in the config file itself.
	Chain string `yaml:"-"`

	// Hardfork Configuration Overrides
	HardForkConfiguration `yaml:",inline"`

	// Optional feature
	Plasma *PlasmaConfig `yaml:"plasma,omitempty"`
}

type PlasmaConfig struct {
	DAChallengeAddress *Address `json:"da_challenge_contract_address" yaml:"-"`
	// DA challenge window value set on the DAC contract. Used in plasma mode
	// to compute when a commitment can no longer be challenged.
	DAChallengeWindow *uint64 `json:"da_challenge_window" yaml:"da_challenge_window"`
	// DA resolve window value set on the DAC contract. Used in plasma mode
	// to compute when a challenge expires and trigger a reorg if needed.
	DAResolveWindow *uint64 `json:"da_resolve_window" yaml:"da_resolve_window"`
}

// SetDefaultHardforkTimestampsToNil sets each hardfork timestamp to nil (to remove the override)
// if the timestamp matches the superchain default
func (c *ChainConfig) SetDefaultHardforkTimestampsToNil(s *SuperchainConfig) {
	cVal := reflect.ValueOf(&c.HardForkConfiguration).Elem()
	sVal := reflect.ValueOf(&s.hardForkDefaults).Elem()

	for i := 0; i < reflect.Indirect(cVal).NumField(); i++ {
		overrideValue := cVal.Field(i)
		defaultValue := sVal.Field(i)
		if reflect.DeepEqual(overrideValue.Interface(), defaultValue.Interface()) {
			overrideValue.Set(reflect.Zero(overrideValue.Type()))
		}
	}
}

// setNilHardforkTimestampsToDefault overwrites each unspecified hardfork activation time override
// with the superchain default, if the default is not nil and is after the SuperchainTime
func (c *ChainConfig) setNilHardforkTimestampsToDefault(s *SuperchainConfig) {
	if c.SuperchainTime == nil {
		return
	}
	cVal := reflect.ValueOf(&c.HardForkConfiguration).Elem()
	sVal := reflect.ValueOf(&s.hardForkDefaults).Elem()

	for i := 0; i < reflect.Indirect(cVal).NumField(); i++ {
		overrideValue := cVal.Field(i)
		defaultValue := sVal.Field(i)
		if overrideValue.IsNil() &&
			!defaultValue.IsNil() &&
			reflect.Indirect(defaultValue).Uint() >= *c.SuperchainTime {
			overrideValue.Set(defaultValue) // use default only if hardfork activated after SuperchainTime
		}
	}

	// This achieves:
	//
	// if c.CanyonTime == nil {
	// 	c.CanyonTime = s.Config.hardForkDefaults.CanyonTime
	// }
	//
	// ...etc for each field in HardForkConfiguration
}

// EnhanceYAML creates a customized yaml string from a RollupConfig. After completion,
// the *yaml.Node pointer can be used with a yaml encoder to write the custom format to file
func (c *ChainConfig) EnhanceYAML(ctx context.Context, node *yaml.Node) error {
	// Check if context is done before processing
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0] // Dive into the document node
	}

	var lastKey string
	for i := 0; i < len(node.Content)-1; i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		// Add blank line AFTER these keys
		if lastKey == "explorer" || lastKey == "superchain_time" || lastKey == "genesis" {
			keyNode.HeadComment = "\n"
		}

		// Add blank line BEFORE these keys
		if keyNode.Value == "genesis" || keyNode.Value == "plasma" {
			keyNode.HeadComment = "\n"
		}

		// Recursive call to check nested fields for "_time" suffix
		if valNode.Kind == yaml.MappingNode {
			if err := c.EnhanceYAML(ctx, valNode); err != nil {
				return err
			}
		}

		if keyNode.Value == "superchain_time" {
			if valNode.Value == "" || valNode.Value == "null" {
				keyNode.LineComment = "Missing hardfork times are NOT yet inherited from superchain.yaml"
			} else if valNode.Value == "0" {
				keyNode.LineComment = "Missing hardfork times are inherited from superchain.yaml"
			} else {
				keyNode.LineComment = "Missing hardfork times after this time are inherited from superchain.yaml"
			}
		}

		// Add human readable timestamp in comment
		if strings.HasSuffix(keyNode.Value, "_time") && valNode.Value != "" && valNode.Value != "null" {
			t, err := strconv.ParseInt(valNode.Value, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to convert yaml string timestamp to int: %w", err)
			}
			timestamp := time.Unix(t, 0).UTC()
			keyNode.LineComment = timestamp.Format("Mon 2 Jan 2006 15:04:05 UTC")
		}

		lastKey = keyNode.Value
	}
	return nil
}

// AddressList represents the set of network specific contracts for a given network.
type AddressList struct {
	AddressManager                    Address `json:"AddressManager"`
	L1CrossDomainMessengerProxy       Address `json:"L1CrossDomainMessengerProxy"`
	L1ERC721BridgeProxy               Address `json:"L1ERC721BridgeProxy"`
	L1StandardBridgeProxy             Address `json:"L1StandardBridgeProxy"`
	L2OutputOracleProxy               Address `json:"L2OutputOracleProxy"`
	OptimismMintableERC20FactoryProxy Address `json:"OptimismMintableERC20FactoryProxy"`
	OptimismPortalProxy               Address `json:"OptimismPortalProxy"`
	SystemConfigProxy                 Address `json:"SystemConfigProxy"`
	ProxyAdmin                        Address `json:"ProxyAdmin"`
}

// AddressFor returns a nonzero address for the supplied contract name, if it has been specified
// (and an error otherwise). Useful for slicing into the struct using a string.
func (a AddressList) AddressFor(contractName string) (Address, error) {
	var address Address
	switch contractName {
	case "AddressManager":
		address = a.AddressManager
	case "ProxyAdmin":
		address = a.ProxyAdmin
	case "L1CrossDomainMessengerProxy":
		address = a.L1CrossDomainMessengerProxy
	case "L1ERC721BridgeProxy":
		address = a.L1ERC721BridgeProxy
	case "L1StandardBridgeProxy":
		address = a.L1StandardBridgeProxy
	case "L2OutputOracleProxy":
		address = a.L2OutputOracleProxy
	case "OptimismMintableERC20FactoryProxy":
		address = a.OptimismMintableERC20FactoryProxy
	case "OptimismPortalProxy":
		address = a.OptimismPortalProxy
	case "SystemConfigProxy":
		address = a.SystemConfigProxy
	default:
		return address, errors.New("no such contract name")
	}
	if address == (Address{}) {
		return address, errors.New("no address or zero address specified")
	}
	return address, nil
}

// ImplementationList represents the set of implementation contracts to be used together
// for a network.
type ImplementationList struct {
	L1CrossDomainMessenger       VersionedContract `json:"L1CrossDomainMessenger"`
	L1ERC721Bridge               VersionedContract `json:"L1ERC721Bridge"`
	L1StandardBridge             VersionedContract `json:"L1StandardBridge"`
	L2OutputOracle               VersionedContract `json:"L2OutputOracle"`
	OptimismMintableERC20Factory VersionedContract `json:"OptimismMintableERC20Factory"`
	OptimismPortal               VersionedContract `json:"OptimismPortal"`
	SystemConfig                 VersionedContract `json:"SystemConfig"`
}

// ContractImplementations represent a set of contract implementations on a given network.
// The key in the map represents the semantic version of the contract and the value is the
// address that the contract is deployed to.
type ContractImplementations struct {
	L1CrossDomainMessenger       AddressSet `yaml:"l1_cross_domain_messenger"`
	L1ERC721Bridge               AddressSet `yaml:"l1_erc721_bridge"`
	L1StandardBridge             AddressSet `yaml:"l1_standard_bridge"`
	L2OutputOracle               AddressSet `yaml:"l2_output_oracle"`
	OptimismMintableERC20Factory AddressSet `yaml:"optimism_mintable_erc20_factory"`
	OptimismPortal               AddressSet `yaml:"optimism_portal"`
	SystemConfig                 AddressSet `yaml:"system_config"`
}

// AddressSet represents a set of addresses for a given
// contract. They are keyed by the semantic version.
type AddressSet map[string]Address

// VersionedContract represents a contract that has a semantic version.
type VersionedContract struct {
	Version string  `json:"version"`
	Address Address `json:"address"`
}

// Get will handle getting semantic versions from the set
// in the case where the semver string is not prefixed with
// a "v" as well as if it does have a "v" prefix.
func (a AddressSet) Get(key string) Address {
	if !strings.HasPrefix(key, "v") {
		key = "v" + key
	}
	if addr, ok := a[strings.TrimPrefix(key, "v")]; ok {
		return addr
	}
	return a[key]
}

// Versions will return the list of semantic versions for a contract.
// It handles the case where the versions are not prefixed with a "v".
func (a AddressSet) Versions() []string {
	keys := maps.Keys(a)
	for i, k := range keys {
		keys[i] = canonicalizeSemver(k)
	}
	semver.Sort(keys)
	return keys
}

// Resolve will return a set of addresses that resolve a given
// semantic version set.
func (c ContractImplementations) Resolve(versions ContractVersions) (ImplementationList, error) {
	var implementations ImplementationList
	var err error
	if implementations.L1CrossDomainMessenger, err = resolve(c.L1CrossDomainMessenger, versions.L1CrossDomainMessenger); err != nil {
		return implementations, fmt.Errorf("L1CrossDomainMessenger: %w", err)
	}
	if implementations.L1ERC721Bridge, err = resolve(c.L1ERC721Bridge, versions.L1ERC721Bridge); err != nil {
		return implementations, fmt.Errorf("L1ERC721Bridge: %w", err)
	}
	if implementations.L1StandardBridge, err = resolve(c.L1StandardBridge, versions.L1StandardBridge); err != nil {
		return implementations, fmt.Errorf("L1StandardBridge: %w", err)
	}
	if implementations.L2OutputOracle, err = resolve(c.L2OutputOracle, versions.L2OutputOracle); err != nil {
		return implementations, fmt.Errorf("L2OutputOracle: %w", err)
	}
	if implementations.OptimismMintableERC20Factory, err = resolve(c.OptimismMintableERC20Factory, versions.OptimismMintableERC20Factory); err != nil {
		return implementations, fmt.Errorf("OptimismMintableERC20Factory: %w", err)
	}
	if implementations.OptimismPortal, err = resolve(c.OptimismPortal, versions.OptimismPortal); err != nil {
		return implementations, fmt.Errorf("OptimismPortal: %w", err)
	}
	if implementations.SystemConfig, err = resolve(c.SystemConfig, versions.SystemConfig); err != nil {
		return implementations, fmt.Errorf("SystemConfig: %w", err)
	}
	return implementations, nil
}

// resolve returns a VersionedContract that matches the passed in semver version
// given a set of addresses.
func resolve(set AddressSet, version string) (VersionedContract, error) {
	version = canonicalizeSemver(version)

	var out VersionedContract
	keys := set.Versions()
	if len(keys) == 0 {
		return out, fmt.Errorf("no implementations found")
	}

	for _, k := range keys {
		res := semver.Compare(k, version)
		if res >= 0 {
			out = VersionedContract{
				Version: k,
				Address: set.Get(k),
			}
			if res == 0 {
				break
			}
		}
	}
	if out == (VersionedContract{}) {
		return out, fmt.Errorf("cannot resolve semver")
	}
	return out, nil
}

// ContractVersions represents the desired semantic version of the contracts
// in the superchain. This currently only supports L1 contracts but could
// represent L2 predeploys in the future.
type ContractVersions struct {
	L1CrossDomainMessenger       string `yaml:"l1_cross_domain_messenger"`
	L1ERC721Bridge               string `yaml:"l1_erc721_bridge"`
	L1StandardBridge             string `yaml:"l1_standard_bridge"`
	L2OutputOracle               string `yaml:"l2_output_oracle"`
	OptimismMintableERC20Factory string `yaml:"optimism_mintable_erc20_factory"`
	OptimismPortal               string `yaml:"optimism_portal"`
	SystemConfig                 string `yaml:"system_config"`
	// Superchain-wide contracts:
	ProtocolVersions string `yaml:"protocol_versions"`
	SuperchainConfig string `yaml:"superchain_config,omitempty"`
}

// VersionFor returns the version for the supplied contract name, if it exits
// (and an error otherwise). Useful for slicing into the struct using a string.
func (c ContractVersions) VersionFor(contractName string) (string, error) {
	var version string
	switch contractName {
	case "L1CrossDomainMessenger":
		version = c.L1CrossDomainMessenger
	case "L1ERC721Bridge":
		version = c.L1ERC721Bridge
	case "L1StandardBridge":
		version = c.L1StandardBridge
	case "L2OutputOracle":
		version = c.L2OutputOracle
	case "OptimismMintableERC20Factory":
		version = c.OptimismMintableERC20Factory
	case "OptimismPortal":
		version = c.OptimismPortal
	case "SystemConfig":
		version = c.SystemConfig
	case "ProtocolVersions":
		version = c.ProtocolVersions
	case "SuperchainConfig":
		version = c.SuperchainConfig
	default:
		return "", errors.New("no such contract name")
	}
	if version == "" {
		return "", errors.New("no version specified")
	}
	return version, nil
}

// Check will sanity check the validity of the semantic version strings
// in the ContractVersions struct. If allowEmptyVersions is true, empty version errors will be ignored.
func (c ContractVersions) Check(allowEmptyVersions bool) error {
	val := reflect.ValueOf(c)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		str, ok := field.Interface().(string)
		if !ok {
			return fmt.Errorf("invalid type for field %s", val.Type().Field(i).Name)
		}
		if str == "" {
			if allowEmptyVersions {
				continue // we allow empty strings and rely on tests to assert (or except) a nonempty version
			}
			return fmt.Errorf("empty version for field %s", val.Type().Field(i).Name)
		}
		str = canonicalizeSemver(str)
		if !semver.IsValid(str) {
			return fmt.Errorf("invalid semver %s for field %s", str, val.Type().Field(i).Name)
		}
	}
	return nil
}

// newContractImplementations returns a new empty ContractImplementations.
// Use this constructor to ensure that none of struct fields are nil.
// It will also merge the local network implementations into the global implementations
// because the global implementations were deployed with create2 and therefore should
// be on every network.
func newContractImplementations(network string) (ContractImplementations, error) {
	var globals ContractImplementations
	globalData, err := implementationsFS.ReadFile(path.Join("implementations", "implementations.yaml"))
	if err != nil {
		return globals, fmt.Errorf("failed to read implementations: %w", err)
	}
	if err := yaml.Unmarshal(globalData, &globals); err != nil {
		return globals, fmt.Errorf("failed to decode implementations: %w", err)
	}
	setAddressSetsIfNil(&globals)
	if network == "" {
		return globals, nil
	}

	filepath := path.Join("implementations", "networks", network+".yaml")
	var impls ContractImplementations
	data, err := implementationsFS.ReadFile(filepath)
	if err != nil {
		return impls, fmt.Errorf("failed to read implementations: %w", err)
	}
	if err := yaml.Unmarshal(data, &impls); err != nil {
		return impls, fmt.Errorf("failed to decode implementations: %w", err)
	}
	setAddressSetsIfNil(&impls)
	globals.Merge(impls)

	return globals, nil
}

// setAddressSetsIfNil will ensure that all of the struct values on a
// ContractImplementations struct are non nil.
func setAddressSetsIfNil(impls *ContractImplementations) {
	if impls.L1CrossDomainMessenger == nil {
		impls.L1CrossDomainMessenger = make(AddressSet)
	}
	if impls.L1ERC721Bridge == nil {
		impls.L1ERC721Bridge = make(AddressSet)
	}
	if impls.L1StandardBridge == nil {
		impls.L1StandardBridge = make(AddressSet)
	}
	if impls.L2OutputOracle == nil {
		impls.L2OutputOracle = make(AddressSet)
	}
	if impls.OptimismMintableERC20Factory == nil {
		impls.OptimismMintableERC20Factory = make(AddressSet)
	}
	if impls.OptimismPortal == nil {
		impls.OptimismPortal = make(AddressSet)
	}
	if impls.SystemConfig == nil {
		impls.SystemConfig = make(AddressSet)
	}
}

// copySemverMap is a concrete implementation of maps.Copy for map[string]Address.
var copySemverMap = maps.Copy[map[string]Address, map[string]Address]

// canonicalizeSemver will ensure that the version string has a "v" prefix.
// This is because the semver library being used requires the "v" prefix,
// even though
func canonicalizeSemver(version string) string {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version
}

// Merge will combine two ContractImplementations into one. Any conflicting keys will
// be overwritten by the arguments. It assumes that nonce of the struct fields are nil.
func (c ContractImplementations) Merge(other ContractImplementations) {
	copySemverMap(c.L1CrossDomainMessenger, other.L1CrossDomainMessenger)
	copySemverMap(c.L1ERC721Bridge, other.L1ERC721Bridge)
	copySemverMap(c.L1StandardBridge, other.L1StandardBridge)
	copySemverMap(c.L2OutputOracle, other.L2OutputOracle)
	copySemverMap(c.OptimismMintableERC20Factory, other.OptimismMintableERC20Factory)
	copySemverMap(c.OptimismPortal, other.OptimismPortal)
	copySemverMap(c.SystemConfig, other.SystemConfig)
}

// Copy will return a shallow copy of the ContractImplementations.
func (c ContractImplementations) Copy() ContractImplementations {
	return ContractImplementations{
		L1CrossDomainMessenger:       maps.Clone(c.L1CrossDomainMessenger),
		L1ERC721Bridge:               maps.Clone(c.L1ERC721Bridge),
		L1StandardBridge:             maps.Clone(c.L1StandardBridge),
		L2OutputOracle:               maps.Clone(c.L2OutputOracle),
		OptimismMintableERC20Factory: maps.Clone(c.OptimismMintableERC20Factory),
		OptimismPortal:               maps.Clone(c.OptimismPortal),
		SystemConfig:                 maps.Clone(c.SystemConfig),
	}
}

type GenesisSystemConfig struct {
	BatcherAddr Address `json:"batcherAddr"`
	Overhead    Hash    `json:"overhead"`
	Scalar      Hash    `json:"scalar"`
	GasLimit    uint64  `json:"gasLimit"`
}

type GenesisAccount struct {
	CodeHash Hash          `json:"codeHash,omitempty"` // code hash only, to reduce overhead of duplicate bytecode
	Storage  map[Hash]Hash `json:"storage,omitempty"`
	Balance  *HexBig       `json:"balance,omitempty"`
	Nonce    uint64        `json:"nonce,omitempty"`
}

type Genesis struct {
	// Block properties
	Nonce         uint64  `json:"nonce"`
	Timestamp     uint64  `json:"timestamp"`
	ExtraData     []byte  `json:"extraData"`
	GasLimit      uint64  `json:"gasLimit"`
	Difficulty    *HexBig `json:"difficulty"`
	Mixhash       Hash    `json:"mixHash"`
	Coinbase      Address `json:"coinbase"`
	Number        uint64  `json:"number"`
	GasUsed       uint64  `json:"gasUsed"`
	ParentHash    Hash    `json:"parentHash"`
	BaseFee       *HexBig `json:"baseFeePerGas"`
	ExcessBlobGas *uint64 `json:"excessBlobGas"` // EIP-4844
	BlobGasUsed   *uint64 `json:"blobGasUsed"`   // EIP-4844
	// State data
	Alloc map[Address]GenesisAccount `json:"alloc"`
	// StateHash substitutes for a full embedded state allocation,
	// for instantiating states with the genesis block only, to be state-synced before operation.
	// Archive nodes should use a full external genesis.json or datadir.
	StateHash *Hash `json:"stateHash,omitempty"`
	// The chain-config is not included. This is derived from the chain and superchain definition instead.
}

type SuperchainL1Info struct {
	ChainID   uint64 `yaml:"chain_id"`
	PublicRPC string `yaml:"public_rpc"`
	Explorer  string `yaml:"explorer"`
}

type SuperchainConfig struct {
	Name string           `yaml:"name"`
	L1   SuperchainL1Info `yaml:"l1"`

	ProtocolVersionsAddr *Address `yaml:"protocol_versions_addr,omitempty"`
	SuperchainConfigAddr *Address `yaml:"superchain_config_addr,omitempty"`

	// Hardfork Configuration. These values may be overridden by individual chains.
	hardForkDefaults HardForkConfiguration
}

// custom unmarshal function to allow yaml to be unmarshalled into unexported fields
func unMarshalSuperchainConfig(data []byte, s *SuperchainConfig) error {
	temp := struct {
		*SuperchainConfig `yaml:",inline"`
		HardForks         *HardForkConfiguration `yaml:",inline"`
	}{
		SuperchainConfig: s,
		HardForks:        &s.hardForkDefaults,
	}

	return yaml.Unmarshal(data, temp)
}

type Superchain struct {
	Config SuperchainConfig

	// Chains that are part of this superchain
	ChainIDs []uint64

	// Superchain identifier, without capitalization or display changes.
	Superchain string
}

// IsEcotone returns true if the EcotoneTime for this chain in the past.
func (c *ChainConfig) IsEcotone() bool {
	if et := c.EcotoneTime; et != nil {
		return int64(*et) < time.Now().Unix()
	}
	return false
}

var Superchains = map[string]*Superchain{}

var OPChains = map[uint64]*ChainConfig{}

var Addresses = map[uint64]*AddressList{}

var GenesisSystemConfigs = map[uint64]*GenesisSystemConfig{}

// Implementations maps superchain name to contract implementations
var Implementations = map[string]ContractImplementations{}

// SuperchainSemver maps superchain name to a contract name : approved semver version structure.
var SuperchainSemver map[string]ContractVersions

func isConfigFile(c fs.DirEntry) bool {
	return (!c.IsDir() &&
		strings.HasSuffix(c.Name(), ".yaml") &&
		c.Name() != "superchain.yaml" &&
		c.Name() != "semver.yaml")
}

// newContractVersions will read the contract versions from semver.yaml
// and check to make sure that it is valid.
func newContractVersions(superchain string) (ContractVersions, error) {
	var versions ContractVersions
	semvers, err := semverFS.ReadFile(path.Join("configs", superchain, "semver.yaml"))
	if err != nil {
		return versions, fmt.Errorf("failed to read semver.yaml: %w", err)
	}
	if err := yaml.Unmarshal(semvers, &versions); err != nil {
		return versions, fmt.Errorf("failed to unmarshal semver.yaml: %w", err)
	}
	return versions, nil
}

func LoadGenesis(chainID uint64) (*Genesis, error) {
	ch, ok := OPChains[chainID]
	if !ok {
		return nil, fmt.Errorf("unknown chain %d", chainID)
	}
	f, err := extraFS.Open(path.Join("extra", "genesis", ch.Superchain, ch.Chain+".json.gz"))
	if err != nil {
		return nil, fmt.Errorf("failed to open chain genesis definition of %d: %w", chainID, err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip reader of genesis data of %d: %w", chainID, err)
	}
	defer r.Close()
	var out Genesis
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode genesis allocation of %d: %w", chainID, err)
	}
	return &out, nil
}

func LoadContractBytecode(codeHash Hash) ([]byte, error) {
	f, err := extraFS.Open(path.Join("extra", "bytecodes", codeHash.String()+".bin.gz"))
	if err != nil {
		return nil, fmt.Errorf("failed to open bytecode %s: %w", codeHash, err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("")
	}
	defer r.Close()
	return io.ReadAll(r)
}

package params

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	gethparams "github.com/ethereum/go-ethereum/params"
)

// chainConfigFields is ChainConfig without its JSON methods, so that chainConfigJSON can
// embed it without recursing into them.
type chainConfigFields ChainConfig

// chainConfigJSON is the JSON form of ChainConfig: the go-ethereum chain config that
// GethChainConfig derives, followed by the ChainConfig fields. Both embedded configs
// declare chainId, which encoding/json would drop as ambiguous, so it is declared
// directly; it leads the encoding, as in go-ethereum's.
type chainConfigJSON struct {
	ChainID *big.Int `json:"chainId"`
	*gethparams.ChainConfig
	chainConfigFields
}

// MarshalJSON encodes c as a go-ethereum chain config: the Ethereum fork schedule that
// GethChainConfig derives from c, followed by c's own fields. Ethereum keys are encoded
// in go-ethereum's declaration order.
func (c ChainConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(chainConfigJSON{
		ChainID:           c.ChainID,
		ChainConfig:       c.GethChainConfig(),
		chainConfigFields: chainConfigFields(c),
	})
}

// UnmarshalJSON decodes a go-ethereum chain config that carries the OP Stack fields.
// Only the OP Stack fields are kept: the Ethereum fork schedule is derived from them
// (see GethChainConfig). An Ethereum fork key that contradicts the derivation is
// rejected, because go-ethereum would build a different chain from the input than from
// the decoded config; absent and unknown keys are accepted.
func (c *ChainConfig) UnmarshalJSON(input []byte) error {
	dec := chainConfigJSON{ChainConfig: new(gethparams.ChainConfig)}
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	out := ChainConfig(dec.chainConfigFields)
	out.ChainID = dec.ChainID
	if err := checkEthereumSchedule(dec.ChainConfig, out.GethChainConfig()); err != nil {
		return err
	}
	*c = out
	return nil
}

// checkEthereumSchedule returns an error naming the first Ethereum fork key that is set
// in dec, a decoded go-ethereum config, to a value other than the one in derived.
func checkEthereumSchedule(dec, derived *gethparams.ChainConfig) error {
	for _, f := range []struct {
		key       string
		got, want *big.Int
	}{
		{"homesteadBlock", dec.HomesteadBlock, derived.HomesteadBlock},
		{"daoForkBlock", dec.DAOForkBlock, derived.DAOForkBlock},
		{"eip150Block", dec.EIP150Block, derived.EIP150Block},
		{"eip155Block", dec.EIP155Block, derived.EIP155Block},
		{"eip158Block", dec.EIP158Block, derived.EIP158Block},
		{"byzantiumBlock", dec.ByzantiumBlock, derived.ByzantiumBlock},
		{"constantinopleBlock", dec.ConstantinopleBlock, derived.ConstantinopleBlock},
		{"petersburgBlock", dec.PetersburgBlock, derived.PetersburgBlock},
		{"istanbulBlock", dec.IstanbulBlock, derived.IstanbulBlock},
		{"muirGlacierBlock", dec.MuirGlacierBlock, derived.MuirGlacierBlock},
		{"berlinBlock", dec.BerlinBlock, derived.BerlinBlock},
		{"londonBlock", dec.LondonBlock, derived.LondonBlock},
		{"arrowGlacierBlock", dec.ArrowGlacierBlock, derived.ArrowGlacierBlock},
		{"grayGlacierBlock", dec.GrayGlacierBlock, derived.GrayGlacierBlock},
		{"mergeNetsplitBlock", dec.MergeNetsplitBlock, derived.MergeNetsplitBlock},
		{"terminalTotalDifficulty", dec.TerminalTotalDifficulty, derived.TerminalTotalDifficulty},
	} {
		if f.got != nil && (f.want == nil || f.got.Cmp(f.want) != 0) {
			return contradiction(f.key, f.got.String(), formatBig(f.want))
		}
	}
	for _, f := range []struct {
		key       string
		got, want *uint64
	}{
		{"shanghaiTime", dec.ShanghaiTime, derived.ShanghaiTime},
		{"cancunTime", dec.CancunTime, derived.CancunTime},
		{"pragueTime", dec.PragueTime, derived.PragueTime},
		{"osakaTime", dec.OsakaTime, derived.OsakaTime},
		{"bpo1Time", dec.BPO1Time, derived.BPO1Time},
		{"bpo2Time", dec.BPO2Time, derived.BPO2Time},
		{"bpo3Time", dec.BPO3Time, derived.BPO3Time},
		{"bpo4Time", dec.BPO4Time, derived.BPO4Time},
		{"bpo5Time", dec.BPO5Time, derived.BPO5Time},
		{"amsterdamTime", dec.AmsterdamTime, derived.AmsterdamTime},
		{"bogotaTime", dec.BogotaTime, derived.BogotaTime},
		{"ubtTime", dec.UBTTime, derived.UBTTime},
	} {
		if f.got != nil && (f.want == nil || *f.got != *f.want) {
			return contradiction(f.key, strconv.FormatUint(*f.got, 10), formatTime(f.want))
		}
	}
	if dec.DAOForkSupport != derived.DAOForkSupport {
		return contradiction("daoForkSupport", strconv.FormatBool(dec.DAOForkSupport), strconv.FormatBool(derived.DAOForkSupport))
	}
	if dec.EnableUBTAtGenesis != derived.EnableUBTAtGenesis {
		return contradiction("enableUBTAtGenesis", strconv.FormatBool(dec.EnableUBTAtGenesis), strconv.FormatBool(derived.EnableUBTAtGenesis))
	}
	if dec.DepositContractAddress != derived.DepositContractAddress {
		return contradiction("depositContractAddress", dec.DepositContractAddress.Hex(), derived.DepositContractAddress.Hex())
	}
	return nil
}

func contradiction(key, got, want string) error {
	return fmt.Errorf("chain config key %q is %s, but the OP Stack fork schedule implies %s", key, got, want)
}

func formatBig(v *big.Int) string {
	if v == nil {
		return "no fork"
	}
	return v.String()
}

func formatTime(v *uint64) string {
	if v == nil {
		return "no fork"
	}
	return strconv.FormatUint(*v, 10)
}

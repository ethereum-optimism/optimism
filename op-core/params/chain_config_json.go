package params

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	gethparams "github.com/ethereum/go-ethereum/params"
)

// chainConfigJSON is the JSON form of ChainConfig: the go-ethereum chain config that
// GethChainConfig derives, with the ChainConfig fields declared directly alongside it.
//
// encoding/json lets a field declared directly dominate a same-named field promoted from
// an embedded struct, so the direct fields take precedence over any OP fields
// go-ethereum's config declares as well. ChainID is declared before the embedded config
// so that it leads the encoding, as in go-ethereum's.
type chainConfigJSON struct {
	ChainID *big.Int `json:"chainId"`

	*gethparams.ChainConfig

	BedrockBlock *big.Int `json:"bedrockBlock,omitempty"`
	RegolithTime *uint64  `json:"regolithTime,omitempty"`
	CanyonTime   *uint64  `json:"canyonTime,omitempty"`
	EcotoneTime  *uint64  `json:"ecotoneTime,omitempty"`
	FjordTime    *uint64  `json:"fjordTime,omitempty"`
	GraniteTime  *uint64  `json:"graniteTime,omitempty"`
	HoloceneTime *uint64  `json:"holoceneTime,omitempty"`
	IsthmusTime  *uint64  `json:"isthmusTime,omitempty"`
	JovianTime   *uint64  `json:"jovianTime,omitempty"`
	KarstTime    *uint64  `json:"karstTime,omitempty"`
	LagoonTime   *uint64  `json:"lagoonTime,omitempty"`

	Optimism *OptimismConfig `json:"optimism,omitempty"`
}

// MarshalJSON encodes c as a go-ethereum chain config: the Ethereum fork schedule that
// GethChainConfig derives from c, followed by c's own fields. Ethereum keys are encoded
// in go-ethereum's declaration order.
func (c ChainConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(chainConfigJSON{
		ChainID:      c.ChainID,
		ChainConfig:  c.GethChainConfig(),
		BedrockBlock: c.BedrockBlock,
		RegolithTime: c.RegolithTime,
		CanyonTime:   c.CanyonTime,
		EcotoneTime:  c.EcotoneTime,
		FjordTime:    c.FjordTime,
		GraniteTime:  c.GraniteTime,
		HoloceneTime: c.HoloceneTime,
		IsthmusTime:  c.IsthmusTime,
		JovianTime:   c.JovianTime,
		KarstTime:    c.KarstTime,
		LagoonTime:   c.LagoonTime,
		Optimism:     c.Optimism,
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
	out := ChainConfig{
		ChainID:      dec.ChainID,
		Optimism:     dec.Optimism,
		BedrockBlock: dec.BedrockBlock,
		RegolithTime: dec.RegolithTime,
		CanyonTime:   dec.CanyonTime,
		EcotoneTime:  dec.EcotoneTime,
		FjordTime:    dec.FjordTime,
		GraniteTime:  dec.GraniteTime,
		HoloceneTime: dec.HoloceneTime,
		IsthmusTime:  dec.IsthmusTime,
		JovianTime:   dec.JovianTime,
		KarstTime:    dec.KarstTime,
		LagoonTime:   dec.LagoonTime,
	}
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
	} {
		if f.got != nil && (f.want == nil || *f.got != *f.want) {
			return contradiction(f.key, strconv.FormatUint(*f.got, 10), formatTime(f.want))
		}
	}
	if dec.DAOForkSupport != derived.DAOForkSupport {
		return contradiction("daoForkSupport", strconv.FormatBool(dec.DAOForkSupport), strconv.FormatBool(derived.DAOForkSupport))
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

package eth

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/params"
)

func GweiToWei(gwei float64) (*big.Int, error) {
	if math.IsNaN(gwei) || math.IsInf(gwei, 0) {
		return nil, fmt.Errorf("invalid gwei value: %v", gwei)
	}

	// convert float GWei value into integer Wei value
	wei, _ := new(big.Float).Mul(
		big.NewFloat(gwei),
		big.NewFloat(params.GWei)).
		Int(nil)

	if wei.Cmp(abi.MaxUint256) == 1 {
		return nil, errors.New("gwei value larger than max uint256")
	}

	return wei, nil
}

var (
	MaxU256Wei           = ETH(uint256.Int{0: ^uint64(0), 1: ^uint64(0), 2: ^uint64(0), 3: ^uint64(0)})
	MaxU128Wei           = ETH(uint256.Int{0: ^uint64(0), 1: ^uint64(0), 2: 0, 3: 0})
	MaxU64Wei            = ETH(uint256.Int{0: ^uint64(0), 1: 0, 2: 0, 3: 0})
	BillionEther         = Ether(1000_000_000)
	MillionEther         = Ether(1000_000)
	ThousandEther        = Ether(1000)
	HundredEther         = Ether(100)
	TenEther             = Ether(10)
	OneEther             = Ether(1)
	HalfEther            = GWei(500_000_000)
	OneTenthEther        = GWei(100_000_000)
	NineHundredthsEther  = GWei(90_000_000)
	EightHundredthsEther = GWei(80_000_000)
	SevenHundredthsEther = GWei(70_000_000)
	SixHundredthsEther   = GWei(60_000_000)
	FiveHundredthsEther  = GWei(50_000_000)
	FourHundredthsEther  = GWei(40_000_000)
	ThreeHundredthsEther = GWei(30_000_000)
	TwoHundredthsEther   = GWei(20_000_000)
	OneHundredthEther    = GWei(10_000_000)
	OneGWei              = GWei(1)
	OneWei               = WeiU64(1)
	ZeroWei              = WeiU64(0)
)

// some internal helper constant values
var (
	weiPerGWei = uint256.NewInt(params.GWei)
	weiPerEth  = uint256.NewInt(params.Ether)
)

// ETH is a typed ETH (test-)currency integer, expressed in number of wei.
// Most methods and usages prefer a flat value presentation, instead of pointer.
// And return the new value, instead of mutating in-place.
// This type is not optimized for speed,
// but is instead designed for readability and to remove mutability foot-guns.
type ETH uint256.Int

// String prints the amount of ETH, with thousands comma-separators, and unit.
// If the amount is perfectly divisible by 1 ether, the amount is printed in ethers.
// If the amount is perfectly divisible by 1 gwei, the amount is printed in gwei.
// If not neatly divisible, the amount is printed in wei.
// This String function is optimized for readability, without precision loss,
// to not hide any precision data during debugging.
func (e ETH) String() string {
	vWei := (*uint256.Int)(&e)
	if vWei.Sign() == 0 {
		return "0 wei"
	}
	var vGWei uint256.Int
	var remainder uint256.Int
	vGWei.DivMod(vWei, weiPerGWei, &remainder)
	if remainder.Sign() == 0 {
		// an exact number of gwei
		var vEth uint256.Int
		vEth.DivMod(vWei, weiPerEth, &remainder)
		if remainder.Sign() == 0 {
			// an exact number of eth
			return vEth.PrettyDec(',') + " ether"
		}
		return vGWei.PrettyDec(',') + " gwei"
	}
	return vWei.PrettyDec(',') + " wei"
}

// Decimal returns the amount, in wei, in decimal form.
func (e ETH) Decimal() string {
	return (*uint256.Int)(&e).String()
}

// Hex returns the amount, in wei, in hexadecimal form with 0x prefix.
func (e ETH) Hex() string {
	return (*uint256.Int)(&e).Hex()
}

// Format implements fmt.Formatter
func (e ETH) Format(s fmt.State, ch rune) {
	(*uint256.Int)(&e).Format(s, ch)
}

// WeiFloat returns the amount as floating point number, in wei (approximate).
// Warning: precision loss. This may not present the exact number of wei.
func (e ETH) WeiFloat() float64 {
	vWei := (*uint256.Int)(&e)
	return vWei.Float64()
}

// EtherString returns the amount, string-ified, forced in ether units (excl. unit suffix)
func (e ETH) EtherString() string {
	var ethers uint256.Int
	var remainder uint256.Int
	ethers.DivMod((*uint256.Int)(&e), OneEther.ToU256(), &remainder)
	if remainder.Sign() == 0 {
		return ethers.Dec()
	}
	// No trailing zeroes
	suffix := strings.TrimRight(fmt.Sprintf("%018d", &remainder), "0")
	return ethers.Dec() + "." + suffix
}

// ToBig converts to *big.Int, in wei.
func (e ETH) ToBig() *big.Int {
	return (*uint256.Int)(&e).ToBig()
}

// Add adds v and returns the result. No value is mutated.
// Add panics if the computation overflows uint256.
func (e ETH) Add(v ETH) (out ETH) {
	var overflow bool
	out, overflow = e.AddOverflow(v)
	if overflow {
		panic(fmt.Errorf("add overflow: %s + %s != %s", e, v, out))
	}
	return
}

// AddOverflow adds v and returns the result. No value is mutated.
// This also returns a boolean indicating if the computation overflowed.
func (e ETH) AddOverflow(v ETH) (out ETH, overflow bool) {
	_, overflow = (*uint256.Int)(&out).AddOverflow((*uint256.Int)(&e), (*uint256.Int)(&v))
	return
}

// Sub subtracts v and returns the result. No value is mutated.
// Sub panics if the computation underflows.
func (e ETH) Sub(v ETH) (out ETH) {
	var underflow bool
	out, underflow = e.SubUnderflow(v)
	if underflow {
		panic(fmt.Errorf("sub underflow: %s - %s != %s", e, v, out))
	}
	return
}

// SubUnderflow subtracts v and returns the result. No value is mutated.
// This also returns a boolean indicating if the computation underflowed.
func (e ETH) SubUnderflow(v ETH) (out ETH, underflow bool) {
	_, underflow = (*uint256.Int)(&out).SubOverflow((*uint256.Int)(&e), (*uint256.Int)(&v))
	return
}

// Mul multiplies by the given uint64 scalar, and returns the result. No value is mutated.
// Mul panics if the given computation overflows uin256.
func (e ETH) Mul(scalar uint64) (out ETH) {
	var overflow bool
	out, overflow = e.MulOverflow(scalar)
	if overflow {
		panic(fmt.Errorf("overflow on ETH mul: %s * %d != %s", e, scalar, out))
	}
	return
}

// MulOverflow multiplies by the given scalar, and returns the result. No value is mutated.
// This also returns a boolean indicating if the result overflowed.
func (e ETH) MulOverflow(scalar uint64) (out ETH, overflow bool) {
	_, overflow = (*uint256.Int)(&out).MulOverflow((*uint256.Int)(&e), uint256.NewInt(scalar))
	return
}

// Div returns the quotient self/denominator.
// Div performs integer division and always rounds down.
// No value is mutated.
// If denominator == 0, this returns 0.
func (e ETH) Div(denominator uint64) (out ETH) {
	(*uint256.Int)(&out).Div((*uint256.Int)(&e), uint256.NewInt(denominator))
	return
}

// Lt returns if this is less than the given ETH value.
func (e ETH) Lt(v ETH) bool {
	return (*uint256.Int)(&e).Lt((*uint256.Int)(&v))
}

// Gt returns if this is greater than the given ETH value.
func (e ETH) Gt(v ETH) bool {
	return (*uint256.Int)(&e).Gt((*uint256.Int)(&v))
}

// IsZero returns if this equals 0.
func (e ETH) IsZero() bool {
	return (*uint256.Int)(&e).IsZero()
}

// ToU256 converts to *uint256.Int, in wei.
// This returns a clone, not the underlying uint256.Int type.
func (e ETH) ToU256() *uint256.Int {
	return (*uint256.Int)(&e).Clone()
}

// Bytes32 converts to [32]byte, as big-endian uint256, in wei.
func (e ETH) Bytes32() [32]byte {
	return (*uint256.Int)(&e).Bytes32()
}

// UnmarshalText supports hexadecimal (0x prefix) and decimal.
func (e *ETH) UnmarshalText(data []byte) error {
	return (*uint256.Int)(e).UnmarshalText(data)
}

// UnmarshalJSON implements json.Unmarshaler. UnmarshalJSON accepts either
// - Quoted string: either hexadecimal OR decimal
// - Not quoted string: only decimal
func (e *ETH) UnmarshalJSON(data []byte) error {
	return (*uint256.Int)(e).UnmarshalJSON(data)
}

// MarshalText marshals as decimal number, without comma separators or unit
func (e ETH) MarshalText() ([]byte, error) {
	return (*uint256.Int)(&e).MarshalText()
}

// WeiBig turns the given big.Int amount of wei into ETH-typed wei.
// This panics if the amount does not fit in 256 bits, or if it is negative.
func WeiBig(wei *big.Int) (out ETH) {
	if wei == nil {
		panic("nil *big.Int input to ETH constructor")
	}
	if wei.Sign() < 0 {
		panic("negative amounts are not supported")
	}
	overflow := (*uint256.Int)(&out).SetFromBig(wei)
	if overflow {
		panic("*big.Int input does not fit in uint256")
	}
	return
}

// WeiU256 turns the given uint256.Int amount of wei into ETH-typed wei.
func WeiU256(wei *uint256.Int) (out ETH) {
	if wei == nil {
		panic("nil *uint256.Int input to ETH constructor")
	}
	return ETH(*wei)
}

// WeiU64 turns the given uint64 amount of wei into ETH-typed wei.
// The upper 192 bits are zeroed.
func WeiU64(wei uint64) (out ETH) {
	(*uint256.Int)(&out).SetUint64(wei)
	return
}

// GWei turns the given amount of GWei into ETH-typed wei.
// I.e. this multiplies the amount by 1e9 to denominate into wei.
func GWei(gwei uint64) (out ETH) {
	var x uint256.Int
	x.SetUint64(gwei)
	x.Mul(&x, weiPerGWei)
	return ETH(x)
}

// Ether turns the given amount of ether into ETH-typed wei.
// I.e. this multiplies the amount by 1e18 to denominate into wei.
func Ether(ether uint64) ETH {
	var x uint256.Int
	x.SetUint64(ether)
	x.Mul(&x, weiPerEth)
	return ETH(x)
}

package eth

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestGweiToWei(t *testing.T) {
	maxUint256p1, _ := new(big.Int).Add(abi.MaxUint256, big.NewInt(1)).Float64()
	for _, tt := range []struct {
		desc string
		gwei float64
		wei  *big.Int
		err  bool
	}{
		{
			desc: "zero",
			gwei: 0,
			wei:  new(big.Int),
		},
		{
			desc: "one-wei",
			gwei: 0.000000001,
			wei:  big.NewInt(1),
		},
		{
			desc: "one-gwei",
			gwei: 1.0,
			wei:  big.NewInt(1e9),
		},
		{
			desc: "one-ether",
			gwei: 1e9,
			wei:  big.NewInt(1e18),
		},
		{
			desc: "err-pos-inf",
			gwei: math.Inf(1),
			err:  true,
		},
		{
			desc: "err-neg-inf",
			gwei: math.Inf(-1),
			err:  true,
		},
		{
			desc: "err-nan",
			gwei: math.NaN(),
			err:  true,
		},
		{
			desc: "err-too-large",
			gwei: maxUint256p1,
			err:  true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			wei, err := GweiToWei(tt.gwei)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wei, wei)
			}
		})
	}
}

func TestEther(t *testing.T) {
	t.Run("constants", func(t *testing.T) {
		require.EqualValues(t, OneGWei, OneWei.Mul(1e9))
		require.EqualValues(t, OneEther, OneGWei.Mul(1e9))
		require.EqualValues(t, ThousandEther, OneEther.Mul(1e3))
		require.EqualValues(t, MillionEther, ThousandEther.Mul(1e3))
		require.EqualValues(t, BillionEther, MillionEther.Mul(1e3))
		require.EqualValues(t, big.NewInt(1), OneWei.ToBig(), "sanity check not mutated value")
	})

	t.Run("constant string", func(t *testing.T) {
		require.Equal(t, "0 wei", ZeroWei.String())
		require.Equal(t, "1 wei", OneWei.String())
		require.Equal(t, "1 gwei", OneGWei.String())
		require.Equal(t, "1 ether", OneEther.String())
		require.Equal(t, "1,000 ether", ThousandEther.String())
		require.Equal(t, "1,000,000 ether", MillionEther.String())
		require.Equal(t, "1,000,000,000 ether", BillionEther.String())
	})

	t.Run("constant float", func(t *testing.T) {
		require.Equal(t, float64(0), ZeroWei.WeiFloat())
		require.Equal(t, float64(1), OneWei.WeiFloat())
		require.Equal(t, float64(1e9), OneGWei.WeiFloat())
	})

	t.Run("ether string force", func(t *testing.T) {
		require.Equal(t, "0.000000000000000001", OneWei.EtherString())
		require.Equal(t, "0.000000001", OneGWei.EtherString())
		require.Equal(t, "1", OneEther.EtherString())
		require.Equal(t, "1000", ThousandEther.EtherString())
		require.Equal(t, "1.000000001", OneEther.Add(OneGWei).EtherString())
		require.Equal(t, "100.000000001", Ether(100).Add(OneGWei).EtherString())
	})

	t.Run("other string", func(t *testing.T) {
		require.Equal(t, "1,234,567 wei", WeiU64(1_234_567).String())
		require.Equal(t, "1,234,567 gwei", GWei(1_234_567).String())
		require.Equal(t, "1,234,567 ether", Ether(1_234_567).String())
		require.Equal(t, "0 wei", WeiU64(0).String())
		require.Equal(t, "0 wei", GWei(0).String())
		require.Equal(t, "0 wei", Ether(0).String())
	})

	t.Run("big strings", func(t *testing.T) {
		require.Equal(t, "1,234,567 wei", WeiU64(1_234_567).String())
		require.Equal(t, "18,446,744,073,709,551,615 wei", MaxU64Wei.String())
		require.Equal(t, "0xffffffffffffffff", MaxU64Wei.Hex())
		require.Equal(t, "340,282,366,920,938,463,463,374,607,431,768,211,455 wei", MaxU128Wei.String())
		require.Equal(t, "0xffffffffffffffffffffffffffffffff", MaxU128Wei.Hex())
		require.Equal(t, "115,792,089,237,316,195,423,570,985,008,687,907,853,269,984,665,640,564,039,457,584,007,913,129,639,935 wei", MaxU256Wei.String())
		require.Equal(t, "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", MaxU256Wei.Hex())
		require.Equal(t, "115792089237316195423570985008687907853269984665640564039457584007913129639935", MaxU256Wei.Decimal())
	})

	t.Run("add", func(t *testing.T) {
		require.Equal(t, Ether(4), OneEther.Add(Ether(3)))
		require.Equal(t, "3,000,000,000,000,000,001 wei", OneWei.Add(Ether(3)).String())

		_, overflowed := MaxU256Wei.AddOverflow(OneWei)
		require.True(t, overflowed)
		require.Panics(t, func() {
			MaxU256Wei.Add(OneWei)
		}, "expect overflow panic")
	})

	t.Run("sub", func(t *testing.T) {
		require.Equal(t, "2,999,999,999,999,999,999 wei", Ether(3).Sub(OneWei).String())

		_, underflowed := ZeroWei.SubUnderflow(OneWei)
		require.True(t, underflowed)
		require.Panics(t, func() {
			ZeroWei.Sub(OneWei)
		}, "expect underflow panic")
	})

	t.Run("mul", func(t *testing.T) {
		require.Equal(t, Ether(3000), ThousandEther.Mul(3))
		require.Equal(t, ZeroWei, ThousandEther.Mul(0))
		require.Equal(t, ZeroWei, ZeroWei.Mul(1))
		require.Equal(t, ZeroWei, OneWei.Mul(0))
		require.Equal(t, OneWei, OneWei.Mul(1))
		_, overflowed := MaxU256Wei.MulOverflow(2)
		require.True(t, overflowed)
		require.Panics(t, func() {
			MaxU256Wei.Mul(2)
		})
		tmp, overflowed := MaxU256Wei.Div(2).MulOverflow(2)
		require.False(t, overflowed)
		require.Equal(t, MaxU256Wei.Sub(OneWei), tmp, "last bit was effectively shifted out and back in as 0")
		var v uint256.Int
		v.Lsh(uint256.NewInt(1), 256-2)
		a := WeiU256(&v)
		a.Mul(2)
		require.Panics(t, func() {
			a.Mul(4)
		})
		_, overflowed = a.MulOverflow(4)
		require.True(t, overflowed)
	})

	t.Run("div", func(t *testing.T) {
		require.Equal(t, ZeroWei, ZeroWei.Div(1))
		require.Equal(t, ZeroWei, ZeroWei.Div(0))
		require.Equal(t, ZeroWei, OneWei.Div(0))
		require.Equal(t, ZeroWei, ThousandEther.Div(0))
		require.Equal(t, OneWei, OneWei.Div(1))
		require.Equal(t, WeiU64(2), WeiU64(4).Div(2))
		require.Equal(t, ThousandEther, MillionEther.Div(1000))
		require.Equal(t, OneGWei, OneEther.Div(1e9))
		require.Equal(t, OneWei, OneGWei.Div(1e9))
		require.Equal(t, WeiU64(16), WeiU64(16*4+3).Div(4), "must round down")
		require.Equal(t, MaxU256Wei, MaxU256Wei.Div(1))
		require.Equal(t, MaxU128Wei, MaxU256Wei.Div(1<<63).Div(1<<63).Div(1<<2))
	})

	t.Run("set", func(t *testing.T) {
		x := OneEther
		require.Equal(t, x, OneEther)
		x = Ether(2)
		require.Equal(t, x, Ether(2))
		require.Equal(t, Ether(1), OneEther, "no mutation of original")
	})

	t.Run("lt", func(t *testing.T) {
		require.True(t, WeiU64(123).Lt(WeiU64(124)))
		require.False(t, WeiU64(124).Lt(WeiU64(124)))
		require.False(t, WeiU64(125).Lt(WeiU64(124)))
	})

	t.Run("gt", func(t *testing.T) {
		require.False(t, WeiU64(123).Gt(WeiU64(124)))
		require.False(t, WeiU64(124).Gt(WeiU64(124)))
		require.True(t, WeiU64(125).Gt(WeiU64(124)))
	})

	t.Run("isZero", func(t *testing.T) {
		require.True(t, ZeroWei.IsZero())
		require.False(t, OneWei.IsZero())
		require.False(t, OneEther.IsZero())
		mostlyZero := [32]byte{0: 1}
		require.False(t, WeiU256(new(uint256.Int).SetBytes(mostlyZero[:])).IsZero())
		require.False(t, MaxU256Wei.IsZero())
	})

	t.Run("convert", func(t *testing.T) {
		require.EqualValues(t, uint256.NewInt(123).String(), WeiU64(123).ToU256().String())
		require.EqualValues(t, big.NewInt(123).String(), WeiU64(123).ToBig().String())
		require.EqualValues(t, [32]byte{31: 123}, WeiU64(123).Bytes32())

		require.Panics(t, func() {
			WeiBig(nil)
		})
		require.Panics(t, func() {
			WeiU256(nil)
		})
		require.Equal(t, MaxU256Wei, WeiBig(MaxU256Wei.ToBig()))
		require.Equal(t, MaxU256Wei, WeiU256(MaxU256Wei.ToU256()))
		v := uint256.Int{0: 0xc0ff_ee12, 1: 0xabcd_1234, 2: 0xef56_7801, 3: 0x1234_5657}
		require.Equal(t, v.String(), WeiU256(&v).ToU256().String())
		require.Equal(t, v.ToBig().String(), WeiU256(&v).ToBig().String())
		// check overflow
		require.Panics(t, func() {
			WeiBig(new(big.Int).Lsh(big.NewInt(1), 256))
		})
		require.Panics(t, func() {
			WeiBig(new(big.Int).Lsh(big.NewInt(123), 300))
		})
		// check negative
		require.Panics(t, func() {
			WeiBig(big.NewInt(-1))
		})
	})

	t.Run("unmarshalText", func(t *testing.T) {
		var x ETH
		require.NoError(t, x.UnmarshalText([]byte("1234")))
		require.Equal(t, "1,234 wei", x.String())
		require.NoError(t, x.UnmarshalText([]byte("0xdeadbeef")))
		require.Equal(t, "0xdeadbeef", x.Hex())
	})

	t.Run("unmarshalJSON", func(t *testing.T) {
		var x ETH
		// basic small unquoted value
		require.NoError(t, x.UnmarshalJSON([]byte("1234")))
		require.Equal(t, "1,234 wei", x.String())
		// hex format is only allowed in strings
		require.ErrorContains(t, x.UnmarshalJSON([]byte("0xdeadbeef")), "invalid")
		// floats are not valid ETH inputs, even if round numbers
		require.ErrorContains(t, x.UnmarshalJSON([]byte("1.0")), "invalid")
		require.ErrorContains(t, x.UnmarshalJSON([]byte("1e18")), "invalid")
		// negative should not work
		require.ErrorContains(t, x.UnmarshalJSON([]byte("-1")), "invalid")
		// Hex is fine when quoted
		require.NoError(t, x.UnmarshalJSON([]byte("\"0xdeadbeef\"")))
		require.Equal(t, WeiU64(0xdeadbeef), x)
		// Decimals, without quotes, should work (cast doesn't add quotes to inputs)
		require.NoError(t, x.UnmarshalJSON([]byte(OneEther.Decimal())))
		require.Equal(t, OneEther, x)
		require.NoError(t, x.UnmarshalJSON([]byte(MaxU256Wei.Decimal())))
		require.Equal(t, MaxU256Wei, x)
		// With quotes should also work
		require.NoError(t, x.UnmarshalJSON([]byte(fmt.Sprintf("%q", OneEther.Decimal()))))
		require.Equal(t, OneEther, x)
		require.NoError(t, x.UnmarshalJSON([]byte(fmt.Sprintf("%q", MaxU256Wei.Decimal()))))
		require.Equal(t, MaxU256Wei, x)
	})

	t.Run("marshal", func(t *testing.T) {
		data, err := OneEther.MarshalText()
		require.NoError(t, err)
		require.Equal(t, "1000000000000000000", string(data))
	})
}

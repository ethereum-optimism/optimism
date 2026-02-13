package script

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

//go:generate go run ./consolegen --abi-txt=console2.txt --out=console2_gen.go

type ConsolePrecompile struct {
	logger log.Logger
	sender func() common.Address
}

func (c *ConsolePrecompile) log(args ...any) {
	sender := c.sender()
	logger := c.logger.With("sender", sender)
	if len(args) == 0 {
		logger.Info("")
		return
	}
	if msg, ok := args[0].(string); ok { // if starting with a string, use it as message. And format with args if needed.
		logger.Info(consoleFormat(msg, args[1:]...))
		return
	} else {
		logger.Info(consoleFormat("", args...))
	}
}

type stringFormat struct{}
type numberFormat struct{}
type objectFormat struct{}
type integerFormat struct{}
type exponentialFormat struct {
	precision int
}
type hexadecimalFormat struct{}

func formatBigInt(x *big.Int, precision int) string {
	if precision < 0 {
		precision = len(new(big.Int).Abs(x).String()) - 1
		return formatBigIntFixedPrecision(x, uint(precision)) + fmt.Sprintf("e%d", precision)
	}
	return formatBigIntFixedPrecision(x, uint(precision))
}

func formatBigIntFixedPrecision(x *big.Int, precision uint) string {
	prec := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	integer, remainder := new(big.Int).QuoRem(x, prec, new(big.Int))
	if remainder.Sign() != 0 {
		decimal := fmt.Sprintf("%0"+fmt.Sprintf("%d", precision)+"d",
			new(big.Int).Abs(remainder))
		decimal = strings.TrimRight(decimal, "0")
		return fmt.Sprintf("%d.%s", integer, decimal)
	} else {
		return fmt.Sprintf("%d", integer)
	}
}

// formatValue formats a value v following the given format-spec.
func formatValue(v any, spec any) string {
	switch x := v.(type) {
	case string:
		switch spec.(type) {
		case stringFormat:
			return x
		case objectFormat:
			return fmt.Sprintf("'%s'", v)
		default:
			return "NaN"
		}
	case bool:
		switch spec.(type) {
		case stringFormat:
			return fmt.Sprintf("%v", x)
		case objectFormat:
			return fmt.Sprintf("'%v'", x)
		case numberFormat:
			if x {
				return "1"
			}
			return "0"
		default:
			return "NaN"
		}
	case *big.Int:
		switch s := spec.(type) {
		case stringFormat, objectFormat, numberFormat, integerFormat:
			return fmt.Sprintf("%d", x)
		case exponentialFormat:
			return formatBigInt(x, s.precision)
		case hexadecimalFormat:
			return (*hexutil.Big)(x).String()
		default:
			return fmt.Sprintf("%d", x)
		}
	case *ABIInt256:
		switch s := spec.(type) {
		case stringFormat, objectFormat, numberFormat, integerFormat:
			return fmt.Sprintf("%d", (*big.Int)(x))
		case exponentialFormat:
			return formatBigInt((*big.Int)(x), s.precision)
		case hexadecimalFormat:
			return (*hexutil.Big)(x).String()
		default:
			return fmt.Sprintf("%d", (*big.Int)(x))
		}
	case common.Address:
		switch spec.(type) {
		case stringFormat, hexadecimalFormat:
			return x.String()
		case objectFormat:
			return fmt.Sprintf("'%s'", x)
		default:
			return "NaN"
		}
	default:
		if typ := reflect.TypeOf(v); (typ.Kind() == reflect.Array || typ.Kind() == reflect.Slice) &&
			typ.Elem().Kind() == reflect.Uint8 {
			switch spec.(type) {
			case stringFormat, hexadecimalFormat:
				return fmt.Sprintf("0x%x", v)
			case objectFormat:
				return fmt.Sprintf("'0x%x'", v)
			default:
				return "NaN"
			}
		}
		return fmt.Sprintf("%v", v)
	}
}

// consoleFormat emulates the foundry-flavor of printf, to format console.log data.
func consoleFormat(fmtMsg string, values ...any) string {
	var sc scanner.Scanner
	sc.Init(bytes.NewReader([]byte(fmtMsg)))
	// default scanner settings are for Go source code parsing. Reset all of that.
	sc.Whitespace = 0
	sc.Mode = 0
	sc.IsIdentRune = func(ch rune, i int) bool {
		return false
	}

	nextValue := func() (v any, ok bool) {
		if len(values) > 0 {
			v = values[0]
			values = values[1:]
			return v, true
		}
		return nil, false
	}

	// Parses a format-spec from a string sequence (excl. the % prefix)
	// Returns the spec (if any), and the consumed characters (to abort with / fall back to)
	formatSpecFromChars := func() (spec any, consumed string) {
		fmtChar := sc.Scan()
		switch fmtChar {
		case 's':
			return stringFormat{}, "s"
		case 'd':
			return numberFormat{}, "d"
		case 'i':
			return integerFormat{}, "i"
		case 'o':
			return objectFormat{}, "o"
		case 'e':
			return exponentialFormat{precision: -1}, "e"
		case 'x':
			return hexadecimalFormat{}, "x"
		case scanner.EOF:
			return nil, ""
		default:
			for ; fmtChar != scanner.EOF; fmtChar = sc.Scan() {
				if fmtChar == 'e' {
					precision, err := strconv.ParseUint(consumed, 10, 16)
					consumed += "e"
					if err != nil {
						return nil, consumed
					}
					return exponentialFormat{precision: int(precision)}, consumed
				}
				consumed += string(fmtChar)
				if !strings.ContainsRune("0123456789", fmtChar) {
					return nil, consumed
				}
			}
			return nil, consumed
		}
	}

	expectFmt := false
	var out strings.Builder
	for sc.Peek() != scanner.EOF {
		if expectFmt {
			expectFmt = false
			spec, consumed := formatSpecFromChars()
			if spec != nil {
				value, ok := nextValue()
				if ok {
					out.WriteString(formatValue(value, spec))
				} else {
					// rather than panic with an .expect() like foundry,
					// just log the original format string
					out.WriteRune('%')
					out.WriteString(consumed)
				}
			} else {
				// on parser failure, write '%' and consumed characters
				out.WriteRune('%')
				out.WriteString(consumed)
			}
		} else {
			tok := sc.Scan()
			if tok == '%' {
				next := sc.Peek()
				switch next {
				case '%': // %% formats as "%"
					out.WriteRune('%')
				case scanner.EOF:
					out.WriteRune(tok)
				default:
					expectFmt = true
				}
			} else {
				out.WriteRune(tok)
			}
		}
	}

	// for all remaining values, append them to the output
	for _, v := range values {
		if out.Len() > 0 {
			out.WriteRune(' ')
		}
		out.WriteString(formatValue(v, stringFormat{}))
	}
	return out.String()
}

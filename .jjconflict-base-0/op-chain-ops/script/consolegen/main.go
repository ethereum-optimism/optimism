package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	var abiTxtPath string
	flag.StringVar(&abiTxtPath, "abi-txt", "console2.txt", "Path of text file with ABI method signatures")
	var outPath string
	flag.StringVar(&outPath, "out", "console2_gen.go", "Path to write output to")
	flag.Parse()
	data, err := os.ReadFile(abiTxtPath)
	if err != nil {
		fmt.Printf("failed to read file: %v\n", err)
		os.Exit(1)
	}
	lines := strings.Split(string(data), "\n")
	var out strings.Builder
	out.WriteString(`// AUTO-GENERATED - DO NOT EDIT
package script

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		byte4ID := crypto.Keccak256([]byte(line))[:4]
		if !strings.HasPrefix(line, "log(") {
			fmt.Printf("unexpected line: %q\n", line)
			os.Exit(1)
		}
		line = strings.TrimPrefix(line, "log(")
		line = strings.TrimSuffix(line, ")")
		params := strings.Split(line, ",")
		out.WriteString("func (c *ConsolePrecompile) Log_")
		out.WriteString(fmt.Sprintf("%x", byte4ID))
		out.WriteString("(")
		for i, p := range params {
			if p == "" {
				continue
			}
			out.WriteString(fmt.Sprintf("p%d ", i))
			name, err := solToGo(p)
			if err != nil {
				fmt.Printf("unexpected param type: %q\n", p)
				os.Exit(1)
			}
			out.WriteString(name)
			if i != len(params)-1 {
				out.WriteString(", ")
			}
		}
		out.WriteString(") {\n")
		out.WriteString("\tc.log(")
		for i, p := range params {
			if p == "" {
				continue
			}
			out.WriteString(prettyArg(fmt.Sprintf("p%d", i), p))
			if i != len(params)-1 {
				out.WriteString(", ")
			}
		}
		out.WriteString(")\n")
		out.WriteString("}\n\n")
	}
	if err := os.WriteFile(outPath, []byte(out.String()), 0644); err != nil {
		fmt.Printf("failed to write output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("done!")
}

func solToGo(name string) (string, error) {
	switch name {
	case "address":
		return "common.Address", nil
	case "uint256":
		return "*big.Int", nil
	case "int256":
		return "*ABIInt256", nil
	case "string", "bool", "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64":
		return name, nil
	case "bytes":
		return "hexutil.Bytes", nil
	default:
		if strings.HasPrefix(name, "bytes") {
			n, err := strconv.Atoi(strings.TrimPrefix(name, "bytes"))
			if err != nil {
				return "", err
			}
			if n > 32 {
				return "", fmt.Errorf("unexpected large bytes slice type: %d", n)
			}
			return fmt.Sprintf("[%d]byte", n), nil
		}
		return "", fmt.Errorf("unrecognized solidity type name: %s", name)
	}
}

func prettyArg(arg string, typ string) string {
	switch typ {
	case "int256":
		return fmt.Sprintf("(*big.Int)(%s)", arg)
	case "bytes":
		return arg
	default:
		if strings.HasPrefix(typ, "bytes") {
			return fmt.Sprintf("hexutil.Bytes(%s[:])", arg)
		}
		return arg
	}
}

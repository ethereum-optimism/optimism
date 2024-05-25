package srcmap

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type LineCol struct {
	Line uint32
	Col  uint32
}

type InstrMapping struct {
	S int32 // start offset in bytes within source (negative when non-existent!)
	L int32 // length in bytes within source (negative when non-existent!)
	F int32 // file index of source (negative when non-existent!)
	J byte  // jump type (i=into, o=out, -=regular)
	M int32 // modifier depth
}

func parseInstrMapping(last InstrMapping, v string) (InstrMapping, error) {
	data := strings.Split(v, ":")
	out := last
	if len(data) < 1 {
		return out, nil
	}
	if len(data) > 5 {
		return out, fmt.Errorf("unexpected length: %d", len(data))
	}
	var err error
	parse := func(x string) int32 {
		p, e := strconv.ParseInt(x, 10, 32)
		err = e
		return int32(p)
	}
	if data[0] != "" {
		out.S = parse(data[0])
	}
	if len(data) < 2 || err != nil {
		return out, err
	}
	if data[1] != "" {
		out.L = parse(data[1])
	}
	if len(data) < 3 || err != nil {
		return out, err
	}
	if data[2] != "" {
		out.F = parse(data[2])
	}
	if len(data) < 4 || err != nil {
		return out, err
	}
	if data[3] != "" {
		out.J = data[3][0]
	}
	if len(data) < 5 || err != nil {
		return out, err
	}
	if data[4] != "" {
		out.M = parse(data[4])
	}
	return out, err
}

type SourceMap struct {
	// source names
	Sources []string
	// per source, source offset -> line/col
	PosData [][]LineCol
	// per bytecode byte, byte index -> instr
	Instr []InstrMapping
}

func (s *SourceMap) Info(pc uint64) (source string, line uint32, col uint32) {
	instr := s.Instr[pc]
	if instr.F < 0 {
		return "generated", 0, 0
	}
	if instr.F >= int32(len(s.Sources)) {
		source = "unknown"
		return
	}
	source = s.Sources[instr.F]
	if instr.S < 0 {
		return
	}
	if s.PosData[instr.F] == nil { // when the source file is known to be unavailable
		return
	}
	if int(instr.S) >= len(s.PosData[instr.F]) { // possibly invalid / truncated source mapping
		return
	}
	lc := s.PosData[instr.F][instr.S]
	line = lc.Line
	col = lc.Col
	return
}

func (s *SourceMap) FormattedInfo(pc uint64) string {
	f, l, c := s.Info(pc)
	return fmt.Sprintf("%s:%d:%d", f, l, c)
}

// ParseSourceMap parses a solidity sourcemap: mapping bytecode indices to source references.
// See https://docs.soliditylang.org/en/latest/internals/source_mappings.html
//
// Sources is the list of source files, which will be read from the filesystem
// to transform token numbers into line/column numbers.
// The sources are as referenced in the source-map by index.
// Not all sources are necessary, some indices may be unknown.
func ParseSourceMap(sources []string, bytecode []byte, sourceMap string) (*SourceMap, error) {
	instructions := strings.Split(sourceMap, ";")

	srcMap := &SourceMap{
		Sources: sources,
		PosData: make([][]LineCol, 0, len(sources)),
		Instr:   make([]InstrMapping, 0, len(bytecode)),
	}
	// map source code position byte offsets to line/column pairs
	for i, s := range sources {
		if strings.HasPrefix(s, "~") {
			srcMap.PosData = append(srcMap.PosData, nil)
			continue
		}
		dat, err := os.ReadFile(s)
		if err != nil {
			return nil, fmt.Errorf("failed to read source %d %q: %w", i, s, err)
		}
		datStr := string(dat)

		out := make([]LineCol, len(datStr))
		line := uint32(1)
		lastLinePos := uint32(0)
		for i, b := range datStr { // iterate the utf8 or the bytes?
			col := uint32(i) - lastLinePos
			out[i] = LineCol{Line: line, Col: col}
			if b == '\n' {
				lastLinePos = uint32(i)
				line += 1
			}
		}
		srcMap.PosData = append(srcMap.PosData, out)
	}

	instIndex := 0

	// bytecode offset to instruction
	lastInstr := InstrMapping{}
	for i := 0; i < len(bytecode); {
		inst := bytecode[i]
		instLen := 1
		if inst >= 0x60 && inst <= 0x7f { // push instructions
			pushDataLen := inst - 0x60 + 1
			instLen += int(pushDataLen)
		}

		var instMapping string
		if instIndex >= len(instructions) {
			// truncated source-map? Or some instruction that's longer than we accounted for?
			// probably the contract-metadata bytes that are not accounted for in source map
		} else {
			instMapping = instructions[instIndex]
		}
		m, err := parseInstrMapping(lastInstr, instMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to parse instr element in source map: %w", err)
		}

		for j := 0; j < instLen; j++ {
			srcMap.Instr = append(srcMap.Instr, m)
		}
		i += instLen
		instIndex += 1
	}
	return srcMap, nil
}

func NewSourceMapTracer(srcMaps map[common.Address]*SourceMap, out io.Writer) *SourceMapTracer {
	return &SourceMapTracer{srcMaps, out}
}

type SourceMapTracer struct {
	srcMaps map[common.Address]*SourceMap
	out     io.Writer
}

func (s *SourceMapTracer) CaptureTxStart(gasLimit uint64) {}

func (s *SourceMapTracer) CaptureTxEnd(restGas uint64) {}

func (s *SourceMapTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

func (s *SourceMapTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {}

func (s *SourceMapTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (s *SourceMapTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (s *SourceMapTracer) info(codeAddr *common.Address, pc uint64) string {
	info := "non-contract"
	if codeAddr != nil {
		srcMap, ok := s.srcMaps[*codeAddr]
		if ok {
			info = srcMap.FormattedInfo(pc)
		} else {
			info = "unknown-contract"
		}
	}
	return info
}

func (s *SourceMapTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if op.IsPush() {
		start := uint64(op) - uint64(vm.PUSH1) + 1
		end := pc + 1 + start
		val := scope.Contract.Code[pc+1 : end]
		fmt.Fprintf(s.out, "%-40s : pc %x opcode %s (%x)\n", s.info(scope.Contract.CodeAddr, pc), pc, op.String(), val)
		return
	}
	fmt.Fprintf(s.out, "%-40s : pc %x opcode %s\n", s.info(scope.Contract.CodeAddr, pc), pc, op.String())
}

func (s *SourceMapTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	fmt.Fprintf(s.out, "%-40s: pc %x opcode %s FAULT %v\n", s.info(scope.Contract.CodeAddr, pc), pc, op.String(), err)
	fmt.Println("----")
	fmt.Fprintf(s.out, "calldata: %x\n", scope.Contract.Input)
	fmt.Println("----")
	fmt.Fprintf(s.out, "memory: %x\n", scope.Memory.Data())
	fmt.Println("----")
	fmt.Fprintf(s.out, "stack:\n")
	stack := scope.Stack.Data()
	for i := range stack {
		fmt.Fprintf(s.out, "%3d: %x\n", -i, stack[len(stack)-1-i].Bytes32())
	}
}

var _ vm.EVMLogger = (*SourceMapTracer)(nil)

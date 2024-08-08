// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { MIPSInstructions as ins } from "src/cannon/libraries/MIPSInstructions.sol";
import { MIPSSyscalls as sys } from "src/cannon/libraries/MIPSSyscalls.sol";
import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";
import { MIPSMemory } from "src/cannon/libraries/MIPSMemory.sol";

/// @title MIPS
/// @notice The MIPS contract emulates a single MIPS instruction.
///         Note that delay slots are isolated instructions:
///         the nextPC in the state pre-schedules where the VM jumps next.
///         The Step input is a packed VM state, with binary-merkle-tree
///         witness data for memory reads/writes.
///         The Step outputs a keccak256 hash of the packed VM State,
///         and logs the resulting state for offchain usage.
/// @dev https://inst.eecs.berkeley.edu/~cs61c/resources/MIPS_Green_Sheet.pdf
/// @dev https://www.cs.cmu.edu/afs/cs/academic/class/15740-f97/public/doc/mips-isa.pdf
///      (page A-177)
/// @dev https://uweb.engr.arizona.edu/~ece369/Resources/spim/MIPSReference.pdf
/// @dev https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats
/// @dev https://github.com/golang/go/blob/master/src/syscall/zerrors_linux_mips.go
///      MIPS linux kernel errors used by Go runtime
contract MIPS is ISemver {
    /// @notice Stores the VM state.
    ///         Total state size: 32 + 32 + 6 * 8 + 1 + 1 + 8 + 32 * 8 = 378 bytes
    ///         If nextPC != pc + 4, then the VM is executing a branch/jump delay slot.
    struct State {
        bytes32 memRoot;
        bytes32 preimageKey;
        uint64 preimageOffset;
        uint64 pc;
        uint64 nextPC;
        uint64 lo;
        uint64 hi;
        uint64 heap;
        uint8 exitCode;
        bool exited;
        uint64 step;
        uint64[32] registers;
    }

    /// @notice The semantic version of the MIPS contract.
    /// @custom:semver 2.0.0-alpha.1
    string public constant version = "2.0.0-alpha.1";

    /// @notice The preimage oracle contract.
    IPreimageOracle internal immutable ORACLE;

    // The offset of the start of proof calldata (_proof.offset) in the step() function
    uint256 internal constant STEP_PROOF_OFFSET = 548;

    /// @param _oracle The address of the preimage oracle contract.
    constructor(IPreimageOracle _oracle) {
        ORACLE = _oracle;
    }

    /// @notice Getter for the pre-image oracle contract.
    /// @return oracle_ The IPreimageOracle contract.
    function oracle() external view returns (IPreimageOracle oracle_) {
        oracle_ = ORACLE;
    }

    /// @notice Computes the hash of the MIPS state.
    /// @return out_ The hashed MIPS state.
    function outputState() internal returns (bytes32 out_) {
        assembly {
            // copies 'size' bytes, right-aligned in word at 'from', to 'to', incl. trailing data
            function copyMem(from, to, size) -> fromOut, toOut {
                mstore(to, mload(add(from, sub(32, size))))
                fromOut := add(from, 32)
                toOut := add(to, size)
            }

            // From points to the MIPS State
            let from := 0x80

            // Copy to the free memory pointer
            let start := mload(0x40)
            let to := start

            // Copy state to free memory
            from, to := copyMem(from, to, 32) // memRoot
            from, to := copyMem(from, to, 32) // preimageKey
            from, to := copyMem(from, to, 8) // preimageOffset
            from, to := copyMem(from, to, 8) // pc
            from, to := copyMem(from, to, 8) // nextPC
            from, to := copyMem(from, to, 8) // lo
            from, to := copyMem(from, to, 8) // hi
            from, to := copyMem(from, to, 8) // heap
            let exitCode := mload(from)
            from, to := copyMem(from, to, 1) // exitCode
            let exited := mload(from)
            from, to := copyMem(from, to, 1) // exited
            from, to := copyMem(from, to, 8) // step
            from := add(from, 32) // offset to registers

            // Copy registers
            for { let i := 0 } lt(i, 32) { i := add(i, 1) } { from, to := copyMem(from, to, 8) }

            // Clean up end of memory
            mstore(to, 0)

            // Log the resulting MIPS state, for debugging
            log0(start, sub(to, start))

            // Determine the VM status
            let status := 0
            switch exited
            case 1 {
                switch exitCode
                // VMStatusValid
                case 0 { status := 0 }
                // VMStatusInvalid
                case 1 { status := 1 }
                // VMStatusPanic
                default { status := 2 }
            }
            // VMStatusUnfinished
            default { status := 3 }

            // Compute the hash of the resulting MIPS state and set the status byte
            out_ := keccak256(start, sub(to, start))
            out_ := or(and(not(shl(248, 0xFF)), out_), shl(248, status))
        }
    }

    /// @notice Handles a syscall.
    /// @param _localContext The local key context for the preimage oracle.
    /// @return out_ The hashed MIPS state.
    function handleSyscall(bytes32 _localContext) internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory
            State memory state;
            assembly {
                state := 0x80
            }

            // Load the syscall numbers and args from the registers
            (uint64 syscall_no, uint64 a0, uint64 a1, uint64 a2,) = sys.getSyscallArgs(state.registers);

            uint64 v0 = 0;
            uint64 v1 = 0;

            if (syscall_no == sys.SYS_MMAP) {
                (v0, v1, state.heap) = sys.handleSysMmap(a0, a1, state.heap);
            } else if (syscall_no == sys.SYS_BRK) {
                // brk: Returns a fixed address for the program break at 0x40000000
                v0 = sys.BRK_START;
            } else if (syscall_no == sys.SYS_CLONE) {
                // clone (not supported) returns 1
                v0 = 1;
            } else if (syscall_no == sys.SYS_EXIT_GROUP) {
                // exit group: Sets the Exited and ExitCode states to true and argument 0.
                state.exited = true;
                state.exitCode = uint8(a0);
                return outputState();
            } else if (syscall_no == sys.SYS_READ) {
                sys.SysReadParams memory args = sys.SysReadParams({
                    a0: a0,
                    a1: a1,
                    a2: a2,
                    preimageKey: state.preimageKey,
                    preimageOffset: state.preimageOffset,
                    localContext: _localContext,
                    oracle: ORACLE,
                    proofOffset: MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1),
                    memRoot: state.memRoot
                });
                (v0, v1, state.preimageOffset, state.memRoot) = sys.handleSysRead(args);
            } else if (syscall_no == sys.SYS_WRITE) {
                (v0, v1, state.preimageKey, state.preimageOffset) = sys.handleSysWrite({
                    _a0: a0,
                    _a1: a1,
                    _a2: a2,
                    _preimageKey: state.preimageKey,
                    _preimageOffset: state.preimageOffset,
                    _proofOffset: MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1),
                    _memRoot: state.memRoot
                });
            } else if (syscall_no == sys.SYS_FCNTL) {
                (v0, v1) = sys.handleSysFcntl(a0, a1);
            }

            st.CpuScalars memory cpu = getCpuScalars(state);
            sys.handleSyscallUpdates(cpu, state.registers, v0, v1);
            setStateCpuScalars(state, cpu);

            out_ = outputState();
        }
    }

    /// @notice Executes a single step of the vm.
    ///         Will revert if any required input state is missing.
    /// @param _stateData The encoded state witness data.
    /// @param _proof The encoded proof data for leaves within the MIPS VM's memory.
    /// @param _localContext The local key context for the preimage oracle. Optional, can be set as a constant
    ///                      if the caller only requires one set of local keys.
    function step(bytes calldata _stateData, bytes calldata _proof, bytes32 _localContext) public returns (bytes32) {
        unchecked {
            State memory state;

            // Packed calldata is ~6 times smaller than state size
            assembly {
                if iszero(eq(state, 0x80)) {
                    // expected state mem offset check
                    revert(0, 0)
                }
                if iszero(eq(_stateData.offset, 132)) {
                    // 32*4+4=132 expected state data offset
                    revert(0, 0)
                }
                if iszero(eq(_proof.offset, STEP_PROOF_OFFSET)) {
                    // 132+32+384=548 expected proof offset
                    revert(0, 0)
                }

                function putField(callOffset, memOffset, size) -> callOffsetOut, memOffsetOut {
                    // calldata is packed, thus starting left-aligned, shift-right to pad and right-align
                    let w := shr(shl(3, sub(32, size)), calldataload(callOffset))
                    mstore(memOffset, w)
                    callOffsetOut := add(callOffset, size)
                    memOffsetOut := add(memOffset, 32)
                }

                // Unpack state from calldata into memory
                let c := _stateData.offset // calldata offset
                let m := 0x80 // mem offset
                c, m := putField(c, m, 32) // memRoot
                c, m := putField(c, m, 32) // preimageKey
                c, m := putField(c, m, 8) // preimageOffset
                c, m := putField(c, m, 8) // pc
                c, m := putField(c, m, 8) // nextPC
                c, m := putField(c, m, 8) // lo
                c, m := putField(c, m, 8) // hi
                c, m := putField(c, m, 8) // heap
                c, m := putField(c, m, 1) // exitCode
                c, m := putField(c, m, 1) // exited
                c, m := putField(c, m, 8) // step

                // Unpack register calldata into memory
                mstore(m, add(m, 32)) // offset to registers
                m := add(m, 32)
                for { let i := 0 } lt(i, 32) { i := add(i, 1) } { c, m := putField(c, m, 8) }
            }

            // Don't change state once exited
            if (state.exited) {
                return outputState();
            }

            state.step += 1;

            // instruction fetch
            uint256 insnProofOffset = MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 0);
            (uint32 insn, uint64 opcode, uint64 fun) =
                ins.getInstructionDetails(state.pc, state.memRoot, insnProofOffset);

            // Handle syscall separately
            // syscall (can read and write)
            if (opcode == 0 && fun == 0xC) {
                return handleSyscall(_localContext);
            }

            // Exec the rest of the step logic
            st.CpuScalars memory cpu = getCpuScalars(state);
            (state.memRoot) = ins.execMipsCoreStepLogic({
                _cpu: cpu,
                _registers: state.registers,
                _memRoot: state.memRoot,
                _memProofOffset: MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1),
                _insn: insn,
                _opcode: opcode,
                _fun: fun
            });
            setStateCpuScalars(state, cpu);

            return outputState();
        }
    }

    function getCpuScalars(State memory _state) internal pure returns (st.CpuScalars memory) {
        return st.CpuScalars({ pc: _state.pc, nextPC: _state.nextPC, lo: _state.lo, hi: _state.hi });
    }

    function setStateCpuScalars(State memory _state, st.CpuScalars memory _cpu) internal pure {
        _state.pc = _cpu.pc;
        _state.nextPC = _cpu.nextPC;
        _state.lo = _cpu.lo;
        _state.hi = _cpu.hi;
    }
}

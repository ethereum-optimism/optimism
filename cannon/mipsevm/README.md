# `mipsevm`

Supported 55 instructions:
| Category             | Instruction   | Description                                  |
|----------------------|---------------|----------------------------------------------|
| `Arithmetic`         | `addi`        | Add immediate (with sign-extension).         |
| `Arithmetic`         | `addiu`       | Add immediate unsigned (no overflow).        |
| `Arithmetic`         | `addu`        | Add unsigned (no overflow).                  |
| `Logical`            | `and`         | Bitwise AND.                                 |
| `Logical`            | `andi`        | Bitwise AND immediate.                       |
| `Branch`             | `b`           | Unconditional branch.                        |
| `Conditional Branch` | `beq`         | Branch on equal.                             |
| `Conditional Branch` | `beqz`        | Branch if equal to zero.                     |
| `Conditional Branch` | `bgez`        | Branch on greater than or equal to zero.     |
| `Conditional Branch` | `bgtz`        | Branch on greater than zero.                 |
| `Conditional Branch` | `blez`        | Branch on less than or equal to zero.        |
| `Conditional Branch` | `bltz`        | Branch on less than zero.                    |
| `Conditional Branch` | `bne`         | Branch on not equal.                         |
| `Conditional Branch` | `bnez`        | Branch if not equal to zero.                 |
| `Logical`            | `clz`         | Count leading zeros.                         |
| `Arithmetic`         | `divu`        | Divide unsigned.                             |
| `Unconditional Jump` | `j`           | Jump.                                        |
| `Unconditional Jump` | `jal`         | Jump and link.                               |
| `Unconditional Jump` | `jalr`        | Jump and link register.                      |
| `Unconditional Jump` | `jr`          | Jump register.                               |
| `Data Transfer`      | `lb`          | Load byte.                                   |
| `Data Transfer`      | `lbu`         | Load byte unsigned.                          |
| `Data Transfer`      | `lui`         | Load upper immediate.                        |
| `Data Transfer`      | `lw`          | Load word.                                   |
| `Data Transfer`      | `lwr`         | Load word right.                             |
| `Data Transfer`      | `mfhi`        | Move from HI register.                       |
| `Data Transfer`      | `mflo`        | Move from LO register.                       |
| `Data Transfer`      | `move`        | Move between registers.                      |
| `Data Transfer`      | `movn`        | Move conditional on not zero.                |
| `Data Transfer`      | `movz`        | Move conditional on zero.                    |
| `Data Transfer`      | `mtlo`        | Move to LO register.                         |
| `Arithmetic`         | `mul`         | Multiply (to produce a word result).         |
| `Arithmetic`         | `multu`       | Multiply unsigned.                           |
| `Arithmetic`         | `negu`        | Negate unsigned.                             |
| `No Op`              | `nop`         | No operation.                                |
| `Logical`            | `not`         | Bitwise NOT (pseudo-instruction in MIPS).    |
| `Logical`            | `or`          | Bitwise OR.                                  |
| `Logical`            | `ori`         | Bitwise OR immediate.                        |
| `Data Transfer`      | `sb`          | Store byte.                                  |
| `Logical`            | `sll`         | Shift left logical.                          |
| `Logical`            | `sllv`        | Shift left logical variable.                 |
| `Comparison`         | `slt`         | Set on less than (signed).                   |
| `Comparison`         | `slti`        | Set on less than immediate.                  |
| `Comparison`         | `sltiu`       | Set on less than immediate unsigned.        |
| `Comparison`         | `sltu`        | Set on less than unsigned.                   |
| `Logical`            | `sra`         | Shift right arithmetic.                      |
| `Logical`            | `srl`         | Shift right logical.                         |
| `Logical`            | `srlv`        | Shift right logical variable.                |
| `Arithmetic`         | `subu`        | Subtract unsigned.                           |
| `Data Transfer`      | `sw`          | Store word.                                  |
| `Data Transfer`      | `swr`         | Store word right.                            |
| `Serialization`      | `sync`        | Synchronize shared memory.                   |
| `System Calls`       | `syscall`     | System call.                                 |
| `Logical`            | `xor`         | Bitwise XOR.                                 |
| `Logical`            | `xori`        | Bitwise XOR immediate.                       |

To run:
1. Load a program into a state, e.g. using `LoadELF`.
2. Patch the program if necessary: e.g. using `PatchGo` for Go programs, `PatchStack` for empty initial stack, etc.
4. Implement the `PreimageOracle` interface
5. Instrument the emulator with the state, and pre-image oracle, using `NewInstrumentedState`
6. Step through the instrumented state with `Step(proof)`,
   where `proof==true` if witness data should be generated. Steps are faster with `proof==false`.
7. Optionally repeat the step on-chain by calling `MIPS.sol` and `PreimageOracle.sol`, using the above witness data.

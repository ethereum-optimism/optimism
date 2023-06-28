# Fault Proof Virtual Machine Specification

## Overview

This is a description of the Fault Proof Virtual Machine (FPVM). The FPVM emulates a minimal Linux-based system running on big-endian 32-bit MIPS32 architecture. Alot of its behaviors are copied from existing Linux/MIPS specification with a few tweaks made for fault proofs.

Operationally, the Fault Proof VM is a state transition function. This state transition is referred to as a *Step*, indicating a single instruction being executed. We say the VM, given a current state $S_{pre}$, steps on a single instruction to produce a new state $S_{post}$.
 $$f(S_{pre}) \rightarrow S_{post}$$

## State
The virtual machine state highlights the effects of running a Fault Proof Program on the VM.
It consists of the following fields:
1. `memRoot` - A `bytes32` value representing the merkle root of VM memory.
2. `preimageKey` - `bytes32` value of the last requested pre-image key (see [pre-image comms spec](fault-proof.md#pre-image-communication))
3. `preimageOffset` - The 32-bit value of the last requested pre-image offset.
4. `pc` - 32-bit program counter.
5. `nextPC` - 32-bit next program counter. Note that this value may not always be $pc+4$ when executing a branch/jump delay slot.
6. `lo` - 32-bit MIPS LO special register.
7. `hi` - 32-bit MIPS HI special register.
8. `heap` - 32-bit address of the base of the heap.
9. `exitCode` - 8-bit exit code.
10. `exited` - 1-bit indicator that the VM has exited.
11. `registers` - General-purpose MIPS32 registers. Each register is a 32-bit value.

The state is represented by packing the above fields, in order, into a 226 byte buffer.

## Memory

Memory is represented as a binary merkle tree. The tree has a fixed-depth of 27 levels, with leaf values of 32 bytes each. This spans the full 32-bit address space, where each leaf contains the memory at that part of the tree.
The effects of memory writes are reflected by the state `memRoot`, representing the merkle root of the tree.

Memory operations are 4-byte aligned. Instructions that reference unaligned addresses will be re-aligned by the FPVM.
Memory access doesn't require any privileges. A step can access any memory location.

### Heap
The FPVM tracks a heap that starts at `0x20000000`. While its program break is at `0x40000000`, the FPVM is permitted to extend the heap beyond this limit via mmap syscalls. Heap pages are bump allocated, per `mmap` syscall.

## Delay Slots

The post-state of a step updates the `nextPC`, indicating instruction following the `pc`. However, in the case of where a branch instruction is being stepped, the `nextPC` post-state is set to the branch target. And the `pc` post-state set to the branch delay slot as usual.

## Syscalls
Syscalls work similar to [Linux/MIPS](https://www.linux-mips.org/wiki/Syscall), including the syscall calling conventions and general syscall handling behavior. However, the FPVM supports a subset of Linux/MIPS syscalls with slightly different behaviors.
The following table list the supported syscalls and their behaviors
| Syscall | Number | Description |
| ------- | ------ | -------- |
| mmap | 4090 | bump allocates a page |
| brk | 4045 | Returns a fixed address for the program break at `0x40000000` |
| clone | 4120 | Not supported. The VM must set the return register to `1`. |
| exit_group | 4246 | Used to indicate VM exit. The FPVM state's `Exited` and `ExitCode` are set to `true` and the input status respectively. |
| read | 4003 | The read(2) syscall. Behavior is identical to Linux/MIPS. |
| write | 4004 | The write(2) syscall. Behavior is identical to Linux/MIPS. |
| fcntl | 4055 | Supports only the F_GETFL command. |

Note that the above syscalls have identical syscall numbers and ABIs as Linux/MIPS.

For all other syscalls, the VM must do nothing except to zero out the syscall return (`$v0`) and errno (`$a3`) registers.

## I/O
The VM does not support open(2). Instead, the VM supports reading/writing to a limited set of file descriptors.
| Name | File descriptor | Description |
| ---- | --------------- | ----------- |
| stdin | 0 | read-only standard input stream. |
| stdout | 1 | write-only standaard output stream. |
| stderr | 2 | write-only standard error stream. |
| hint response | 3 | read-only. Used to read the status of [pre-image hinting](./fault-proof.md#hinting). |
| hint request | 4 | write-only. Used to provide [pre-image hints](./fault-proof.md#hinting) |
| pre-image response | 5 | read-only. Used to read pre-images. |
| pre-image request | 6 | write-only. Used to request pre-images. |

Syscalls referencing unnkown file descriptors fail with an `EBADF` errno as done on Linux/MIPS.

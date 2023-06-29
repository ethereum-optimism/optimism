# Fault Proof Virtual Machine Specification

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [State](#state)
- [Memory](#memory)
  - [Heap](#heap)
- [Delay Slots](#delay-slots)
- [Syscalls](#syscalls)
- [I/O](#io)
  - [Pre-image Communication](#pre-image-communication)
- [Exceptions](#exceptions)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

This is a description of the Fault Proof Virtual Machine (FPVM). The FPVM emulates a minimal Linux-based system running on big-endian 32-bit MIPS32 architecture. Alot of its behaviors are copied from existing Linux/MIPS specification with a few tweaks made for fault proofs.

Operationally, the Fault Proof VM is a state transition function. This state transition is referred to as a *Step*, indicating a single instruction being executed. We say the VM is a function $f$, given an input state $S_{pre}$, steps on a single instruction encoded in the state to produce a new state $S_{post}$.
$$f(S_{pre}) \rightarrow S_{post}$$

## State
The virtual machine state highlights the effects of running a Fault Proof Program on the VM.
It consists of the following fields:
1. `memRoot` - A `bytes32` value representing the merkle root of VM memory.
2. `preimageKey` - `bytes32` value of the last requested pre-image key.
3. `preimageOffset` - The 32-bit value of the last requested pre-image offset.
4. `pc` - 32-bit program counter.
5. `nextPC` - 32-bit next program counter. Note that this value may not always be $pc+4$ when executing a branch/jump delay slot.
6. `lo` - 32-bit MIPS LO special register.
7. `hi` - 32-bit MIPS HI special register.
8. `heap` - 32-bit address of the base of the heap.
9. `exitCode` - 8-bit exit code.
10. `exited` - 1-bit indicator that the VM has exited.
11. `registers` - General-purpose MIPS32 registers. Each register is a 32-bit value.

The state is represented by packing the above fields, in order, into a 226-byte buffer.

## Memory

Memory is represented as a binary merkle tree. The tree has a fixed-depth of 27 levels, with leaf values of 32 bytes each. This spans the full 32-bit address space, where each leaf contains the memory at that part of the tree.
The state `memRoot` represents the merkle root of the tree, reflecting the effects of memory writes. As a result of this memory representation, all memory operations are 4-byte aligned.

Memory access doesn't require any privileges. An instruction step can access any memory location.

### Heap
FPVM state contains a `heap` tracking the current address of the free store used for memory allocation. Heap pages are bump allocated at the page boundary, per `mmap` syscall. The page size is 4096.

The FPVM has a fixed program break at `0x40000000`. However, the FPVM is permitted to extend the heap beyond this limit via mmap syscalls. For simplicity, there are no memory protections against "heap overruns" against other conceptual segments.
Such steps are considered valid state transitions.

The actual memory mappings is outside the scope of this specification as it is irrelevant to the VM state. FPVM implementers may refer to the Linux/MIPS kernel for inspiration.

## Delay Slots

The post-state of a step updates the `nextPC`, indicating instruction following the `pc`. However, in the case of where a branch instruction is being stepped, the `nextPC` post-state is set to the branch target. And the `pc` post-state set to the branch delay slot as usual.

## Syscalls
Syscalls work similar to [Linux/MIPS](https://www.linux-mips.org/wiki/Syscall), including the syscall calling conventions and general syscall handling behavior. However, the FPVM supports a subset of Linux/MIPS syscalls with slightly different behaviors.
The following table list summarizes the supported syscalls and their behaviors

| $v0 | system call | $a0 | $a1 | $a2 | Effect |
| -- | -- | -- | -- | -- | -- |
| 4090 | mmap | uint32 addr | uint32 len | | Allocates a page from the heap. See [heap](#heap) for details. |
| 4045 | brk | | | | Returns a fixed address for the program break at `0x40000000` |
| 4120 | clone | | | | Returns 1 |
| 4246 | exit_group | uint8 exit_code | | | Sets the Exited and ExitCode states to `true` and `$a0` respectively. |
| 4003 | read | uint32 fd | char *buf | uint32 count | Similar behavior as Linux/MIPS with support for unaligned reads. See [I/O](#io) for more details. |
| 4004 | write | uint32 fd | char *buf | uint32 count | Similar behavior as Linux/MIPS with support for unaligned writes. See [I/O](#io) for more details. |
| 4055 | fcntl | uint32 fd | int32 cmd | | Similar behavior as Linux/MIPS. Only the `F_GETFL` (3) cmd is supported. |

For all of the above syscalls, an error is indicated by setting the return register (`$v0`) to `0xFFFFFFFF` and `errno` (`$a3`) is set accordingly. For all other syscalls, the VM must do nothing except to zero out the syscall return (`$v0`) and errno (`$a3`) registers.

Note that the above syscalls have identical syscall numbers and ABIs as Linux/MIPS.

## I/O
The VM does not support open(2). Only a preset file descriptors can be read from and written to.
| Name | File descriptor | Description |
| ---- | --------------- | ----------- |
| stdin | 0 | read-only standard input stream. |
| stdout | 1 | write-only standaard output stream. |
| stderr | 2 | write-only standard error stream. |
| hint response | 3 | read-only. Used to read the status of [pre-image hinting](./fault-proof.md#hinting). |
| hint request | 4 | write-only. Used to provide [pre-image hints](./fault-proof.md#hinting) |
| pre-image response | 5 | read-only. Used to [read pre-images](./fault-proof.md#pre-image-communication). |
| pre-image request | 6 | write-only. Used to [request pre-images](./fault-proof.md#pre-image-communication). |

Syscalls referencing unnkown file descriptors fail with an `EBADF` errno as done on Linux/MIPS.

Writing to and reading from standard output, input and error streams have no effect on the FPVM state.
FPVM implementations may use them for debugging purposes as long as I/O is stateless.

All I/O operations are restricted to a maximum of 4 bytes per operation.
Any read or write syscall request exceeding this limit will be truncated to 4 bytes.
Consequently, the return value of read syscalls is 4, indicating the actual number of bytes read.

### Pre-image Communication
The `preimageKey` and `preimageOffset` state are updated via read/write syscalls to the pre-image read and write file descriptors (see [I/O](#io)).
The `preimageKey` buffers the stream of bytes written to the pre-image write fd.
The `preimageKey` buffer is shifted to accomodate new bytes written to the end of it.
Each write also resets the `preimageOffset` to 0, indicating the intent to read a new pre-image.

When handling pre-image reads, the `preimageKey` is used to lookup the pre-imgae data from an oracle. A max 4-byte chunk of the pre-image at the `preimageOffset` is returned by the read.
Each read operation increases the `preimageOffset` by the number of bytes requested (capped at 4 bytes).

## Exceptions
The FPVM may raise an exception rather than output a post-state to signal an invalid state transition. Nominally, the FPVM must raise an exception in at least the following cases:
- Invalid instruction (either via an invalid opcode or an instruction referencing registers outside the general purpose registers).
- Pre-image read at an offset larger than the size of the pre-image.

VM implementations may raise an exception in other cases that is specific to the implementation. For example, an on-chain FPVM that relies on pre-supplied merkle proofs of memory access may raise an exception if the supplied merkle root proof doees not match the pre-state `memRoot`.

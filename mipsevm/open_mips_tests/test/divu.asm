###############################################################################
# File         : divu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'divu' instruction.
#
###############################################################################


    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff         # Load the base address 0xbffffff0
    ori     $s0, 0xfff0
    ori     $s1, $0, 1          # Prepare the 'done' status

    #### Test code start ####

    lui     $t0, 0x1234
    ori     $t0, 0x5678
    lui     $t1, 0xc001
    ori     $t1, 0xcafe
    divu    $t1, $t0            # 0xa (q), 0x09f66a4e (r)
    mfhi    $t2
    mflo    $t3
    lui     $t4, 0x09f6
    ori     $t4, 0x6a4e
    lui     $t5, 0x0000
    ori     $t5, 0x000a
    subu    $t6, $t2, $t4
    subu    $t7, $t3, $t5
    sltiu   $v0, $t6, 1
    sltiu   $v1, $t7, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

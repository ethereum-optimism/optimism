###############################################################################
# File         : msubu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'msubu' instruction.
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

    lui     $t0, 0x1234         # Multiply A 0x12345678
    ori     $t0, 0x5678
    lui     $t1, 0xc001         # Multiply B 0xc001cafe
    ori     $t1, 0xcafe
    lui     $t2, 0x3141         # Fused sum 0x3141592631415926
    ori     $t2, 0x5926
    mthi    $t2
    mtlo    $t2
    msubu   $t0, $t1            # 0x2399f7a87ea0de16
    mfhi    $t3
    mflo    $t4
    lui     $t5, 0x2399
    ori     $t5, 0xf7a8
    lui     $t6, 0x7ea0
    ori     $t6, 0xde16
    subu    $t7, $t3, $t5
    subu    $t8, $t4, $t6
    sltiu   $v0, $t7, 1
    sltiu   $v1, $t8, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

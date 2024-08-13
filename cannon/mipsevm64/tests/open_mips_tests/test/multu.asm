###############################################################################
# File         : multu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'multu' instruction.
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
    multu   $t0, $t1            # 0x0da7617db2a07b10
    mfhi    $t2
    mflo    $t3
    lui     $t4, 0x0da7
    ori     $t4, 0x617d
    lui     $t5, 0xb2a0
    ori     $t5, 0x7b10
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

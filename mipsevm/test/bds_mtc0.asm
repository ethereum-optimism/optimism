###############################################################################
# File         : bds_mtc0.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'mtc0' instruction in a branch delay slot
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

    lui     $t0, 0xc001
    ori     $t0, 0xcafe
    j       $check
    mtc0    $t0, $11, 0
    j       $end
    move    $v0, $0

$check:
    mfc0    $t1, $11, 0
    subu    $t2, $t0, $t1
    sltiu   $v0, $t2, 1

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

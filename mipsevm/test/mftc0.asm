###############################################################################
# File         : mftc0.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'mfc0' and 'mtc0' instructions.
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

    # Read and set the compare register (Reg 11 Sel 0)
    lui     $t0, 0xc001
    ori     $t0, 0xcafe
    mtc0    $t0, $11, 0
    mfc0    $t1, $11, 0
    subu    $t2, $t0, $t1
    sltiu   $v0, $t2, 1

    # TODO: Add more tests

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : sll.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'sll' instruction.
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

    lui     $t0, 0xdeaf         # A = 0xdeafbeef
    ori     $t0, 0xbeef
    sll     $t1, $t0, 4         # B = 0xdeafbeef << 4 = 0xeafbeef0
    lui     $t2, 0xeafb         # C = 0xeafbeef0
    ori     $t2, 0xeef0
    subu    $t3, $t1, $t2
    sltiu   $v0, $t3, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

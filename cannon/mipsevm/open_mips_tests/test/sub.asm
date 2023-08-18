###############################################################################
# File         : sub.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'sub' instruction.
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

    lui     $t0, 0xffff         # A = 0xfffffffd (-3)
    ori     $t0, 0xfffd
    sub     $t1, $t0, $t0       # B = A - A = 0
    sub     $t2, $t1, $t0       # C = B - A = 0 - A = 3
    ori     $t3, $0, 3          # D = 2
    sub     $t4, $t2, $t3       # E = C - D = C - 2 = 0
    sltiu   $v0, $t4, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

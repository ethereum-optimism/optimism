###############################################################################
# File         : xori.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'xori' instruction.
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

    ori     $t0, $0, 0xdeaf     # A = 0xdeaf
    xori    $t1, $t0, 0x3141    # B = xor(A, 0x3141) = 0xefee
    xori    $t2, $t1, 0xefef    # C = xor(B, 0xefef) = 0x1
    xori    $t3, $t2, 1         # D = xor(C, 1) = 0
    sltiu   $v0, $t3, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

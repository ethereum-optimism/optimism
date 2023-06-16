###############################################################################
# File         : subu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'subu' instruction.
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
    ori     $t1, $0, 4          # B = 4
    subu    $t2, $t0, $t1       # C = A - B = 0xfffffff9 (-7)
    lui     $t3, 0xffff         # D = 0xfffffff8 (like -8 mod 2^32)
    ori     $t3, 0xfff8
    subu    $t4, $t2, $t3       # F = C - D = 1
    subu    $t5, $t4, $s1       # G = F - 1 = 0
    sltiu   $v0, $t5, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

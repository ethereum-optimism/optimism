###############################################################################
# File         : nor.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'nor' instruction.
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
    ori     $t0, $t0, 0xbeef
    lui     $t1, 0x3141         # B = 0x31415926
    ori     $t1, $t1, 0x5926
    lui     $t2, 0xffff         # C = 0xfffffffe
    ori     $t2, $t2, 0xfffe
    nor     $t3, $t0, $t1       # D = nor(A,B) = 0x00100010
    nor     $v0, $t2, $t3       # E = nor(C,D) = 0x1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

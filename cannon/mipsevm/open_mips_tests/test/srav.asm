###############################################################################
# File         : srav.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'srav' instruction.
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
    ori     $t1, $0, 12
    srav    $t2, $t0, $t1       # B = 0xdeafbeef >> 12 = 0xfffdeafb
    lui     $t3, 0xfffd
    ori     $t3, 0xeafb
    subu    $t4, $t2, $t3
    sltiu   $v0, $t4, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

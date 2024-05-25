###############################################################################
# File         : movn.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'movn' instruction.
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

    lui     $t0, 0xdeaf
    ori     $t0, $t0, 0xbeef
    ori     $t1, $0, 0
    movn    $t2, $t0, $s1       # $t2 gets 0xdeafbeef
    movn    $t1, $t0, $0        # $t1 remains 0
    subu    $t3, $t2, $t0
    sltiu   $v0, $t3, 1
    sltiu   $v1, $t1, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

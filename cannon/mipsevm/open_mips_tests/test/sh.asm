###############################################################################
# File         : sh.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'sh' instruction.
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

    lui     $t0, 0xbfc0         # Load address 0xbfc007fc (last word in 2KB starting
    ori     $t0, 0x07fc         #  from 0xbfc00000)
    sw      $0,  0($t0)
    ori     $t1, $0, 0xc001
    ori     $t2, $0, 0xcafe
    sh      $t1, 0($t0)
    sh      $t2, 2($t0)
    lw      $t3, 0($t0)
    .ifdef big_endian
    lui     $t4, 0xc001
    ori     $t4, 0xcafe
    .else
    lui     $t4, 0xcafe
    ori     $t4, 0xc001
    .endif
    subu    $t5, $t3, $t4
    sltiu   $v0, $t5, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

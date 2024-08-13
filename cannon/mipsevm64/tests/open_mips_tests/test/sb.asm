###############################################################################
# File         : sb.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'sb' instruction.
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
    ori     $t1, $0, 0xc0
    ori     $t2, $0, 0x01
    ori     $t3, $0, 0xca
    ori     $t4, $0, 0xfe
    sb      $t1, 0($t0)
    sb      $t2, 1($t0)
    sb      $t3, 2($t0)
    sb      $t4, 3($t0)
    lw      $t5, 0($t0)
    .ifdef big_endian
    lui     $t6, 0xc001
    ori     $t6, 0xcafe
    .else
    lui     $t6, 0xfeca
    ori     $t6, 0x01c0
    .endif
    subu    $t7, $t5, $t6
    sltiu   $v0, $t7, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : lbu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'lbu' instruction.
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
    lui     $t1, 0xc001
    ori     $t1, 0x7afe
    sw      $t1, 0($t0)
    lbu     $t2, 0($t0)
    lbu     $t3, 1($t0)
    lbu     $t4, 2($t0)
    lbu     $t5, 3($t0)
    .ifdef big_endian
    ori     $t6, $0, 0x00c0
    ori     $t7, $0, 0x0001
    ori     $t8, $0, 0x007a
    ori     $t9, $0, 0x00fe
    .else
    ori     $t6, $0, 0x00fe
    ori     $t7, $0, 0x007a
    ori     $t8, $0, 0x0001
    ori     $t9, $0, 0x00c0
    .endif
    subu    $v1, $t2, $t6
    sltiu   $v0, $v1, 1
    subu    $v1, $t3, $t7
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t4, $t8
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t5, $t9
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

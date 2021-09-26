###############################################################################
# File         : lhu.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'lhu' instruction.
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
    lui     $t1, 0x7001
    ori     $t1, 0xcafe
    sw      $t1, 0($t0)
    lhu     $t2, 0($t0)
    lhu     $t3, 2($t0)
    .ifdef big_endian
    ori     $t4, $0, 0x7001
    ori     $t5, $0, 0xcafe
    .else
    ori     $t4, $0, 0xcafe
    ori     $t5, $0, 0x7001
    .endif
    subu    $v1, $t2, $t4
    sltiu   $v0, $v1, 1
    subu    $v1, $t3, $t5
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    # Repeat with halves swapped (sign extension corner cases)
    lui     $t1, 0xcafe
    ori     $t1, 0x7001
    sw      $t1, 0($t0)
    lhu     $t2, 0($t0)
    lhu     $t3, 2($t0)
    .ifdef big_endian
    ori     $t4, $0, 0xcafe
    ori     $t5, $0, 0x7001
    .else
    ori     $t4, $0, 0x7001
    ori     $t5, $0, 0xcafe
    .endif
    subu    $v1, $t2, $t4
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t3, $t5
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : swr.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'swr' instruction.
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

    lui     $t0, 0xbfc0         # Load address 0xbfc007ec (last four words in 2KB starting
    ori     $t0, 0x07ec         # from 0xbfc00000)
    lui     $t1, 0xc001         # Memory words are 0xc001cafe
    ori     $t1, 0xcafe
    sw      $t1, 0($t0)
    sw      $t1, 4($t0)
    sw      $t1, 8($t0)
    sw      $t1, 12($t0)
    lui     $t2, 0xdeaf         # Register word is 0xdeafbeef
    ori     $t2, 0xbeef
    swr     $t2, 0($t0)
    swr     $t2, 5($t0)
    swr     $t2, 10($t0)
    swr     $t2, 15($t0)
    lw      $s2, 0($t0)
    lw      $s3, 4($t0)
    lw      $s4, 8($t0)
    lw      $s5, 12($t0)
    .ifdef big_endian
    lui     $t3, 0xef01         # 0xef01cafe
    ori     $t3, 0xcafe
    lui     $t4, 0xbeef         # 0xbeefcafe
    ori     $t4, 0xcafe
    lui     $t5, 0xafbe         # 0xafbeeffe
    ori     $t5, 0xeffe
    lui     $t6, 0xdeaf         # 0xdeafbeef
    ori     $t6, 0xbeef
    .else
    lui     $t3, 0xdeaf         # 0xdeafbeef
    ori     $t3, 0xbeef
    lui     $t4, 0xafbe         # 0xafbeeffe
    ori     $t4, 0xeffe
    lui     $t5, 0xbeef         # 0xbeefcafe
    ori     $t5, 0xcafe
    lui     $t6, 0xef01         # 0xef01cafe
    ori     $t6, 0xcafe
    .endif
    subu    $t7, $s2, $t3
    sltiu   $v0, $t7, 1
    subu    $t7, $s3, $t4
    sltiu   $v1, $t7, 1
    and     $v0, $v0, $v1
    subu    $t7, $s4, $t5
    sltiu   $v1, $t7, 1
    and     $v0, $v0, $v1
    subu    $t7, $s5, $t6
    sltiu   $v1, $t7, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

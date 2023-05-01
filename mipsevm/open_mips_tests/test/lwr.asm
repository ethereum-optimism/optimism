###############################################################################
# File         : lwr.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'lwr' instruction.
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
    ori     $t0, 0x07fc         # from 0xbfc00000)
    lui     $t1, 0xc001         # Memory word is 0xc001cafe
    ori     $t1, 0xcafe
    sw      $t1, 0($t0)
    lui     $t2, 0xdeaf         # Register word is 0xdeafbeef
    ori     $t2, 0xbeef
    or      $t3, $0, $t2
    or      $t4, $0, $t2
    or      $t5, $0, $t2
    or      $t6, $0, $t2
    lwr     $t3, 0($t0)
    lwr     $t4, 1($t0)
    lwr     $t5, 2($t0)
    lwr     $t6, 3($t0)
    .ifdef big_endian
    lui     $s3, 0xdeaf         # 0xdeafbec0
    ori     $s3, 0xbec0
    lui     $s4, 0xdeaf         # 0xdeafc001
    ori     $s4, 0xc001
    lui     $s5, 0xdec0         # 0xdec001ca
    ori     $s5, 0x01ca
    lui     $s6, 0xc001         # 0xc001cafe
    ori     $s6, 0xcafe
    .else
    lui     $s3, 0xc001         # 0xc001cafe
    ori     $s3, 0xcafe
    lui     $s4, 0xdec0         # 0xdec001ca
    ori     $s4, 0x01ca
    lui     $s5, 0xdeaf         # 0xdeafc001
    ori     $s5, 0xc001
    lui     $s6, 0xdeaf         # 0xdeafbec0
    ori     $s6, 0xbec0
    .endif
    subu    $s2, $t3, $s3
    sltiu   $v0, $s2, 1
    subu    $s2, $t4, $s4
    sltiu   $v1, $s2, 1
    and     $v0, $v0, $v1
    subu    $s2, $t5, $s5
    sltiu   $v1, $s2, 1
    and     $v0, $v0, $v1
    subu    $s2, $t6, $s6
    sltiu   $v1, $s2, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

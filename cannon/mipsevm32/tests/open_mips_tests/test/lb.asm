###############################################################################
# File         : lb.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'lb' instruction.
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
    lb      $t2, 0($t0)
    lb      $t3, 1($t0)
    lb      $t4, 2($t0)
    lb      $t5, 3($t0)
    .ifdef big_endian
    lui     $t6, 0xffff
    ori     $t6, 0xffc0
    lui     $t7, 0x0000
    ori     $t7, 0x0001
    lui     $t8, 0x0000
    ori     $t8, 0x007a
    lui     $t9, 0xffff
    ori     $t9, 0xfffe
    .else
    lui     $t6, 0xffff
    ori     $t6, 0xfffe
    lui     $t7, 0x0000
    ori     $t7, 0x007a
    lui     $t8, 0x0000
    ori     $t8, 0x0001
    lui     $t9, 0xffff
    ori     $t9, 0xffc0
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

    # Repeat with halves swapped (sign extension corner cases)
    lui     $t1, 0x7afe
    ori     $t1, 0xc001
    sw      $t1, 0($t0)
    lb      $t2, 0($t0)
    lb      $t3, 1($t0)
    lb      $t4, 2($t0)
    lb      $t5, 3($t0)
    .ifdef big_endian
    lui     $t6, 0x0000
    ori     $t6, 0x007a
    lui     $t7, 0xffff
    ori     $t7, 0xfffe
    lui     $t8, 0xffff
    ori     $t8, 0xffc0
    lui     $t9, 0x0000
    ori     $t9, 0x0001
    .else
    lui     $t6, 0x0000
    ori     $t6, 0x0001
    lui     $t7, 0xffff
    ori     $t7, 0xffc0
    lui     $t8, 0xffff
    ori     $t8, 0xfffe
    lui     $t9, 0x0000
    ori     $t9, 0x007a
    .endif
    subu    $v1, $t2, $t6
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
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

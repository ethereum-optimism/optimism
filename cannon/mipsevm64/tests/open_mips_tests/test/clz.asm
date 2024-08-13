###############################################################################
# File         : clz.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'clz' instruction.
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

    lui     $t2, 0xffff         # 0
    ori     $t2, 0xffff
    ori     $t3, $0, 0x0100     # 23
    lui     $t4, 0x0700         # 5
    lui     $t5, 0x0f00         # 4
    lui     $t6, 0x7fff         # 1
    ori     $t7, $0, 0          # 32
    clz     $s2, $t2
    clz     $s3, $t3
    clz     $s4, $t4
    clz     $s5, $t5
    clz     $s6, $t6
    clz     $s7, $t7
    addiu   $s2, 0
    addiu   $s3, -23
    addiu   $s4, -5
    addiu   $s5, -4
    addiu   $s6, -1
    addiu   $s7, -32
    or      $v1, $s2, $s3
    or      $v1, $v1, $s4
    or      $v1, $v1, $s5
    or      $v1, $v1, $s6
    or      $v1, $v1, $s7
    sltiu   $v0, $v1, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : clo.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'clo' instruction.
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

    lui     $t2, 0xffff         # 32
    ori     $t2, 0xffff
    lui     $t3, 0xffff         # 18
    ori     $t3, 0xc000
    lui     $t4, 0xf800         # 5
    lui     $t5, 0xf000         # 4
    lui     $t6, 0x7fff         # 0
    ori     $t7, $0, 0          # 0
    clo     $s2, $t2
    clo     $s3, $t3
    clo     $s4, $t4
    clo     $s5, $t5
    clo     $s6, $t6
    clo     $s7, $t7
    addiu   $s2, -32
    addiu   $s3, -18
    addiu   $s4, -5
    addiu   $s5, -4
    addiu   $s6, 0
    addiu   $s7, 0
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

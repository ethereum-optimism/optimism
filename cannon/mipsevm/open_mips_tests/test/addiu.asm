###############################################################################
# File         : addiu.asm
# Project      : MIPS32 MUX
# Author       : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'addiu' instruction.
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

    lui     $t0, 0xffff         # A = 0xfffffffd (-3)
    ori     $t0, 0xfffd
    addiu   $t1, $t0, 5         # B = A + 5 = 2
    addiu   $t2, $t1, 0xfffe    # C = B + -2 = 0
    sltiu   $v0, $t2, 1         # D = 1 if C == 0

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : and.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'and' instruction.
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

    lui     $t0, 0xdeaf         # A = 0xdeafbeef
    lui     $t1, 0xaaaa         # B = 0xaaaaaaaa
    lui     $t2, 0x5555         # C = 0x55555555
    ori     $t0, 0xbeef
    ori     $t1, 0xaaaa
    ori     $t2, 0x5555
    and     $t3, $t0, $t1       # D = A & B = 0x8aaaaaaa
    and     $t4, $t2, $t3       # E = B & D = 0
    sltiu   $v0, $t4, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

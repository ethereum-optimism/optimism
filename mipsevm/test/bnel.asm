###############################################################################
# File         : bnel.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'bnel' instruction.
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

    ori     $t0, $0,  0xcafe
    ori     $t1, $0,  0xcafe
    ori     $v0, $0,  0         # The test result starts as a failure
    ori     $t2, $0,  0
    ori     $t3, $0,  0
    bnel    $t0, $t1, $finish   # No branch, no BDS
    ori     $t2, $0,  1
    bnel    $t0, $v0, $target
    ori     $t3, $0,  1

$finish:
    sw      $v0, 8($s0)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop
    j       $finish             # Early-by-1 branch detection

$target:
    nop
    bnel    $t0, $0, $likely
    nop
    j       $finish
    nop

$likely:
    subu    $t4, $t3, $t2       # Should be t4 = 1 - 0
    addiu   $t5, $t4, -1        # Should be t5 = 1 - 1 = 0
    bnel    $t0, $v0, $finish
    sltiu   $v0, $t5, 1
    j       $finish
    ori     $v0, $0, 0

    #### Test code end ####

    .end test

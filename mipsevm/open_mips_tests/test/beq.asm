###############################################################################
# File         : beq.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'beq' instruction.
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
    beq     $t0, $v0, $finish   # No branch
    nop
    beq     $t0, $t1, $target
    nop

$finish:
    sw      $v0, 8($s0)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop
    j       $finish             # Early-by-1 branch detection

$target:
    nop
    ori     $v0, $0, 1          # Set the result to pass
    beq     $0, $0, $finish     # Late-by-1 branch detection (result not stored)
    nop
    j       $finish
    nop

    #### Test code end ####

    .end test

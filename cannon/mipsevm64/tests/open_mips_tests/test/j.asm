###############################################################################
# File         : j.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'j' instruction.
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

    j       $target
    ori     $v0, $0, 0          # The test result starts as a failure

$finish:
    sw      $v0, 8($s0)
    sw      $s1, 4($s0)
    jr      $ra
    nop
    j       $finish             # Early-by-1 detection

$target:
    nop
    ori     $v0, $0, 1          # Set the result to pass
    j       $finish             # Late-by 1 detection (result not written)
    nop

    #### Test code end ####

    .end test

###############################################################################
# File         : jalr.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'jalr' instruction.
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

    ori     $v1, $ra, 0         # Save $ra
    la      $t0, $target
    jalr    $t0
    ori     $v0, $0, 0          # The test result starts as a failure

$finish:
    sw      $v0, 8($s0)
    ori     $ra, $v1, 0         # Restore $ra
    sw      $s1, 4($s0)
    jr      $ra
    nop
    j       $finish             # Early-by-1 detection

$target:
    nop
    ori     $v0, $0, 1          # Set the result to pass
    jr      $ra
    nop

    #### Test code end ####

    .end test

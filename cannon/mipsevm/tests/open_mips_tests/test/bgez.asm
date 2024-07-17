###############################################################################
# File         : bgez.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'bgez' instruction.
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

    lui     $t0, 0xffff
    bgez    $t0, $finish        # No branch
    nop
    bgez    $s1, $target
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
    bgez    $0, $finish         # Late-by-1 branch detection (result not stored)
    nop
    j       $finish             # Broken branch recovery
    nop

    #### Test code end ####

    .end test

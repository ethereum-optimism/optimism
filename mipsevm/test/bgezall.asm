###############################################################################
# File         : bgezall.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'bgezall' instruction.
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
    ori     $v0, $0, 0          # The test result starts as a failure
    lui     $t0, 0xffff
    ori     $t1, $0,  0
    ori     $t2, $0,  0
    bgezall $t0, $finish        # No branch, no BDS
    ori     $t1, $0, 0x1
    bgezall $s1, $target
    ori     $t2, $0, 0x1
    j       $likely
    nop

$finish:
    sw      $v0, 8($s0)
    ori     $ra, $v1, 0         # Restore $ra
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop
    j       $finish             # Early-by-1 branch detection

$target:
    nop
    jr      $ra
    nop

$likely:
    subu    $t3, $t2, $t1       # Should be t3 = 1 - 0
    addiu   $t4, $t3, -1        # Should be t4 = 1 - 1 = 0
    bgezall $s1, $finish
    sltiu   $v0, $t4, 1

    #### Test code end ####

    .end test

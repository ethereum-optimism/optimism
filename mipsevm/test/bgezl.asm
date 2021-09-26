###############################################################################
# File         : bgezl.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'bgezl' instruction.
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
    ori     $t1, $0,  0
    ori     $t2, $0,  0
    bgezl   $t0, $finish        # No branch, no BDS
    ori     $t1, $0, 0x1
    bgezl   $s1, $target
    ori     $t2, $0, 0x1

$finish:
    sw      $v0, 8($s0)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop
    j       $finish             # Early-by-1 branch detection

$target:
    nop
    bgezl   $0, $likely
    nop
    j       $finish             # Broken branch recovery
    nop

$likely:
    subu    $t3, $t2, $t1       # Should be t3 = 1 - 0
    subu    $t4, $t3, $s1       # Should be t4 = 1 - 1 = 0
    bgezl   $s1, $finish
    sltiu   $v0, $t4, 1
    j       $finish
    ori     $v0, $0, 0

    #### Test code end ####

    .end test

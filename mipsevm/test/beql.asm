###############################################################################
# File         : beql.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'beql' instruction.
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
    beql    $t0, $v0, $finish   # Expect no branch, no BDS
    ori     $t2, $0,  0xcafe
    beql    $t0, $t1, $target   # Expect branch and BDS
    nop

$finish:
    sw      $v0, 8($s0)         # Late-by-1 branch detection (result not stored)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop
    j       $finish             # Early-by-1 branch detection

$target:
    nop
    beql    $0,  $0,  $likely
    ori     $t3, $0,  0xcafe
    j       $finish
    nop

$likely:
    subu    $t4, $t3, $t2      # Should be t4 = 0xcafe - 0
    subu    $t5, $t4, $t0      # Should be t5 = 0xcafe - 0xcafe = 0
    beql    $0,  $0,  $finish
    sltiu   $v0, $t5, 1        # Set the result to pass

    #### Test code end ####

    .end test

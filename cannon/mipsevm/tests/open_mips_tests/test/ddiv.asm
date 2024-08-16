###############################################################################
# File         : ddiv.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'ddiv' instruction.
#
###############################################################################


    .section .test, "x"
    .balign 8
    .set    noreorder
    .set    mips64
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff         # Load the base address 0xbffffff0
    ori     $s0, 0xfff0
    ori     $s1, $0, 1          # Prepare the 'done' status

    #### Test code start ####

    # In bounds
    li      $t1, 0xbeef         # 0xbeef
    li      $t2, 0x2            # 0x2
    ddiv    $t1, $t2            # Perform the division
    mfhi    $t3
    li      $s2, 0x1
    bne     $t3, $s2, $finish   # Check the result
    nop
    mflo    $t3
    li      $s2, 0x5F77
    bne     $t3, $s2, $finish   # Check the result
    nop

    # Set success flag
    ori     $v0, $0, 1          # Set test result to success

    #### Test code end ####

$finish:
    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

###############################################################################
# File         : dmult.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dmult' instruction.
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
    li      $t3, 0x17DDE        # Expected result
    dmult   $t1, $t2            # Perform the multiplication
    mflo    $t4
    bne     $t4, $t3, $finish   # Check the result
    nop

    # 128-bit multiplication
    li      $t1, 0xFFFFFFFF
    li      $t2, 0x2
    dmult   $t1, $t2
    mfhi    $t3
    li      $s2, 0x1
    bne     $t3, $s2, $finish
    nop
    mflo    $t4
    li      $s2, 0xfffffffe
    bne     $t4, $s2, $finish
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

###############################################################################
# File         : dsra32.asm
# Author       : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsra32' instruction.
#
###############################################################################

    .section .test, "x"
    .balign 4
    .set    noreorder
    .set    mips64
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff         # Load the base address 0xbffffff0
    ori     $s0, 0xfff0
    ori     $s1, $0, 1          # Prepare the 'done' status

    #### Test code start ####

    # Load initial value into $t0 (example value: 0xFFFF123456789ABC)
    li      $t0, 0xFFFF1234
    dsll32  $t0, $t0, 0         # $t0 = 0xFFFF123400000000
    ori     $t0, $t0, 0xBEEF    # $t0 = 0xFFFF12340000BEEF

    # Calculate expected result manually (arithmetic shift right by 32 bits)
    # Expected: 0xFFFFFFFFFFFF1234
    li      $t2, 0xFFFF1234     # Expected result (sign extended - 0xFFFFFFFFFFFF1234)

    # Perform the dsra32 operation
    dsra32  $t1, $t0, 0         # $t1 = $t0 >> 32 (arithmetic shift right by 32 bits)

    # Compare the results
    bne     $t1, $t2, $finish   # If $t1 != $t2, fail the test
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

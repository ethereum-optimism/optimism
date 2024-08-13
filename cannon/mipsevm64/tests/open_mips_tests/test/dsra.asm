###############################################################################
# File         : dsra.asm
# Author       : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsra' instruction.
#
###############################################################################

    .section .test, "x"
    .balign 4
    .set    noreorder
    .set    mips64
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff          # Load the base address 0xbffffff0
    ori     $s0, 0xfff0
    ori     $s1, $0, 1           # Prepare the 'done' status

    #### Test code start ####

    # Load initial value into $t0 (example value: 0xFFFF123456789ABC)
    li      $t0, 0xFFFF1234

    # Calculate expected result manually (arithmetic shift right by 12 bits)
    # Expected: 0x00FFFFF1
    li      $t2, 0xFFFFFFF1     # Expected result

    # Perform the dsra32 operation
    dsra    $t1, $t0, 12        # $t1 = $t0 >> 12 (arithmetic shift right by 12 bits)

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

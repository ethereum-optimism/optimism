###############################################################################
# File         : dsll32.asm
# Author       : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsll32' instruction.
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

    # Load initial value into $t0
    lui     $t0, 0x1234
    ori     $t0, $t0, 0x5678    # $t0 = 0x0000000012345678

    # Calculate expected result manually (shift left by 8 bits)
    # Expected: 0x0000001234567800
    lui     $t2, 0x1234
    ori     $t2, $t2, 0x5678    # $t2 = 0x0000000012345678
    dsll32  $t2, $t2, 0         # Shift left by 32 bits: 0x1234567800000000
    dsrl    $t2, $t2, 24        # Shift right by 24 bits: 0x0000001234567800
    dsll    $t1, $t0, 8         # $t1 = $t0 << 8 = 0x0000001234567800

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

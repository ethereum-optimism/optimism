###############################################################################
# File         : scd.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'scd' instruction.
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

    # Test SCD
    li      $t0, 0x12345678     # Load the value 0x12345678
    dsll32  $t0, $t0, 0         # Shift $t0 left by 32 bits
    li      $t1, 0x87654321     # Load the value 0x87654321
    dsll32  $t1, $t1, 0         # Shift $t1 left by 32 bits
    dsrl32  $t1, $t1, 0         # Shift $t1 right by 32 bits
    or      $t0, $t0, $t1       # Combine $t0 and $t1
    scd     $t0, 8($s0)         # Store the combined value to the base address

    # Check that `rt` was set to 1
    li      $s2, 0x1            # Load the expected value 1
    bne     $t0, $s2, $finish   # Check the return value
    nop

    lw      $t1, 8($s0)         # Load the upper 32 bits of the stored value
    lw      $t2, 12($s0)        # Load the lower 32 bits of the stored value
    li      $s2, 0x12345678     # Load the expected value 0x12345678
    bne     $t1, $s2, $finish   # Check the upper 32 bits
    nop
    li      $s2, 0x87654321     # Load the expected value 0x87654321
    bne     $t2, $s2, $finish   # Check the lower 32 bits
    nop

    # Test SCD (conditional = false)
    scd     $0, 8($s0)          # SCD w/ rt = $zero
    li      $s2, 0x0            # Load the expected value 0
    bne     $0, $s2, $finish    # Check the return value
    nop

    # Check that the link reg was not updated
    li     $s2, 0x0             # Load the expected value 0
    bne    $0, $s2, $finish     # Check the return value
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

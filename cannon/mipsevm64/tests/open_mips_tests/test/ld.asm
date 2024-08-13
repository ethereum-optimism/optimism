###############################################################################
# File         : ld.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'ld' instruction.
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

    # Test LD
    li      $t0, 0x12345678     # Load the value 0x12345678
    li      $t1, 0x87654321     # Load the value 0x87654321
    sw      $t0, 8($s0)         # Store $t0 at 0xbffffff8
    sw      $t1, 12($s0)        # Store $t1 at 0xbffffffc
    ld      $t2, 8($s0)         # Load the doubleword into $t2

    dsll32  $t0, $t0, 0         # Shift $t0 left by 32 bits
    dsll32  $t1, $t1, 0         # Shift $t1 left by 32 bits
    dsrl32  $t1, $t1, 0         # Shift $t1 right by 32 bits
    or      $t0, $t0, $t1       # Combine $t0 and $t1
    bne     $t0, $t2, $finish   # If $t0 != $t2, fail
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

###############################################################################
# File         : dsub.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsub' instruction.
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

    # Check that basic subtraction works
    lui     $t0, 0xffff         # Load upper 48 bits into $t0 (0xFFFF_FFFF_FFFF_0000) - (top 32 bits are sign extended)
    ori     $t0, 0xffff         # Load lower 16 bits into $t0 (0xFFFF_FFFF_FFFF_FFFF)
    dsub    $t1, $t0, $t0       # B = A - A
    bne     $t1, $0, $finish    # If B != 0, fail
    nop

    # Extended subtraction test
    dsrl    $t0, $t0, 16        # Shift right by 16 bits (0x0000_FFFF_FFFF_FFFF)
    dsll    $t0, $t0, 16        # Shift left by 16 bits (0xFFFF_FFFF_FFFF_0000)
    ori     $t0, $t0, 0xeeee    # Set lower 16 bits to 0xFFFF (0xFFFF_FFFF_FFFF_EEEE)
    ori     $t1, $0, 0xeeee     # B = 0xEEEE
    dsub    $t2, $t0, $t1       # C = A - B

    # Check that the result is 0xFFFF_FFFF_FFFF_0000
    lui     $t3, 0xffff         # Load upper 48 bits into $t3 (0xFFFF_FFFF_FFFF_0000)
    bne     $t2, $t3, $finish   # If C != 0xFFFF_FFFF_FFFF_0000, fail
    nop

    # Check standard 64-bit arithmetic
    ori     $v0, $0, 1          # Set test result to success

    #### Test code end ####

$finish:
    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

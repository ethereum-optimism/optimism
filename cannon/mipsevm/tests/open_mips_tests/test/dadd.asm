###############################################################################
# File         : dadd.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dadd' instruction.
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

    # Check that overflow is detected
    lui     $t0, 0xffff         # Load upper 48 bits into $t0 (0xFFFF_FFFF_FFFF_0000) - (top 32 bits are sign extended)
    ori     $t0, 0xfffd         # Load lower 16 bits into $t0 (0xFFFF_FFFF_FFFF_FFFD)
    ori     $t1, $0, 0x3        # B = 0x3
    dadd    $t2, $t0, $t1       # C = A + B = 0
    bne     $t2, $0, $finish    # If C != 0, fail
    nop

    # Add standard unsigned 64-bit numbers, no overflow
    lui     $t3, 0xffff         # Load upper 48 bits into $t3 (0xFFFF_FFFF_FFFF_0000) - (top 32 bits are sign extended)
    ori     $t3, $t3, 0x3       # Load lower 16 bits into $t3 (0xFFFF_FFFF_FFFF_0003)

    dsrl    $t0, $t0, 16        # Shift $t0 right by 16 bits (0x0000_FFFF_FFFF_FFFF)
    dsll    $t0, $t0, 16        # Shift $t0 left by 16 bits (0xFFFF_FFFF_FFFF_0000)
    ori     $t1, $0, 0x3        # B = 0x3
    dadd    $t2, $t0, $t1       # C = A + B = 0xFFFF_FFFF_FFFF_0003
    bne     $t2, $t3, $finish   # If C != 0xFFFF_FFFF_FFFF_0003, fail
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

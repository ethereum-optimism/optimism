###############################################################################
# File         : lwu.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'lwu' instruction.
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

    # Test LWU
    li      $t0, 0xFFFFFFFF     # Load 0xFFFFFFFF into $t0
    dsrl32  $t0, $t0, 0         # Clear the upper 32 bits of $t0
    sw      $t0, 8($s0)         # Store 0xFFFFFFFF at 0xbffffff8
    lwu     $t1, 8($s0)         # Load 0xFFFFFFFF from 0xbffffff8
    bne     $t1, $t0, $finish   # Fail if $t1 != $t0
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

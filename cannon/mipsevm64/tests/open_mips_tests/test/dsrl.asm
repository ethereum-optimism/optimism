###############################################################################
# File         : dsrl.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsrl' instruction.
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

    lui     $t0, 0xbfc0         # Load address 0xbfc007f8 (last double word in 2KB starting
    ori     $t0, 0x07f8         # from 0xbfc00000)

    li      $t1, 0x0FFFFFFF     # Load 0xFFFFFFF set into $t1
    dsrl    $t1, $t1, 16        # Shift $t1 right by 16 bits
    li      $t2, 0xFFF          # Load 0x00000FFF into $t2
    bne     $t1, $t2, $finish   # Check if $t1 == $t2
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

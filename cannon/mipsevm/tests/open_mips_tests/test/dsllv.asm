###############################################################################
# File         : dsllv.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'dsllv' instruction.
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

    li      $t1, 0x0000FFFF     # Load 0x0000FFFF set into $t1
    ori     $s2, $s2, 0x8       # Load shamt into $s2
    dsllv   $t1, $t1, $s2       # Shift $t1 left by $s2
    li      $t2, 0x00FFFF00     # Load 0x00FFFF00 set into $t2
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

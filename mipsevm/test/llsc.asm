###############################################################################
# File         : llsc.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'll' and 'sc' instructions.
#
###############################################################################


    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff         # Load the base address 0xbffffff0
    ori     $s0, 0xfff0
    ori     $s1, $0, 1          # Prepare the 'done' status

    #### Test code start ####

    lui     $s2, 0xbfc0         # Load address 0xbfc007fc (last word in 2KB starting
    ori     $s2, 0x07fc         # from 0xbfc00000)
    lui     $s3, 0xdeaf         # Original memory word: 0xdeafbeef
    ori     $s3, 0xbeef
    sw      $s3, 0($s2)
    lui     $s4, 0xc001         # New memory word: 0xc001cafe
    ori     $s4, 0xcafe

    ### Test: Success
    move    $t0, $s3
    move    $t1, $s4
    ll      $t2, 0($s2)
    sc      $t1, 0($s2)
    subu    $v1, $t2, $s3       # Make sure the load worked
    sltiu   $v0, $v1, 1
    lw      $t3, 0($s2)         # Memory should have the new value
    subu    $v1, $t3, $s4
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    addiu   $v1, $t1, -1        # The sc dest reg should be 1
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    ### Test: Failure
    move    $t4, $s4
    sw      $s3, 0($s2)
    ll      $t5, 0($s2)
    sw      $0, 0($s2)
    sc      $t4, 0($s2)
    subu    $v1, $t5, $s3       # Make sure the loads worked
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $t7, 0($s2)         # Memory should have the old value
    sltiu   $v1, $t7, 1
    and     $v0, $v0, $v1
    sltiu   $v1, $t4, 1         # The sc dest reg should be 0
    and     $v0, $v0, $v1

    ### Test: Failure (Eret)
    sw      $s3, 0($s2)
    move    $t8, $s4
    mfc0    $k0, $12, 0         # Load the Status register
    lui     $k1, 0x1dbf         # Disable CP1-3, No RE, No BEV
    ori     $k1, 0x00e6         # Disable ints, kernel mode
    and     $k0, $k0, $k1
    mtc0    $k0, $12, 0
    la      $k1, $post_eret
    mtc0    $k1, $30, 0
    ll      $t9, 0($s2)
    eret
$post_eret:
    sc      $t8, 0($s2)
    subu    $v1, $t9, $s3       # Make sure the load worked
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $s5, 0($s2)         # Memory should have the old value
    subu    $v1, $s5, $s3
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    sltiu   $v1, $t8, 1         # The sc dest reg should be 0
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

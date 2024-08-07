###############################################################################
# File         : sdr.asm
# Author:      : clabby (github.com/clabby)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'sdr' instruction.
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

    lui     $t1, 0xc001         # Memory double word is 0xc001cafe_babef00d
    ori     $t1, 0xcafe
    dsll    $t1, $t1, 32        # Shift left 32 bits
    ori     $t2, $0, 0xbabe
    dsll    $t2, $t2, 16        # Shift left 16 bits
    ori     $t2, $t2, 0xf00d
    daddu   $t1, $t1, $t2       # Combine the two words into $t1
    sd      $t1, 0($t0)

    # Test 1
    sdr     $t1, 8($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll32  $t2, $t2, 24
    bne     $s2, $t2, $finish
    nop

    # Test 2
    sdr     $t1, 9($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll32  $t2, $t2, 16
    bne     $s2, $t2, $finish
    nop

    # Test 3
    sdr     $t1, 10($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll32  $t2, $t2, 8
    bne     $s2, $t2, $finish
    nop

    # Test 4
    sdr     $t1, 11($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll32  $t2, $t2, 0
    bne     $s2, $t2, $finish
    nop

    # Test 5
    sdr     $t1, 12($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll    $t2, $t2, 24
    bne     $s2, $t2, $finish
    nop

    # Test 6
    sdr     $t1, 13($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll    $t2, $t2, 16
    bne     $s2, $t2, $finish
    nop

    # Test 7
    sdr     $t1, 14($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    dsll    $t2, $t2, 8
    bne     $s2, $t2, $finish
    nop

    # Test 8
    sdr     $t1, 15($t0)
    ld      $s2, 8($t0)
    or      $t2, $0, $t1
    bne     $s2, $t2, $finish
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

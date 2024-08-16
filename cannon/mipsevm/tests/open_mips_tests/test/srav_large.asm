###############################################################################
# Description:
#   Tests that the 'srav' instruction properly only utilizes the lower 5 bits
#   of the rs register rather than using the entire 32 bits.
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

    lui     $t0, 0xdeaf         # A = 0xdeafbeef
    ori     $t0, 0xbeef
    ori     $t1, $0, 0x2c
    srav    $t2, $t0, $t1       # B = 0xdeafbeef >> (0x2c & 0x1f) = 0xdeadbeef >> 12 = 0xfffdeafb
    lui     $t3, 0xfffd
    ori     $t3, 0xeafb
    subu    $t4, $t2, $t3
    sltiu   $v0, $t4, 1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

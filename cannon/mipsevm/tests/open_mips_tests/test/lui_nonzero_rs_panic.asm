###############################################################################
# Description:
#   Tests that the 'lui' instruction panics when the rs field is nonzero.
#
###############################################################################

    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff
    ori     $s0, 0xfff0
    ori     $s1, $0, 1

    # Invalid LUI (nonzero rs field)
    .word 0x3d200280

    sw      $zero, 8($s0)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop

    .end test

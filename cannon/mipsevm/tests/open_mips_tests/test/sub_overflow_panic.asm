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

    lui     $t0, 0x7fff         # A = 0x7fffffff (maximum positive 32-bit integer)
    ori     $t0, 0xffff
    lui     $t1, 0xffff         # B = 0xffffffff (-1)
    ori     $t1, 0xffff
    sub     $t2, $t0, $t1       # C = A - B = 0x7fffffff - (-1) = 0x80000000 (overflow)

    # Unreachable...
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

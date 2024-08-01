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

    lui     $t0, 0x8000         # A = 0x80000000 (minimum negative 32-bit integer)
    ori     $t0, 0x0000
    ori     $t1, $0, 1          # B = 1
    sub     $t2, $t0, $t1       # C = A - B = 0x80000000 - 1 (underflow)

    # Unreachable...
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

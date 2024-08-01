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

    lui     $t0, 0xffff         # A = 0xffffffff (maximum unsigned 32-bit integer)
    ori     $t0, 0xffff
    ori     $t1, $0, 1          # B = 1
    addu    $t2, $t0, $t1       # C = A + B (simulate overflow)
    sltu    $v0, $t2, $t0       # D = 1 if overflow (C < A)

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

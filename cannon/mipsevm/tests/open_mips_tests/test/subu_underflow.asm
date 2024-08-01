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

    ori     $t0, $0, 1          # A = 1
    ori     $t1, $0, 2          # B = 2
    subu    $t2, $t0, $t1       # C = A - B = 0xFFFFFFFF
    sltu    $v0, $t0, $t1       # D = 1 if underflow occurred (A < B)

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

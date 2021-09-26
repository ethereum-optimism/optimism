###############################################################################
# File         : interrupt_sw.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the basic functionality of software interrupts.
#   This triggers SW Int 0 to the general exception vector (0x80000180)
#   and then SW Int 1 to the interrupt vector (0x80000200), thus
#   performing basic verification of the exception vectors and offsets too.
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
    j       $setup              # Enable interrupts and exit boot mode
    nop

$run:
    move    $s2, $ra            # Save $ra since we'll use it
    sw      $0, 12($s0)         # Clear the scratch register
    mfc0    $k0, $13, 0         # Fire sw interrupt 0 to 0x80000180 (general)
    ori     $k0, 0x0100
    mtc0    $k0, $13, 0
    jal     busy
    nop
    lw      $t0, 12($s0)        # Check scratch register for 0x1
    addiu   $t1, $t0, -1
    bne     $t1, $0, $fail
    nop
    mfc0    $k0, $13, 0
    lui     $t2, 0x0080         # Enable the 'special' interrupt vector
    ori     $t2, 0x0200         # Fire sw interrupt 1 to 0x80000200 (iv)
    or      $k1, $k0, $t2
    mtc0    $k1, $13, 0
    jal     busy
    nop
    lw      $t3, 12($s0)        # Check scratch register for 0x2
    addiu   $t4, $t3, -2
    bne     $t4, $0, $fail
    nop
    sw      $s1, 8($s0)         # Set the test to pass
    sw      $s1, 4($s0)         # Set 'done'
    jr      $s2

$fail:
    sw      $0, 8($s0)          # Set the test result
    sw      $s1, 4($s0)         # Set 'done'
    jr      $s2

$setup:
    mfc0    $k0, $12, 0         # Load the Status register
    lui     $k1, 0x1000         # Allow access to CP0
    ori     $k1, 0x0301         # Enable sw Int 0,1
    or      $k0, $k0, $k1
    lui     $k1, 0x1dbf         # Disable CP3-1, No RE, No BEV
    ori     $k1, 0x03e7         # Disable hw ints, sw int 1-0, kernel mode, IE
    and     $k0, $k0, $k1
    mtc0    $k0, $12, 0         # Commit the new Status register
    la      $k0, $run           # Set ErrorEPC address to main test body
    mtc0    $k0, $30, 0
    eret

busy:                           # Allow time for an interrupt to be detected
    nop
    nop
    nop
    nop
    nop
    nop
    nop
    nop
    nop
    jr      $ra
    nop

    #### Test code end ####

    .end test

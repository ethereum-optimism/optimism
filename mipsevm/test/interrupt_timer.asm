###############################################################################
# File         : int_timer_cache.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the timer interrupt, i.e., hardware interrupt 5 with the interrupt
#   vector running from the cache
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
    lw      $t0, 0($a0)         # Return success if the counter is >=30
    sltiu   $t1, $t0, 30
    bne     $t1, $0, $run
    nop
    sw      $s1, 8($s0)
    sw      $s1, 4($s0)
$loop:
    j       $loop
    nop

$setup:
    mfc0    $k0, $16, 0         # Config: Enable kseg0 caching
    lui     $k1, 0xffff
    ori     $k1, 0xfff8
    and     $k0, $k0, $k1
    ori     $k0, 0x3
    mtc0    $k0, $16, 0
    mfc0    $k0, $12, 0         # Status: CP0, timer (int 5), ~CP3-1, ~RE, ~BEV, kernel, IE
    lui     $k1, 0x1dbf
    ori     $k1, 0x80e7
    and     $k0, $k0, $k1
    lui     $k1, 0x1000
    ori     $k1, 0x8001
    or      $k0, $k0, $k1
    mtc0    $k0, $12, 0
    mfc0    $k0, $13, 0         # Cause: Use the special interrupt vector
    lui     $k1, 0x0080
    or      $k0, $k0, $k1
    mtc0    $k0, $13, 0
    mfc0    $k0, $9, 0          # Set Compare to the near future (+200 cycles)
    addiu   $k0, 200
    mtc0    $k0, $11, 0
    la      $k0, $run           # Set ErrorEPC address to main test body (cached)
    lui     $k1, 0xdfff
    ori     $k1, 0xffff
    and     $k0, $k0, $k1
    mtc0    $k0, $30, 0
    la      $a0, data           # Use $a0 to hold the address of the iteration count
    eret

data:
    .word 0x0

    #### Test code end ####

    .end test

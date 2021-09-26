###############################################################################
# File         : xop_flush.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test for proper flushing caused by serialized XOP instructions
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
setup:
    mfc0    $k0, $12, 0         # Load the Status register
    lui     $k1, 0x1000         # Allow access to CP0
    or      $k0, $k0, $k1
    lui     $k1, 0x1dff         # Disable CP3-1, No RE, keep BEV
    ori     $k1, 0x00e6         # No interrupts, use kernel mode
    and     $k0, $k0, $k1
    mtc0    $k0, $12, 0
    la      $k0, enable_cache   # Set ErrorEPC so we continue after the reset exception
    mtc0    $k0, $30, 0
    eret

enable_cache:
    mfc0    $t0, $16, 0         # Enable kseg0 caching (Config:K0 = 0x3)
    lui     $t1, 0xffff
    ori     $t1, 0xfff8
    and     $t0, $t0, $t1
    ori     $t0, 0x3
    mtc0    $t0, $16, 0
    la      $t1, $cache_on      # Run the rest of the code with the i-cache enabled (kseg0)
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $t1, $t1, $t0       # Clearing bit 29 of a kseg1 address moves it to kseg0
    jr      $t1
    li      $v0, 1              # Initialize the test result (1 is pass)
$cache_on:
    jal     test_1
    nop
    jal     test_2
    nop
    j       $end
    nop

test_1:   # Interleave XOP/regular instructions, verify execution counts
    li      $t0, 0
    mtc0    $0,  $11, 0         # Use 'Compare' as a scratch register in CP0
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    mtc0    $0,  $11, 0
    addiu   $t0, $t0, 1
    addiu   $v1, $t0, -20
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    jr      $ra
    nop

test_2:   # Cause an exception in a flush slot. It shouldn't fire
    lui     $t0, 0x7fff
    ori     $t0, 0xffff
    syscall                     # The handler will set t0 to 0
    addi    $t0, 1              # Arithmetic overflow exception
    addi    $t0, 1
    addi    $t0, 1
    addi    $t0, 1
    addi    $t0, 1
    addi    $t0, 1
    addi    $t0, 1
    addi    $t0, 1
    jr      $ra
    nop

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

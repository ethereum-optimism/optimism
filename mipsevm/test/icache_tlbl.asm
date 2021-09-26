###############################################################################
# File         : icache_tlbl.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test that an icache operation causes a TLB exception when needed
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

    jal     $setup
    nop
    xor     $v0, $v0, $v0
    la      $ra, $end
cache_inst:
    j       $end
    cache   0x10, 0($0)         # Cause a TLB miss for L1-ICache HitInv
    nop

$setup:
    mfc0    $k0, $12, 0         # Load the Status register for general setup
    lui     $k1, 0x1000         # Allow access to CP0
    or      $k0, $k0, $k1
    lui     $k1, 0x1dff         # Disable CP3-1, No RE, BEV
    ori     $k1, 0x00e6         # Disable all interrupts, kernel mode
    and     $k0, $k0, $k1
    mtc0    $k0, $12, 0         # Commit new Status register
    mtc0    $ra, $30, 0         # Set ErrorEPC to the return address of this call
    la      $a0, cache_inst     # Used by exception handler for verification
    eret                        # Exit boot mode and return

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

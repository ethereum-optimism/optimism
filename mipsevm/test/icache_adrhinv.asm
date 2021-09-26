###############################################################################
# File         : icache_adrhinv.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the address hit invalidate operation on the icache
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

    mfc0    $t0, $16, 0         # Enable kseg0 caching (Config:K0 = 0x3)
    lui     $t1, 0xffff
    ori     $t1, 0xfff8
    and     $t0, $t0, $t1
    ori     $t0, 0x3
    mtc0    $t0, $16, 0
    la      $t0, $cache_on      # Run the remainder of the code from the i-cache
    lui     $t1, 0xdfff
    ori     $t1, 0xffff
    and     $t0, $t0, $t1
    jr      $t0
    nop
$cache_on:
    la      $s2, $mutable       # Run the mutable code once in kseg0 to cache it
    lui     $t0, 0xdfff         #   (Clear bit 29 to change the address from kseg1 to kseg0)
    ori     $t0, 0xffff
    and     $s2, $s2, $t0
    jalr    $s2
    nop
    addiu   $v1, $s7, -123      # Sanity check the call result
    sltiu   $v0, $v1, 1
    la      $t0, $mutable       # Replace the mutable instruction with "li $s7, 321"
    lui     $s3, 0x2417
    ori     $s3, 0x0141
    sw      $s3, 0($t0)         # This address is in kseg1, hence uncacheable so it will go to mem
    jalr    $s2                 # Call the cacheable version again to verify it was actually
    nop                         #   cached--if so it will give the old value
    addiu   $v1, $s7, -123
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    cache   0x10, 0($t0)        # Test 1: Invalidate via the kseg1 address
                                # 0x10 <=> {100, 00} <=> {AddrHitInvalidate, L1-ICache}
    jalr    $s2                 # Call the cacheable version. If the invalidation worked it will
    nop                         #   pull the new instruction from memory which sets '321'. Otherwise
                                #   the invalidation (or uncacheable store) failed.
    addiu   $v1, $s7, -321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lui     $s4, 0x2417         # Replace the new instruction with the original version
    ori     $s4, 0x007b
    sw      $s4, 0($t0)
    cache   0x10, 0($t0)        # Test 2: Invalidate via the kseg0 address
    jalr    $s2                 # Cacheable version again. Should give '123'
    nop
    addiu   $v1, $s7, -123
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .balign 4096                # Place this code at address 0xXXXXXaa0, which is the arbitrary
    .skip 0xaa0, 0              #   index 0xaa (170) for an 8 KiB 2-way 16-byte block cache
$mutable:
    li      $s7, 123            # The arbitrary number 123 indicates the instruction is unchanged
    jr      $ra
    nop

    .end test

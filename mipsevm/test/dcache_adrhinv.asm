###############################################################################
# File         : dcache_adrhinv.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the address hit invalidate operation on the dcache
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
    la      $t1, $cache_on      # Run this code with the i-cache enabled by jumping to the cacheable address
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $t1, $t1, $t0
    jr      $t1
    li      $v0, 1              # Initialize the test result (1 is pass)
$cache_on:
    la      $s2, word           # Uncacheable address for 'word' in kseg1
    la      $s3, word           # Cacheable address of 'word' in kseg0 (set below)
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $s3, $s3, $t0       # Clearing bit 29 changes a kseg1 address to kseg0
    jal     test1
    nop
    jal     test2
    nop
    jal     test3
    nop
    j       $end
    nop

test1: # No Writeback: Load to cache, modify, invalidate => memory/uncached value should not be updated
    lw      $t1, 0($s3)         # Load 'word' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    li      $t0, 4321           # Modify 'word' in the cache
    sw      $t0, 0($s3)
    lw      $t0, 0($s3)         # Sanity check that the store worked
    addiu   $v1, $t0, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    cache   0x11, 0($s3)        # Invalidate 'word' in the cache (11 <=> {3'AdrHitInv, 2'L1-DCache})
    lw      $t2, 0($s2)         # Check that the (uncached) memory word was not updated
    addiu   $v1, $t2, -1234
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    jr      $ra
    nop

test2: # Invalidate: Load to cache, invalidate, write uncached, load to cache => new value in cache
    lw      $t1, 0($s3)         # Load 'word' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    cache   0x15, 0($s3)        # Remove 'word' from the cache
    li      $t0, 4321           # Modify 'word' in memory
    sw      $t0, 0($s2)
    lw      $t1, 0($s3)         # Load 'word' to the cache. It should have the new value
    addiu   $v1, $t1, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    li      $t0, 1234           # Restore 'word' for the next test
    sw      $t0, 0($s2)
    jr      $ra
    sw      $t0, 0($s3)

test3: # Invalidate: Load to cache, modify, write uncached, invalidate, load to cache => new value in cache
    lw      $t1, 0($s3)         # Load 'word' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    li      $t0, 777            # Modify 'word' in the cache
    sw      $t0, 0($s3)
    li      $t0, 4321           # Modify 'word' in memory
    sw      $t0, 0($s2)
    cache   0x11, 0($s3)        # Remove 'word' from the cache
    lw      $t1, 0($s3)         # Load 'word' to the cache. It should have the new uncached value
    addiu   $v1, $t1, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    jr      $ra
    nop

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .balign 4096                # Place this code at address 0xXXXXX{3,7,b,f}30, which is the arbitrary
    .skip 0xaa0, 0              #   index 0x33 (51) for a 2 KiB 2-way 16-byte-block cache
word:
    .word 0x000004d2            # Arbitrary value (1234) indicating the data is unchanged

    .end test

###############################################################################
# File         : dcache_stag1.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the index store tag operation on the dcache: Invalidate a block
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
    la      $t1, $cache_on      # Run the rest of the code with the i-cache enabled (kseg0)
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $t1, $t1, $t0       # Clearing bit 29 of a kseg1 address moves it to kseg0
    j       $t1
    nop
$cache_on:
    la      $s2, word           # Uncacheable address for 'word' in kseg1
    la      $s3, word           # Cacheable address of 'word' in kseg0
    and     $s3, $s3, $t0       # Clearing bit 29 of a kseg1 address moves it to kseg0
    lw      $t1, 0($s3)         # Load 'word' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v0, $v1, 1
    li      $t2, 4321           # Store a new value to 'word' in the cache
    sw      $t2, 0($s3)
    lw      $t1, 0($s3)         # Sanity check that the store worked (new value)
    addiu   $v1, $t1, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    mtc0    $0, $28, 0          # Prepare TagHi/TagLo for invalidation
    mtc0    $0, $29, 0
    lui     $t0, 0x8000         # Invalidate index 0x33 (51) for both ways (1 KiB apart)
    cache   0x9, 0x0330($t0)    # 0x9 <=> {010, 01} <=> {IdxStoreTag, L1-DCache}
    cache   0x9, 0x0730($t0)
    lw      $t3, 0($s2)         # Load 'word' from memory. It should be the old value
    addiu   $v1, $t3, -1234
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $t4, 0($s3)         # Load again but cached. Expect the old value still
    addiu   $v1, $t4, -1234
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    j       $end                # Set the result and finish
    nop

    .balign 1024                # Place this code at address 0xXXXXX{3,7,b,f}30, which is the arbitrary
    .skip 0x330, 0              #   index 0x33 (51) for a 2 KiB 2-way 16-byte-block cache
word:
    .word 0x000004d2            # Arbitrary value (1234) indicating the data is unchanged

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

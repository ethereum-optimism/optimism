###############################################################################
# File         : dcache_idxwbinv.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the index writeback invalidate operation on the dcache
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
    la      $s2, word           # Uncacheable address for 'word' in kseg1
    la      $s3, word           # Cacheable address of 'word' in kseg0
    lui     $t0, 0xdfff         #  (clear bit 29 to change from kseg1 to kseg0)
    ori     $t0, 0xffff
    and     $s3, $s3, $t0
    lw      $t1, 0($s3)         # Load 'word' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v0, $v1, 1
    li      $t0, 4321           # Store a new value to 'word' in the cache
    sw      $t0, 0($s3)
    lw      $t2, 0($s3)         # Sanity check that the store worked (new value)
    addiu   $v1, $t2, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $t3, 0($s2)         # Verify that the uncached value did not update
    addiu   $v1, $t3, -1234
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lui     $t0, 0x8000         # Invalidate index 0x33 (51) for both ways (1 KiB apart)
    cache   0x1, 0x0330($t0)    # 0x1 <=> {000, 01} <=> {IdxWbInv, L1-DCache}
    cache   0x1, 0x0730($t0)
    lw      $t4, 0($s2)         # Load 'word1' uncached (should have new value)
    addiu   $v1, $t4, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v1, $v1
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

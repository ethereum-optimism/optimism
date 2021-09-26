###############################################################################
# File         : dcache_stag2.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test index store tag operation on the dcache: Change a tag to a new address
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
    la      $s2, word1          # Uncacheable address of 'word1' in kseg1
    la      $s3, word1          # Cacheable address of 'word1' in kseg1 (set below)
    la      $s4, word2          # Uncacheable address of 'word2' in kseg1
    la      $s5, word2          # Cacheable address of 'word2' in kseg0 (set below)
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $s3, $s3, $t0       # Clearing bit 29 changes a kseg1 address to kseg0
    and     $s5, $s5, $t0
    lw      $t1, 0($s3)         # Load 'word1' into the cache
    addiu   $v1, $t1, -1234     # Sanity check that the load worked
    sltiu   $v0, $v1, 1
    li      $t0, 7777
    sw      $t0, 0($s3)         # Change the value of 'word1' in the cache
    lw      $t2, 0($s3)         # Sanity check that the store worked
    addiu   $v1, $t2, -7777
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lui     $t0, 0x1fff         # Compute the physical address of 'word2'
    ori     $t0, 0xffff
    and     $t3, $t0, $s4
    srl     $t3, $t3, 1         # TagLo format is {1'b0, 23'b(Tag), 2'b(State), 6'b0}
    lui     $t0, 0xffff
    ori     $t0, 0xff00
    and     $t3, $t3, $t0
    ori     $t3, 0x00c0         # If any bit of State (TagLo[7:6]) is high => valid (if both => dirty)
    mtc0    $t3, $28, 0         # Send the tag to TagLo and zero TagHi
    mtc0    $0,  $29, 0
    lui     $t0, 0x8000         # Change the tag of 'word1' to 'word2' (assumes data is in set A!)
    cache   0x9, 0x0730($t0)    # 0x9 <=> {010, 01} <=> {IdxStoreTag, L1-DCache}
    lw      $t4, 0($s5)         # Load 'word2' from the cache. It should contain '7777'
    addiu   $v1, $t4, -7777
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $t5, 0($s4)         # Load 'word2' uncached. It should be the original value
    addiu   $v1, $t5, -4321
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lw      $t6, 0($s3)         # Load 'word1' cached. It should be the original value
    addiu   $v1, $t6, -1234
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    j       $end                # Set the result and finish
    nop

    .balign 1024                # Place this code at address 0xXXXXX{3,7,b,f}30, which is the arbitrary
    .skip 0x330, 0              #   index 0x33 (51) for a 2 KiB 2-way 16-byte-block cache
word1:
    .word 0x000004d2            # Arbitrary value (1234) indicating the data is unchanged

    .balign 1024                # Move to another address with the same set index
    .skip 0x330, 0
word2:
    .word 0x000010e1            # Arbitrary value (4321)

$end:
    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

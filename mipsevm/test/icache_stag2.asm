###############################################################################
# File         : icache_stag2.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test index store tag operation on the icache: Change a tag to a new address
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

    # Procedure: Load procA to cache, change tag to procB, jump to procB and it should execute procA.

    mfc0    $t0, $16, 0         # Enable kseg0 caching (Config:K0 = 0x3)
    lui     $t1, 0xffff
    ori     $t1, 0xfff8
    and     $t0, $t0, $t1
    ori     $t0, 0x3
    mtc0    $t0, $16, 0
    la      $t0, $cache_on      # Run the remainder of the code from the i-cache
    lui     $t8, 0xdfff
    ori     $t8, 0xffff
    and     $t0, $t0, $t8
    jr      $t0
    nop
$cache_on:
    la      $s2, $procedureA    # Run procedure A once in kseg0 to cache it
    lui     $t0, 0xdfff         #   (Clear bit 29 to change the address from kseg1 to kseg0)
    ori     $t0, 0xffff
    and     $s2, $s2, $t0
    jalr    $s2
    nop
    addiu   $v1, $s7, -123      # Sanity check the call result
    sltiu   $v0, $v1, 1
    la      $s3, $procedureB    # Compute the physical tag of procedureB
    lui     $t0, 0xdfff
    ori     $t0, 0xffff
    and     $s3, $s3, $t0       # kseg0 address of procedureB
    lui     $t0, 0x1fff
    ori     $t0, 0xffff
    and     $t1, $t0, $s3       # Physical address of procedureB
    srl     $t2, $t1, 1         # TagLo format is {1'b0, 23'b(Tag), 2'b(State), 6'b0}
    lui     $t0, 0xffff
    ori     $t0, 0xff00
    and     $t2, $t2, $t0
    ori     $t2, 0x00c0         # If any bit of State (TagLo[7:6]) is high => valid
    mtc0    $t2, $28, 0         # Send the tag to TagLo and zero TagHi
    mtc0    $0,  $29, 0
    lui     $t1, 0x8000         # Change the tag for procedureA to procedureB
    cache   0x8, 0x1aa0($t1)    # 0x8 <=> {010, 00} <=> {IndexStoreTag, L1-ICache}
                                # NOTE: Assumes the data is in set A! This could break.
    nop                         # Inject NOPs to prevent the next jump target from entering
    nop                         #   the pipeline and being 'prefetched' into the cache
    nop                         #   before the 'cache' operation retires
    jalr    $s3                 # Call procedureB, which should actually be procedureA
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
$procedureA:
    li      $s7, 123            # The arbitrary number 123 indicates the instruction is unchanged
    jr      $ra
    nop

    .balign 4096
    .skip 0xaa0, 0
$procedureB:
    li      $s7, 321
    jr      $ra
    nop

    .end test

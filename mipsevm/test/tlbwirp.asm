###############################################################################
# File         : tlbwirp.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'tlbwi' 'tlbr' and 'tlbp' instructions.
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

    ori     $t0, $0, 4          # Reserve (wire) 3 TLB entries
    mtc0    $t0, $6, 0
    ori     $t1, $0, 2          # Set the TLB index to 2 (third entry)
    mtc0    $t1, $0, 0
    lui     $t2, 0x0200         # Set the PFN to 2GB, cacheable, dirty, valid,
    ori     $t2, 0x003f         #  for EntryLo0/EntryLo1
    mtc0    $t2, $2, 0
    mtc0    $t2, $3, 0
    lui     $t3, 0x0001         # Set the page size to 64KB (0xf) in PageMask
    ori     $t3, 0xe000
    mtc0    $t3, $5, 0
    ori     $t4, $0, 100        # Set VPN2 to map the first 64-KiB page. Set ASID to 100
    mtc0    $t4, $10, 0
    tlbwi                       # Write Index
    mtc0    $0, $2, 0           # Clear EntryLo0,EntryLo1,EntryHi,PageMask
    mtc0    $0, $3, 0
    mtc0    $0, $5, 0
    mtc0    $0, $10, 0
    tlbr                        # Read TLB index 2
    mfc0    $s2, $2, 0          # (EntryLo0)
    mfc0    $s3, $3, 0          # (EntryLo1)
    mfc0    $s4, $5, 0          # (PageMask)
    mfc0    $s5, $10, 0         # (EntryHi)
    subu    $v1, $t2, $s2       # Validate read
    sltiu   $v0, $v1, 1
    subu    $v1, $t2, $s3
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t3, $s4
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t4, $s5
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    mtc0    $0, $0, 0           # Clear Index for tlbp
    tlbp
    mfc0    $s6, $0, 0          # (Index)
    subu    $v1, $t1, $s6       # Verify tlbp hit
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    lui     $t5, 0xffff         # Set a bogus value to EntryHi
    ori     $t5, 0xffff
    mtc0    $t5, $10, 0
    tlbp
    lui     $t6, 0x8000         # Verify tlbp miss
    mfc0    $s7, $10, 0
    and     $s7, $s7, $t6
    subu    $v1, $t6, $s7
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

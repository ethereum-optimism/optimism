###############################################################################
# File         : tlbwr.asm
# Project      : MIPS32 MUX
# Author:      : Grant Ayers (ayers@cs.stanford.edu)
#
# Standards/Formatting:
#   MIPS gas, soft tab, 80 column
#
# Description:
#   Test the functionality of the 'tlbwr' 'tlbr' and 'tlbp' instructions.
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

    ori     $t0, $0, 8          # Reserve (wire) 8 TLB entries
    mtc0    $t0, $6, 0
    ori     $t1, $0, 3          # Set the TLB index to 2 (third entry)
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
    tlbwr                       # Write Random
    mtc0    $0, $2, 0           # Clear EntryLo0,EntryLo1,PageMask
    mtc0    $0, $3, 0
    mtc0    $0, $5, 0
    tlbp                        # Verify TLB hit
    mfc0    $s2, $0, 0
    srl     $v1, $s2, 31
    sltiu   $v0, $v1, 1
    sltiu   $v1, $s2, 16        # Verify index is in bounds     (idx<16)  (7<idx)
    and     $v0, $v0, $v1
    ori     $t5, $0, 7
    sltu    $v1, $t5, $s2
    and     $v0, $v0, $v1
    mtc0    $0, $10, 0          # Verify the index data
    tlbr
    mfc0    $s3, $2, 0          # (EntryLo0)
    mfc0    $s4, $3, 0          # (EntryLo1)
    mfc0    $s5, $5, 0          # (PageMask)
    mfc0    $s6, $10, 0         # (EntryHi)
    subu    $v1, $t2, $s3       # Validate read
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t2, $s4
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t3, $s5
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1
    subu    $v1, $t4, $s6
    sltiu   $v1, $v1, 1
    and     $v0, $v0, $v1

    #### Test code end ####

    sw      $v0, 8($s0)         # Set the test result
    sw      $s1, 4($s0)         # Set 'done'

$done:
    jr      $ra
    nop

    .end test

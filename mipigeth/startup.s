    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test
test:

lui     $sp, 0x7fff
ori     $sp, 0xd000

# http://articles.manugarg.com/aboutelfauxiliaryvectors.html
# _AT_PAGESZ = 6
ori $t0, $0, 6
sw $t0, 0xC($sp)
ori $t0, $0, 0x1000
sw $t0, 0x10($sp)

lw $ra, dat($0)
jr $ra
nop

dat:

.end test

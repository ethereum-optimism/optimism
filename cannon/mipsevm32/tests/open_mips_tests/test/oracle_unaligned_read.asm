.section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test

# load hash at 0x30001000
# 0x47173285 a8d7341e 5e972fc6 77286384 f802f8ef 42a5ec5f 03bbfa25 4cb01fad = keccak("hello world")
# 0x02173285 a8d7341e 5e972fc6 77286384 f802f8ef 42a5ec5f 03bbfa25 4cb01fad = keccak("hello world").key
test:
  lui $s0, 0x3000
  ori $s0, 0x1000

  lui $t0, 0x0217
  ori $t0, 0x3285
  sw $t0, 0($s0)
  lui $t0, 0xa8d7
  ori $t0, 0x341e
  sw $t0, 4($s0)
  lui $t0, 0x5e97
  ori $t0, 0x2fc6
  sw $t0, 8($s0)
  lui $t0, 0x7728
  ori $t0, 0x6384
  sw $t0, 0xc($s0)
  lui $t0, 0xf802
  ori $t0, 0xf8ef
  sw $t0, 0x10($s0)
  lui $t0, 0x42a5
  ori $t0, 0xec5f
  sw $t0, 0x14($s0)
  lui $t0, 0x03bb
  ori $t0, 0xfa25
  sw $t0, 0x18($s0)
  lui $t0, 0x4cb0
  ori $t0, 0x1fad
  sw $t0, 0x1c($s0)

# preimage request - write(fdPreimageWrite, preimageData, 32)
  li $a0, 6
  li $a1, 0x30001000
  li $t0, 8
  li $a2, 4
$writeloop:
  li $v0, 4004
  syscall
  addiu $a1, $a1, 4
  addiu $t0, $t0, -1
  bnez $t0, $writeloop
  nop

# preimage response to 0x30002000 - read(fdPreimageRead, addr, count)
# read preimage length to unaligned addr. This will read only up to the nearest aligned byte so we have to read again.
  li $a0, 5
  li $a1, 0x31000001
  li $a2, 4
  li $v0, 4003
  syscall
  li $a1, 0x31000004
  li $v0, 4003
  syscall
  li $a1, 0x31000008
  li $a2, 1
  li $v0, 4003
  syscall
# read the preimage data
  li $a1, 0x31000009
  li $t0, 11
$readloop:
  li $v0, 4003
  li $a2, 4
  syscall
  addu $a1, $a1, $v0
  subu $t0, $t0, $v0
  bnez $t0, $readloop
  nop

# length at 0x31000001. We also check that the lower 32 bits are zero
  li $s1, 0x31000001
  lb $t0, 0($s1)
  lb $t2, 1($s1)
  sll $t2, $t2, 8
  or $t0, $t0, $t2
  lb $t2, 2($s1)
  sll $t2, $t2, 16
  or $t0, $t0, $t2
# assert len[0:3] == 0
  sltiu $v0, $t0, 1

# assert len[4:8] == 0
  addiu $s1, $s1, 3
  lw $t1, 0($s1)
  sltiu $v1, $t1, 1
  and $v0, $v0, $v1

# assert len[8:9] == 11
  addiu $s1, $s1, 4
  lb $t2, 0($s1)
  li $t4, 11
  subu $t5, $t2, $t4
  sltiu $v1, $t5, 1
  and $v0, $v0, $v1

# data at 0x31000009
  addiu $s1, $s1, 1
  lb $t0, 0($s1)
  lb $t2, 1($s1)
  sll $t0, $t0, 8
  or $t0, $t0, $t2
  lb $t2, 2($s1)
  sll $t0, $t0, 8
  or $t0, $t0, $t2
  lb $t2, 3($s1)
  sll $t0, $t0, 8
  or $t0, $t0, $t2

  #lw $t0, 0($s1)
  lui $t4, 0x6865
  ori $t4, 0x6c6c
  subu $t5, $t0, $t4
  sltiu $v1, $t5, 1

  and $v0, $v0, $v1

# save results
  lui     $s0, 0xbfff         # Load the base address 0xbffffff0
  ori     $s0, 0xfff0
  ori     $s1, $0, 1          # Prepare the 'done' status

  sw      $v0, 8($s0)         # Set the test result
  sw      $s1, 4($s0)         # Set 'done'

$done:
  jr      $ra
  nop

  .end test

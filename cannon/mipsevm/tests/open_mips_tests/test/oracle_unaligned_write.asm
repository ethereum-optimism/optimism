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
# create stuffed buffer containing the first byte of the hash - [garbage, hash[0], garbage]
  lui $s1, 0x3200
  ori $s1, 0x0000
  lui $t0, 0xFFFF
  ori $t0, 0x02FF
  sw $t0, 0($s1)

# initial unaligned write for stuffed buffer
  li $a0, 6
  li $a1, 0x32000002
  li $a2, 1
  li $v0, 4004
  syscall

# write 3 bytes for realignment
  li $a0, 6
  li $a1, 0x30001001
  li $a2, 3
  li $v0, 4004
  syscall

  li $a0, 6
  li $a1, 0x30001004
  li $t0, 7
  li $a2, 4
$writeloop:
  li $v0, 4004
  syscall
  addiu $a1, $a1, 4
  addiu $t0, $t0, -1
  bnez $t0, $writeloop
  nop

# preimage response to 0x30002000 - read(fdPreimageRead, addr, count)
# read preimage length
  li $a0, 5
  li $a1, 0x31000000
  li $a2, 4
  li $v0, 4003
  syscall
  li $a1, 0x31000004
  li $v0, 4003
  syscall
# read the preimage data
  li $a1, 0x31000008
  li $t0, 3
$readloop:
  li $v0, 4003
  syscall
  addiu $a1, $a1, 4
  addiu $t0, $t0, -1
  bnez $t0, $readloop
  nop

# length at 0x31000000. We also check that the lower 32 bits are zero
  lui $s1, 0x3100
  lw $t0, 0($s1)
  sltiu $t6, $t0, 1
  li $s1, 0x31000004
  lw $t0, 0($s1)
# should be len("hello world") == 11
  li $t4, 11
  subu $t5, $t0, $t4
  sltiu $v0, $t5, 1
  and $v0, $v0, $t6

# data at 0x31000008
  lw $t0, 4($s1)
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

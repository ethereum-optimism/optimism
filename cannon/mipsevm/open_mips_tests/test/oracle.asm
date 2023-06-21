    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test

# load hash at 0x30001000
# 0x47173285 a8d7341e 5e972fc6 77286384 f802f8ef 42a5ec5f 03bbfa25 4cb01fad = "hello world"
test:
  lui $s0, 0x3000
  ori $s0, 0x1000

  lui $t0, 0x4717
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

# syscall 4020 to trigger
  li $v0, 4020
  syscall

# length at 0x31000000
  lui $s1, 0x3100
  lw $t0, 0($s1)

# should be len("hello world") == 11
  li $t4, 11
  subu $t5, $t0, $t4
  sltiu $v0, $t5, 1 

# data at 0x31000004
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

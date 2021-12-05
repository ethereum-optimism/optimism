    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test

# load hash at 0x30001000
# 0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad = "hello world"
test:
  lui $s0, 0x3000
  ori $s0, 0x1000
  li $t0, 0x4717
  sh $t0, 0($s0)
  li $t0, 0x3285
  sh $t0, 1($s0)

# length at 0x31000000
  lui $s1, 0x3100
  lw $s1, 0($t0)

# should be 9
  li $t4, 9
  subu $t5, $t0, $t4
  sltiu $v0, $t5, 1 

# data at 0x31000004

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

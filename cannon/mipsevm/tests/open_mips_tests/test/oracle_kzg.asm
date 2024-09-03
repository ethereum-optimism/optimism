.section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test

# load hash at 0x30001000
# requiredGas is 50_000
# point evaluation precompile input (requiredGas ++ precompileInput) - 000000000000c35001e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a
# 0x3efd5c3c 1c555298 0c63aee5 4570c276 cbff7532 796b4d75 3132d51a 6bedf0c6 = keccak(address(0xa) ++ required_gas ++ precompile_input)
# 0x06fd5c3c 1c555298 0c63aee5 4570c276 cbff7532 796b4d75 3132d51a 6bedf0c6 = keccak(address(0xa) ++ required_gas ++ precompile_input).key (precompile)
test:
  lui $s0, 0x3000
  ori $s0, 0x1000

  lui $t0, 0x06fd
  ori $t0, 0x5c3c
  sw $t0, 0($s0)
  lui $t0, 0x1c55
  ori $t0, 0x5298
  sw $t0, 4($s0)
  lui $t0, 0x0c63
  ori $t0, 0xaee5
  sw $t0, 8($s0)
  lui $t0, 0x4570
  ori $t0, 0xc276
  sw $t0, 0xc($s0)
  lui $t0, 0xcbff
  ori $t0, 0x7532
  sw $t0, 0x10($s0)
  lui $t0, 0x796b
  ori $t0, 0x4d75
  sw $t0, 0x14($s0)
  lui $t0, 0x3132
  ori $t0, 0xd51a
  sw $t0, 0x18($s0)
  lui $t0, 0x6bed
  ori $t0, 0xf0c6
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
# read preimage length
  li $a0, 5
  li $a1, 0x31000000
  li $a2, 4
  li $v0, 4003
  syscall
  li $a1, 0x31000004
  li $v0, 4003
  syscall
# read the 1 byte precompile status and 3 bytes of return data
  li $a1, 0x31000008
  li $v0, 4003
  syscall
  nop

# length at 0x31000000. We also check that the lower 32 bits are zero
  lui $s1, 0x3100
  lw $t0, 0($s1)
  sltiu $t6, $t0, 1
  li $s1, 0x31000004
  lw $t0, 0($s1)
# should be 1 + len(blobPrecompileReturnValue) = 65
  li $t4, 65
  subu $t5, $t0, $t4
  sltiu $v0, $t5, 1
  and $v0, $v0, $t6

# data at 0x31000008
# first byte is 01 status. Next 3 bytes are 0
  lw $t0, 4($s1)
  lui $t4, 0x0100
  ori $t4, 0x0000
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

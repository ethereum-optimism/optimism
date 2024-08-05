# Utility for generating assembly test files for the MIPS instructions that panic when specific
# fields in the encoded instruction are nonzero. Using a script to do this is much less prone to
# human error than writing these tests by hand.

import os

# Format is (opcode, funct, shamt, rd, rt, rs, field_to_make_nonzero)
tests = {
    'ADD': [('000000', '100000', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'ADDU': [('000000', '100001', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'AND': [('000000', '100100', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'DIV': [
      ('000000', '011010', '00001', '01010', '00000', '01001', 'rd'),  # RD nonzero
      ('000000', '011010', '00001', '00000', '01010', '01001', 'shamt'),  # SHAMT nonzero
    ],
    'DIVU': [
      ('000000', '011011', '00001', '01010', '00000', '01001', 'rd'),  # RD nonzero
      ('000000', '011011', '00001', '00000', '01010', '01001', 'shamt'),  # SHAMT nonzero
    ],
    'JALR': [
        ('000000', '001001', '00000', '01011', '01001', '01010', 'rt'),  # RT nonzero
        ('000000', '001001', '00001', '01011', '01001', '00000', 'shamt'),  # SHAMT nonzero
    ],
    'JR': [
        ('000000', '001000', '00000', '01011', '01001', '00000', 'rd'),  # RD nonzero
        ('000000', '001000', '00000', '00000', '01001', '01010', 'rt'),  # RT nonzero
        ('000000', '001000', '00001', '00000', '01001', '00000', 'shamt'),  # SHAMT nonzero
    ],
    'LUI': [('001111', '00000', '01010', '00000', '00000', '01001', 'rs')],  # RS nonzero
    'MFHI': [
        ('000000', '010000', '00000', '01011', '01010', '00000', 'rt'),  # RT nonzero
        ('000000', '010000', '00001', '01011', '00000', '00000', 'shamt'),  # SHAMT nonzero
        ('000000', '010000', '00000', '01011', '00000', '01001', 'rs'),  # RS nonzero
    ],
    'MFLO': [
        ('000000', '010010', '00000', '01011', '01010', '00000', 'rt'),  # RT nonzero
        ('000000', '010010', '00001', '01011', '00000', '00000', 'shamt'),  # SHAMT nonzero
        ('000000', '010010', '00000', '01011', '00000', '01001', 'rs'),  # RS nonzero
    ],
    'MTHI': [
        ('000000', '010001', '00000', '01011', '01010', '00000', 'rd'),  # RD nonzero
        ('000000', '010001', '00000', '01001', '01010', '00000', 'rt'),  # RT nonzero
        ('000000', '010001', '00001', '01001', '00000', '00000', 'shamt'),  # SHAMT nonzero
    ],
    'MTLO': [
        ('000000', '010011', '00000', '01011', '01010', '00000', 'rd'),  # RD nonzero
        ('000000', '010011', '00000', '01001', '01010', '00000', 'rt'),  # RT nonzero
        ('000000', '010011', '00001', '01001', '00000', '00000', 'shamt'),  # SHAMT nonzero
    ],
    'MULT': [
        ('000000', '011000', '00000', '01011', '01010', '00000', 'rd'),  # RD nonzero
        ('000000', '011000', '00001', '00000', '01010', '01001', 'shamt'),  # SHAMT nonzero
    ],
    'MULTU': [
        ('000000', '011001', '00000', '01011', '01010', '00000', 'rd'),  # RD nonzero
        ('000000', '011001', '00001', '00000', '01010', '01001', 'shamt'),  # SHAMT nonzero
    ],
    'NOR': [('000000', '100111', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'OR': [('000000', '100101', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SLL': [('000000', '000000', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SLT': [('000000', '101010', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SLTU': [('000000', '101011', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SRA': [
        ('000000', '000011', '00001', '01011', '01010', '01001', 'shamt'),  # SHAMT nonzero
        ('000000', '000011', '00000', '01011', '01010', '01001', 'rs'),  # RS nonzero
    ],
    'SRL': [
        ('000000', '000010', '00001', '01011', '01010', '01001', 'shamt'),  # SHAMT nonzero
        ('000000', '000010', '00000', '01011', '01010', '01001', 'rs'),  # RS nonzero
    ],
    'SUB': [('000000', '100010', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SUBU': [('000000', '100011', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
    'SYNC': [('000000', '001111', '00001', '00000', '00000', '00000', 'shamt')],  # SHAMT nonzero
    'XOR': [('000000', '100110', '00001', '01011', '01010', '01001', 'shamt')],  # SHAMT nonzero
}

def pad_to_length(binary_str, length):
    return binary_str.zfill(length)

def generate_instruction(opcode, rs, rt, rd, shamt, funct):
    binary_instruction = (
        pad_to_length(opcode, 6) +
        pad_to_length(rs, 5) +
        pad_to_length(rt, 5) +
        pad_to_length(rd, 5) +
        pad_to_length(shamt, 5) +
        pad_to_length(funct, 6)
    )
    hex_instruction = hex(int(binary_instruction, 2))[2:].zfill(8)
    return '0x' + hex_instruction

def create_test_file(instruction, field, hex_instruction):
    return f"""###############################################################################
# Description:
#   Tests that the '{instruction.lower()}' instruction panics when the {field} field is nonzero.
#
###############################################################################

    .section .test, "x"
    .balign 4
    .set    noreorder
    .global test
    .ent    test
test:
    lui     $s0, 0xbfff
    ori     $s0, 0xfff0
    ori     $s1, $0, 1

    # Invalid {instruction} (nonzero {field} field)
    .word {hex_instruction}

    sw      $zero, 8($s0)
    sw      $s1, 4($s0)

$done:
    jr      $ra
    nop

    .end test
"""

def write_test_file(filename, content):
    os.makedirs('./test', exist_ok=True)
    with open(f'./test/{filename}', 'w') as file:
        file.write(content)

for instr, cases in tests.items():
    for params in cases:
        opcode, funct, shamt, rd, rt, rs, field = params
        hex_instr = generate_instruction(opcode, rs, rt, rd, shamt, funct)
        test_content = create_test_file(instr, field, hex_instr)
        filename = f"{instr.lower()}_nonzero_{field}_panic.asm"
        write_test_file(filename, test_content)

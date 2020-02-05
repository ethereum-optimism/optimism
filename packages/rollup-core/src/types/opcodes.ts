export interface EVMOpcode {
  name: string
  code: Buffer
  programBytesConsumed: number
}

export interface EVMOpcodeAndBytes {
  opcode: EVMOpcode
  consumedBytes: Buffer
}

export type EVMBytecode = EVMOpcodeAndBytes[]

export class Opcode {
  public static readonly STOP: EVMOpcode = {
    code: Buffer.from('00', 'hex'),
    name: 'STOP',
    programBytesConsumed: 0,
  }
  public static readonly ADD: EVMOpcode = {
    code: Buffer.from('01', 'hex'),
    name: 'ADD',
    programBytesConsumed: 0,
  }
  public static readonly MUL: EVMOpcode = {
    code: Buffer.from('02', 'hex'),
    name: 'MUL',
    programBytesConsumed: 0,
  }
  public static readonly SUB: EVMOpcode = {
    code: Buffer.from('03', 'hex'),
    name: 'SUB',
    programBytesConsumed: 0,
  }
  public static readonly DIV: EVMOpcode = {
    code: Buffer.from('04', 'hex'),
    name: 'DIV',
    programBytesConsumed: 0,
  }
  public static readonly SDIV: EVMOpcode = {
    code: Buffer.from('05', 'hex'),
    name: 'SDIV',
    programBytesConsumed: 0,
  }
  public static readonly MOD: EVMOpcode = {
    code: Buffer.from('06', 'hex'),
    name: 'MOD',
    programBytesConsumed: 0,
  }
  public static readonly SMOD: EVMOpcode = {
    code: Buffer.from('07', 'hex'),
    name: 'SMOD',
    programBytesConsumed: 0,
  }
  public static readonly ADDMOD: EVMOpcode = {
    code: Buffer.from('08', 'hex'),
    name: 'ADDMOD',
    programBytesConsumed: 0,
  }
  public static readonly MULMOD: EVMOpcode = {
    code: Buffer.from('09', 'hex'),
    name: 'MULMOD',
    programBytesConsumed: 0,
  }
  public static readonly EXP: EVMOpcode = {
    code: Buffer.from('0a', 'hex'),
    name: 'EXP',
    programBytesConsumed: 0,
  }
  public static readonly SIGNEXTEND: EVMOpcode = {
    code: Buffer.from('0b', 'hex'),
    name: 'SIGNEXTEND',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly LT: EVMOpcode = {
    code: Buffer.from('10', 'hex'),
    name: 'LT',
    programBytesConsumed: 0,
  }
  public static readonly GT: EVMOpcode = {
    code: Buffer.from('11', 'hex'),
    name: 'GT',
    programBytesConsumed: 0,
  }
  public static readonly SLT: EVMOpcode = {
    code: Buffer.from('12', 'hex'),
    name: 'SLT',
    programBytesConsumed: 0,
  }
  public static readonly SGT: EVMOpcode = {
    code: Buffer.from('13', 'hex'),
    name: 'SGT',
    programBytesConsumed: 0,
  }
  public static readonly EQ: EVMOpcode = {
    code: Buffer.from('14', 'hex'),
    name: 'EQ',
    programBytesConsumed: 0,
  }
  public static readonly ISZERO: EVMOpcode = {
    code: Buffer.from('15', 'hex'),
    name: 'ISZERO',
    programBytesConsumed: 0,
  }
  public static readonly AND: EVMOpcode = {
    code: Buffer.from('16', 'hex'),
    name: 'AND',
    programBytesConsumed: 0,
  }
  public static readonly OR: EVMOpcode = {
    code: Buffer.from('17', 'hex'),
    name: 'OR',
    programBytesConsumed: 0,
  }
  public static readonly XOR: EVMOpcode = {
    code: Buffer.from('18', 'hex'),
    name: 'XOR',
    programBytesConsumed: 0,
  }
  public static readonly NOT: EVMOpcode = {
    code: Buffer.from('19', 'hex'),
    name: 'NOT',
    programBytesConsumed: 0,
  }
  public static readonly BYTE: EVMOpcode = {
    code: Buffer.from('1a', 'hex'),
    name: 'BYTE',
    programBytesConsumed: 0,
  }
  public static readonly SHL: EVMOpcode = {
    code: Buffer.from('1b', 'hex'),
    name: 'SHL',
    programBytesConsumed: 0,
  }
  public static readonly SHR: EVMOpcode = {
    code: Buffer.from('1c', 'hex'),
    name: 'SHR',
    programBytesConsumed: 0,
  }
  public static readonly SAR: EVMOpcode = {
    code: Buffer.from('1d', 'hex'),
    name: 'SAR',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly SHA3: EVMOpcode = {
    code: Buffer.from('20', 'hex'),
    name: 'SHA3',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly ADDRESS: EVMOpcode = {
    code: Buffer.from('30', 'hex'),
    name: 'ADDRESS',
    programBytesConsumed: 0,
  }
  public static readonly BALANCE: EVMOpcode = {
    code: Buffer.from('31', 'hex'),
    name: 'BALANCE',
    programBytesConsumed: 0,
  }
  public static readonly ORIGIN: EVMOpcode = {
    code: Buffer.from('32', 'hex'),
    name: 'ORIGIN',
    programBytesConsumed: 0,
  }
  public static readonly CALLER: EVMOpcode = {
    code: Buffer.from('33', 'hex'),
    name: 'CALLER',
    programBytesConsumed: 0,
  }
  public static readonly CALLVALUE: EVMOpcode = {
    code: Buffer.from('34', 'hex'),
    name: 'CALLVALUE',
    programBytesConsumed: 0,
  }
  public static readonly CALLDATALOAD: EVMOpcode = {
    code: Buffer.from('35', 'hex'),
    name: 'CALLDATALOAD',
    programBytesConsumed: 0,
  }
  public static readonly CALLDATASIZE: EVMOpcode = {
    code: Buffer.from('36', 'hex'),
    name: 'CALLDATASIZE',
    programBytesConsumed: 0,
  }
  public static readonly CALLDATACOPY: EVMOpcode = {
    code: Buffer.from('37', 'hex'),
    name: 'CALLDATACOPY',
    programBytesConsumed: 0,
  }
  public static readonly CODESIZE: EVMOpcode = {
    code: Buffer.from('38', 'hex'),
    name: 'CODESIZE',
    programBytesConsumed: 0,
  }
  public static readonly CODECOPY: EVMOpcode = {
    code: Buffer.from('39', 'hex'),
    name: 'CODECOPY',
    programBytesConsumed: 0,
  }
  public static readonly GASPRICE: EVMOpcode = {
    code: Buffer.from('3a', 'hex'),
    name: 'GASPRICE',
    programBytesConsumed: 0,
  }
  public static readonly EXTCODESIZE: EVMOpcode = {
    code: Buffer.from('3b', 'hex'),
    name: 'EXTCODESIZE',
    programBytesConsumed: 0,
  }
  public static readonly EXTCODECOPY: EVMOpcode = {
    code: Buffer.from('3c', 'hex'),
    name: 'EXTCODECOPY',
    programBytesConsumed: 0,
  }
  public static readonly RETURNDATASIZE: EVMOpcode = {
    code: Buffer.from('3d', 'hex'),
    name: 'RETURNDATASIZE',
    programBytesConsumed: 0,
  }
  public static readonly RETURNDATACOPY: EVMOpcode = {
    code: Buffer.from('3e', 'hex'),
    name: 'RETURNDATACOPY',
    programBytesConsumed: 0,
  }
  public static readonly EXTCODEHASH: EVMOpcode = {
    code: Buffer.from('3f', 'hex'),
    name: 'EXTCODEHASH',
    programBytesConsumed: 0,
  }
  public static readonly BLOCKHASH: EVMOpcode = {
    code: Buffer.from('40', 'hex'),
    name: 'BLOCKHASH',
    programBytesConsumed: 0,
  }
  public static readonly COINBASE: EVMOpcode = {
    code: Buffer.from('41', 'hex'),
    name: 'COINBASE',
    programBytesConsumed: 0,
  }
  public static readonly TIMESTAMP: EVMOpcode = {
    code: Buffer.from('42', 'hex'),
    name: 'TIMESTAMP',
    programBytesConsumed: 0,
  }
  public static readonly NUMBER: EVMOpcode = {
    code: Buffer.from('43', 'hex'),
    name: 'NUMBER',
    programBytesConsumed: 0,
  }
  public static readonly DIFFICULTY: EVMOpcode = {
    code: Buffer.from('44', 'hex'),
    name: 'DIFFICULTY',
    programBytesConsumed: 0,
  }
  public static readonly GASLIMIT: EVMOpcode = {
    code: Buffer.from('45', 'hex'),
    name: 'GASLIMIT',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly POP: EVMOpcode = {
    code: Buffer.from('50', 'hex'),
    name: 'POP',
    programBytesConsumed: 0,
  }
  public static readonly MLOAD: EVMOpcode = {
    code: Buffer.from('51', 'hex'),
    name: 'MLOAD',
    programBytesConsumed: 0,
  }
  public static readonly MSTORE: EVMOpcode = {
    code: Buffer.from('52', 'hex'),
    name: 'MSTORE',
    programBytesConsumed: 0,
  }
  public static readonly MSTORE8: EVMOpcode = {
    code: Buffer.from('53', 'hex'),
    name: 'MSTORE8',
    programBytesConsumed: 0,
  }
  public static readonly SLOAD: EVMOpcode = {
    code: Buffer.from('54', 'hex'),
    name: 'SLOAD',
    programBytesConsumed: 0,
  }
  public static readonly SSTORE: EVMOpcode = {
    code: Buffer.from('55', 'hex'),
    name: 'SSTORE',
    programBytesConsumed: 0,
  }
  public static readonly JUMP: EVMOpcode = {
    code: Buffer.from('56', 'hex'),
    name: 'JUMP',
    programBytesConsumed: 0,
  }
  public static readonly JUMPI: EVMOpcode = {
    code: Buffer.from('57', 'hex'),
    name: 'JUMPI',
    programBytesConsumed: 0,
  }
  public static readonly PC: EVMOpcode = {
    code: Buffer.from('58', 'hex'),
    name: 'PC',
    programBytesConsumed: 0,
  }
  public static readonly MSIZE: EVMOpcode = {
    code: Buffer.from('59', 'hex'),
    name: 'MSIZE',
    programBytesConsumed: 0,
  }
  public static readonly GAS: EVMOpcode = {
    code: Buffer.from('5a', 'hex'),
    name: 'GAS',
    programBytesConsumed: 0,
  }
  public static readonly JUMPDEST: EVMOpcode = {
    code: Buffer.from('5b', 'hex'),
    name: 'JUMPDEST',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly PUSH1: EVMOpcode = {
    code: Buffer.from('60', 'hex'),
    name: 'PUSH1',
    programBytesConsumed: 1,
  }
  public static readonly PUSH2: EVMOpcode = {
    code: Buffer.from('61', 'hex'),
    name: 'PUSH2',
    programBytesConsumed: 2,
  }
  public static readonly PUSH3: EVMOpcode = {
    code: Buffer.from('62', 'hex'),
    name: 'PUSH3',
    programBytesConsumed: 3,
  }
  public static readonly PUSH4: EVMOpcode = {
    code: Buffer.from('63', 'hex'),
    name: 'PUSH4',
    programBytesConsumed: 4,
  }
  public static readonly PUSH5: EVMOpcode = {
    code: Buffer.from('64', 'hex'),
    name: 'PUSH5',
    programBytesConsumed: 5,
  }
  public static readonly PUSH6: EVMOpcode = {
    code: Buffer.from('65', 'hex'),
    name: 'PUSH6',
    programBytesConsumed: 6,
  }
  public static readonly PUSH7: EVMOpcode = {
    code: Buffer.from('66', 'hex'),
    name: 'PUSH7',
    programBytesConsumed: 7,
  }
  public static readonly PUSH8: EVMOpcode = {
    code: Buffer.from('67', 'hex'),
    name: 'PUSH8',
    programBytesConsumed: 8,
  }
  public static readonly PUSH9: EVMOpcode = {
    code: Buffer.from('68', 'hex'),
    name: 'PUSH9',
    programBytesConsumed: 9,
  }
  public static readonly PUSH10: EVMOpcode = {
    code: Buffer.from('69', 'hex'),
    name: 'PUSH10',
    programBytesConsumed: 10,
  }
  public static readonly PUSH11: EVMOpcode = {
    code: Buffer.from('6a', 'hex'),
    name: 'PUSH11',
    programBytesConsumed: 11,
  }
  public static readonly PUSH12: EVMOpcode = {
    code: Buffer.from('6b', 'hex'),
    name: 'PUSH12',
    programBytesConsumed: 12,
  }
  public static readonly PUSH13: EVMOpcode = {
    code: Buffer.from('6c', 'hex'),
    name: 'PUSH13',
    programBytesConsumed: 13,
  }
  public static readonly PUSH14: EVMOpcode = {
    code: Buffer.from('6d', 'hex'),
    name: 'PUSH14',
    programBytesConsumed: 14,
  }
  public static readonly PUSH15: EVMOpcode = {
    code: Buffer.from('6e', 'hex'),
    name: 'PUSH15',
    programBytesConsumed: 15,
  }
  public static readonly PUSH16: EVMOpcode = {
    code: Buffer.from('6f', 'hex'),
    name: 'PUSH16',
    programBytesConsumed: 16,
  }
  public static readonly PUSH17: EVMOpcode = {
    code: Buffer.from('70', 'hex'),
    name: 'PUSH17',
    programBytesConsumed: 17,
  }
  public static readonly PUSH18: EVMOpcode = {
    code: Buffer.from('71', 'hex'),
    name: 'PUSH18',
    programBytesConsumed: 18,
  }
  public static readonly PUSH19: EVMOpcode = {
    code: Buffer.from('72', 'hex'),
    name: 'PUSH19',
    programBytesConsumed: 19,
  }
  public static readonly PUSH20: EVMOpcode = {
    code: Buffer.from('73', 'hex'),
    name: 'PUSH20',
    programBytesConsumed: 20,
  }
  public static readonly PUSH21: EVMOpcode = {
    code: Buffer.from('74', 'hex'),
    name: 'PUSH21',
    programBytesConsumed: 21,
  }
  public static readonly PUSH22: EVMOpcode = {
    code: Buffer.from('75', 'hex'),
    name: 'PUSH22',
    programBytesConsumed: 22,
  }
  public static readonly PUSH23: EVMOpcode = {
    code: Buffer.from('76', 'hex'),
    name: 'PUSH23',
    programBytesConsumed: 23,
  }
  public static readonly PUSH24: EVMOpcode = {
    code: Buffer.from('77', 'hex'),
    name: 'PUSH24',
    programBytesConsumed: 24,
  }
  public static readonly PUSH25: EVMOpcode = {
    code: Buffer.from('78', 'hex'),
    name: 'PUSH25',
    programBytesConsumed: 25,
  }
  public static readonly PUSH26: EVMOpcode = {
    code: Buffer.from('79', 'hex'),
    name: 'PUSH26',
    programBytesConsumed: 26,
  }
  public static readonly PUSH27: EVMOpcode = {
    code: Buffer.from('7a', 'hex'),
    name: 'PUSH27',
    programBytesConsumed: 27,
  }
  public static readonly PUSH28: EVMOpcode = {
    code: Buffer.from('7b', 'hex'),
    name: 'PUSH28',
    programBytesConsumed: 28,
  }
  public static readonly PUSH29: EVMOpcode = {
    code: Buffer.from('7c', 'hex'),
    name: 'PUSH29',
    programBytesConsumed: 29,
  }
  public static readonly PUSH30: EVMOpcode = {
    code: Buffer.from('7d', 'hex'),
    name: 'PUSH30',
    programBytesConsumed: 30,
  }
  public static readonly PUSH31: EVMOpcode = {
    code: Buffer.from('7e', 'hex'),
    name: 'PUSH31',
    programBytesConsumed: 31,
  }
  public static readonly PUSH32: EVMOpcode = {
    code: Buffer.from('7f', 'hex'),
    name: 'PUSH32',
    programBytesConsumed: 32,
  }

  public static readonly DUP1: EVMOpcode = {
    code: Buffer.from('80', 'hex'),
    name: 'DUP1',
    programBytesConsumed: 0,
  }
  public static readonly DUP2: EVMOpcode = {
    code: Buffer.from('81', 'hex'),
    name: 'DUP2',
    programBytesConsumed: 0,
  }
  public static readonly DUP3: EVMOpcode = {
    code: Buffer.from('82', 'hex'),
    name: 'DUP3',
    programBytesConsumed: 0,
  }
  public static readonly DUP4: EVMOpcode = {
    code: Buffer.from('83', 'hex'),
    name: 'DUP4',
    programBytesConsumed: 0,
  }
  public static readonly DUP5: EVMOpcode = {
    code: Buffer.from('84', 'hex'),
    name: 'DUP5',
    programBytesConsumed: 0,
  }
  public static readonly DUP6: EVMOpcode = {
    code: Buffer.from('85', 'hex'),
    name: 'DUP6',
    programBytesConsumed: 0,
  }
  public static readonly DUP7: EVMOpcode = {
    code: Buffer.from('86', 'hex'),
    name: 'DUP7',
    programBytesConsumed: 0,
  }
  public static readonly DUP8: EVMOpcode = {
    code: Buffer.from('87', 'hex'),
    name: 'DUP8',
    programBytesConsumed: 0,
  }
  public static readonly DUP9: EVMOpcode = {
    code: Buffer.from('88', 'hex'),
    name: 'DUP9',
    programBytesConsumed: 0,
  }
  public static readonly DUP10: EVMOpcode = {
    code: Buffer.from('89', 'hex'),
    name: 'DUP10',
    programBytesConsumed: 0,
  }
  public static readonly DUP11: EVMOpcode = {
    code: Buffer.from('8a', 'hex'),
    name: 'DUP11',
    programBytesConsumed: 0,
  }
  public static readonly DUP12: EVMOpcode = {
    code: Buffer.from('8b', 'hex'),
    name: 'DUP12',
    programBytesConsumed: 0,
  }
  public static readonly DUP13: EVMOpcode = {
    code: Buffer.from('8c', 'hex'),
    name: 'DUP13',
    programBytesConsumed: 0,
  }
  public static readonly DUP14: EVMOpcode = {
    code: Buffer.from('8d', 'hex'),
    name: 'DUP14',
    programBytesConsumed: 0,
  }
  public static readonly DUP15: EVMOpcode = {
    code: Buffer.from('8e', 'hex'),
    name: 'DUP15',
    programBytesConsumed: 0,
  }
  public static readonly DUP16: EVMOpcode = {
    code: Buffer.from('8f', 'hex'),
    name: 'DUP16',
    programBytesConsumed: 0,
  }

  public static readonly SWAP1: EVMOpcode = {
    code: Buffer.from('90', 'hex'),
    name: 'SWAP1',
    programBytesConsumed: 0,
  }
  public static readonly SWAP2: EVMOpcode = {
    code: Buffer.from('91', 'hex'),
    name: 'SWAP2',
    programBytesConsumed: 0,
  }
  public static readonly SWAP3: EVMOpcode = {
    code: Buffer.from('92', 'hex'),
    name: 'SWAP3',
    programBytesConsumed: 0,
  }
  public static readonly SWAP4: EVMOpcode = {
    code: Buffer.from('93', 'hex'),
    name: 'SWAP4',
    programBytesConsumed: 0,
  }
  public static readonly SWAP5: EVMOpcode = {
    code: Buffer.from('94', 'hex'),
    name: 'SWAP5',
    programBytesConsumed: 0,
  }
  public static readonly SWAP6: EVMOpcode = {
    code: Buffer.from('95', 'hex'),
    name: 'SWAP6',
    programBytesConsumed: 0,
  }
  public static readonly SWAP7: EVMOpcode = {
    code: Buffer.from('96', 'hex'),
    name: 'SWAP7',
    programBytesConsumed: 0,
  }
  public static readonly SWAP8: EVMOpcode = {
    code: Buffer.from('97', 'hex'),
    name: 'SWAP8',
    programBytesConsumed: 0,
  }
  public static readonly SWAP9: EVMOpcode = {
    code: Buffer.from('98', 'hex'),
    name: 'SWAP9',
    programBytesConsumed: 0,
  }
  public static readonly SWAP10: EVMOpcode = {
    code: Buffer.from('99', 'hex'),
    name: 'SWAP10',
    programBytesConsumed: 0,
  }
  public static readonly SWAP11: EVMOpcode = {
    code: Buffer.from('9a', 'hex'),
    name: 'SWAP11',
    programBytesConsumed: 0,
  }
  public static readonly SWAP12: EVMOpcode = {
    code: Buffer.from('9b', 'hex'),
    name: 'SWAP12',
    programBytesConsumed: 0,
  }
  public static readonly SWAP13: EVMOpcode = {
    code: Buffer.from('9c', 'hex'),
    name: 'SWAP13',
    programBytesConsumed: 0,
  }
  public static readonly SWAP14: EVMOpcode = {
    code: Buffer.from('9d', 'hex'),
    name: 'SWAP14',
    programBytesConsumed: 0,
  }
  public static readonly SWAP15: EVMOpcode = {
    code: Buffer.from('9e', 'hex'),
    name: 'SWAP15',
    programBytesConsumed: 0,
  }
  public static readonly SWAP16: EVMOpcode = {
    code: Buffer.from('9f', 'hex'),
    name: 'SWAP16',
    programBytesConsumed: 0,
  }

  public static readonly LOG0: EVMOpcode = {
    code: Buffer.from('a0', 'hex'),
    name: 'LOG0',
    programBytesConsumed: 0,
  }
  public static readonly LOG1: EVMOpcode = {
    code: Buffer.from('a1', 'hex'),
    name: 'LOG1',
    programBytesConsumed: 0,
  }
  public static readonly LOG2: EVMOpcode = {
    code: Buffer.from('a2', 'hex'),
    name: 'LOG2',
    programBytesConsumed: 0,
  }
  public static readonly LOG3: EVMOpcode = {
    code: Buffer.from('a3', 'hex'),
    name: 'LOG3',
    programBytesConsumed: 0,
  }
  public static readonly LOG4: EVMOpcode = {
    code: Buffer.from('a4', 'hex'),
    name: 'LOG4',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly CREATE: EVMOpcode = {
    code: Buffer.from('f0', 'hex'),
    name: 'CREATE',
    programBytesConsumed: 0,
  }
  public static readonly CALL: EVMOpcode = {
    code: Buffer.from('f1', 'hex'),
    name: 'CALL',
    programBytesConsumed: 0,
  }
  public static readonly CALLCODE: EVMOpcode = {
    code: Buffer.from('f2', 'hex'),
    name: 'CALLCODE',
    programBytesConsumed: 0,
  }
  public static readonly RETURN: EVMOpcode = {
    code: Buffer.from('f3', 'hex'),
    name: 'RETURN',
    programBytesConsumed: 0,
  }
  public static readonly DELEGATECALL: EVMOpcode = {
    code: Buffer.from('f4', 'hex'),
    name: 'DELEGATECALL',
    programBytesConsumed: 0,
  }
  public static readonly CREATE2: EVMOpcode = {
    code: Buffer.from('f5', 'hex'),
    name: 'CREATE2',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly STATICCALL: EVMOpcode = {
    code: Buffer.from('fa', 'hex'),
    name: 'STATICCALL',
    programBytesConsumed: 0,
  }

  // gap

  public static readonly REVERT: EVMOpcode = {
    code: Buffer.from('fd', 'hex'),
    name: 'REVERT',
    programBytesConsumed: 0,
  }
  public static readonly INVALID: EVMOpcode = {
    code: Buffer.from('fe', 'hex'),
    name: 'INVALID',
    programBytesConsumed: 0,
  }
  public static readonly SELFDESTRUCT: EVMOpcode = {
    code: Buffer.from('ff', 'hex'),
    name: 'SELFDESTRUCT',
    programBytesConsumed: 0,
  }

  public static readonly ALL_OP_CODES: EVMOpcode[] = [
    Opcode.STOP,
    Opcode.ADD,
    Opcode.MUL,
    Opcode.SUB,
    Opcode.DIV,
    Opcode.SDIV,
    Opcode.MOD,
    Opcode.SMOD,
    Opcode.ADDMOD,
    Opcode.MULMOD,
    Opcode.EXP,
    Opcode.SIGNEXTEND,

    Opcode.LT,
    Opcode.GT,
    Opcode.SLT,
    Opcode.SGT,
    Opcode.EQ,
    Opcode.ISZERO,
    Opcode.AND,
    Opcode.OR,
    Opcode.XOR,
    Opcode.NOT,
    Opcode.BYTE,
    Opcode.SHL,
    Opcode.SHR,
    Opcode.SAR,

    Opcode.SHA3,

    Opcode.ADDRESS,
    Opcode.BALANCE,
    Opcode.ORIGIN,
    Opcode.CALLER,
    Opcode.CALLVALUE,
    Opcode.CALLDATALOAD,
    Opcode.CALLDATASIZE,
    Opcode.CALLDATACOPY,
    Opcode.CODESIZE,
    Opcode.CODECOPY,
    Opcode.GASPRICE,
    Opcode.EXTCODESIZE,
    Opcode.EXTCODECOPY,
    Opcode.RETURNDATASIZE,
    Opcode.RETURNDATACOPY,
    Opcode.EXTCODEHASH,
    Opcode.BLOCKHASH,
    Opcode.COINBASE,
    Opcode.TIMESTAMP,
    Opcode.NUMBER,
    Opcode.DIFFICULTY,
    Opcode.GASLIMIT,

    Opcode.POP,
    Opcode.MLOAD,
    Opcode.MSTORE,
    Opcode.MSTORE8,
    Opcode.SLOAD,
    Opcode.SSTORE,
    Opcode.JUMP,
    Opcode.JUMPI,
    Opcode.PC,
    Opcode.MSIZE,
    Opcode.GAS,
    Opcode.JUMPDEST,

    Opcode.PUSH1,
    Opcode.PUSH2,
    Opcode.PUSH3,
    Opcode.PUSH4,
    Opcode.PUSH5,
    Opcode.PUSH6,
    Opcode.PUSH7,
    Opcode.PUSH8,
    Opcode.PUSH9,
    Opcode.PUSH10,
    Opcode.PUSH11,
    Opcode.PUSH12,
    Opcode.PUSH13,
    Opcode.PUSH14,
    Opcode.PUSH15,
    Opcode.PUSH16,
    Opcode.PUSH17,
    Opcode.PUSH18,
    Opcode.PUSH19,
    Opcode.PUSH20,
    Opcode.PUSH21,
    Opcode.PUSH22,
    Opcode.PUSH23,
    Opcode.PUSH24,
    Opcode.PUSH25,
    Opcode.PUSH26,
    Opcode.PUSH27,
    Opcode.PUSH28,
    Opcode.PUSH29,
    Opcode.PUSH30,
    Opcode.PUSH31,
    Opcode.PUSH32,

    Opcode.DUP1,
    Opcode.DUP2,
    Opcode.DUP3,
    Opcode.DUP4,
    Opcode.DUP5,
    Opcode.DUP6,
    Opcode.DUP7,
    Opcode.DUP8,
    Opcode.DUP9,
    Opcode.DUP10,
    Opcode.DUP11,
    Opcode.DUP12,
    Opcode.DUP13,
    Opcode.DUP14,
    Opcode.DUP15,
    Opcode.DUP16,

    Opcode.SWAP1,
    Opcode.SWAP2,
    Opcode.SWAP3,
    Opcode.SWAP4,
    Opcode.SWAP5,
    Opcode.SWAP6,
    Opcode.SWAP7,
    Opcode.SWAP8,
    Opcode.SWAP9,
    Opcode.SWAP10,
    Opcode.SWAP11,
    Opcode.SWAP12,
    Opcode.SWAP13,
    Opcode.SWAP14,
    Opcode.SWAP15,
    Opcode.SWAP16,

    Opcode.LOG0,
    Opcode.LOG1,
    Opcode.LOG2,
    Opcode.LOG3,
    Opcode.LOG4,

    Opcode.CREATE,
    Opcode.CALL,
    Opcode.CALLCODE,
    Opcode.RETURN,
    Opcode.DELEGATECALL,
    Opcode.CREATE2,

    Opcode.STATICCALL,

    Opcode.REVERT,
    Opcode.INVALID,
    Opcode.SELFDESTRUCT,
  ]

  public static readonly HALTING_OP_CODES: EVMOpcode[] = [
    Opcode.STOP,
    Opcode.JUMP,
    Opcode.RETURN,
    Opcode.REVERT,
    Opcode.INVALID,
  ]

  private static readonly nameToOpcode: Map<string, EVMOpcode> = new Map<
    string,
    EVMOpcode
  >(Opcode.ALL_OP_CODES.map((x) => [x.name, x]))
  private static readonly codeToOpcode: Map<string, EVMOpcode> = new Map<
    string,
    EVMOpcode
  >(Opcode.ALL_OP_CODES.map((x) => [x.code.toString('hex'), x]))

  public static parseByName(name: string): EVMOpcode | undefined {
    return this.nameToOpcode.get(name)
  }

  public static parseByCode(code: Buffer): EVMOpcode | undefined {
    if (!code) {
      return undefined
    }

    return this.codeToOpcode.get(code.toString('hex'))
  }

  public static parseByNumber(code: number): EVMOpcode | undefined {
    if (code === undefined || code === null) {
      return undefined
    }

    if (code < 16) {
      return this.codeToOpcode.get(`0${code.toString(16)}`)
    }

    return this.codeToOpcode.get(code.toString(16))
  }
}

export interface CodeLine {
  line: number;
}

export interface InstructionTrace {
  line: number;
  pc: number;
  op: string;
  idx: number;
  gasCost: number;
}

export interface LineTrace {
  line: number;
  code: string;
  gasUsed: number;
  instructions: InstructionTrace[];
}

export interface CodeTrace {
  [line: number]: LineTrace;
}

export interface StructLog {
  depth: number;
  error: string;
  gas: number;
  gasCost: number;
  memory: string[];
  op: string;
  pc: number;
  stack: string[];
  storage: {
    [key: string]: string;
  }
}

export enum JumpType {
  FUNCTION_IN, FUNCTION_OUT, STANDARD
}

export interface SourceMapChunk {
  start: number;
  length: number;
  index: number;
  jump: JumpType;
}
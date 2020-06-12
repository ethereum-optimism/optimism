import * as fs from 'fs';
import * as colors from 'colors';
import { JsonRpcProvider } from 'ethers/providers';
import { ContractJson } from '../interfaces/contract.interface';
import { JumpType, StructLog, SourceMapChunk, CodeLine, CodeTrace, InstructionTrace } from '../interfaces/trace.interface';

const JUMP_TYPES = {
  'i': JumpType.FUNCTION_IN,
  'o': JumpType.FUNCTION_OUT,
  '-': JumpType.STANDARD,
};

/**
 * Pulls structlogs via `debug_traceTransaction`.
 * @param provider `ethers` provider to use for the call.
 * @param txhash Hash of the transaction to pull logs for.
 * @returns Array of structlogs.
 */
const getStructLogs = async (provider: JsonRpcProvider, txhash: string): Promise<StructLog[]> => {
  const response = await provider.send('debug_traceTransaction', [
    txhash,
    {
      // Disable some optional return values we don't really need.
      disableStack: true,
      disableMemory: true,
      disableStorage: true
    }
  ]);

  return response.structLogs;
};

/**
 * Parses encoded source map chunks (s:l:f:j;) into a more useful format.
 * @param encodedChunk encoded source map chunk.
 * @returns standardized source map chunk.
 */
const parseSourceMapChunk = (encodedChunk: string): SourceMapChunk => {
  const splitChunk = encodedChunk.split(':');
  return {
    start: parseInt(splitChunk[0], 10),
    length: parseInt(splitChunk[1], 10),
    index: parseInt(splitChunk[2], 10),
    jump: JUMP_TYPES[splitChunk[3]],
  };
};

/**
 * Picks "good" pieces out of either the current source map chunk or the previous one.
 * Source map chunks take the form (s:l:f:j), but one more variables may not be present.
 * When this is the case, we carry over the variable value from the previous chunk.
 * @param previousChunk previous source map chunk.
 * @param currentChunk current source map chunk.
 * @returns a normalized source map chunk.
 */
const pickGoodChunk = (previousChunk: string, currentChunk: string): string => {
  const previousChunkSplit = previousChunk.split(':');
  const currentChunkSplit = currentChunk.split(':');

  if (currentChunkSplit[0] === '') {
    return previousChunk;
  }

  const goodChunks: string[] = [];
  for (let i = 0; i < 4; i++) {
    goodChunks.push(currentChunkSplit[i] || previousChunkSplit[i]);
  }
  return goodChunks.join(':');
}

/**
 * Parses a source map into a more useful format.
 * @param sourceMap contract source map to parse.
 * @returns parsed source map.
 */
const parseSourceMap = (sourceMap: string): SourceMapChunk[] => {
  let previousChunk: string = ':::';
  const sourceMapChunks: SourceMapChunk[] = []
  for (const currentChunk of sourceMap.split(';')) {
    const chunk = pickGoodChunk(previousChunk, currentChunk);
    const parsed = parseSourceMapChunk(chunk);
    sourceMapChunks.push(parsed);
    previousChunk = chunk;
  }

  return sourceMapChunks;
};

/**
 * Determines the line number for a given index within a string.
 * @param source string containing various lines.
 * @param index index within the source.
 * @returns line number of the index.
 */
const parseLineNumber = (source: string, index: number): number => {
  if (index === 0) {
    return 0;
  }

  const lines = source.split('\n');
  let offset = 0;
  for (let i = 0; i < lines.length; i++) {
    if (index <= offset) {
      return i - 1;
    }

    offset += Buffer.byteLength(lines[i] + '\n', 'utf8');
  }
}

/**
 * Given a source map, returns a list of line numbers for corresponding chunks.
 * @param source file to select line numbers from.
 * @param sourceMap file source map.
 * @returns corresponding code line numbers.
 */
const parseSourceLines = (source: string, sourceMap: string): CodeLine[] => {
  const sourceMapChunks = parseSourceMap(sourceMap);
  const firstSourceIndex = sourceMapChunks[0].index;

  let lastGoodLine: number;
  const sourceLines: CodeLine[] = [];
  for (const sourceMapChunk of sourceMapChunks) {
    if (sourceMapChunk.index === firstSourceIndex) {
      const line = parseLineNumber(source, sourceMapChunk.start)
      sourceLines.push({
        line: line,
        chunk: sourceMapChunk,
      });
      lastGoodLine = line;
    } else {
      sourceLines.push({
        line: lastGoodLine,
        chunk: sourceMapChunk,
      });
    }
  }

  return sourceLines;
}

/**
 * Constructs a mapping from contract binary to the source map.
 * @param binary contract binary to parse.
 * @returns mapping between instructions and the source map.
 */
const parseInstructionIndices = (binary: Buffer | string): number[] => {
  if (typeof binary === 'string') {
    binary = Buffer.from(binary, 'hex');
  }

  const instructionIndices: number[] = [];

  let instructionIndex = 0;
  let byteIndex = 0;
  while (byteIndex < binary.length) {
    const instruction = binary[byteIndex];

    let instructionLength = 1;
    if (instruction >= 0x60 && instruction <= 0x7f) {
      instructionLength = 1 + instruction - 0x5f;
    }

    for (let i = 0; i < instructionLength; i++) {
      instructionIndices.push(instructionIndex);
    }

    byteIndex += instructionLength;
    instructionIndex += 1;
  }

  return instructionIndices;
};

/**
 * Utility; produces a useful structure representing a trace for a given file.
 * @param source code to produce an empty trace for.
 * @returns an empty code trace.
 */
const parseCodeLines = (source: string): CodeTrace => {
  const codeTrace: CodeTrace = {};
  const lines = source.split('\n');
  for (let i = 0; i < lines.length; i++) {
    codeTrace[i] = {
      line: i,
      code: lines[i],
      gasUsed: 0,
      instructions: [],
    };
  }
  return codeTrace;
}

const seekJumpDest = (structLogs: StructLog[], instructionIndices: number[], sourceLines: CodeLine[], start: number): InstructionTrace[] => {
  let intermediateLogs: StructLog[] = [];
  let logIndex = start;

  let structLog: StructLog;
  let instructionIndex: number;
  let sourceLine: CodeLine;
  do {
    structLog = structLogs[logIndex];
    instructionIndex = instructionIndices[structLog.pc];
    sourceLine = sourceLines[instructionIndex];
    
    intermediateLogs.push(structLog);
    logIndex++;
  }
  while (
    structLog.op !== 'JUMPDEST' ||
    sourceLine.chunk.index !== sourceLines[0].chunk.index ||
    sourceLine.line === sourceLines[0].line
  );

  let intermediateTraces: InstructionTrace[] = [];
  for (const intermediateLog of intermediateLogs) {
    intermediateTraces.push({
      line: sourceLine.line,
      pc: intermediateLog.pc,
      op: intermediateLog.op,
      idx: instructionIndex,
      gasCost: intermediateLog.gasCost,
    });
  }

  return intermediateTraces;
}

/**
 * Helper; prettifies a code trace into a convenient string.
 * @param trace code trace to prettify.
 * @returns a pretty trace.
 */
export const prettifyTransactionTrace = (trace: CodeTrace): string => {
  let pretty = '';
  for (const key in trace) {
    pretty += `${trace[key].gasUsed}\t┋ ${trace[key].code}\n`;
  }
  return pretty;
}

/**
 * Helper; automatically generates a code trace for a given transaction.
 * @param provider ethers json-rpc provider.
 * @param source path to the source file for this contract.
 * @param contract compiled contract JSON object.
 * @param txhash hash of the transaction to trace.
 * @returns a code trace for the given transaction.
 */
export const getTransactionTrace = async (
  provider: JsonRpcProvider,
  sourcePath: string,
  contract: ContractJson,
  txhash: string,
): Promise<CodeTrace> => {
  const source = fs.readFileSync(sourcePath, 'utf8');
  const structLogs = await getStructLogs(provider, txhash);
  const sourceMap = contract.evm.deployedBytecode.sourceMap;
  const sourceLines = parseSourceLines(source, sourceMap);
  const binary = contract.evm.deployedBytecode.object;
  const instructionIndices = parseInstructionIndices(binary);

  let instructionTraces: InstructionTrace[] = [];
  for (let i = 0; i < structLogs.length; i++) {
    const structLog = structLogs[i];
    const instructionIndex = instructionIndices[structLog.pc];
    const sourceLine = sourceLines[instructionIndex];

    if (structLog.op === 'JUMP' && sourceLine.chunk.index === sourceLines[0].chunk.index) {
      const intermediateTraces = seekJumpDest(structLogs, instructionIndices, sourceLines, i);
      instructionTraces = instructionTraces.concat(intermediateTraces);
      i += intermediateTraces.length - 1;
    } else {
      instructionTraces.push({
        line: sourceLine.line,
        pc: structLog.pc,
        op: structLog.op,
        idx: instructionIndex,
        gasCost: structLog.gasCost,
      });
    }
  }

  const codeTrace: CodeTrace = parseCodeLines(source);
  for (const instructionTrace of instructionTraces) {
    codeTrace[instructionTrace.line].instructions.push(instructionTrace);
    codeTrace[instructionTrace.line].gasUsed += instructionTrace.gasCost;
  }

  return codeTrace;
};
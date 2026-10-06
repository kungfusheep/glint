/**
 * TypeScript type definitions for Glint decoder
 */

export interface DecoderOptions {
  maxStringLength?: number;
  maxArrayLength?: number;
  maxMapSize?: number;
  maxNestingDepth?: number;
}

export const DEFAULT_LIMITS: Required<DecoderOptions> = {
  maxStringLength: 10 * 1024 * 1024,    // 10MB
  maxArrayLength: 1000000,              // 1M elements
  maxMapSize: 100000,                   // 100K entries
  maxNestingDepth: 100,
};

export interface SchemaField {
  name: string;
  wireType: number;
  offset?: number;
  subSchema?: SchemaField[];  // For nested structs
}

// Decoded value types
export type DecodedValue = 
  | boolean 
  | number 
  | bigint
  | string 
  | Uint8Array 
  | Date
  | DecodedObject 
  | DecodedArray
  | null;

export type DecodedObject = { [key: string]: DecodedValue };
export type DecodedArray = DecodedValue[];

export interface DecodeContext {
  depth: number;
  limits: Required<DecoderOptions>;
}

export class GlintError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'GlintError';
  }
}

export class GlintLimitError extends GlintError {
  constructor(message: string) {
    super(message);
    this.name = 'GlintLimitError';
  }
}
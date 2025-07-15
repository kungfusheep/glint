/**
 * Glint TypeScript Decoder
 * Zero-dependency decoder for Glint binary format
 */

export { GlintDecoder } from './decoder';
export { BinaryReader } from './reader';
export * from './wire-types';
export * from './types';

import { GlintDecoder } from './decoder';
import { DecoderOptions, DecodedObject } from './types';

/**
 * Convenience function to decode Glint data
 */
export function decode(data: Uint8Array, options?: DecoderOptions): DecodedObject {
  const decoder = new GlintDecoder(options);
  return decoder.decode(data);
}

/**
 * Check if data appears to be a valid Glint document
 */
export function isGlintDocument(data: Uint8Array): boolean {
  return data.length >= 5;  // Minimum: flags(1) + crc(4)
}

// Version info
export const VERSION = '0.1.0';
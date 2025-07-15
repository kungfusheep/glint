/**
 * BinaryReader provides efficient reading of Glint binary data
 */

import { GlintError } from './types';

export class BinaryReader {
  private data: Uint8Array;
  private position: number = 0;
  private textDecoder: any; // TextDecoder from util module

  constructor(data: Uint8Array) {
    this.data = data;
    // Use Node.js TextDecoder if available, otherwise create a simple fallback
    try {
      const { TextDecoder } = require('util');
      this.textDecoder = new TextDecoder('utf-8', { fatal: true });
    } catch {
      // Fallback for environments without TextDecoder
      this.textDecoder = {
        decode: (buffer: Uint8Array) => Buffer.from(buffer).toString('utf-8')
      };
    }
  }

  get offset(): number {
    return this.position;
  }

  get bytesLeft(): number {
    return this.data.length - this.position;
  }

  get remaining(): Uint8Array {
    return this.data.subarray(this.position);
  }

  advance(n: number): void {
    if (this.position + n > this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    this.position += n;
  }

  readByte(): number {
    if (this.position >= this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    return this.data[this.position++];
  }

  readBytes(n: number): Uint8Array {
    if (this.position + n > this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    const result = this.data.subarray(this.position, this.position + n);
    this.position += n;
    return result;
  }

  /**
   * Read unsigned LEB128 varint
   */
  readVarint(): number {
    let result = 0;
    let shift = 0;
    
    while (this.position < this.data.length) {
      const byte = this.data[this.position++];
      
      if ((byte & 0x80) === 0) {
        result |= byte << shift;
        return result >>> 0;  // Ensure unsigned
      }
      
      result |= (byte & 0x7f) << shift;
      shift += 7;
      
      if (shift >= 35) {
        throw new GlintError('Varint overflow');
      }
    }
    
    throw new GlintError('Unexpected end of data');
  }

  /**
   * Read unsigned LEB128 as BigInt for 64-bit values
   */
  readVarintBigInt(): bigint {
    let result = 0n;
    let shift = 0n;
    
    while (this.position < this.data.length) {
      const byte = BigInt(this.data[this.position++]);
      
      if ((byte & 0x80n) === 0n) {
        result |= byte << shift;
        return result;
      }
      
      result |= (byte & 0x7fn) << shift;
      shift += 7n;
      
      if (shift >= 70n) {
        throw new GlintError('Varint overflow');
      }
    }
    
    throw new GlintError('Unexpected end of data');
  }

  /**
   * Read zigzag-encoded signed varint
   */
  readZigzag(): number {
    const n = this.readVarint();
    return (n >>> 1) ^ -(n & 1);
  }

  /**
   * Read zigzag-encoded signed varint as BigInt
   */
  readZigzagBigInt(): bigint {
    const n = this.readVarintBigInt();
    return (n >> 1n) ^ -(n & 1n);
  }

  // Type-specific readers

  readBool(): boolean {
    return this.readByte() !== 0;
  }

  readInt8(): number {
    const val = this.readByte();
    return val > 127 ? val - 256 : val;
  }

  readUint8(): number {
    return this.readByte();
  }

  readInt16(): number {
    return this.readZigzag();
  }

  readUint16(): number {
    return this.readVarint();
  }

  readInt32(): number {
    return this.readZigzag();
  }

  readUint32(): number {
    return this.readVarint();
  }

  readInt64(): bigint {
    return this.readVarintBigInt();
  }

  readUint64(): bigint {
    return this.readVarintBigInt();
  }

  readInt(): number {
    return this.readZigzag();
  }

  readUint(): number {
    return this.readVarint();
  }

  readFloat32(): number {
    const bits = this.readVarint();
    // Create a temporary buffer to reinterpret bits as float
    const buf = new ArrayBuffer(4);
    new DataView(buf).setUint32(0, bits, true);
    return new DataView(buf).getFloat32(0, true);
  }

  readFloat64(): number {
    const bits = this.readVarint();
    // Create a temporary buffer to reinterpret bits as float
    const buf = new ArrayBuffer(8);
    new DataView(buf).setUint32(0, bits, true);
    new DataView(buf).setUint32(4, 0, true);
    return new DataView(buf).getFloat64(0, true);
  }

  readString(maxLength?: number): string {
    const length = this.readVarint();
    
    if (maxLength !== undefined && length > maxLength) {
      throw new GlintError(`String length ${length} exceeds maximum ${maxLength}`);
    }
    
    const bytes = this.readBytes(length);
    return this.textDecoder.decode(bytes);
  }

  readByteArray(): Uint8Array {
    const length = this.readVarint();
    return this.readBytes(length);
  }

  /**
   * Read time.Time encoded as Go's MarshalBinary format
   */
  readTime(): Date {
    const length = this.readVarint();
    if (length === 0) {
      return new Date(0);
    }
    
    const bytes = this.readBytes(length);
    
    // Go time.Time MarshalBinary format:
    // version(1) + seconds(8) + nanoseconds(4) + zone offset(2)
    if (bytes[0] !== 1) {
      throw new GlintError(`Unsupported time encoding version: ${bytes[0]}`);
    }
    
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    const seconds = view.getBigInt64(1, false);  // Big-endian
    const nanos = view.getInt32(9, false);      // Big-endian
    
    // Convert to milliseconds
    const millis = Number(seconds) * 1000 + Math.floor(nanos / 1000000);
    return new Date(millis);
  }

  /**
   * Create a sub-reader for nested data
   */
  subReader(length: number): BinaryReader {
    const subData = this.readBytes(length);
    return new BinaryReader(subData);
  }
}
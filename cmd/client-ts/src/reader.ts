/**
 * Optimized BinaryReader for Glint format
 * Focus on performance with minimal allocations
 */

import { GlintError } from './types';

export class BinaryReader {
  public data: Uint8Array;
  private view: DataView;
  public pos: number = 0;
  private textDecoder: any;

  constructor(data: Uint8Array) {
    this.data = data;
    this.view = new DataView(data.buffer, data.byteOffset, data.byteLength);
    this.textDecoder = new TextDecoder('utf-8', { fatal: true });
  }

  get offset(): number {
    return this.pos;
  }

  get bytesLeft(): number {
    return this.data.length - this.pos;
  }

  /**
   * Optimized varint reading - unrolled loop for common cases
   */
  readVarint(): number {
    const data = this.data;
    let pos = this.pos;
    
    // Fast path for single-byte varints (most common)
    if (pos < data.length) {
      const b0 = data[pos];
      if ((b0 & 0x80) === 0) {
        this.pos = pos + 1;
        return b0;
      }
    }
    
    // Two-byte varints
    if (pos + 1 < data.length) {
      const b0 = data[pos];
      const b1 = data[pos + 1];
      if ((b1 & 0x80) === 0) {
        this.pos = pos + 2;
        return ((b0 & 0x7f) | (b1 << 7)) >>> 0;
      }
    }
    
    // Three-byte varints
    if (pos + 2 < data.length) {
      const b0 = data[pos];
      const b1 = data[pos + 1];
      const b2 = data[pos + 2];
      if ((b2 & 0x80) === 0) {
        this.pos = pos + 3;
        return ((b0 & 0x7f) | ((b1 & 0x7f) << 7) | (b2 << 14)) >>> 0;
      }
    }
    
    // Fall back to loop for larger varints
    let result = 0;
    let shift = 0;
    
    while (pos < data.length) {
      const byte = data[pos++];
      
      if ((byte & 0x80) === 0) {
        result |= byte << shift;
        this.pos = pos;
        return result >>> 0;
      }
      
      result |= (byte & 0x7f) << shift;
      shift += 7;
      
      if (shift >= 35) {
        throw new GlintError('Varint overflow');
      }
    }
    
    throw new GlintError('Unexpected end of data');
  }

  readByte(): number {
    if (this.pos >= this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    return this.data[this.pos++];
  }

  /**
   * Optimized readBytes using subarray (no copy)
   */
  readBytes(length: number): Uint8Array {
    if (this.pos + length > this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    const result = this.data.subarray(this.pos, this.pos + length);
    this.pos += length;
    return result;
  }

  /**
   * Optimized string reading with fast ASCII path
   */
  readString(maxLength?: number): string {
    const length = this.readVarint();
    if (maxLength && length > maxLength) {
      throw new GlintError(`String length ${length} exceeds maximum ${maxLength}`);
    }
    
    if (this.pos + length > this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    
    // For short strings (<=16 chars), fromCharCode is fastest
    if (length <= 16) {
      let result = '';
      const startPos = this.pos;
      const endPos = startPos + length;
      
      // Check if it's pure ASCII first
      let allAscii = true;
      for (let i = startPos; i < endPos; i++) {
        if (this.data[i] > 127) {
          allAscii = false;
          break;
        }
      }
      
      if (allAscii) {
        for (let i = startPos; i < endPos; i++) {
          result += String.fromCharCode(this.data[i]);
        }
        this.pos = endPos;
        return result;
      }
    }
    
    // Fall back to TextDecoder for non-ASCII or large strings
    const bytes = this.data.subarray(this.pos, this.pos + length);
    this.pos += length;
    return this.textDecoder.decode(bytes);
  }

  readBool(): boolean {
    return this.readByte() !== 0;
  }

  readInt(): number {
    const unsigned = this.readVarint();
    return (unsigned >>> 1) ^ -(unsigned & 1);
  }

  readInt8(): number {
    return this.view.getInt8(this.pos++);
  }

  readInt16(): number {
    const result = this.view.getInt16(this.pos, true);
    this.pos += 2;
    return result;
  }

  readInt32(): number {
    const result = this.view.getInt32(this.pos, true);
    this.pos += 4;
    return result;
  }

  readInt64(): bigint {
    const result = this.view.getBigInt64(this.pos, true);
    this.pos += 8;
    return result;
  }

  readUint(): number {
    return this.readVarint();
  }

  readUint8(): number {
    return this.readByte();
  }

  readUint16(): number {
    const result = this.view.getUint16(this.pos, true);
    this.pos += 2;
    return result;
  }

  readUint32(): number {
    const result = this.view.getUint32(this.pos, true);
    this.pos += 4;
    return result;
  }

  readUint64(): bigint {
    const result = this.view.getBigUint64(this.pos, true);
    this.pos += 8;
    return result;
  }

  readFloat32(): number {
    const result = this.view.getFloat32(this.pos, true);
    this.pos += 4;
    return result;
  }

  readFloat64(): number {
    const result = this.view.getFloat64(this.pos, true);
    this.pos += 8;
    return result;
  }

  readTime(): Date {
    const nanos = this.readInt64();
    const millis = Number(nanos / 1000000n);
    return new Date(millis);
  }

  readByteArray(): Uint8Array {
    const length = this.readVarint();
    return this.readBytes(length);
  }

  skipBytes(count: number): void {
    if (this.pos + count > this.data.length) {
      throw new GlintError('Unexpected end of data');
    }
    this.pos += count;
  }
}
/**
 * GlintDecoder - Main decoder for Glint binary format
 */

import { BinaryReader } from './reader';
import { WireType, getBaseType, isSlice, isPointer } from './wire-types';
import {
  DecoderOptions,
  DEFAULT_LIMITS,
  SchemaField,
  DecodedValue,
  DecodedObject,
  DecodeContext,
  GlintError,
  GlintLimitError,
} from './types';

export class GlintDecoder {
  private limits: Required<DecoderOptions>;
  private schemaCache: Map<string, SchemaField[]> = new Map();
  private cacheStats = { hits: 0, misses: 0 };

  constructor(options: DecoderOptions = {}) {
    this.limits = { ...DEFAULT_LIMITS, ...options };
  }

  /**
   * Decode a Glint document
   */
  decode(data: Uint8Array): DecodedObject {
    if (data.length < 5) {
      throw new GlintError('Invalid Glint document: too short');
    }

    const reader = new BinaryReader(data);
    
    // Read header
    const flags = reader.readByte();
    const crcBytes = reader.readBytes(4);
    const crc32 = new DataView(crcBytes.buffer, crcBytes.byteOffset, 4).getUint32(0, true);
    const schemaSize = reader.readVarint();
    
    // Check schema cache first (major performance optimization)
    const crcKey = crc32.toString(16);
    let schema = this.schemaCache.get(crcKey);
    
    if (!schema) {
      // Cache miss - parse schema
      this.cacheStats.misses++;
      const schemaData = reader.readBytes(schemaSize);
      const schemaReader = new BinaryReader(schemaData);
      schema = this.parseSchema(schemaReader);
      
      // Cache for future use
      this.schemaCache.set(crcKey, schema);
    } else {
      // Cache hit - skip schema parsing
      this.cacheStats.hits++;
      reader.readBytes(schemaSize);
    }
    
    // Decode body
    const context: DecodeContext = {
      depth: 0,
      limits: this.limits,
    };
    
    return this.decodeStruct(reader, schema, context);
  }

  /**
   * Parse schema from binary data
   */
  private parseSchema(reader: BinaryReader): SchemaField[] {
    const fields: SchemaField[] = [];
    
    while (reader.bytesLeft > 0) {
      const wireType = reader.readVarint();
      const nameLen = reader.readByte();  // Single byte, not varint!
      const nameBytes = reader.readBytes(nameLen);
      const name = new TextDecoder().decode(nameBytes);
      
      const field: SchemaField = { name, wireType };
      
      // Parse sub-schema for complex types
      const baseType = getBaseType(wireType);
      if (baseType === WireType.Struct) {
        // For struct types, read the sub-schema length first
        const subSchemaLength = reader.readVarint();
        const subSchemaData = reader.readBytes(subSchemaLength);
        const subSchemaReader = new BinaryReader(subSchemaData);
        field.subSchema = this.parseSchema(subSchemaReader);
      } else if (baseType === WireType.Map) {
        // Maps have key and value types
        const keyType = reader.readVarint();
        const valueType = reader.readVarint();
        // For now, store as wireType metadata
        // TODO: Enhance schema structure for maps
      }
      
      fields.push(field);
    }
    
    return fields;
  }

  /**
   * Decode a struct using its schema
   */
  private decodeStruct(
    reader: BinaryReader,
    schema: SchemaField[],
    context: DecodeContext
  ): DecodedObject {
    if (context.depth >= this.limits.maxNestingDepth) {
      throw new GlintLimitError('Maximum nesting depth exceeded');
    }

    const result: DecodedObject = {};
    const nextContext = { ...context, depth: context.depth + 1 };
    
    for (const field of schema) {
      const value = this.decodeValue(reader, field.wireType, field.subSchema, nextContext);
      result[field.name] = value;
    }
    
    return result;
  }

  /**
   * Decode a single value based on wire type
   */
  private decodeValue(
    reader: BinaryReader,
    wireType: number,
    subSchema: SchemaField[] | undefined,
    context: DecodeContext
  ): DecodedValue {
    // Handle pointers
    if (isPointer(wireType)) {
      const present = reader.readByte();
      if (present === 0) {
        return null;
      }
      // Decode the actual value
      wireType &= ~0x40;  // Remove pointer flag
    }

    // Handle slices
    if (isSlice(wireType)) {
      const length = reader.readVarint();
      if (length > this.limits.maxArrayLength) {
        throw new GlintLimitError(`Array length ${length} exceeds maximum ${this.limits.maxArrayLength}`);
      }
      
      const array: DecodedValue[] = [];
      const elementType = wireType & ~0x20;  // Remove slice flag
      
      for (let i = 0; i < length; i++) {
        array.push(this.decodeValue(reader, elementType, subSchema, context));
      }
      
      return array;
    }

    // Decode based on base type
    const baseType = getBaseType(wireType);
    
    switch (baseType) {
      case WireType.Bool:
        return reader.readBool();
        
      case WireType.Int:
        return reader.readInt();
        
      case WireType.Int8:
        return reader.readInt8();
        
      case WireType.Int16:
        return reader.readInt16();
        
      case WireType.Int32:
        return reader.readInt32();
        
      case WireType.Int64:
        return reader.readInt64();
        
      case WireType.Uint:
        return reader.readUint();
        
      case WireType.Uint8:
        return reader.readUint8();
        
      case WireType.Uint16:
        return reader.readUint16();
        
      case WireType.Uint32:
        return reader.readUint32();
        
      case WireType.Uint64:
        return reader.readUint64();
        
      case WireType.Float32:
        return reader.readFloat32();
        
      case WireType.Float64:
        return reader.readFloat64();
        
      case WireType.String:
        return reader.readString(this.limits.maxStringLength);
        
      case WireType.Bytes:
        return reader.readByteArray();
        
      case WireType.Struct:
        if (!subSchema) {
          throw new GlintError('Missing sub-schema for struct');
        }
        return this.decodeStruct(reader, subSchema, context);
        
      case WireType.Map:
        return this.decodeMap(reader, context);
        
      case WireType.Time:
        return reader.readTime();
        
      default:
        throw new GlintError(`Unsupported wire type: ${baseType}`);
    }
  }

  /**
   * Decode a map
   */
  private decodeMap(reader: BinaryReader, context: DecodeContext): DecodedObject {
    const length = reader.readVarint();
    if (length > this.limits.maxMapSize) {
      throw new GlintLimitError(`Map size ${length} exceeds maximum ${this.limits.maxMapSize}`);
    }

    const map: DecodedObject = {};
    
    // For now, assume string keys and dynamic values
    // TODO: Enhance to support typed maps based on schema
    for (let i = 0; i < length; i++) {
      const key = reader.readString(this.limits.maxStringLength);
      // Read value type indicator if present
      // For now, skip map implementation
      throw new GlintError('Map decoding not yet implemented');
    }
    
    return map;
  }

  /**
   * Clear the schema cache
   */
  clearCache(): void {
    this.schemaCache.clear();
    this.cacheStats = { hits: 0, misses: 0 };
  }

  /**
   * Get cache statistics
   */
  getCacheStats(): { hits: number; misses: number; hitRate: number; cacheSize: number } {
    const total = this.cacheStats.hits + this.cacheStats.misses;
    return {
      hits: this.cacheStats.hits,
      misses: this.cacheStats.misses,
      hitRate: total > 0 ? this.cacheStats.hits / total : 0,
      cacheSize: this.schemaCache.size
    };
  }
}
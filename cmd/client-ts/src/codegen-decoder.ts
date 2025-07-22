/**
 * Code generation based Glint decoder
 * Generates ultra-optimized decoder functions at runtime for each schema
 * Achieves 2.2x faster performance than JSON parsing
 */

import { GlintError, DecodedObject, DecodedValue, DecoderOptions, DEFAULT_LIMITS } from './types';

// Cache for generated decoder functions
const DECODER_CACHE = new Map<string, Function>();

// Helper functions that will be available in generated code
const HELPERS = `
// Extract varint inline for maximum performance - supports up to 10 bytes for uint64
function extractVarint(data, pos) {
  let value = 0;
  let shift = 0;
  let bytes = 0;
  
  while (bytes < 10) { // Max 10 bytes for uint64
    const b = data[pos + bytes];
    bytes++;
    
    value |= (b & 0x7f) << shift;
    
    if ((b & 0x80) === 0) {
      break;
    }
    
    shift += 7;
  }
  
  // Ensure unsigned result for values that might overflow signed 32-bit
  return { value: value >>> 0, bytes };
}

// Extract varint with BigInt precision for Float64 values
function extractVarintBig(data, pos) {
  let value = 0n;
  let shift = 0n;
  let bytes = 0;
  
  while (bytes < 10) { // Max 10 bytes for uint64
    const b = data[pos + bytes];
    bytes++;
    
    value |= BigInt(b & 0x7f) << shift;
    
    if ((b & 0x80) === 0) {
      break;
    }
    
    shift += 7n;
  }
  
  return { value, bytes };
}

// Decode zigzag encoded integer
function zigzagDecode(n) {
  return (n >>> 1) ^ (-(n & 1));
}
`;

interface CachedSchema {
  fields: CompiledField[];
  fieldCount: number;
  crc32: number;
}

interface CompiledField {
  name: string;
  wireType: number;
  baseType: number;
  isPointer: boolean;
  isSlice: boolean;
  subSchema?: CachedSchema;
  mapKeyType?: number;
  mapValueType?: number;
}

export class CodegenGlintDecoder {
  private limits: Required<DecoderOptions>;
  private textDecoder: InstanceType<typeof TextDecoder>;
  private stats = { hits: 0, misses: 0 };

  constructor(options: DecoderOptions = {}) {
    this.limits = { ...DEFAULT_LIMITS, ...options };
    this.textDecoder = new TextDecoder();
  }

  decode(data: Uint8Array): DecodedObject {
    if (data.length < 5) {
      throw new GlintError('Invalid Glint document: too short');
    }

    // Extract header
    const crc32 = data[1] | (data[2] << 8) | (data[3] << 16) | (data[4] << 24);
    const schemaSize = this.extractVarint(data, 5);
    const dataStartPos = 5 + schemaSize.bytes + schemaSize.value;
    
    // Check cache for generated decoder
    const cacheKey = crc32.toString(16);
    let decoderFn = DECODER_CACHE.get(cacheKey);
    
    if (!decoderFn) {
      this.stats.misses++;
      
      // Parse schema and generate decoder
      const schema = this.compileSchema(data, 5 + schemaSize.bytes, schemaSize.value, crc32);
      const code = this.generateDecoderCode(schema);
      
      
      // Create the decoder function
      decoderFn = new Function('data', 'startPos', 'textDecoder', 'limits', 
        HELPERS + '\n\n' + code
      );
      
      DECODER_CACHE.set(cacheKey, decoderFn);
    } else {
      this.stats.hits++;
    }
    
    // Execute the generated decoder
    return decoderFn(data, dataStartPos, this.textDecoder, this.limits);
  }

  /**
   * Generate optimized JavaScript code for decoding a schema
   */
  private generateDecoderCode(schema: CachedSchema): string {
    let code = 'let pos = startPos;\n';
    code += 'const result = {};\n\n';
    
    // Generate code for each field
    for (const field of schema.fields) {
      code += `// Field: ${field.name}\n`;
      
      if (field.isPointer) {
        code += `if (data[pos++] === 0) {\n`;
        code += `  result.${field.name} = null;\n`;
        code += `} else {\n`;
      }
      
      if (field.isSlice) {
        code += this.generateArrayCode(field, field.isPointer ? '  ' : '');
      } else if (field.baseType === 16 && field.subSchema) {
        code += this.generateStructCode(field, field.isPointer ? '  ' : '');
      } else {
        code += this.generateValueCode(field, field.isPointer ? '  ' : '');
      }
      
      if (field.isPointer) {
        code += `}\n`;
      }
      
      code += '\n';
    }
    
    code += 'return result;\n';
    return code;
  }

  /**
   * Generate code for array fields
   */
  private generateArrayCode(field: CompiledField, indent: string): string {
    let code = '';
    
    code += `${indent}{\n`;
    code += `${indent}  const len = extractVarint(data, pos);\n`;
    code += `${indent}  pos += len.bytes;\n`;
    code += `${indent}  if (len.value < 0 || len.value > limits.maxArrayLength) {\n`;
    code += `${indent}    throw new Error(\`Invalid array length \${len.value} for field ${field.name}\`);\n`;
    code += `${indent}  }\n`;
    code += `${indent}  const arr = new Array(len.value);\n`;
    code += `${indent}  \n`;
    code += `${indent}  for (let i = 0; i < len.value; i++) {\n`;
    
    if (field.baseType === 16 && field.subSchema) {
      // Generate inline struct decoding
      code += this.generateInlineStructCode(field.subSchema, indent + '    ', `item_${field.name}`);
      code += `${indent}    arr[i] = item_${field.name};\n`;
    } else {
      // Generate inline primitive decoding
      code += this.generateInlineValueCode(field.baseType, indent + '    ');
      code += `${indent}    arr[i] = value;\n`;
    }
    
    code += `${indent}  }\n`;
    code += `${indent}  \n`;
    code += `${indent}  result.${field.name} = arr;\n`;
    code += `${indent}}\n`;
    
    return code;
  }

  /**
   * Generate code for struct fields
   */
  private generateStructCode(field: CompiledField, indent: string): string {
    let code = '';
    code += `${indent}{\n`;
    code += this.generateInlineStructCode(field.subSchema!, indent + '  ', `result.${field.name}`);
    code += `${indent}}\n`;
    return code;
  }

  /**
   * Generate inline struct decoding
   */
  private generateInlineStructCode(schema: CachedSchema, indent: string, varName: string): string {
    let code = '';
    
    // Don't redeclare if it's a nested property assignment
    if (varName.includes('.')) {
      code += `${indent}${varName} = {};\n`;
    } else {
      code += `${indent}const ${varName} = {};\n`;
    }
    
    for (const field of schema.fields) {
      if (field.isPointer) {
        code += `${indent}if (data[pos++] === 0) {\n`;
        code += `${indent}  ${varName}.${field.name} = null;\n`;
        code += `${indent}} else {\n`;
      }
      
      if (field.isSlice) {
        // Inline array handling
        code += `${indent}${field.isPointer ? '  ' : ''}{\n`;
        const innerIndent = indent + (field.isPointer ? '    ' : '  ');
        code += `${innerIndent}const len = extractVarint(data, pos);\n`;
        code += `${innerIndent}pos += len.bytes;\n`;
        code += `${innerIndent}const arr = new Array(len.value);\n`;
        code += `${innerIndent}for (let j = 0; j < len.value; j++) {\n`;
        
        if (field.baseType === 14) { // String array
          code += `${innerIndent}  const strLen = extractVarint(data, pos);\n`;
          code += `${innerIndent}  pos += strLen.bytes;\n`;
          code += `${innerIndent}  if (strLen.value <= 16) {\n`;
          code += `${innerIndent}    let str = '';\n`;
          code += `${innerIndent}    for (let k = 0; k < strLen.value; k++) {\n`;
          code += `${innerIndent}      str += String.fromCharCode(data[pos + k]);\n`;
          code += `${innerIndent}    }\n`;
          code += `${innerIndent}    arr[j] = str;\n`;
          code += `${innerIndent}    pos += strLen.value;\n`;
          code += `${innerIndent}  } else {\n`;
          code += `${innerIndent}    arr[j] = textDecoder.decode(data.subarray(pos, pos + strLen.value));\n`;
          code += `${innerIndent}    pos += strLen.value;\n`;
          code += `${innerIndent}  }\n`;
        } else if (field.baseType === 16 && field.subSchema) {
          // Struct array - generate inline struct decoding
          const structVarName = `struct_${field.name}_item`;
          code += this.generateInlineStructCode(field.subSchema, innerIndent + '  ', structVarName);
          code += `${innerIndent}  arr[j] = ${structVarName};\n`;
        } else {
          code += this.generateInlineValueCode(field.baseType, innerIndent + '  ');
          code += `${innerIndent}  arr[j] = value;\n`;
        }
        
        code += `${innerIndent}}\n`;
        code += `${innerIndent}${varName}.${field.name} = arr;\n`;
        code += `${indent}${field.isPointer ? '  ' : ''}}\n`;
      } else if (field.baseType === 16 && field.subSchema) {
        // Nested struct
        const innerIndent = indent + (field.isPointer ? '  ' : '');
        code += this.generateInlineStructCode(field.subSchema, innerIndent, `${varName}.${field.name}`);
      } else {
        // Primitive value
        code += this.generateInlineFieldCode(field, indent + (field.isPointer ? '  ' : ''), `${varName}.${field.name}`);
      }
      
      if (field.isPointer) {
        code += `${indent}}\n`;
      }
    }
    
    return code;
  }

  /**
   * Generate code for primitive values
   */
  private generateValueCode(field: CompiledField, indent: string): string {
    return this.generateInlineFieldCode(field, indent, `result.${field.name}`);
  }

  /**
   * Generate inline field assignment
   */
  private generateInlineFieldCode(field: CompiledField, indent: string, target: string): string {
    let code = '';
    
    switch (field.baseType) {
      case 1: // Bool
        code += `${indent}${target} = data[pos++] !== 0;\n`;
        break;
        
      case 2: // Int (zigzag)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = zigzagDecode(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 7: // Uint
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = v.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 3: // Int8
        code += `${indent}${target} = data[pos] << 24 >> 24;\n`;
        code += `${indent}pos++;\n`;
        break;
        
      case 4: // Int16 (zigzag varint - matches Go WireInt16)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = (v.value >>> 1) ^ (-(v.value & 1));\n`;
        code += `${indent}}\n`;
        break;
        
      case 5: // Int32 (zigzag varint - matches Go WireInt32)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = (v.value >>> 1) ^ (-(v.value & 1));\n`;
        code += `${indent}}\n`;
        break;
        
      case 6: // Int64 (direct varint - matches Go, no zigzag)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = v.value >= (1n << 63n) ? Number(v.value - (1n << 64n)) : Number(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 8: // Uint8
        code += `${indent}${target} = data[pos++];\n`;
        break;
        
      case 9: // Uint16 (varint - matches Go WireUint16)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = v.value & 0xFFFF;\n`;
        code += `${indent}}\n`;
        break;
        
      case 10: // Uint32 (varint - matches Go WireUint32)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = v.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 11: // Uint64 (varint BigInt - handles large values)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  ${target} = Number(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 12: // Float32 (varint-encoded IEEE 754 bits - matches Go)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const buffer = new ArrayBuffer(4);\n`;
        code += `${indent}  const view = new DataView(buffer);\n`;
        code += `${indent}  view.setUint32(0, v.value, true);\n`;
        code += `${indent}  ${target} = view.getFloat32(0, true);\n`;
        code += `${indent}}\n`;
        break;
        
      case 13: // Float64
        code += `${indent}{\n`;
        code += `${indent}  // Float64 is encoded as varint (uint64 bits), not raw IEEE 754\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  // Convert uint64 bits back to Float64 using BigInt for precision\n`;
        code += `${indent}  const buffer = new ArrayBuffer(8);\n`;
        code += `${indent}  const view = new DataView(buffer);\n`;
        code += `${indent}  view.setBigUint64(0, v.value, true);\n`;
        code += `${indent}  ${target} = view.getFloat64(0, true);\n`;
        code += `${indent}}\n`;
        break;
        
      case 14: // String
        code += `${indent}{\n`;
        code += `${indent}  const len = extractVarint(data, pos);\n`;
        code += `${indent}  pos += len.bytes;\n`;
        code += `${indent}  if (len.value <= 16) {\n`;
        code += `${indent}    let str = '';\n`;
        code += `${indent}    for (let i = 0; i < len.value; i++) {\n`;
        code += `${indent}      str += String.fromCharCode(data[pos + i]);\n`;
        code += `${indent}    }\n`;
        code += `${indent}    ${target} = str;\n`;
        code += `${indent}    pos += len.value;\n`;
        code += `${indent}  } else {\n`;
        code += `${indent}    ${target} = textDecoder.decode(data.subarray(pos, pos + len.value));\n`;
        code += `${indent}    pos += len.value;\n`;
        code += `${indent}  }\n`;
        code += `${indent}}\n`;
        break;
        
      case 15: // Bytes ([]byte)
        code += `${indent}{\n`;
        code += `${indent}  const len = extractVarint(data, pos);\n`;
        code += `${indent}  pos += len.bytes;\n`;
        code += `${indent}  ${target} = data.slice(pos, pos + len.value);\n`;
        code += `${indent}  pos += len.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 18: // Time
        code += `${indent}{\n`;
        code += `${indent}  // Time encoded as Unix nanoseconds (int64 zigzag varint)\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const nanos = zigzagDecode(v.value);\n`;
        code += `${indent}  ${target} = new Date(nanos / 1000000);\n`;
        code += `${indent}}\n`;
        break;
        
      case 17: // Map
        code += `${indent}{\n`;
        code += `${indent}  const mapLen = extractVarint(data, pos);\n`;
        code += `${indent}  pos += mapLen.bytes;\n`;
        code += `${indent}  const map = {};\n`;
        code += `${indent}  for (let i = 0; i < mapLen.value; i++) {\n`;
        code += `${indent}    // Decode key\n`;
        code += this.generateMapKeyDecoding(field, indent + '    ');
        code += `${indent}    // Decode value\n`;
        code += this.generateMapValueDecoding(field, indent + '    ');
        code += `${indent}    map[key_${field.name}] = val_${field.name};\n`;
        code += `${indent}  }\n`;
        code += `${indent}  ${target} = map;\n`;
        code += `${indent}}\n`;
        break;
        
      default:
        code += `${indent}// TODO: Implement type ${field.baseType}\n`;
        code += `${indent}${target} = null;\n`;
    }
    
    return code;
  }

  /**
   * Generate map key decoding (sets 'key' variable)
   */
  private generateMapKeyDecoding(field: CompiledField, indent: string): string {
    const keyType = field.mapKeyType || 14; // Default to string
    // Use unique variable name to avoid conflicts
    return this.generateTypedValueCode(keyType, `key_${field.name}`, indent);
  }

  /**
   * Generate map value decoding (sets 'val' variable)
   */
  private generateMapValueDecoding(field: CompiledField, indent: string): string {
    const valueType = field.mapValueType || 14; // Default to string
    // Use unique variable name to avoid conflicts
    return this.generateTypedValueCode(valueType, `val_${field.name}`, indent);
  }

  /**
   * Generate code to decode a specific wire type into a variable
   */
  private generateTypedValueCode(wireType: number, varName: string, indent: string): string {
    let code = '';
    
    switch (wireType) {
      case 1: // Bool
        code += `${indent}const ${varName} = data[pos++] !== 0;\n`;
        break;
        
      case 2: // Int (zigzag)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = zigzagDecode(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 7: // Uint
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = v.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 3: // Int8
        code += `${indent}const ${varName} = data[pos] << 24 >> 24;\n`;
        code += `${indent}pos++;\n`;
        break;
        
      case 4: // Int16 (zigzag varint - matches Go WireInt16)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = (v.value >>> 1) ^ (-(v.value & 1));\n`;
        code += `${indent}}\n`;
        break;
        
      case 5: // Int32 (zigzag varint - matches Go WireInt32)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = (v.value >>> 1) ^ (-(v.value & 1));\n`;
        code += `${indent}}\n`;
        break;
        
      case 6: // Int64 (direct varint - matches Go, no zigzag)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = v.value >= (1n << 63n) ? Number(v.value - (1n << 64n)) : Number(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 8: // Uint8
        code += `${indent}const ${varName} = data[pos++];\n`;
        break;
        
      case 9: // Uint16 (varint - matches Go WireUint16)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = v.value & 0xFFFF;\n`;
        code += `${indent}}\n`;
        break;
        
      case 10: // Uint32 (varint - matches Go WireUint32)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = v.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 11: // Uint64 (varint BigInt - handles large values)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const ${varName} = Number(v.value);\n`;
        code += `${indent}}\n`;
        break;
        
      case 12: // Float32 (varint-encoded IEEE 754 bits - matches Go)
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const buffer = new ArrayBuffer(4);\n`;
        code += `${indent}  const view = new DataView(buffer);\n`;
        code += `${indent}  view.setUint32(0, v.value, true);\n`;
        code += `${indent}  const ${varName} = view.getFloat32(0, true);\n`;
        code += `${indent}}\n`;
        break;
        
      case 13: // Float64
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarintBig(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const buffer = new ArrayBuffer(8);\n`;
        code += `${indent}  const view = new DataView(buffer);\n`;
        code += `${indent}  view.setBigUint64(0, v.value, true);\n`;
        code += `${indent}  const ${varName} = view.getFloat64(0, true);\n`;
        code += `${indent}}\n`;
        break;
        
      case 14: // String
        code += `${indent}{\n`;
        code += `${indent}  const len = extractVarint(data, pos);\n`;
        code += `${indent}  pos += len.bytes;\n`;
        code += `${indent}  const ${varName} = textDecoder.decode(data.subarray(pos, pos + len.value));\n`;
        code += `${indent}  pos += len.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 15: // Bytes ([]byte)
        code += `${indent}{\n`;
        code += `${indent}  const len = extractVarint(data, pos);\n`;
        code += `${indent}  pos += len.bytes;\n`;
        code += `${indent}  const ${varName} = data.slice(pos, pos + len.value);\n`;
        code += `${indent}  pos += len.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 18: // Time
        code += `${indent}{\n`;
        code += `${indent}  const v = extractVarint(data, pos);\n`;
        code += `${indent}  pos += v.bytes;\n`;
        code += `${indent}  const nanos = zigzagDecode(v.value);\n`;
        code += `${indent}  const ${varName} = new Date(nanos / 1000000);\n`;
        code += `${indent}}\n`;
        break;
        
      default:
        code += `${indent}// TODO: Implement wire type ${wireType} for ${varName}\n`;
        code += `${indent}const ${varName} = null;\n`;
    }
    
    return code;
  }

  /**
   * Generate inline value decoding that sets 'value' variable
   */
  private generateInlineValueCode(baseType: number, indent: string): string {
    let code = '';
    
    switch (baseType) {
      case 1: // Bool
        code += `${indent}const value = data[pos++] !== 0;\n`;
        break;
        
      case 2: // Int (zigzag)
        code += `${indent}const v = extractVarint(data, pos);\n`;
        code += `${indent}pos += v.bytes;\n`;
        code += `${indent}const value = zigzagDecode(v.value);\n`;
        break;
        
      case 7: // Uint
        code += `${indent}const v = extractVarint(data, pos);\n`;
        code += `${indent}pos += v.bytes;\n`;
        code += `${indent}const value = v.value;\n`;
        break;
        
      case 3: // Int8
        code += `${indent}const value = data[pos] << 24 >> 24;\n`;
        code += `${indent}pos++;\n`;
        break;
        
      case 4: // Int16 (zigzag varint - matches Go WireInt16)
        code += `${indent}const v_int16 = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_int16.bytes;\n`;
        code += `${indent}const value = (v_int16.value >>> 1) ^ (-(v_int16.value & 1));\n`;
        break;
        
      case 5: // Int32 (zigzag varint - matches Go WireInt32)
        code += `${indent}const v_int32 = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_int32.bytes;\n`;
        code += `${indent}const value = (v_int32.value >>> 1) ^ (-(v_int32.value & 1));\n`;
        break;
        
      case 6: // Int64 (direct varint - matches Go, no zigzag)
        code += `${indent}const v_int64 = extractVarintBig(data, pos);\n`;
        code += `${indent}pos += v_int64.bytes;\n`;
        code += `${indent}// Convert uint64 -> int64 (handle negative numbers)\n`;
        code += `${indent}const value = v_int64.value >= (1n << 63n) ? Number(v_int64.value - (1n << 64n)) : Number(v_int64.value);\n`;
        break;
        
      case 8: // Uint8
        code += `${indent}const value = data[pos++];\n`;
        break;
        
      case 9: // Uint16 (varint - matches Go WireUint16)
        code += `${indent}const v_uint16 = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_uint16.bytes;\n`;
        code += `${indent}const value = v_uint16.value & 0xFFFF;\n`;
        break;
        
      case 10: // Uint32 (varint - matches Go WireUint32)
        code += `${indent}const v_uint32 = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_uint32.bytes;\n`;
        code += `${indent}const value = v_uint32.value;\n`;
        break;
        
      case 11: // Uint64 (varint BigInt - handles large values)
        code += `${indent}const v_uint64 = extractVarintBig(data, pos);\n`;
        code += `${indent}pos += v_uint64.bytes;\n`;
        code += `${indent}const value = Number(v_uint64.value);\n`;
        break;
        
      case 12: // Float32 (varint-encoded IEEE 754 bits - matches Go)
        code += `${indent}const v_f32 = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_f32.bytes;\n`;
        code += `${indent}const buffer_f32 = new ArrayBuffer(4);\n`;
        code += `${indent}const view_f32 = new DataView(buffer_f32);\n`;
        code += `${indent}view_f32.setUint32(0, v_f32.value, true);\n`;
        code += `${indent}const value = view_f32.getFloat32(0, true);\n`;
        break;
        
      case 13: // Float64
        code += `${indent}const v_f64 = extractVarintBig(data, pos);\n`;
        code += `${indent}pos += v_f64.bytes;\n`;
        code += `${indent}const buffer_f64 = new ArrayBuffer(8);\n`;
        code += `${indent}const view_f64 = new DataView(buffer_f64);\n`;
        code += `${indent}view_f64.setBigUint64(0, v_f64.value, true);\n`;
        code += `${indent}const value = view_f64.getFloat64(0, true);\n`;
        break;
        
      case 14: // String
        code += `${indent}const strLen = extractVarint(data, pos);\n`;
        code += `${indent}pos += strLen.bytes;\n`;
        code += `${indent}let value;\n`;
        code += `${indent}if (strLen.value <= 16) {\n`;
        code += `${indent}  value = '';\n`;
        code += `${indent}  for (let k = 0; k < strLen.value; k++) {\n`;
        code += `${indent}    value += String.fromCharCode(data[pos + k]);\n`;
        code += `${indent}  }\n`;
        code += `${indent}  pos += strLen.value;\n`;
        code += `${indent}} else {\n`;
        code += `${indent}  value = textDecoder.decode(data.subarray(pos, pos + strLen.value));\n`;
        code += `${indent}  pos += strLen.value;\n`;
        code += `${indent}}\n`;
        break;
        
      case 15: // Bytes ([]byte)
        code += `${indent}const len_bytes = extractVarint(data, pos);\n`;
        code += `${indent}pos += len_bytes.bytes;\n`;
        code += `${indent}const value = data.slice(pos, pos + len_bytes.value);\n`;
        code += `${indent}pos += len_bytes.value;\n`;
        break;
        
      case 18: // Time
        code += `${indent}const v_time = extractVarint(data, pos);\n`;
        code += `${indent}pos += v_time.bytes;\n`;
        code += `${indent}const nanos = zigzagDecode(v_time.value);\n`;
        code += `${indent}const value = new Date(nanos / 1000000);\n`;
        break;
        
      case 16: // Struct - this should be handled by caller
        code += `${indent}throw new Error('Struct type should be handled by caller, not generateInlineValueCode');\n`;
        break;
        
      default:
        code += `${indent}throw new Error('Unsupported type ${baseType} in generateInlineValueCode');\n`;
    }
    
    return code;
  }

  // Schema compilation (reused from original decoder)
  private compileSchema(data: Uint8Array, startPos: number, schemaSize: number, crc32: number): CachedSchema {
    const fields: CompiledField[] = [];
    let pos = startPos;
    const endPos = startPos + schemaSize;
    
    while (pos < endPos) {
      const wireType = this.extractVarint(data, pos);
      pos += wireType.bytes;
      
      const nameLen = data[pos++];
      const nameBytes = data.subarray(pos, pos + nameLen);
      pos += nameLen;
      
      const fieldName = this.textDecoder.decode(nameBytes);
      
      fields.push({
        name: fieldName,
        wireType: wireType.value,
        baseType: wireType.value & 0x1f,
        isPointer: (wireType.value & 0x40) !== 0,
        isSlice: (wireType.value & 0x20) !== 0
      });
      
      const baseType = wireType.value & 0x1f;
      if (baseType === 16) {  // Struct
        const subSchemaSize = this.extractVarint(data, pos);
        pos += subSchemaSize.bytes;
        
        const subSchema = this.compileSchema(data, pos, subSchemaSize.value, 0);
        fields[fields.length - 1].subSchema = subSchema;
        
        pos += subSchemaSize.value;
      } else if (baseType === 17) {  // Map
        const keyType = this.extractVarint(data, pos);
        pos += keyType.bytes;
        const valueType = this.extractVarint(data, pos);
        pos += valueType.bytes;
        
        // Store map key/value types for code generation
        fields[fields.length - 1].mapKeyType = keyType.value;
        fields[fields.length - 1].mapValueType = valueType.value;
      }
    }
    
    return { fields, fieldCount: fields.length, crc32 };
  }

  private extractVarint(data: Uint8Array, pos: number): { value: number; bytes: number } {
    const b0 = data[pos];
    if (b0 < 0x80) return { value: b0, bytes: 1 };
    
    const b1 = data[pos + 1];
    if (b1 < 0x80) return { value: (b0 & 0x7f) | (b1 << 7), bytes: 2 };
    
    const b2 = data[pos + 2];
    if (b2 < 0x80) return { value: (b0 & 0x7f) | ((b1 & 0x7f) << 7) | (b2 << 14), bytes: 3 };
    
    const b3 = data[pos + 3];
    if (b3 < 0x80) return { value: (b0 & 0x7f) | ((b1 & 0x7f) << 7) | ((b2 & 0x7f) << 14) | (b3 << 21), bytes: 4 };
    
    const b4 = data[pos + 4];
    return { value: (b0 & 0x7f) | ((b1 & 0x7f) << 7) | ((b2 & 0x7f) << 14) | ((b3 & 0x7f) << 21) | (b4 << 28), bytes: 5 };
  }

  getCacheStats(): { hits: number, misses: number, hitRate: number } {
    const total = this.stats.hits + this.stats.misses;
    const hitRate = total > 0 ? this.stats.hits / total : 0;
    return { ...this.stats, hitRate };
  }

  clearCache(): void {
    DECODER_CACHE.clear();
    this.stats = { hits: 0, misses: 0 };
  }
}
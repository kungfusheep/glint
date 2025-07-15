/**
 * Debug body decoding to find UTF-8 error
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoderOptimized } from '../src/decoder-optimized';

// Create a custom decoder that logs field decoding
class DebuggingDecoder extends GlintDecoderOptimized {
  private fieldCount = 0;
  
  protected decodeValue(reader: any, wireType: number, subSchema: any, context: any): any {
    this.fieldCount++;
    const startPos = reader.offset;
    
    try {
      console.log(`\n🔍 Decoding field ${this.fieldCount}:`);
      console.log(`  Position: ${startPos}`);
      console.log(`  Wire type: ${wireType} (0x${wireType.toString(16)})`);
      console.log(`  Base type: ${wireType & 0x1f}`);
      console.log(`  Is pointer: ${(wireType & 0x40) !== 0}`);
      console.log(`  Is slice: ${(wireType & 0x20) !== 0}`);
      
      // Log specific type
      const baseType = wireType & 0x1f;
      const typeNames: { [key: number]: string } = {
        1: 'Bool', 2: 'Int', 3: 'Int8', 4: 'Int16', 5: 'Int32', 6: 'Int64',
        7: 'Uint', 8: 'Uint8', 9: 'Uint16', 10: 'Uint32', 11: 'Uint64',
        12: 'Float32', 13: 'Float64', 14: 'String', 15: 'Bytes',
        16: 'Struct', 17: 'Map', 18: 'Time'
      };
      console.log(`  Type: ${typeNames[baseType] || 'Unknown'}`);
      
      if (baseType === 14) { // String type
        console.log('  📝 Decoding string...');
        // Peek at the length
        const lengthPos = reader.offset;
        const length = reader.readVarint();
        console.log(`  String length: ${length} (at position ${lengthPos})`);
        
        // Reset and let parent decode
        reader.pos = startPos;
      }
      
      const result = super.decodeValue(reader, wireType, subSchema, context);
      
      console.log(`  ✅ Success! Result type: ${typeof result}`);
      if (typeof result === 'string' && result.length < 50) {
        console.log(`  Value: "${result}"`);
      } else if (Array.isArray(result)) {
        console.log(`  Array length: ${result.length}`);
      }
      
      return result;
      
    } catch (error) {
      console.log(`  ❌ Error: ${(error as Error).message}`);
      console.log(`  Position when failed: ${reader.offset}`);
      
      // Show next few bytes
      if (reader.bytesLeft > 0) {
        const nextBytes = reader.data.slice(reader.offset, Math.min(reader.offset + 20, reader.data.length));
        console.log(`  Next 20 bytes: ${Array.from(nextBytes).map((b) => (b as number).toString(16).padStart(2, '0')).join(' ')}`);
        console.log(`  As ASCII: "${Array.from(nextBytes).map((b) => (b as number) >= 32 && (b as number) < 127 ? String.fromCharCode(b as number) : '.').join('')}"`);
      }
      
      throw error;
    }
  }
}

function debugBodyDecode(): void {
  console.log('🔍 Debugging Body Decoding for Medium Dataset\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const data = new Uint8Array(fs.readFileSync(path.join(testDir, 'medium.glint')));
  
  const decoder = new DebuggingDecoder();
  
  try {
    console.log('Starting decode...');
    const result = decoder.decode(data);
    console.log('\n✅ Decode successful!');
    console.log(`Top-level keys: ${Object.keys(result).join(', ')}`);
  } catch (error) {
    console.log(`\n💥 Decode failed: ${(error as Error).message}`);
    console.log('Stack:', (error as Error).stack);
  }
}

// Run debug
debugBodyDecode();
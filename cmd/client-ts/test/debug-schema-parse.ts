/**
 * Debug schema parsing step by step
 */

import * as fs from 'fs';
import * as path from 'path';
import { BinaryReader } from '../src/reader';

function debugSchemaParsing(): void {
  console.log('🔍 Debugging Schema Parsing Step by Step');
  console.log('');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const datasets = ['simple', 'complex', 'medium'];
  
  for (const dataset of datasets) {
    console.log(`\n📊 Testing ${dataset} dataset:`);
    
    const glintPath = path.join(testDir, `${dataset}.glint`);
    const data = new Uint8Array(fs.readFileSync(glintPath));
    
    console.log(`Data length: ${data.length}`);
    
    const reader = new BinaryReader(data);
    
    try {
      // Read header
      const flags = reader.readByte();
      console.log(`Flags: ${flags} (position: ${reader.offset})`);
      
      const crcBytes = reader.readBytes(4);
      const crc32 = new DataView(crcBytes.buffer, crcBytes.byteOffset, 4).getUint32(0, true);
      console.log(`CRC32: 0x${crc32.toString(16)} (position: ${reader.offset})`);
      
      const schemaSize = reader.readVarint();
      console.log(`Schema size: ${schemaSize} (position: ${reader.offset})`);
      console.log(`Bytes left: ${reader.bytesLeft}`);
      
      // Read schema data
      const schemaData = reader.readBytes(schemaSize);
      console.log(`Schema data length: ${schemaData.length}`);
      
      // Now parse the schema step by step
      const schemaReader = new BinaryReader(schemaData);
      console.log(`\n🔍 Parsing schema fields:`);
      
      let fieldIndex = 0;
      while (schemaReader.bytesLeft > 0) {
        console.log(`\n  Field ${fieldIndex}:`);
        console.log(`    Position: ${schemaReader.offset}`);
        console.log(`    Bytes left: ${schemaReader.bytesLeft}`);
        
        if (schemaReader.bytesLeft < 1) {
          console.log(`    ❌ No bytes left for wire type`);
          break;
        }
        
        const wireType = schemaReader.readVarint();
        console.log(`    Wire type: ${wireType} (position: ${schemaReader.offset})`);
        
        if (schemaReader.bytesLeft < 1) {
          console.log(`    ❌ No bytes left for name length`);
          break;
        }
        
        const nameLen = schemaReader.readVarint();
        console.log(`    Name length: ${nameLen} (position: ${schemaReader.offset})`);
        
        if (nameLen > schemaReader.bytesLeft) {
          console.log(`    ❌ Name length ${nameLen} > bytes left ${schemaReader.bytesLeft}`);
          break;
        }
        
        const nameBytes = schemaReader.readBytes(nameLen);
        const name = new TextDecoder().decode(nameBytes);
        console.log(`    Name: "${name}" (position: ${schemaReader.offset})`);
        
        // Check for sub-schema (simplified logic)
        const baseType = wireType & 0x1f; // Remove flags
        if (baseType === 16) { // WireStruct
          console.log(`    ⚠️  Struct type detected - would need sub-schema parsing`);
          // This is where the issue might be - we need to handle nested schemas
        }
        
        fieldIndex++;
        
        if (fieldIndex > 20) {
          console.log(`    ⚠️  Too many fields, stopping debug`);
          break;
        }
      }
      
      console.log(`\n  ✅ Schema parsing completed`);
      console.log(`  Final position: ${schemaReader.offset}`);
      console.log(`  Bytes left: ${schemaReader.bytesLeft}`);
      
      // Show remaining body size
      console.log(`\n📄 Body data after schema:`);
      console.log(`  Position: ${reader.offset}`);
      console.log(`  Bytes left: ${reader.bytesLeft}`);
      
    } catch (error) {
      console.log(`❌ Error: ${(error as Error).message}`);
      console.log(`Position when error occurred: ${reader.offset}`);
      console.log(`Stack: ${(error as Error).stack?.split('\n')[1]?.trim()}`);
    }
  }
}

// Run debug if this file is executed directly
if (require.main === module) {
  debugSchemaParsing();
}

export { debugSchemaParsing };
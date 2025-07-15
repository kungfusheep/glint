/**
 * Debug UTF-8 decoding errors
 */

import * as fs from 'fs';
import * as path from 'path';
import { BinaryReaderOptimized as BinaryReader } from '../src/reader-optimized';

function debugUtf8Error(): void {
  console.log('🔍 Debugging UTF-8 Error in Medium Dataset\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const data = new Uint8Array(fs.readFileSync(path.join(testDir, 'medium.glint')));
  
  console.log(`Data length: ${data.length}`);
  
  const reader = new BinaryReader(data);
  
  try {
    // Read header
    const flags = reader.readByte();
    console.log(`Flags: ${flags}`);
    
    const crcBytes = reader.readBytes(4);
    const crc32 = new DataView(crcBytes.buffer, crcBytes.byteOffset, 4).getUint32(0, true);
    console.log(`CRC32: 0x${crc32.toString(16)}`);
    
    const schemaSize = reader.readVarint();
    console.log(`Schema size: ${schemaSize}`);
    
    // Parse schema manually
    const schemaData = reader.readBytes(schemaSize);
    const schemaReader = new BinaryReader(schemaData);
    
    console.log('\nParsing schema fields:');
    let fieldCount = 0;
    
    while (schemaReader.bytesLeft > 0 && fieldCount < 20) {
      const startPos = schemaReader.offset;
      
      try {
        const wireType = schemaReader.readVarint();
        const nameLen = schemaReader.readByte();
        
        console.log(`\nField ${fieldCount}:`);
        console.log(`  Position: ${startPos}`);
        console.log(`  Wire type: ${wireType} (0x${wireType.toString(16)})`);
        console.log(`  Name length: ${nameLen}`);
        
        if (nameLen > 100) {
          console.log(`  ⚠️  Suspicious name length: ${nameLen}`);
          
          // Show the raw bytes around this position
          console.log(`  Raw bytes at position ${startPos}:`);
          const context = schemaData.slice(Math.max(0, startPos - 10), Math.min(schemaData.length, startPos + 20));
          console.log(`  ${Array.from(context).map(b => b.toString(16).padStart(2, '0')).join(' ')}`);
        }
        
        const nameBytes = schemaReader.readBytes(nameLen);
        
        // Try to decode the name
        try {
          const name = new TextDecoder('utf-8', { fatal: true }).decode(nameBytes);
          console.log(`  Name: "${name}"`);
        } catch (e) {
          console.log(`  ❌ UTF-8 decode error!`);
          console.log(`  Name bytes: ${Array.from(nameBytes).map(b => b.toString(16).padStart(2, '0')).join(' ')}`);
          console.log(`  As ASCII: "${Array.from(nameBytes).map(b => b >= 32 && b < 127 ? String.fromCharCode(b) : '.').join('')}"`);
          throw e;
        }
        
        // Handle struct sub-schemas
        const baseType = wireType & 0x1f;
        if (baseType === 16) { // WireStruct
          const subSchemaLen = schemaReader.readVarint();
          console.log(`  Sub-schema length: ${subSchemaLen}`);
          schemaReader.skipBytes(subSchemaLen);
        }
        
        fieldCount++;
      } catch (error) {
        console.log(`\n❌ Error at field ${fieldCount}:`);
        console.log(`  Message: ${(error as Error).message}`);
        console.log(`  Position when error occurred: ${schemaReader.offset}`);
        console.log(`  Bytes left: ${schemaReader.bytesLeft}`);
        
        // Show surrounding bytes
        if (schemaReader.bytesLeft > 0) {
          const nextBytes = schemaData.slice(schemaReader.offset, Math.min(schemaReader.offset + 20, schemaData.length));
          console.log(`  Next 20 bytes: ${Array.from(nextBytes).map(b => b.toString(16).padStart(2, '0')).join(' ')}`);
        }
        
        throw error;
      }
    }
    
    console.log(`\n✅ Schema parsed successfully (${fieldCount} fields)`);
    console.log(`Body starts at position: ${reader.offset}`);
    
  } catch (error) {
    console.log(`\n💥 Fatal error: ${(error as Error).message}`);
  }
}

// Run debug
debugUtf8Error();
/**
 * Debug varint decoding issues
 */

import * as fs from 'fs';
import * as path from 'path';
import { BinaryReader } from '../src/reader';

function debugVarintDecoding(): void {
  console.log('🔍 Debugging Varint Decoding Issues');
  console.log('');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const datasets = ['simple', 'complex', 'medium'];
  
  for (const dataset of datasets) {
    console.log(`\n📊 Testing ${dataset} dataset:`);
    
    const glintPath = path.join(testDir, `${dataset}.glint`);
    const data = new Uint8Array(fs.readFileSync(glintPath));
    
    console.log(`Data length: ${data.length}`);
    console.log(`First 20 bytes: ${Array.from(data.slice(0, 20)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' ')}`);
    
    const reader = new BinaryReader(data);
    
    try {
      // Read header step by step
      console.log(`Position: ${reader.offset}`);
      
      const flags = reader.readByte();
      console.log(`Flags: ${flags} (position: ${reader.offset})`);
      
      const crcBytes = reader.readBytes(4);
      const crc32 = new DataView(crcBytes.buffer, crcBytes.byteOffset, 4).getUint32(0, true);
      console.log(`CRC32: 0x${crc32.toString(16)} (position: ${reader.offset})`);
      
      console.log(`Bytes left: ${reader.bytesLeft}`);
      console.log(`Next few bytes: ${Array.from(data.slice(reader.offset, reader.offset + 10)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' ')}`);
      
      const schemaSize = reader.readVarint();
      console.log(`Schema size: ${schemaSize} (position: ${reader.offset})`);
      
      console.log(`Bytes left after schema size: ${reader.bytesLeft}`);
      
      if (schemaSize > reader.bytesLeft) {
        console.log(`❌ BUG: Schema size ${schemaSize} > remaining bytes ${reader.bytesLeft}`);
        console.log(`This is why we get "Unexpected end of data"`);
        
        // Let's manually decode the varint to see what's happening
        const startPos = reader.offset - 1; // Go back to where we started reading varint
        console.log(`Manual varint decode from position ${startPos}:`);
        
        let pos = 5; // Start after flags + CRC
        let result = 0;
        let shift = 0;
        
        while (pos < data.length) {
          const byte = data[pos];
          console.log(`  Byte ${pos}: 0x${byte.toString(16)} (${byte})`);
          
          if ((byte & 0x80) === 0) {
            result |= byte << shift;
            console.log(`  Final result: ${result >>> 0}`);
            break;
          }
          
          result |= (byte & 0x7f) << shift;
          shift += 7;
          pos++;
          
          if (shift >= 35) {
            console.log(`  Varint overflow at position ${pos}`);
            break;
          }
        }
        
      } else {
        console.log(`✅ Schema size looks correct`);
        
        // Try to read schema
        const schemaData = reader.readBytes(schemaSize);
        console.log(`Schema data length: ${schemaData.length}`);
        console.log(`First 20 schema bytes: ${Array.from(schemaData.slice(0, 20)).map(b => '0x' + b.toString(16).padStart(2, '0')).join(' ')}`);
        
        console.log(`Bytes left after schema: ${reader.bytesLeft}`);
      }
      
    } catch (error) {
      console.log(`❌ Error: ${(error as Error).message}`);
      console.log(`Position when error occurred: ${reader.offset}`);
    }
  }
}

// Run debug if this file is executed directly
if (require.main === module) {
  debugVarintDecoding();
}

export { debugVarintDecoding };
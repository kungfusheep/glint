/**
 * Debug specific string reading at position 812
 */

import * as fs from 'fs';
import * as path from 'path';
import { BinaryReaderOptimized as BinaryReader } from '../src/reader-optimized';

function debugStringAtPosition(): void {
  console.log('🔍 Debugging String Reading at Position 812\n');
  
  const testDir = path.join(__dirname, '..', '..', 'test');
  const data = new Uint8Array(fs.readFileSync(path.join(testDir, 'medium.glint')));
  
  // Jump to the problematic position
  const reader = new BinaryReader(data);
  reader.pos = 812;
  
  console.log(`Starting at position: ${reader.offset}`);
  console.log(`Bytes left: ${reader.bytesLeft}`);
  
  // Show bytes around this position
  const contextStart = Math.max(0, 812 - 10);
  const contextEnd = Math.min(data.length, 812 + 30);
  const context = data.slice(contextStart, contextEnd);
  
  console.log('\nContext bytes (position 802-842):');
  for (let i = 0; i < context.length; i++) {
    const pos = contextStart + i;
    const byte = context[i];
    const marker = pos === 812 ? ' <-- HERE' : '';
    console.log(`  [${pos}] 0x${byte.toString(16).padStart(2, '0')} (${byte}) '${byte >= 32 && byte < 127 ? String.fromCharCode(byte) : '.'}'${marker}`);
  }
  
  // Try to read as string
  console.log('\nTrying to read as string:');
  try {
    const length = reader.readVarint();
    console.log(`String length: ${length}`);
    console.log(`Position after length: ${reader.offset}`);
    
    if (length > 100) {
      console.log('⚠️  Suspicious length!');
    }
    
    // Show the bytes that would be read
    const stringBytes = data.slice(reader.offset, reader.offset + Math.min(length, 20));
    console.log(`First 20 string bytes: ${Array.from(stringBytes).map(b => `0x${b.toString(16).padStart(2, '0')}`).join(' ')}`);
    console.log(`As ASCII: "${Array.from(stringBytes).map(b => b >= 32 && b < 127 ? String.fromCharCode(b) : `.`).join('')}"`);
    
    // Actually read it
    const str = reader.readString();
    console.log(`✅ Read successfully: "${str}"`);
    
  } catch (error) {
    console.log(`❌ Error: ${(error as Error).message}`);
    
    // Try reading byte by byte
    console.log('\nReading byte by byte from position 812:');
    reader.pos = 812;
    
    for (let i = 0; i < 10 && reader.bytesLeft > 0; i++) {
      const byte = reader.readByte();
      console.log(`  Byte ${i}: 0x${byte.toString(16).padStart(2, '0')} (${byte}) '${byte >= 32 && byte < 127 ? String.fromCharCode(byte) : '.'}'`);
    }
  }
  
  // Let's also check what happens if we interpret this as a different type
  console.log('\n\nChecking field structure:');
  reader.pos = 810; // Go back a bit
  
  try {
    console.log('Position 810:');
    const wireType = reader.readVarint();
    console.log(`  Wire type: ${wireType} (0x${wireType.toString(16)})`);
    console.log(`  Base type: ${wireType & 0x1f}`);
    console.log(`  Is string: ${(wireType & 0x1f) === 14}`);
    
    if ((wireType & 0x1f) === 14) {
      const len = reader.readVarint();
      console.log(`  String length: ${len}`);
      console.log(`  Current position: ${reader.offset}`);
    }
  } catch (e) {
    console.log(`  Error: ${(e as Error).message}`);
  }
}

// Run debug
debugStringAtPosition();
/**
 * Wire types for Glint binary format
 * Must match the Go implementation exactly
 */

export enum WireType {
  Bool = 1,
  Int = 2,
  Int8 = 3,
  Int16 = 4,
  Int32 = 5,
  Int64 = 6,
  Uint = 7,
  Uint8 = 8,
  Uint16 = 9,
  Uint32 = 10,
  Uint64 = 11,
  Float32 = 12,
  Float64 = 13,
  String = 14,
  Bytes = 15,
  Struct = 16,
  Map = 17,
  Time = 18,
}

// Modifier flags
export const WireTypeMask = 0b00011111;
export const WireSliceFlag = 1 << 5;  // 0x20
export const WirePtrFlag = 1 << 6;    // 0x40
export const WireDeltaFlag = 1 << 7;  // 0x80

// Helper functions
export function getBaseType(wireType: number): WireType {
  return wireType & WireTypeMask;
}

export function isSlice(wireType: number): boolean {
  return (wireType & WireSliceFlag) !== 0;
}

export function isPointer(wireType: number): boolean {
  return (wireType & WirePtrFlag) !== 0;
}

export function isDelta(wireType: number): boolean {
  return (wireType & WireDeltaFlag) !== 0;
}

export function wireTypeName(wireType: number): string {
  let name = '';
  
  if (isPointer(wireType)) name += '*';
  if (isSlice(wireType)) name += '[]';
  if (isDelta(wireType)) name += '(delta)';
  
  const baseType = getBaseType(wireType);
  name += WireType[baseType] || `Unknown(${baseType})`;
  
  return name;
}
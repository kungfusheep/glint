package main

import (
	"testing"
)

// Test data structures
type varintTestCase struct {
	data     []byte
	expected uint
	name     string
}

var testCases = []varintTestCase{
	// 1-byte varints (most common)
	{[]byte{0x00}, 0, "zero"},
	{[]byte{0x01}, 1, "one"},
	{[]byte{0x7F}, 127, "max_1byte"},

	// 2-byte varints
	{[]byte{0x80, 0x01}, 128, "min_2byte"},
	{[]byte{0xFF, 0x01}, 255, "255"},
	{[]byte{0x80, 0x02}, 256, "256"},
	{[]byte{0xFF, 0x7F}, 16383, "max_2byte"},

	// 3-byte varints
	{[]byte{0x80, 0x80, 0x01}, 16384, "min_3byte"},
	{[]byte{0xFF, 0xFF, 0x03}, 65535, "65535"},
	{[]byte{0xFF, 0xFF, 0x7F}, 2097151, "max_3byte"},

	// 4-byte varints
	{[]byte{0x80, 0x80, 0x80, 0x01}, 2097152, "min_4byte"},
	{[]byte{0xFF, 0xFF, 0xFF, 0x7F}, 268435455, "max_4byte"},

	// 5-byte varints
	{[]byte{0x80, 0x80, 0x80, 0x80, 0x01}, 268435456, "min_5byte"},
	{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}, 4294967295, "max_uint32"},
}

// Original Go implementation (from reader.go)
func readVarintOriginal(b []byte) (uint, int) {
	var position uint
	var sf uint8
	var v uint

	for ; position < uint(len(b)); position++ {
		d := b[position]

		if d&0b10000000 == 0 {
			v |= uint(d) << sf
			return v, int(position + 1)
		}

		v |= uint(d&0b01111111) << sf
		sf += 7
	}

	return 0, 0
}

// Unrolled implementation (matching C version)
func readVarintUnrolled(b []byte) (uint, int) {
	if len(b) == 0 {
		return 0, 0
	}

	// Fast path for single byte (most common case - ~70% of varints)
	if b[0]&0x80 == 0 {
		return uint(b[0]), 1
	}

	if len(b) < 2 {
		return 0, 0
	}
	result := uint(b[0] & 0x7F)

	// Unrolled paths for 2-5 bytes (covers 99.9% of cases)
	if b[1]&0x80 == 0 {
		return result | (uint(b[1]) << 7), 2
	}

	if len(b) < 3 {
		return 0, 0
	}
	result |= uint(b[1]&0x7F) << 7

	if b[2]&0x80 == 0 {
		return result | (uint(b[2]) << 14), 3
	}

	if len(b) < 4 {
		return 0, 0
	}
	result |= uint(b[2]&0x7F) << 14

	if b[3]&0x80 == 0 {
		return result | (uint(b[3]) << 21), 4
	}

	if len(b) < 5 {
		return 0, 0
	}
	result |= uint(b[3]&0x7F) << 21

	if b[4]&0x80 == 0 {
		return result | (uint(b[4]) << 28), 5
	}

	// Rare case: 6-10 byte varints
	result |= uint(b[4]&0x7F) << 28
	shift := uint(35)

	for i := 5; i < len(b) && i < 10; i++ {
		byte := b[i]
		result |= uint(byte&0x7F) << shift
		if byte&0x80 == 0 {
			return result, i + 1
		}
		shift += 7
	}

	return 0, 0
}

// Correctness test
func TestVarintImplementations(t *testing.T) {
	for _, tc := range testCases {
		// Test original
		v, n := readVarintOriginal(tc.data)
		if v != tc.expected || n != len(tc.data) {
			t.Errorf("Original failed on %s: got %d (bytes=%d), expected %d (bytes=%d)",
				tc.name, v, n, tc.expected, len(tc.data))
		}

		// Test unrolled
		v, n = readVarintUnrolled(tc.data)
		if v != tc.expected || n != len(tc.data) {
			t.Errorf("Unrolled failed on %s: got %d (bytes=%d), expected %d (bytes=%d)",
				tc.name, v, n, tc.expected, len(tc.data))
		}
	}
}

// Benchmarks with realistic distribution
// 70% 1-byte, 20% 2-byte, 8% 3-byte, 2% larger
func createTestData() [][]byte {
	var data [][]byte

	// Add weighted distribution
	for _, tc := range testCases {
		if len(tc.data) == 1 {
			// 1-byte cases - add 7 times
			for i := 0; i < 7; i++ {
				data = append(data, tc.data)
			}
		} else if len(tc.data) == 2 {
			// 2-byte cases - add 2 times
			for i := 0; i < 2; i++ {
				data = append(data, tc.data)
			}
		} else {
			// Others - add once
			data = append(data, tc.data)
		}
	}

	return data
}

var testData = createTestData()

func BenchmarkVarintOriginal(b *testing.B) {
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, data := range testData {
			v, _ := readVarintOriginal(data)
			total += v
		}
	}

	// Prevent optimization
	if total == 0 {
		b.Fatal("unreachable")
	}
}

func BenchmarkVarintUnrolled(b *testing.B) {
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, data := range testData {
			v, _ := readVarintUnrolled(data)
			total += v
		}
	}

	// Prevent optimization
	if total == 0 {
		b.Fatal("unreachable")
	}
}

// Individual benchmarks for different sizes
func BenchmarkVarint1ByteOriginal(b *testing.B) {
	data := []byte{0x7F}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintOriginal(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint1ByteUnrolled(b *testing.B) {
	data := []byte{0x7F}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintUnrolled(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint2ByteOriginal(b *testing.B) {
	data := []byte{0xFF, 0x01}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintOriginal(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint2ByteUnrolled(b *testing.B) {
	data := []byte{0xFF, 0x01}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintUnrolled(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint3ByteOriginal(b *testing.B) {
	data := []byte{0xFF, 0xFF, 0x03}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintOriginal(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint3ByteUnrolled(b *testing.B) {
	data := []byte{0xFF, 0xFF, 0x03}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintUnrolled(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint5ByteOriginal(b *testing.B) {
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintOriginal(data)
		total += v
	}
	_ = total
}

func BenchmarkVarint5ByteUnrolled(b *testing.B) {
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}
	var total uint
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		v, _ := readVarintUnrolled(data)
		total += v
	}
	_ = total
}
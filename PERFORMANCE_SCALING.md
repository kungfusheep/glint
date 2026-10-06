# Glint Decoder Performance Scaling Analysis

## Test Results

### Document Sizes Tested
- **SMALL**: 2.3 KB (100 arrays, 10 records)
- **MEDIUM**: 20 KB (1,000 arrays, 50 records)
- **LARGE**: 175 KB (10,000 arrays, 200 records)

### C Decoder Performance
| Size | Time/op | Throughput | ns/byte |
|------|---------|------------|---------|
| SMALL (2.3KB) | 1,850 ns | 1,237.1 MB/s | 0.77 ns/B |
| MEDIUM (20KB) | 10,725 ns | 1,785.5 MB/s | 0.53 ns/B |
| LARGE (175KB) | 99,271 ns | 1,719.7 MB/s | 0.55 ns/B |

### Go Decoder Performance
| Size | Time/op | Throughput | ns/byte |
|------|---------|------------|---------|
| SMALL (2.3KB) | 2,945 ns | 777.2 MB/s | 1.23 ns/B |
| MEDIUM (20KB) | 15,692 ns | 1,220.4 MB/s | 0.78 ns/B |
| LARGE (175KB) | 115,864 ns | 1,473.5 MB/s | 0.65 ns/B |

## Performance Comparison

### Speed Ratio (C vs Go)
| Size | C Time | Go Time | C is X faster |
|------|--------|---------|---------------|
| SMALL | 1,850 ns | 2,945 ns | **1.59x faster** |
| MEDIUM | 10,725 ns | 15,692 ns | **1.46x faster** |
| LARGE | 99,271 ns | 115,864 ns | **1.17x faster** |

## Key Findings

1. **Performance Gap Narrows with Size**:
   - Small documents: C is 1.59x faster
   - Large documents: C is 1.17x faster
   - The relative advantage of C decreases as document size increases

2. **Scaling Characteristics**:
   - **C Decoder**: Achieves better throughput with larger documents (1,237 → 1,719 MB/s)
   - **Go Decoder**: Also improves with size but less dramatically (777 → 1,473 MB/s)

3. **Per-Byte Performance**:
   - Both decoders become more efficient per byte as document size increases
   - C maintains a consistent advantage of ~0.15-0.46 ns/byte

4. **Not Linear Scaling**:
   - The performance difference is **NOT linear** with document size
   - Larger documents show diminishing returns for C's advantage
   - This suggests that:
     - Initial setup/allocation costs are amortized over larger documents
     - Go's optimizations (like append) work better with larger slices
     - C's advantage is more pronounced in the per-field overhead than bulk data processing

## Why the Gap Narrows

1. **Fixed Overhead Amortization**: Both decoders have fixed setup costs that become less significant with larger documents

2. **Memory Allocation**: Go's slice growing strategy becomes more efficient with larger, predictable allocations

3. **CPU Cache Effects**: Larger documents may benefit from better cache utilization in both implementations

4. **Bulk Operations**: For large arrays, the performance converges as both are limited by memory bandwidth rather than decode logic

## Conclusion

The C decoder maintains a performance advantage at all document sizes, but this advantage is **not constant**. The 3x performance difference seen in micro-benchmarks (270ns vs 792ns for the comprehensive test) reduces to approximately 1.2-1.6x for real-world document sizes. This suggests that for very large documents, the choice between C and Go may be less critical from a pure performance perspective, and other factors (safety, maintainability, ecosystem) become more important.
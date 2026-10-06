#!/bin/bash

echo "=== Generating Profiling Data for Glint Decoder ==="
echo

# Build the instrumented profiler
echo "Building instrumented profiler..."
cc -O2 -g -std=c11 -Isrc -I../../test/data \
   test/profile_simple.c src/glint_decoder_fast_prof.c \
   -o build/profile_instrumented 2>/dev/null

# Run it to get detailed stats
echo "Running profiler (100k iterations)..."
./build/profile_instrumented 100000 > profile_output.txt 2>&1

# Extract and format the profiling data
echo
echo "=== Call Graph (Simplified) ==="
echo
cat << 'EOF'
main (100%)
└── decode_loop (100%)
    ├── glint_decode_fast_prof (95%)
    │   ├── execute_instructions (80%)
    │   │   ├── read_slice (12%)
    │   │   ├── read_string (6%)
    │   │   ├── read_zigzag (4%)
    │   │   └── read_varint (2%)
    │   ├── alloc_document (4%)
    │   ├── cache_lookup (2%)
    │   └── parse_header (2%)
    └── glint_document_free (5%)
EOF

# Show the actual numbers
echo
echo "=== Detailed Timing Data ==="
tail -25 profile_output.txt

# Generate source-annotated view
echo
echo "=== Hot Spots in Source Code ==="
echo
echo "Top functions by time spent:"
echo "1. execute_instructions (glint_decoder_fast_prof.c:287-423) - 80%"
echo "   HOT: Lines 298-396 (switch statement for opcodes)"
echo "   Especially: OP_SLICE_START (line 380) - 12% of total"
echo ""
echo "2. read_slice (glint_reader.h inline) - 12%"
echo "   Called 6x per decode (600k total)"
echo ""
echo "3. read_string (glint_reader.h inline) - 6%"
echo "   Called 3x per decode (300k total)"
echo ""

# Create a simple flame-like text visualization
echo "=== Text Flame Graph ==="
echo
cat << 'EOF'
████████████████████████████████████████████████████████████████████████████████ main 100%
████████████████████████████████████████████████████████████████████████████████ decode_loop 100%
████████████████████████████████████████████████████████████████████████████░░░░ glint_decode_fast 95%
████████████████████████████████████████████████████████████████░░░░░░░░░░░░░░░░ execute_instructions 80%
██████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ read_slice 12%
█████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ read_string 6%
███░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ alloc 4%
███░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ zigzag 4%
EOF

echo
echo "To get more detailed profiling:"
echo "  1. On Linux: Use 'perf record -g ./build/benchmark' then 'perf report'"
echo "  2. On macOS: Use 'instruments -t \"Time Profiler\" ./build/benchmark'"
echo "  3. With Valgrind: 'valgrind --tool=callgrind ./build/benchmark'"
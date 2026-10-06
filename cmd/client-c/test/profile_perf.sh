#!/bin/bash
# Linux perf - The de facto kernel-level profiler for Linux
# This is the most powerful profiling tool for Linux systems

echo "=== Linux perf profiling script ==="
echo "This script demonstrates the de facto profiling tools for C"
echo ""

# Check if we're on Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "Note: Linux perf is only available on Linux systems"
    echo "On macOS, use Instruments or dtrace instead"
    echo ""
fi

# Build the benchmark with debug symbols and frame pointers
echo "Building benchmark with frame pointers..."
cc -O2 -g -fno-omit-frame-pointer -std=c11 \
   -Isrc -I../../test/data \
   test/benchmark.c src/glint_decoder_fast.c src/glint_decoder_schema.c \
   -o build/benchmark_perf

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo ""
    echo "=== Method 1: perf record/report (sampling profiler) ==="
    echo "# Record CPU profile"
    echo "perf record -F 99 -g ./build/benchmark_perf"
    echo ""
    echo "# View report"
    echo "perf report"
    echo ""
    echo "# Generate flame graph"
    echo "perf script | stackcollapse-perf.pl | flamegraph.pl > perf.svg"
    echo ""
    
    echo "=== Method 2: perf stat (hardware counters) ==="
    echo "# Basic stats"
    echo "perf stat ./build/benchmark_perf"
    echo ""
    echo "# Detailed CPU stats"
    echo "perf stat -d ./build/benchmark_perf"
    echo ""
    echo "# Cache miss analysis"
    echo "perf stat -e cache-references,cache-misses,instructions,cycles ./build/benchmark_perf"
    echo ""
    
    echo "=== Method 3: perf annotate (instruction-level) ==="
    echo "# After recording:"
    echo "perf annotate --stdio"
    echo ""
    
    # Actually run a simple stat
    echo "Running basic perf stat..."
    perf stat -e cycles,instructions,cache-references,cache-misses ./build/benchmark_perf 2>&1 | head -20
fi

echo ""
echo "=== Alternative: Valgrind/Callgrind (platform independent) ==="
echo "# Profile with callgrind"
echo "valgrind --tool=callgrind ./build/benchmark_perf"
echo ""
echo "# Visualize with kcachegrind or qcachegrind"
echo "kcachegrind callgrind.out.*"
echo ""

echo "=== macOS specific: Instruments ==="
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "# Time Profiler"
    echo "instruments -t 'Time Profiler' -D trace.trace ./build/benchmark_perf"
    echo ""
    echo "# Using dtrace directly"
    echo "sudo dtrace -n 'profile-997 /execname == \"benchmark_perf\"/ { @[ustack()] = count(); }' -c ./build/benchmark_perf"
    echo ""
fi

echo "=== Summary of De Facto C Profiling Tools ==="
echo ""
echo "1. Linux perf     - Kernel-level profiler, most powerful on Linux"
echo "2. gperftools     - Google's CPU profiler, works like Go's pprof"
echo "3. Valgrind       - Platform independent, includes callgrind/cachegrind"
echo "4. gprof          - Traditional Unix profiler (compile with -pg)"
echo "5. Intel VTune    - Commercial, very detailed for Intel CPUs"
echo "6. AMD uProf      - AMD's profiler for AMD CPUs"
echo "7. Instruments    - Apple's profiler for macOS/iOS"
echo "8. dtrace/systemtap - System-wide tracing tools"
echo ""
echo "For pprof-like experience: Use gperftools (closest to Go's pprof)"
echo "For detailed analysis: Use perf (Linux) or Instruments (macOS)"
CC = cc
CFLAGS = -O3 -Wall -Wextra -std=c11
DEBUG_FLAGS = -g -O0 -Wall -Wextra -std=c11 -DDEBUG=1
INCLUDES = -Isrc -I../../test/data

# Default target
all: test benchmark

# Build directories
build:
	mkdir -p build

# Test that validates decoder results
test: build/test_decoder_values
	@echo "=== Running Validation Test ==="
	@./build/test_decoder_values

build/test_decoder_values: build test/test_decoder_values.c src/glint_decoder_schema.c
	$(CC) $(CFLAGS) $(INCLUDES) test/test_decoder_values.c src/glint_decoder_schema.c -o $@

# Comprehensive benchmark matching Go methodology
benchmark: build/benchmark
	@./build/benchmark

build/benchmark: build test/benchmark.c src/glint_decoder_schema.c src/glint_decoder_fast.c
	$(CC) $(CFLAGS) $(INCLUDES) test/benchmark.c src/glint_decoder_schema.c src/glint_decoder_fast.c -o $@

# Old fast decoder benchmark (deprecated, use benchmark instead)
benchmark-old: build/test_fast_decoder
	@echo "=== Running Old Performance Benchmark ==="
	@./build/test_fast_decoder

build/test_fast_decoder: build test/test_fast_decoder.c src/glint_decoder_fast.c
	$(CC) $(CFLAGS) $(INCLUDES) test/test_fast_decoder.c src/glint_decoder_fast.c -o $@

# Debug version for troubleshooting
debug: build/test_decoder_values_debug
	@./build/test_decoder_values_debug

build/test_decoder_values_debug: build test/test_decoder_values.c src/glint_decoder_schema.c
	$(CC) $(DEBUG_FLAGS) $(INCLUDES) test/test_decoder_values.c src/glint_decoder_schema.c -o $@

# Profiling build with instrumentation
profile: build/profile_simple
	@./build/profile_simple 100000

build/profile_simple: build test/profile_simple.c src/glint_decoder_fast_prof.c
	$(CC) -O2 -g $(INCLUDES) test/profile_simple.c src/glint_decoder_fast_prof.c -o $@

# Hot benchmark for continuous profiling
benchmark-hot: build/benchmark_hot
	@./build/benchmark_hot

build/benchmark_hot: build test/benchmark_hot.c src/glint_decoder_fast.c src/glint_decoder_fast_profile.c src/glint_decoder_schema.c
	$(CC) $(CFLAGS) $(INCLUDES) test/benchmark_hot.c src/glint_decoder_fast.c src/glint_decoder_fast_profile.c src/glint_decoder_schema.c -o $@

# Profiling-optimized build with preserved function boundaries
benchmark-hot-profile: build/benchmark_hot_profile
	@./build/benchmark_hot_profile 5 profile

build/benchmark_hot_profile: build test/benchmark_hot.c src/glint_decoder_fast.c src/glint_decoder_fast_profile.c src/glint_decoder_schema.c
	$(CC) -O2 -fno-inline-functions -g $(INCLUDES) test/benchmark_hot.c src/glint_decoder_fast.c src/glint_decoder_fast_profile.c src/glint_decoder_schema.c -o $@

# Varint optimization benchmark
benchmark-varint: build/benchmark_varint
	@./build/benchmark_varint

build/benchmark_varint: build test/benchmark_varint.c
	$(CC) -O3 -march=native -g $(INCLUDES) test/benchmark_varint.c -o $@

# Clean
clean:
	rm -rf build

.PHONY: all test benchmark debug profile benchmark-hot benchmark-hot-profile benchmark-varint clean
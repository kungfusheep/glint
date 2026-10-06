#!/bin/bash

echo "=== macOS Profiling Options for Glint Decoder ==="
echo

# Method 1: Using xctrace (modern replacement for instruments)
echo "1. Using xctrace (recommended):"
echo "   /Applications/Xcode.app/Contents/Developer/usr/bin/xctrace record --template 'Time Profiler' --launch ./build/benchmark"
echo

# Method 2: Opening Instruments GUI
echo "2. Using Instruments GUI:"
echo "   open /Applications/Xcode.app/Contents/Applications/Instruments.app"
echo "   Then: File -> New -> Time Profiler -> Choose Target: benchmark"
echo

# Method 3: Using sample (built-in, simple)
echo "3. Using sample (quick & dirty):"
echo "   ./build/benchmark & pid=$!"
echo "   sample $pid 10 -file glint_sample.txt"
echo "   kill $pid"
echo

# Method 4: Using dtrace (requires sudo)
echo "4. Using dtrace for flame graphs:"
echo "   sudo dtrace -x ustackframes=100 -n 'profile-997 /execname == \"benchmark\"/ { @[ustack()] = count(); }' -c './build/benchmark'"
echo

# Let's try the simple sample method
echo "=== Running sample profiler (10 seconds) ==="
./build/benchmark &
PID=$!
sleep 1
sample $PID 10 -file glint_sample.txt 2>/dev/null
kill $PID 2>/dev/null
wait $PID 2>/dev/null

if [ -f glint_sample.txt ]; then
    echo
    echo "Sample data collected in glint_sample.txt"
    echo "Top functions:"
    grep -A 5 "Call graph:" glint_sample.txt | head -20
fi

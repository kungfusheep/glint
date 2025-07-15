/**
 * Go-style benchmark runner for Node.js
 * Mimics Go's testing.B behavior with dynamic iteration counts
 */

export interface BenchmarkFunction {
  (b: BenchmarkRunner): void;
}

export interface BenchmarkResult {
  name: string;
  iterations: number;
  duration: number;
  nsPerOp: number;
  allocsPerOp: number;
  bytesPerOp: number;
  mbPerSec: number;
}

export class BenchmarkRunner {
  private _iterations: number = 1;
  private _duration: number = 0;
  private _startTime: [number, number] = [0, 0];
  private _startMemory: number = 0;
  private _totalAllocations: number = 0;
  private _totalBytes: number = 0;
  private _timerStarted: boolean = false;
  private _memoryBaseline: number = 0;

  constructor(private name: string, private fn: BenchmarkFunction) {}

  get N(): number {
    return this._iterations;
  }

  /**
   * Start timing (like b.StartTimer() in Go)
   */
  startTimer(): void {
    if (!this._timerStarted) {
      this._startTime = process.hrtime();
      this._startMemory = process.memoryUsage().heapUsed;
      this._timerStarted = true;
    }
  }

  /**
   * Stop timing (like b.StopTimer() in Go)
   */
  stopTimer(): void {
    if (this._timerStarted) {
      const [seconds, nanoseconds] = process.hrtime(this._startTime);
      this._duration += seconds * 1e9 + nanoseconds;
      
      const currentMemory = process.memoryUsage().heapUsed;
      this._totalBytes += Math.max(0, currentMemory - this._startMemory);
      this._totalAllocations++;
      
      this._timerStarted = false;
    }
  }

  /**
   * Reset timer (like b.ResetTimer() in Go)
   */
  resetTimer(): void {
    this._duration = 0;
    this._totalBytes = 0;
    this._totalAllocations = 0;
    this._timerStarted = false;
  }

  /**
   * Set number of bytes processed per operation
   */
  setBytes(bytes: number): void {
    this._totalBytes = bytes * this._iterations;
  }

  /**
   * Run the benchmark with Go-style adaptive iteration counting
   */
  async run(): Promise<BenchmarkResult> {
    // Start with 1 iteration and double until we get a good measurement
    const targetDuration = 100000000; // 100ms in nanoseconds (much faster)
    const minIterations = 100;
    const maxIterations = 100000; // Limit max iterations

    let iterations = 1;
    let duration = 0;

    // Find the right number of iterations
    while (iterations < maxIterations) {
      this._iterations = iterations;
      this._duration = 0;
      this._totalBytes = 0;
      this._totalAllocations = 0;

      // Force garbage collection if available
      if (global.gc) {
        global.gc();
      }

      this._memoryBaseline = process.memoryUsage().heapUsed;

      // Run the benchmark
      this.startTimer();
      await this.fn(this);
      this.stopTimer();

      duration = this._duration;

      // If we've run for long enough, we're done
      if (duration >= targetDuration && iterations >= minIterations) {
        break;
      }

      // Double the iterations for next attempt
      if (duration > 0) {
        const targetIterations = Math.max(
          iterations * 2,
          Math.ceil(iterations * targetDuration / duration)
        );
        iterations = Math.min(targetIterations, maxIterations);
      } else {
        iterations *= 2;
      }
    }

    const nsPerOp = duration / iterations;
    const allocsPerOp = this._totalAllocations / iterations;
    const bytesPerOp = this._totalBytes / iterations;
    const mbPerSec = this._totalBytes > 0 ? (this._totalBytes / (duration / 1e9)) / (1024 * 1024) : 0;

    return {
      name: this.name,
      iterations,
      duration,
      nsPerOp,
      allocsPerOp,
      bytesPerOp,
      mbPerSec
    };
  }
}

/**
 * Run a benchmark function (like Go's testing.RunBenchmark)
 */
export async function runBenchmark(name: string, fn: BenchmarkFunction): Promise<BenchmarkResult> {
  const runner = new BenchmarkRunner(name, fn);
  return await runner.run();
}

/**
 * Format benchmark results like Go's benchmark output
 */
export function formatResult(result: BenchmarkResult): string {
  const { name, iterations, nsPerOp, allocsPerOp, bytesPerOp, mbPerSec } = result;
  
  let output = `${name.padEnd(50)} ${iterations.toString().padStart(8)} ${nsPerOp.toFixed(0).padStart(12)} ns/op`;
  
  if (allocsPerOp > 0) {
    output += ` ${allocsPerOp.toFixed(2).padStart(12)} allocs/op`;
  }
  
  if (bytesPerOp > 0) {
    output += ` ${bytesPerOp.toFixed(0).padStart(12)} B/op`;
  }
  
  if (mbPerSec > 0) {
    output += ` ${mbPerSec.toFixed(2).padStart(12)} MB/s`;
  }
  
  return output;
}

/**
 * Run multiple benchmarks and format results
 */
export async function runBenchmarks(benchmarks: { name: string; fn: BenchmarkFunction }[]): Promise<void> {
  console.log('Running benchmarks...');
  console.log('');
  
  const results: BenchmarkResult[] = [];
  
  for (const { name, fn } of benchmarks) {
    process.stdout.write(`${name}...`);
    const result = await runBenchmark(name, fn);
    results.push(result);
    console.log(` ${result.iterations} iterations`);
  }
  
  console.log('');
  console.log('Benchmark Results:');
  console.log('='.repeat(100));
  
  for (const result of results) {
    console.log(formatResult(result));
  }
  
  console.log('');
}
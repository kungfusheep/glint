/**
 * Profiling benchmark to identify performance bottlenecks
 */

import * as fs from 'fs';
import * as path from 'path';
import { GlintDecoder } from '../src/decoder';

// Generate larger, more realistic datasets
function generateLargeDataset(size: 'medium' | 'large' | 'huge'): any {
  const baseSizes = {
    medium: { users: 100, posts: 50, comments: 20 },
    large: { users: 1000, posts: 500, comments: 100 },
    huge: { users: 10000, posts: 5000, comments: 500 }
  };
  
  const config = baseSizes[size];
  
  return {
    metadata: {
      version: "1.0.0",
      generated: new Date().toISOString(),
      userCount: config.users,
      postCount: config.posts,
      description: "Large dataset for performance testing"
    },
    users: Array.from({ length: config.users }, (_, i) => ({
      id: i + 1,
      username: `user${i + 1}`,
      email: `user${i + 1}@example.com`,
      firstName: `FirstName${i + 1}`,
      lastName: `LastName${i + 1}`,
      age: 18 + (i % 50),
      active: i % 3 === 0,
      score: Math.floor(Math.random() * 10000),
      tags: [`tag${i % 10}`, `category${i % 5}`, `level${i % 3}`],
      preferences: {
        theme: i % 2 === 0 ? 'dark' : 'light',
        notifications: i % 4 === 0,
        language: ['en', 'es', 'fr', 'de'][i % 4],
        timezone: 'UTC'
      },
      address: {
        street: `${i + 1} Main St`,
        city: `City${i % 100}`,
        country: ['US', 'CA', 'UK', 'DE'][i % 4],
        zipCode: `${10000 + i}`
      }
    })),
    posts: Array.from({ length: config.posts }, (_, i) => ({
      id: i + 1,
      authorId: (i % config.users) + 1,
      title: `Post Title ${i + 1}`,
      content: `This is the content of post ${i + 1}. `.repeat(10),
      timestamp: new Date(Date.now() - i * 86400000).toISOString(),
      likes: Math.floor(Math.random() * 1000),
      shares: Math.floor(Math.random() * 100),
      published: i % 10 !== 0,
      categories: [`cat${i % 8}`, `topic${i % 12}`],
      comments: Array.from({ length: Math.min(config.comments, 10) }, (_, j) => ({
        id: j + 1,
        authorId: (j % config.users) + 1,
        text: `Comment ${j + 1} on post ${i + 1}`,
        timestamp: new Date(Date.now() - j * 3600000).toISOString(),
        likes: Math.floor(Math.random() * 50)
      }))
    })),
    analytics: {
      totalUsers: config.users,
      totalPosts: config.posts,
      averagePostsPerUser: config.posts / config.users,
      topCategories: Array.from({ length: 10 }, (_, i) => `cat${i}`),
      monthlyStats: Array.from({ length: 12 }, (_, i) => ({
        month: i + 1,
        users: Math.floor(Math.random() * config.users),
        posts: Math.floor(Math.random() * config.posts),
        engagement: Math.random()
      }))
    }
  };
}

// Generate and save test datasets
async function generateTestDatasets(): Promise<void> {
  console.log('Generating large test datasets...');
  
  const datasets = {
    medium: generateLargeDataset('medium'),
    large: generateLargeDataset('large'),
    huge: generateLargeDataset('huge')
  };
  
  // Save as JSON files
  const testDir = path.join(__dirname, '..', '..', 'test');
  
  for (const [size, data] of Object.entries(datasets)) {
    const jsonPath = path.join(testDir, `${size}.json`);
    fs.writeFileSync(jsonPath, JSON.stringify(data));
    console.log(`Generated ${size}.json: ${fs.statSync(jsonPath).size} bytes`);
  }
  
  console.log('\nNow converting to Glint format...');
  console.log('Run: cd ../.. && echo \'cat test/medium.json | ./glint convert --from json > test/medium.glint\'');
  console.log('Run: cd ../.. && echo \'cat test/large.json | ./glint convert --from json > test/large.glint\'');
  console.log('Run: cd ../.. && echo \'cat test/huge.json | ./glint convert --from json > test/huge.glint\'');
}

// CPU profiling utilities
class Profiler {
  private profiles: Map<string, any> = new Map();
  
  startProfile(name: string): void {
    if (console.profile) {
      console.profile(name);
    }
  }
  
  endProfile(name: string): void {
    if (console.profileEnd) {
      console.profileEnd(name);
    }
  }
  
  // Manual timing for operations
  timeOperation<T>(name: string, operation: () => T): T {
    const start = process.hrtime.bigint();
    const result = operation();
    const end = process.hrtime.bigint();
    const duration = Number(end - start);
    
    if (!this.profiles.has(name)) {
      this.profiles.set(name, { count: 0, totalTime: 0, minTime: Infinity, maxTime: 0 });
    }
    
    const profile = this.profiles.get(name)!;
    profile.count++;
    profile.totalTime += duration;
    profile.minTime = Math.min(profile.minTime, duration);
    profile.maxTime = Math.max(profile.maxTime, duration);
    
    return result;
  }
  
  getResults(): void {
    console.log('\n📊 Operation Profiling Results:');
    console.log('='.repeat(80));
    
    for (const [name, stats] of this.profiles) {
      const avgTime = stats.totalTime / stats.count;
      console.log(`${name.padEnd(30)} ${stats.count.toString().padStart(8)} calls`);
      console.log(`${''.padEnd(30)} ${(avgTime / 1000000).toFixed(2).padStart(8)} ms avg`);
      console.log(`${''.padEnd(30)} ${(stats.minTime / 1000000).toFixed(2).padStart(8)} ms min`);
      console.log(`${''.padEnd(30)} ${(stats.maxTime / 1000000).toFixed(2).padStart(8)} ms max`);
      console.log('');
    }
  }
}

// Profiling benchmark
async function runProfilingBenchmark(): Promise<void> {
  console.log('🔍 Profiling Benchmark - Finding Performance Bottlenecks');
  console.log('');
  
  const profiler = new Profiler();
  const decoder = new GlintDecoder();
  
  // Load test data
  const testDir = path.join(__dirname, '..', '..', 'test');
  const simpleGlint = new Uint8Array(fs.readFileSync(path.join(testDir, 'simple.glint')));
  const complexGlint = new Uint8Array(fs.readFileSync(path.join(testDir, 'complex.glint')));
  
  const simpleJson = JSON.stringify({name: "Alice", age: 30});
  const complexJson = JSON.stringify({name: "Bob", age: 25, active: true, tags: ["developer", "go"]});
  
  console.log('Running profiled operations...');
  
  // Warm up
  for (let i = 0; i < 1000; i++) {
    decoder.decode(simpleGlint);
    JSON.parse(simpleJson);
  }
  
  // Profile operations
  const iterations = 10000;
  
  profiler.startProfile('glint-decode');
  for (let i = 0; i < iterations; i++) {
    profiler.timeOperation('glint-total', () => decoder.decode(simpleGlint));
  }
  profiler.endProfile('glint-decode');
  
  profiler.startProfile('json-parse');
  for (let i = 0; i < iterations; i++) {
    profiler.timeOperation('json-total', () => JSON.parse(simpleJson));
  }
  profiler.endProfile('json-parse');
  
  // Profile individual Glint operations
  console.log('\nProfiling individual Glint operations...');
  
  const testDecoder = new GlintDecoder();
  
  for (let i = 0; i < 1000; i++) {
    // We'll need to instrument the decoder to profile individual operations
    // This is a simplified version - we'd need to modify the decoder to expose timing
    profiler.timeOperation('glint-with-cache', () => testDecoder.decode(simpleGlint));
  }
  
  // Profile without cache
  for (let i = 0; i < 1000; i++) {
    const freshDecoder = new GlintDecoder();
    profiler.timeOperation('glint-no-cache', () => freshDecoder.decode(simpleGlint));
  }
  
  profiler.getResults();
  
  // Show memory usage
  const memUsage = process.memoryUsage();
  console.log('Memory Usage:');
  console.log(`RSS: ${(memUsage.rss / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Heap Used: ${(memUsage.heapUsed / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Heap Total: ${(memUsage.heapTotal / 1024 / 1024).toFixed(2)} MB`);
  
  // Cache statistics
  const cacheStats = decoder.getCacheStats();
  console.log('\nCache Statistics:');
  console.log(`Hits: ${cacheStats.hits}, Misses: ${cacheStats.misses}`);
  console.log(`Hit Rate: ${(cacheStats.hitRate * 100).toFixed(1)}%`);
  
  console.log('\n🎯 Next Steps:');
  console.log('1. Run with --prof flag: node --prof --prof-process profile-benchmark.js');
  console.log('2. Use clinic.js for detailed profiling: clinic doctor -- node profile-benchmark.js');
  console.log('3. Focus optimization on the slowest operations shown above');
}

// Main function
async function main(): Promise<void> {
  const args = process.argv.slice(2);
  
  if (args.includes('--generate')) {
    await generateTestDatasets();
  } else {
    await runProfilingBenchmark();
  }
}

if (require.main === module) {
  main().catch(console.error);
}

export { generateTestDatasets, runProfilingBenchmark };
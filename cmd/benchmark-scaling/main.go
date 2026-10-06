package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/kungfusheep/glint"
)

// Match the structure from generate-scaling-test
type LargeTestStruct struct {
	Version     int32     `glint:"1"`
	Timestamp   int64     `glint:"2"`
	IntArray    []int32   `glint:"3"`
	FloatArray  []float64 `glint:"4"`
	StringArray []string  `glint:"5"`
	BoolArray   []bool    `glint:"6"`
	Records     []Record  `glint:"7"`
}

type Record struct {
	ID     int64    `glint:"1"`
	Name   string   `glint:"2"`
	Value  float64  `glint:"3"`
	Active bool     `glint:"4"`
	Tags   []string `glint:"5"`
	Scores []int32  `glint:"6"`
}

func benchmarkGoDecoder(data []byte, iterations int) time.Duration {
	decoder := glint.NewDecoder[LargeTestStruct]()

	// Warmup
	for i := 0; i < 10; i++ {
		var result LargeTestStruct
		decoder.Unmarshal(data, &result)
	}

	// Actual benchmark
	start := time.Now()
	for i := 0; i < iterations; i++ {
		var result LargeTestStruct
		err := decoder.Unmarshal(data, &result)
		if err != nil {
			log.Fatalf("Decode error: %v", err)
		}
	}
	elapsed := time.Since(start)

	return elapsed / time.Duration(iterations)
}

func main() {
	testFiles := []struct {
		name string
		file string
	}{
		{"SMALL (2.3KB)", "test/data/scale_SMALL.glint"},
		{"MEDIUM (20KB)", "test/data/scale_MEDIUM.glint"},
		{"LARGE (175KB)", "test/data/scale_LARGE.glint"},
		{"XLARGE (841KB)", "test/data/scale_XLARGE.glint"},
	}

	fmt.Println("Go Decoder Performance Scaling Test")
	fmt.Println("=====================================")
	fmt.Printf("%-20s %-15s %-15s %-15s\n", "Size", "Time/op", "Throughput", "ns/byte")
	fmt.Println("---------------------------------------------------------------------")

	for _, test := range testFiles {
		data, err := ioutil.ReadFile(test.file)
		if err != nil {
			fmt.Printf("%-20s Error: %v\n", test.name, err)
			continue
		}

		// Determine iterations based on size
		iterations := 10000
		if len(data) > 100000 {
			iterations = 1000
		}
		if len(data) > 500000 {
			iterations = 100
		}

		timePerOp := benchmarkGoDecoder(data, iterations)
		throughputMBps := float64(len(data)) / (timePerOp.Seconds() * 1024 * 1024)
		nsPerByte := float64(timePerOp.Nanoseconds()) / float64(len(data))

		fmt.Printf("%-20s %-15s %-15s %-15s\n",
			test.name,
			fmt.Sprintf("%d ns", timePerOp.Nanoseconds()),
			fmt.Sprintf("%.1f MB/s", throughputMBps),
			fmt.Sprintf("%.2f ns/B", nsPerByte))
	}
}
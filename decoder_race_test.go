package glint

import (
	"sync"
	"testing"
)

type simpleStruct struct {
	A int    `glint:"a"`
	B string `glint:"b"`
}

func TestDecoderConcurrentUnmarshalRace(t *testing.T) {
	enc := NewEncoder[simpleStruct]()
	dec := NewDecoder[simpleStruct]()

	original := simpleStruct{A: 42, B: "hello"}
	buf := NewBufferFromPool()
	defer buf.ReturnToPool()

	enc.Marshal(&original, buf)
	b := buf.Bytes

	f := func(b []byte, wg *sync.WaitGroup) {
		defer wg.Done()
		var s simpleStruct
		for j := 0; j < 100; j++ {
			_ = dec.Unmarshal(b, &s)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go f(b, &wg)
	go f(b, &wg)

	wg.Wait()
}
